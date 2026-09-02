// RECONSTRUCCION DE GestionSO V57.
// Este archivo NO es el fuente original. Es una reimplementacion basada en
// simbolos, strings y comportamiento observable del binario V57.
//
// HECHO VERIFICADO: el binario contiene simbolos para las funciones XLSX,
// persistencia, vistas, logging, opciones y simulador documentadas en
// docs/EVIDENCIA_BINARIO.md.
// INFERENCIA: firmas, estructuras internas y algunos detalles de formato que
// no pueden recuperarse literalmente del binario se implementan de forma
// conservadora para mantener una base compilable y trazable.

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Los nombres de estos tipos aparecen entre los simbolos del binario real.
type MasterRow map[string]string

type MasterData struct {
	Headers []string
	Rows    []MasterRow
	Path    string
}

type Line struct {
	Values    map[string]string
	Source    string
	RowNumber int
}

type ColumnDef struct {
	Name   string
	Width  int
	Hidden bool
}

type xlsxDoc struct {
	SharedStrings []string
	Sheets        map[string][][]string
}

// INFERENCIA: el binario tiene LoadConfig/SaveConfig/defaultConfig, pero el
// tipo exacto de configuracion no es recuperable como fuente Go original.
type configData struct {
	MasterPath string
	EnginePath string
	Mode       string
}

// panicGuard es una envoltura defensiva; el simbolo existe en el binario.
func panicGuard(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logf("panicGuard recovered: %v", r)
		}
	}()
	fn()
}

func defaultConfig() configData { return configData{Mode: "FACTURAS"} }

func LoadConfig() configData {
	c := defaultConfig()
	p := filepath.Join(os.TempDir(), "GestionSO-config.txt")
	b, err := os.ReadFile(p)
	if err != nil {
		return c
	}
	for _, ln := range strings.Split(string(b), "\n") {
		kv := strings.SplitN(strings.TrimSpace(ln), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "MasterPath":
			c.MasterPath = kv[1]
		case "EnginePath":
			c.EnginePath = kv[1]
		case "Mode":
			c.Mode = kv[1]
		}
	}
	return c
}

func SaveConfig(c configData) error {
	p := filepath.Join(os.TempDir(), "GestionSO-config.txt")
	return saveCfg(p, c)
}

func saveCfg(path string, c configData) error {
	data := fmt.Sprintf("MasterPath=%s\nEnginePath=%s\nMode=%s\n", c.MasterPath, c.EnginePath, c.Mode)
	return os.WriteFile(path, []byte(data), 0644)
}

// ---------- XLSX: ZIP + XML, sin depender de Excel ----------

func ReadXLSX(path string) (*xlsxDoc, error) { return readXLSXDoc(path) }

func readXLSXDoc(path string) (*xlsxDoc, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer z.Close()

	d := &xlsxDoc{Sheets: map[string][][]string{}}
	if ss, e := readZipEntry(z, "xl/sharedStrings.xml"); e == nil {
		d.SharedStrings = parseSharedStrings(ss)
	}

	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml") {
			b, e := readZipEntry(z, f.Name)
			if e == nil {
				d.Sheets[filepath.Base(f.Name)] = decodeSheet(b, d.SharedStrings)
			}
		}
	}
	return d, nil
}

