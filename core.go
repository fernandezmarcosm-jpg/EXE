// RECONSTRUCCION DE GestionSO V57.
// Este archivo NO es el fuente original. Es una reimplementacion basada en
// simbolos, strings y comportamiento observable del binario V57.
// HECHO VERIFICADO: existen simbolos para XLSX, persistencia, vistas,
// configuracion, opciones y simulador.
// INFERENCIA: estructuras, formulas y conteos no recuperables son conservadores.
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

type MasterRow map[string]string

type MasterData struct {
	Headers []string
	Rows    []MasterRow
	Path    string
}

type Line struct {
	Values   map[string]string
	Source   string
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

type configData struct {
	MasterPath string
	EnginePath string
	Mode       string
}

func panicGuard(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logf("panicGuard recovered: %v", r)
		}
	}()
	fn()
}

func logf(format string, args ...interface{}) {
	p := filepath.Join(os.TempDir(), "GestionSO-V57-debug.log")
	f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if e != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", args...)
}

func initLog() { logf("initLog: GestionSO V57 reconstruccion") }

func defaultConfig() configData { return configData{Mode: "MODO: FACTURAS"} }

func LoadConfig() configData {
	c := defaultConfig()
	p := filepath.Join(os.TempDir(), "GestionSO-config.txt")
	b, e := os.ReadFile(p)
	if e != nil {
		return c
	}
	for _, ln := range strings.Split(string(b), "\n") {
		kv := strings.SplitN(ln, "=", 2)
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
	return saveCfg(filepath.Join(os.TempDir(), "GestionSO-config.txt"), c)
}

func saveCfg(path string, c configData) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("MasterPath=%s\nEnginePath=%s\nMode=%s\n", c.MasterPath, c.EnginePath, c.Mode)), 0644)
}

func ReadXLSX(path string) (*xlsxDoc, error) { return readXLSXDoc(path) }

func readXLSXDoc(path string) (*xlsxDoc, error) {
	z, e := zip.OpenReader(path)
	if e != nil {
		return nil, e
	}
	defer z.Close()
	d := &xlsxDoc{Sheets: map[string][][]string{}}
	if ss, e := readZipEntry(z, "xl/sharedStrings.xml"); e == nil {
		d.SharedStrings = parseSharedStrings(ss)
	}
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml") {
			if b, e := readZipEntry(z, f.Name); e == nil {
				d.Sheets[f.Name] = decodeSheet(b, d.SharedStrings)
			}
		}
	}
	return d, nil
}

func readZipEntry(z *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range z.File {
		if f.Name == name {
			r, e := f.Open()
			if e != nil {
				return nil, e
			}
			defer r.Close()
			return io.ReadAll(r)
		}
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
	o := make([]string, len(x.SI))
	for i, s := range x.SI {
		if s.T != "" {
			o[i] = s.T
		} else {
			for _, r := range s.R {
				o[i] += r.T
			}
		}
	}
	return o
}

type sheetCellXML struct {
	Ref    string `xml:"r,attr"`
	Type   string `xml:"t,attr"`
	Value  string `xml:"v"`
	Inline struct {
		Texts []string `xml:"t"`
	} `xml:"is"`
}

type sheetRowXML struct {
	Cells []sheetCellXML `xml:"c"`
}

type sheetXML struct {
	Rows []sheetRowXML `xml:"row"`
}

func decodeSheet(b []byte, ss []string) [][]string {
	var x sheetXML
	if err := xml.Unmarshal(b, &x); err != nil {
		return nil
	}
	return parseRows(x.Rows, ss)
}

func parseRows(rows []sheetRowXML, ss []string) [][]string {
	out := make([][]string, 0, len(rows))
	for _, r := range rows {
		maxCol := -1
		for i, c := range r.Cells {
			col := i
			if c.Ref != "" {
				if parsed := colFromRef(c.Ref); parsed > 0 {
					col = parsed - 1
				}
			}
			if col > maxCol {
				maxCol = col
			}
		}
		if maxCol < 0 {
			out = append(out, nil)
			continue
		}
		row := make([]string, maxCol+1)
		for i, c := range r.Cells {
			col := i
			if c.Ref != "" {
				if parsed := colFromRef(c.Ref); parsed > 0 {
					col = parsed - 1
				}
			}
			val := c.Value
			switch c.Type {
			case "s":
				if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && n >= 0 && n < len(ss) {
					val = ss[n]
				}
			case "inlineStr":
				if len(c.Inline.Texts) > 0 {
					val = strings.Join(c.Inline.Texts, "")
				}
			}
			row[col] = val
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
	n := 0
	for _, r := range strings.ToUpper(ref) {
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
			}
		}
	}
	return score
}

func headerRowIndex(rows [][]string) int {
	best, bestScore := 0, -1
	for i, r := range rows {
		if s := headerScore(r); s > bestScore {
			best, bestScore = i, s
		}
	}
	return best
}

func normalizedHeader(v string, index int) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fmt.Sprintf("C%d", index+1)
	}
	return v
}

