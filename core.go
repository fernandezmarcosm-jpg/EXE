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
type MasterData struct{
	Headers []string
	Rows []MasterRow
	Path string
}
type Line struct{
	Values map[string]string
	Source string
	RowNumber int
}
type ColumnDef struct{Name string;Width int;Hidden bool}
type xlsxDoc struct{SharedStrings []string;Sheets map[string][][]string}
type configData struct{MasterPath string;EnginePath string;Mode string}

func panicGuard(fn func()){
	defer func(){if r:=recover();r!=nil{logf("panicGuard recovered: %v",r)}}()
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
	if e != nil { return c }
	for _, ln := range strings.Split(string(b), "\n") {
		kv := strings.SplitN(ln, "=", 2)
		if len(kv) != 2 { continue }
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
func SaveConfig(c configData) error { return saveCfg(filepath.Join(os.TempDir(), "GestionSO-config.txt"), c) }
func saveCfg(path string, c configData) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("MasterPath=%s\nEnginePath=%s\nMode=%s\n", c.MasterPath, c.EnginePath, c.Mode)), 0644)
}

func ReadXLSX(path string)(*xlsxDoc,error){return readXLSXDoc(path)}
func readXLSXDoc(path string)(*xlsxDoc,error){
	z, e := zip.OpenReader(path)
	if e != nil { return nil, e }
	defer z.Close()
	d := &xlsxDoc{Sheets: map[string][][]string{}}
	if ss, e := readZipEntry(z, "xl/sharedStrings.xml"); e == nil {
		d.SharedStrings = parseSharedStrings(ss)
	}
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml") {
			if b, e := readZipEntry(z, f.Name); e == nil {
				s := decodeSheet(b, d.SharedStrings)
				d.Sheets[f.Name] = s
			}
		}
	}
	return d, nil
}
func readZipEntry(z *zip.ReadCloser, name string)([]byte,error){
	for _, f := range z.File {
		if f.Name == name {
			r, e := f.Open()
			if e != nil { return nil, e }
			defer r.Close()
			return io.ReadAll(r)
		}
	}
	return nil, os.ErrNotExist
}

type sharedXML struct{SI []struct{T string `xml:"t"`;R []struct{T string `xml:"t"`} `xml:"r"`} `xml:"si"`}

func parseSharedStrings(b []byte) []string {
	var x sharedXML
	if xml.Unmarshal(b, &x) != nil { return nil }
	o := make([]string, len(x.SI))
	for i, s := range x.SI {
		if s.T != "" {
			o[i] = s.T
		} else {
			for _, r := range s.R { o[i] += r.T }
		}
	}
	return o
}


type sheetXML struct{Rows []struct{Cells []struct{Ref string `xml:"r,attr"`;Type string `xml:"t,attr"`;Value string `xml:"v"`;Inline struct{T string `xml:"t"`} `xml:"is"`} `xml:"c"`} `xml:"row"`}

func decodeSheet(b []byte, ss []string) [][]string {
	var x sheetXML
	if xml.Unmarshal(b, &x) != nil { return nil }
	return parseRows(x.Rows, ss)
}

func parseRows(rows []struct{Cells []struct{Ref string `xml:"r,attr"`;Type string `xml:"t,attr"`;Value string `xml:"v"`;Inline struct{T string `xml:"t"`} `xml:"is"`} `xml:"c"`}, ss []string) [][]string {
	out := [][]string{}
	for _, r := range rows {
		row := []string{}
		for _, c := range r.Cells {
			val := c.Value
			if c.Type == "s" {
				if i, err := strconv.Atoi(val); err == nil && i >= 0 && i < len(ss) {
					val = ss[i]
				}
			} else if c.Inline.T != "" {
				val = c.Inline.T
			}
			row = append(row, val)
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
		if i > 0 { b.WriteByte('\n') }
		for j, v := range r {
			if j > 0 { b.WriteByte(',') }
			b.WriteString(xmlEscape(v))
		}
	}
	return b.Bytes()
}

func rewriteRowNumber(ref string, n int) string {
	if n < 1 { return ref }
	i := 0
	for i < len(ref) && ((ref[i] >= 'A' && ref[i] <= 'Z') || (ref[i] >= 'a' && ref[i] <= 'z')) {
		i++
	}
	return ref[:i] + strconv.Itoa(n)
}

func xmlEscape(s string) string { return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&apos;").Replace(s) }

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
		if r < 'A' || r > 'Z' { break }
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
			if strings.Contains(s, k) { score++ }
		}
	}
	return score
}

func headerRowIndex(rows [][]string) int {
	best, bestScore := 0, -1
	for i, r := range rows {
		if s := headerScore(r); s > bestScore { best, bestScore = i, s }
	}
	return best
}

func normalizedHeader(v string, index int) string {
	v = strings.TrimSpace(v)
	if v == "" { return fmt.Sprintf("C%d", index+1) }
	return v
}

func uniqueHeaders(row []string) []string {
	o := make([]string, len(row))
	seen := map[string]int{}
	for i, v := range row {
		h := normalizedHeader(v, i)
		n := seen[h]
		if n > 0 {
			n++
			seen[h] = n
			h = fmt.Sprintf("%s_%d", h, n)
		} else {
			seen[h] = 1
		}
		o[i] = h
	}
	return o
}