func readZipEntry(z *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range z.File {
		if f.Name != name {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	}
	return nil, os.ErrNotExist
}

type sharedXML struct {
	SI []struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

func parseSharedStrings(b []byte) []string {
	var x sharedXML
	if xml.Unmarshal(b, &x) != nil {
		return nil
	}
	out := make([]string, len(x.SI))
	for i, s := range x.SI {
		if s.T != "" {
			out[i] = s.T
			continue
		}
		for _, r := range s.R {
			out[i] += r.T
		}
	}
	return out
}

type sheetXML struct {
	Rows []struct {
		Cells []struct {
			Ref      string `xml:"r,attr"`
			Type     string `xml:"t,attr"`
			Value    string `xml:"v"`
			Inline   struct{ T string `xml:"t"` } `xml:"is"`
		} `xml:"c"`
	} `xml:"row"`
}

func decodeSheet(b []byte, ss []string) [][]string {
	var x sheetXML
	if xml.Unmarshal(b, &x) != nil {
		return nil
	}
	return parseRows(x.Rows, ss)
}

func parseRows(rows []struct {
	Cells []struct {
		Ref    string `xml:"r,attr"`
		Type   string `xml:"t,attr"`
		Value  string `xml:"v"`
		Inline struct{ T string `xml:"t"` } `xml:"is"`
	} `xml:"c"`
}, ss []string) [][]string {
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		m := map[int]string{}
		max := 0
		for _, c := range r.Cells {
			col := colFromRef(c.Ref)
			if col < 1 {
				continue
			}
			v := c.Value
			switch c.Type {
			case "s":
				if n, e := strconv.Atoi(v); e == nil && n >= 0 && n < len(ss) {
					v = ss[n]
				}
			case "inlineStr":
				v = c.Inline.T
			}
			m[col] = v
			if col > max {
				max = col
			}
		}
		row := make([]string, max)
		for c, v := range m {
			row[c-1] = v
		}
		out = append(out, row)
	}
	return normalizeRows(out)
}

func normalizeRows(rows [][]string) [][]string {
	for i := range rows {
		for len(rows[i]) > 0 && strings.TrimSpace(rows[i][len(rows[i])-1]) == "" {
			rows[i] = rows[i][:len(rows[i])-1]
		}
	}
	return rows
}

func normalizeRowXML(b []byte) []byte { return bytes.TrimSpace(b) }

func buildMergedSheet(rows [][]string) []byte {
	var b bytes.Buffer
	for i, r := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for j, v := range r {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString(xmlEscape(v))
		}
	}
	return b.Bytes()
}

func rewriteRowNumber(ref string, n int) string {
	if n < 1 {
		return ref
	}
	i := 0
	for i < len(ref) && ((ref[i] >= 'A' && ref[i] <= 'Z') || (ref[i] >= 'a' && ref[i] <= 'z')) {
		i++
	}
	return ref[:i] + strconv.Itoa(n)
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;").Replace(s)
}

func hashRow(r []string) string {
	h := sha256.New()
	for _, v := range r {
		_, _ = h.Write([]byte(v))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func colFromRef(ref string) int {
	ref = strings.ToUpper(ref)
	n := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' {
			break
		}
		n = n*26 + int(r-'A'+1)
	}
	return n
}

func headerScore(row []string) int {
	score := 0
	keys := []string{"factura", "cliente", "fecha", "cuit", "sku", "producto", "cantidad", "importe", "so"}
	for _, v := range row {
		s := strings.ToLower(strings.TrimSpace(v))
		for _, k := range keys {
			if strings.Contains(s, k) {
				score++
				break
			}
		}
	}
	return score
}

func mergeXLSX(paths []string) ([][]string, error) {
	var all [][]string
	for _, p := range paths {
		d, err := ReadXLSX(p)
		if err != nil {
			return nil, err
		}
		var best [][]string
		bestScore := -1
		for _, rows := range d.Sheets {
			score := -1
			if len(rows) > 0 {
				score = headerScore(rows[0])
			}
			if score > bestScore {
				bestScore = score
				best = rows
			}
		}
		if len(best) == 0 {
			continue
		}
		if len(all) == 0 {
			all = append(all, best...)
			continue
		}
		start := 0
		if len(best) > 0 && len(all) > 0 && hashRow(best[0]) == hashRow(all[0]) {
			start = 1
		}
		all = append(all, best[start:]...)
	}
	return normalizeRows(all), nil
}

// ---------- Configuracion / persistencia ----------

func locateMaster() string {
	c := LoadConfig()
	if c.MasterPath != "" {
		return c.MasterPath
	}
	return filepath.Join(os.TempDir(), "GestionSO_Datos.csv")
}