func uniqueHeaders(row []string) []string {
	o := make([]string, len(row))
	seen := map[string]int{}
	for i, v := range row {
		h := normalizedHeader(v, i)
		key := strings.ToLower(h)
		n := seen[key]
		if n > 0 {
			n++
			seen[key] = n
			h = fmt.Sprintf("%s_%d", h, n)
		} else {
			seen[key] = 1
		}
		o[i] = h
	}
	return o
}

func mergeXLSX(paths []string) ([][]string, error) {
	var all [][]string
	for _, p := range paths {
		d, err := ReadXLSX(p)
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(d.Sheets))
		for name := range d.Sheets {
			names = append(names, name)
		}
		sort.Strings(names)
		// INFERENCIA: se toma una hoja por XLSX. Elegir el primer nombre
		// ordenado hace que el resultado sea determinista.
		if len(names) > 0 {
			all = append(all, d.Sheets[names[0]]...)
		}
	}
	return all, nil
}

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
	f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return e
	}
	if st.Size() == 0 {
		w := csv.NewWriter(f)
		_ = w.Write(headers)
		w.Flush()
		return w.Error()
	}
	return nil
}

func LoadMaster(path ...string) (*MasterData, error) {
	p := locateMaster()
	if len(path) > 0 && path[0] != "" {
		p = path[0]
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, e := r.ReadAll()
	if e != nil {
		return nil, e
	}
	md := &MasterData{Path: p}
	if len(rows) > 0 {
		md.Headers = rows[0]
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			mr := MasterRow{}
			for j, h := range md.Headers {
				if j < len(row) {
					mr[h] = row[j]
				} else {
					mr[h] = ""
				}
			}
			md.Rows = append(md.Rows, mr)
		}
	}
	return md, nil
}