func mergeXLSX(paths []string) ([][]string, error) {
	var all [][]string
	for _, p := range paths {
		d, e := ReadXLSX(p)
		if e != nil { return nil, e }
		for _, rows := range d.Sheets {
			all = append(all, rows...)
			break
		}
	}
	return all, nil
}

func locateMaster() string {
	c := LoadConfig()
	if c.MasterPath != "" { return c.MasterPath }
	return filepath.Join(os.TempDir(), "GestionSO_Datos.csv")
}

func openMasterCSV() (*os.File, error) { return os.OpenFile(locateMaster(), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644) }

func ensureMasterHeaders(path string, headers []string) error {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if e != nil { return e }
	defer f.Close()
	st, e := f.Stat()
	if e != nil { return e }
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
	if len(path) > 0 && path[0] != "" { p = path[0] }
	f, e := os.Open(p)
	if e != nil { return nil, e }
	defer f.Close()
	r := csv.NewReader(f)
	rows, e := r.ReadAll()
	if e != nil { return nil, e }
	md := &MasterData{Path: p}
	if len(rows) > 0 {
		md.Headers = rows[0]
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			mr := MasterRow{}
			for j, h := range md.Headers {
				if j < len(row) { mr[h] = row[j] } else { mr[h] = "" }
			}
			md.Rows = append(md.Rows, mr)
		}
	}
	return md, nil
}

func SaveWithBackup(path string, data []byte) error {
	if b, e := os.ReadFile(path); e == nil {
		if e := os.WriteFile(path+".bak", b, 0644); e != nil { return e }
	}
	return os.WriteFile(path, data, 0644)
}

func EnsureSKU(m *MasterData, sku string) MasterRow {
	for _, r := range m.Rows { if r["SKU"] == sku { return r } }
	r := MasterRow{"SKU": sku}
	m.Rows = append(m.Rows, r)
	return r
}

func SetSO(r MasterRow, so string) { r["SO"] = so }
func SOState(r MasterRow) string { return r["SO"] }

func BuildLines(rows [][]string, source string) []Line {
	if len(rows) == 0 { return nil }
	hi := headerRowIndex(rows)
	headers := uniqueHeaders(rows[hi])
	out := make([]Line, 0, len(rows)-hi-1)
	for i := hi+1; i < len(rows); i++ {
		values := map[string]string{}
		for j, v := range rows[i] {
			if j < len(headers) { values[headers[j]] = v }
		}
		out = append(out, Line{Values: values, Source: source, RowNumber: i})
	}
	return out
}

func findFieldKey(l Line, candidates ...string) string {
	for _, cand := range candidates {
		cl := strings.ToLower(cand)
		for k := range l.Values {
			kl := strings.ToLower(strings.TrimSpace(k))
			if kl == cl || strings.Contains(kl, cl) { return k }
		}
	}
	for k := range l.Values { return k }
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

func BuildFilteredSortedView(lines []Line, filter string) []Line { return BuildFilteredSortedViewByHeaders(lines, map[string]string{"__all__": filter}) }

func BuildFilteredSortedViewByHeaders(lines []Line, filters map[string]string) []Line {
	o := make([]Line, 0, len(lines))
	for _, l := range lines {
		ok := true
		for field, value := range filters {
			value = strings.TrimSpace(value)
			if value == "" { continue }
			if !FilterValue(l, value) { ok = false; break }
		}
		if ok { o = append(o, l) }
	}
	return o
}

func fieldValue(l Line, field string) string {
	if v, ok := l.Values[field]; ok { return v }
	for k, v := range l.Values { if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(field)) { return v } }
	return ""
}

func FilterValue(l Line, filter string) bool { f := strings.ToLower(filter); for _, v := range l.Values { if strings.Contains(strings.ToLower(v), f) { return true } }; return false }

func DisplayValue(l Line, col string) string { return fieldValue(l, col) }
func rawVal(l Line, col string) string { return fieldValue(l, col) }
func rawDisplay(l Line, col string) string { return fieldValue(l, col) }

func availableColumns(lines []Line) []ColumnDef {
	if len(lines) == 0 { return nil }
	keys := make([]string, 0, len(lines[0].Values))
	for k := range lines[0].Values { keys = append(keys, k) }
	sort.Strings(keys)
	o := make([]ColumnDef, 0, len(keys))
	for _, k := range keys { o = append(o, ColumnDef{Name: k, Width: 100}) }
	return o
}

func defaultVisible(c []ColumnDef) []ColumnDef { return c }
func defaultHeaderFilters(c []ColumnDef) map[string]string { m := map[string]string{}; for _, x := range c { m[x.Name] = "" }; return m }