func openMasterCSV() (*os.File, error) {
	return os.OpenFile(locateMaster(), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
}

func ensureMasterHeaders(path string, headers []string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if st.Size() != 0 {
		return nil
	}
	w := csv.NewWriter(f)
	if err := w.Write(headers); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func LoadMaster(path ...string) (*MasterData, error) {
	p := locateMaster()
	if len(path) > 0 && path[0] != "" {
		p = path[0]
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	m := &MasterData{Path: p}
	if len(rows) == 0 {
		return m, nil
	}
	m.Headers = rows[0]
	for _, rr := range rows[1:] {
		mr := MasterRow{}
		for i, h := range m.Headers {
			if i < len(rr) {
				mr[h] = rr[i]
			}
		}
		m.Rows = append(m.Rows, mr)
	}
	return m, nil
}

func SaveWithBackup(path string, data []byte) error {
	if b, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+".bak", b, 0644); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0644)
}

func EnsureSKU(m *MasterData, sku string) MasterRow {
	for _, r := range m.Rows {
		if r["SKU"] == sku {
			return r
		}
	}
	r := MasterRow{"SKU": sku}
	m.Rows = append(m.Rows, r)
	return r
}

func SetSO(r MasterRow, so string) { r["SO"] = so }
func SOState(r MasterRow) string   { return r["SO"] }

// ---------- Vistas / proceso ----------

func BuildLines(rows [][]string, source string) []Line {
	out := make([]Line, 0, len(rows))
	for i, r := range rows {
		m := map[string]string{}
		for j, v := range r {
			m[fmt.Sprintf("C%d", j+1)] = v
		}
		out = append(out, Line{Values: m, Source: source, RowNumber: i + 1})
	}
	return out
}

func GroupLines(lines []Line) map[string][]Line {
	g := map[string][]Line{}
	for _, l := range lines {
		g[l.Values["C1"]] = append(g[l.Values["C1"]], l)
	}
	return g
}

func BuildFilteredSortedView(lines []Line, filter string) []Line {
	out := make([]Line, 0, len(lines))
	for _, l := range lines {
		if filter == "" || FilterValue(l, filter) {
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return lineSortKey(out[i]) < lineSortKey(out[j]) })
	return out
}

func FilterValue(l Line, filter string) bool {
	filter = strings.ToLower(filter)
	for _, v := range l.Values {
		if strings.Contains(strings.ToLower(v), filter) {
			return true
		}
	}
	return false
}

func DisplayValue(l Line, col string) string { return l.Values[col] }
func rawVal(l Line, col string) string       { return l.Values[col] }
func rawDisplay(v string) string             { return strings.TrimSpace(v) }
func lineSortKey(l Line) string              { return strings.ToLower(l.Values["C1"]) }
func groupSortKey(k string) string            { return strings.ToLower(k) }
func cmpKey(a, b string) int                 { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) }

func exportVisible(path string, lines []Line) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for _, l := range lines {
		row := make([]string, 0, len(l.Values))
		for _, v := range l.Values {
			row = append(row, v)
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

type csvWriter struct{ w *csv.Writer }

// ---------- Opciones / simulador ----------
// Los simbolos existen en el binario, pero no hay evidencia suficiente en el
// material disponible para reconstruir su logica exacta. Se dejan stubs
// compilables y explicitamente marcados como pendientes.

func openOption(hwnd uintptr) { logf("openOption: reconstructed stub hwnd=%x", hwnd) }
func optWndProc(hwnd, msg, w, l uintptr) uintptr {
	logf("optWndProc: reconstructed stub hwnd=%x msg=%x", hwnd, msg)
	return 0
}
func createOptControls(hwnd uintptr) {}
func layoutOpt(hwnd uintptr)           {}
func optChecked(hwnd uintptr) bool     { return false }
func applyOption(hwnd uintptr)          {}
func openSimulator(hwnd uintptr)        { logf("openSimulator: reconstructed stub hwnd=%x", hwnd) }
func simWndProc(hwnd, msg, w, l uintptr) uintptr { return 0 }
func createSimControls(hwnd uintptr)            {}
func layoutSim(hwnd uintptr)                    {}
func rebuildSimColumns(hwnd uintptr)            {}
func captureSimState(hwnd uintptr)              {}
func simKey(hwnd uintptr) string                { return "" }
func simAdd(hwnd uintptr)                       {}
func simApply(hwnd uintptr)                     {}
func simRemove(hwnd uintptr)                    {}
func simPopulate(hwnd uintptr)                  {}
func handleSimNotify(hwnd, w, l uintptr) uintptr { return 0 }