func SaveWithBackup(path string, data []byte) error {
	if b, e := os.ReadFile(path); e == nil {
		if e := os.WriteFile(path+".bak", b, 0644); e != nil {
			return e
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

func BuildLines(rows [][]string, source string) []Line {
	if len(rows) == 0 {
		return nil
	}
	hi := headerRowIndex(rows)
	headers := uniqueHeaders(rows[hi])
	out := make([]Line, 0, len(rows)-hi-1)
	for i := hi + 1; i < len(rows); i++ {
		values := map[string]string{}
		for j, v := range rows[i] {
			if j < len(headers) {
				values[headers[j]] = v
			}
		}
		out = append(out, Line{Values: values, Source: source, RowNumber: i})
	}
	return out
}

func findFieldKey(l Line, candidates ...string) string {
	keys := make([]string, 0, len(l.Values))
	for k := range l.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, cand := range candidates {
		cl := strings.ToLower(strings.TrimSpace(cand))
		for _, k := range keys {
			if strings.ToLower(strings.TrimSpace(k)) == cl {
				return k
			}
		}
	}
	for _, cand := range candidates {
		cl := strings.ToLower(strings.TrimSpace(cand))
		for _, k := range keys {
			if strings.Contains(strings.ToLower(strings.TrimSpace(k)), cl) {
				return k
			}
		}
	}
	if len(keys) > 0 {
		return keys[0]
	}
	return ""
}

func GroupLines(lines []Line) map[string][]Line {
	g := map[string][]Line{}
	for _, l := range lines {
		k := findFieldKey(l, "so", "factura", "cliente")
		g[l.Values[k]] = append(g[l.Values[k]], l)
	}
	return g
}

func BuildFilteredSortedView(lines []Line, filter string) []Line {
	return BuildFilteredSortedViewByHeaders(lines, map[string]string{"__all__": filter})
}

func BuildFilteredSortedViewByHeaders(lines []Line, filters map[string]string) []Line {
	o := make([]Line, 0, len(lines))
	for _, l := range lines {
		ok := true
		for field, value := range filters {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if field == "__all__" {
				if !FilterValue(l, value) {
					ok = false
					break
				}
			} else if !filterFieldValue(l, field, value) {
				ok = false
				break
			}
		}
		if ok {
			o = append(o, l)
		}
	}
	sort.SliceStable(o, func(i, j int) bool {
		return cmpKey(lineSortKey(o[i]), lineSortKey(o[j])) < 0
	})
	return o
}

func filterFieldValue(l Line, field, filter string) bool {
	v := fieldValue(l, field)
	if v == "" {
		return false
	}
	return strings.Contains(strings.ToLower(v), strings.ToLower(filter))
}

func fieldValue(l Line, field string) string {
	if v, ok := l.Values[field]; ok {
		return v
	}
	for k, v := range l.Values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(field)) {
			return v
		}
	}
	return ""
}

func FilterValue(l Line, filter string) bool {
	f := strings.ToLower(filter)
	for _, v := range l.Values {
		if strings.Contains(strings.ToLower(v), f) {
			return true
		}
	}
	return false
}

func DisplayValue(l Line, col string) string { return fieldValue(l, col) }
func rawVal(l Line, col string) string        { return fieldValue(l, col) }
func rawDisplay(l Line, col string) string    { return fieldValue(l, col) }

func availableColumns(lines []Line) []ColumnDef {
	if len(lines) == 0 {
		return nil
	}
	keys := make([]string, 0, len(lines[0].Values))
	for k := range lines[0].Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	o := make([]ColumnDef, 0, len(keys))
	for _, k := range keys {
		o = append(o, ColumnDef{Name: k, Width: 100})
	}
	return o
}

func defaultVisible(c []ColumnDef) []ColumnDef { return c }

func defaultHeaderFilters(c []ColumnDef) map[string]string {
	m := map[string]string{}
	for _, x := range c {
		m[x.Name] = ""
	}
	return m
}

func lineSortKey(l Line) string {
	return strings.ToLower(l.Values[findFieldKey(l, "so", "factura", "cliente")])
}
func groupSortKey(l Line) string { return lineSortKey(l) }
func keyText(l Line) string      { return lineSortKey(l) }
func keyAuto(l Line) string      { return lineSortKey(l) }

func cmpKey(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func fmtPct(v float64) string { return fmt.Sprintf("%.2f%%", v) }

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "'", "")
	if strings.Contains(s, ",") && strings.Contains(s, ".") {
		// INFERENCIA: el último separador se interpreta como decimal.
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	} else if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		if len(parts) == 2 && len(parts[1]) <= 2 {
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	}
	if x, err := strconv.ParseFloat(s, 64); err == nil {
		return x, true
	}
	return 0, false
}

func numericByNames(l Line, names ...string) float64 {
	keys := make([]string, 0, len(l.Values))
	for k := range l.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		for _, k := range keys {
			if strings.ToLower(strings.TrimSpace(k)) == n {
				if x, ok := parseNumber(l.Values[k]); ok {
					return x
				}
			}
		}
	}
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		for _, k := range keys {
			if strings.Contains(strings.ToLower(strings.TrimSpace(k)), n) {
				if x, ok := parseNumber(l.Values[k]); ok {
					return x
				}
			}
		}
	}
	return 0
}

func SimDisplay(v float64) string { return fmt.Sprintf("%.2f", v) }

func exportVisible(lines []Line, path string) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	w := csv.NewWriter(f)
	cols := availableColumns(lines)
	h := make([]string, len(cols))
	for i, c := range cols {
		h[i] = c.Name
	}
	if e := w.Write(h); e != nil {
		return e
	}
	for _, l := range lines {
		row := make([]string, len(cols))
		for i, c := range cols {
			row[i] = DisplayValue(l, c.Name)
		}
		_ = w.Write(row)
	}
	w.Flush()
	return w.Error()
}