func lineSortKey(l Line) string { return strings.ToLower(l.Values[findFieldKey(l, "so", "factura", "cliente")]) }
func groupSortKey(l Line) string { return lineSortKey(l) }
func keyText(l Line) string { return lineSortKey(l) }
func keyAuto(l Line) string { return lineSortKey(l) }
func cmpKey(a, b string) int { a = strings.ToLower(a); b = strings.ToLower(b); if a < b { return -1 }; if a > b { return 1 }; return 0 }
func fmtPct(v float64) string { return fmt.Sprintf("%.2f%%", v) }

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" { return 0, false }
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, " ", "")
	if strings.Contains(s, ",") && strings.Contains(s, ".") { s = strings.ReplaceAll(s, ",", "") }
	if x, err := strconv.ParseFloat(s, 64); err == nil { return x, true }
	return 0, false
}

func numericByNames(l Line, names ...string) float64 {
	for k, v := range l.Values {
		kl := strings.ToLower(strings.TrimSpace(k))
		for _, n := range names {
			if kl == n || strings.Contains(kl, n) {
				if x, ok := parseNumber(v); ok { return x }
			}
		}
	}
	return 0
}

// Stubs for simulator/options - safe no-ops (documented)
func SimDisplay(v float64) string { return fmt.Sprintf("%.2f", v) }
func exportVisible(lines []Line, path string) error {
	f, e := os.Create(path)
	if e != nil { return e }
	defer f.Close()
	w := csv.NewWriter(f)
	cols := availableColumns(lines)
	h := make([]string, len(cols))
	for i, c := range cols { h[i] = c.Name }
	if e := w.Write(h); e != nil { return e }
	for _, l := range lines {
		row := make([]string, len(cols))
		for i, c := range cols { row[i] = DisplayValue(l, c.Name) }
		_ = w.Write(row)
	}
	w.Flush()
	return w.Error()
}
func csvWriter(w io.Writer, rows [][]string) error { cw := csv.NewWriter(w); for _, r := range rows { if e := cw.Write(r); e != nil { return e } }; cw.Flush(); return cw.Error() }

func newModeConfig() configData { return defaultConfig() }
func openOption(_ uintptr) {}
func optWndProc(hwnd, msg, w, l uintptr) uintptr { return 0 }
func createOptControls(_ uintptr) {}
func layoutOpt(_ uintptr) {}
func optChecked(_ uintptr) bool { return false }
func applyOption(_ uintptr) {}
func openSimulator(_ uintptr) {}
func simWndProc(hwnd, msg, w, l uintptr) uintptr { return 0 }
func createSimControls(_ uintptr) {}
func layoutSim(_ uintptr) {}
func rebuildSimColumns(_ uintptr) {}
func captureSimState(_ uintptr) []string { return nil }
func simKey(_ Line) string { return "" }
func simAdd(_ Line) {}
func simApply() {}
func simRemove(_ int) {}
func simPopulate(_ []Line) {}
func handleSimNotify(_ uintptr, _ uintptr, _ uintptr) uintptr { return 0 }
func CalcSimFromMaster(_ *MasterData) float64 { return 0 }
func masterScore(_ MasterRow) int { return 0 }
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

// ========== FUNCIONES AGREGADAS PARA COMPILAR ==========

// BuildStatusBar construye la cadena de la barra de estado
func BuildStatusBar(mode string, lines []Line, filterCount int, extra string) string {
	total := len(lines)
	var retenidas, liberadas int
	soSet := make(map[string]bool)
	for _, l := range lines {
		estado := fieldValue(l, "Estado")
		if estado == "RETENIDA" || estado == "RETENIDAS" {
			retenidas++
		} else if estado == "LIBERADA" || estado == "LIBERADAS" {
			liberadas++
		}
		if so := fieldValue(l, "SO"); so != "" {
			soSet[so] = true
		}
	}
	mode = strings.TrimPrefix(mode, "MODO: ")
	if mode == "" { mode = "SO RETENIDAS" }
	return fmt.Sprintf("MODO: %s | RETENIDAS %d | LIBERADAS %d | SO %d | LINEAS %d | %d filtros | %s | CSV",
		mode, retenidas, liberadas, len(soSet), total, filterCount, extra)
}

// CalculateSOSubtotals calcula subtotales por SO (para mostrar en grid)
func CalculateSOSubtotals(lines []Line) map[string]map[string]float64 {
	result := make(map[string]map[string]float64)
	for _, l := range lines {
		so := fieldValue(l, "SO")
		if so == "" { continue }
		if _, ok := result[so]; !ok {
			result[so] = make(map[string]float64)
		}
		if v, ok := parseNumber(fieldValue(l, "NETO SO")); ok {
			result[so]["NETO SO"] += v
		}
		if v, ok := parseNumber(fieldValue(l, "UNIDADES")); ok {
			result[so]["UNIDADES"] += v
		}
		if v, ok := parseNumber(fieldValue(l, "PK")); ok {
			result[so]["PK"] += v
		}
		if v, ok := parseNumber(fieldValue(l, "TN SO")); ok {
			result[so]["TN SO"] += v
		}
	}
	return result
}

// findAnyValue busca en todas las keys de la línea por coincidencia parcial
func findAnyValue(l Line, substr string) string {
	sub := strings.ToLower(substr)
	for k, v := range l.Values {
		if strings.Contains(strings.ToLower(k), sub) {
			return v
		}
	}
	return ""
}