func csvWriter(w io.Writer, rows [][]string) error {
	cw := csv.NewWriter(w)
	for _, r := range rows {
		if e := cw.Write(r); e != nil {
			return e
		}
	}
	cw.Flush()
	return cw.Error()
}

func newModeConfig() configData { return defaultConfig() }

// Stubs: no-op because the internal behavior is not recoverable from the
// available evidence. Keeping the verified symbols preserves the API surface.
func openOption(_ uintptr)                                      {}
func optWndProc(hwnd, msg, w, l uintptr) uintptr               { return 0 }
func createOptControls(_ uintptr)                              {}
func layoutOpt(_ uintptr)                                      {}
func optChecked(_ uintptr) bool                                { return false }
func applyOption(_ uintptr)                                    {}
func openSimulator(_ uintptr)                                  {}
func simWndProc(hwnd, msg, w, l uintptr) uintptr               { return 0 }
func createSimControls(_ uintptr)                              {}
func layoutSim(_ uintptr)                                      {}
func rebuildSimColumns(_ uintptr)                              {}
func captureSimState(_ uintptr) []string                       { return nil }
func simKey(_ Line) string                                     { return "" }
func simAdd(_ Line)                                             {}
func simApply()                                                 {}
func simRemove(_ int)                                           {}
func simPopulate(_ []Line)                                      {}
func handleSimNotify(_ uintptr, _ uintptr, _ uintptr) uintptr  { return 0 }
func CalcSimFromMaster(_ *MasterData) float64                   { return 0 }
func masterScore(_ MasterRow) int                               { return 0 }

func insertAfter(s []string, after string, value string) []string {
	for i, v := range s {
		if v == after {
			res := make([]string, 0, len(s)+1)
			res = append(res, s[:i+1]...)
			res = append(res, value)
			res = append(res, s[i+1:]...)
			return res
		}
	}
	return append(s, value)
}

// BuildStatusBar: formato observado en la captura V54. Los conteos se derivan
// conservadoramente de las líneas cargadas; no representan la fórmula interna.
func BuildStatusBar(mode string, lines []Line, filterCount int, extra string) string {
	total := len(lines)
	var retenidas, liberadas int
	soSet := make(map[string]bool)
	for _, l := range lines {
		estado := strings.ToUpper(strings.TrimSpace(fieldValue(l, "Estado")))
		switch estado {
		case "RETENIDA", "RETENIDAS":
			retenidas++
		case "LIBERADA", "LIBERADAS":
			liberadas++
		}
		if so := strings.TrimSpace(fieldValue(l, "SO")); so != "" {
			soSet[so] = true
		}
	}
	mode = strings.TrimPrefix(strings.TrimSpace(mode), "MODO: ")
	if mode == "" {
		mode = "SO RETENIDAS"
	}
	return fmt.Sprintf("MODO: %s | RETENIDAS %d | LIBERADAS %d | SO %d | LINEAS %d | %d filtros | %s | CSV",
		mode, retenidas, liberadas, len(soSet), total, filterCount, extra)
}

// CalculateSOSubtotals: suma conservadora por grupo de SO. No pretende
// reproducir las fórmulas internas de negocio del binario original.
func CalculateSOSubtotals(lines []Line) map[string]map[string]float64 {
	result := make(map[string]map[string]float64)
	for _, l := range lines {
		so := strings.TrimSpace(fieldValue(l, "SO"))
		if so == "" {
			continue
		}
		if _, ok := result[so]; !ok {
			result[so] = make(map[string]float64)
		}
		for _, field := range []string{
			"BULTOS", "PALL", "PK", "UNIDADES", "NETO PK", "NETO SO",
			"TN SO", "CMG", "PPP SO", "RESULTADO",
		} {
			if v, ok := parseNumber(fieldValue(l, field)); ok {
				result[so][field] += v
			}
		}
	}
	return result
}

func findAnyValue(l Line, substr string) string {
	sub := strings.ToLower(strings.TrimSpace(substr))
	keys := make([]string, 0, len(l.Values))
	for k := range l.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.ToLower(strings.TrimSpace(k)) == sub {
			return l.Values[k]
		}
	}
	for _, k := range keys {
		if strings.Contains(strings.ToLower(strings.TrimSpace(k)), sub) {
			return l.Values[k]
		}
	}
	return ""
}
