// RECONSTRUCCION DE GestionSO V57.
// Este archivo contiene la lectura XLSX y la capa base de datos en memoria.
// La informacion importada se mantiene separada de la presentacion y de las
// formulas. Los nombres visibles de las columnas no son sus identificadores.
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
	ByKey   map[string]MasterRow
}

// Line se conserva como estructura de compatibilidad para las funciones
// existentes. La nueva capa de trabajo es MemoryWorkbook.
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

// ValueType es el tipo de dato que se asigna una sola vez al incorporar una
// celda a memoria. El valor original Raw siempre se conserva.
type ValueType uint8

const (
	ValueEmpty ValueType = iota
	ValueText
	ValueNumber
	ValueDate
)

// MemoryValue es la unidad de informacion que circula por el programa.
// ColumnID identifica la columna; Raw conserva exactamente lo leido y los
// campos tipados permiten calcular sin volver al XLSX.
type MemoryValue struct {
	ColumnID string
	Raw      string
	Type     ValueType
	Number   float64
}

type MemoryColumn struct {
	ID       string
	Title    string
	Index    int
	Type     ValueType
	Width    int
	Visible  bool
}

type MemoryRow struct {
	ID        string
	Source    string
	SourceRow int
	Values    map[string]MemoryValue
}

type MemorySheet struct {
	ID          string
	Name        string
	HeaderIndex int
	Columns     []MemoryColumn
	Rows        []MemoryRow
}

type MemoryWorkbook struct {
	Sheets      []MemorySheet
	TotalRows   int
	TotalValues int
}

type xlsxDoc struct {
	SharedStrings []string
	// Sheets conserva el resultado crudo del lector. Memory es la estructura
	// normalizada que se usa para trabajar posteriormente.
	Sheets map[string][][]string
	Memory *MemoryWorkbook
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

func defaultConfig() configData { return configData{Mode: "MODO: SO RETENIDAS"} }

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

// ReadXLSX es el unico punto de entrada de un XLSX. Lee primero y luego crea
// una representacion independiente en memoria. El CSV maestro NO se mezcla
// aqui: se consultara cuando una formula lo necesite.
func ReadXLSX(path string) (*xlsxDoc, error) {
	d, err := readXLSXDoc(path)
	if err != nil {
		return nil, err
	}
	d.Memory = BuildMemoryWorkbook(d)
	return d, nil
}

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

// BuildMemoryWorkbook transforma el resultado crudo en la base estable de
// trabajo. Solo las filas posteriores a la fila de titulos se convierten en
// registros. La identidad de una columna es su ID, no su titulo visible.
func BuildMemoryWorkbook(doc *xlsxDoc) *MemoryWorkbook {
	mw := &MemoryWorkbook{}
	if doc == nil {
		return mw
	}
	names := make([]string, 0, len(doc.Sheets))
	for name := range doc.Sheets {
		names = append(names, name)
	}
	sort.Strings(names)
	for si, name := range names {
		rows := doc.Sheets[name]
		if len(rows) == 0 {
			continue
		}
		hi := headerRowIndexStrict(rows)
		if hi < 0 || hi >= len(rows) {
			continue
		}
		headers := uniqueHeaders(rows[hi])
		columnCount := maxColumns(rows)
		if columnCount < len(headers) {
			columnCount = len(headers)
		}
		sheet := MemorySheet{
			ID: fmt.Sprintf("S%03d", si+1), Name: name, HeaderIndex: hi,
			Columns: make([]MemoryColumn, 0, columnCount), Rows: make([]MemoryRow, 0, len(rows)-hi-1),
		}
		for ci := 0; ci < columnCount; ci++ {
			title := fmt.Sprintf("C%d", ci+1)
			if ci < len(headers) && strings.TrimSpace(headers[ci]) != "" {
				title = headers[ci]
			}
			t := inferColumnType(rows, hi, ci)
			sheet.Columns = append(sheet.Columns, MemoryColumn{
			ID: fmt.Sprintf("%sC%03d", sheet.ID, ci+1), Title: title,
			Index: ci, Type: t, Width: 140, Visible: true,
			})
		}
		for ri := hi + 1; ri < len(rows); ri++ {
			row := rows[ri]
			if memoryRowEmpty(row) {
				continue
			}
			mr := MemoryRow{ID: fmt.Sprintf("%sR%06d", sheet.ID, len(sheet.Rows)+1), Source: name, SourceRow: ri + 1, Values: map[string]MemoryValue{}}
			for ci, col := range sheet.Columns {
				raw := ""
				if ci < len(row) {
					raw = strings.TrimSpace(row[ci])
				}
				v := makeMemoryValue(col.ID, raw, col.Type)
				if v.Type != ValueEmpty {
					mr.Values[col.ID] = v
				}
			}
			sheet.Rows = append(sheet.Rows, mr)
		}
		mw.TotalRows += len(sheet.Rows)
		for _, r := range sheet.Rows {
			mw.TotalValues += len(r.Values)
		}
		mw.Sheets = append(mw.Sheets, sheet)
	}
	return mw
}

// headerRowIndexStrict identifica la fila de titulos por su estructura, no
// por nombres concretos. Como el formato de entrada es estable, buscamos la
// primera fila no vacia con varias celdas textuales y con registros debajo.
func headerRowIndexStrict(rows [][]string) int {
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		if memoryRowEmpty(row) || nonEmptyCount(row) < 2 {
			continue
		}
		textLike, numeric := 0, 0
		for _, v := range row {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, ok := parseNumber(v); ok {
				numeric++
			} else {
				textLike++
			}
		}
		if textLike >= 2 && textLike >= numeric {
			if hasDataBelow(rows, i) {
				return i
			}
		}
	}
	return -1
}

func nonEmptyCount(row []string) int {
	n := 0
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			n++
		}
	}
	return n
}

func hasDataBelow(rows [][]string, headerIndex int) bool {
	seen := 0
	for i := headerIndex + 1; i < len(rows) && seen < 3; i++ {
		if !memoryRowEmpty(rows[i]) {
			seen++
		}
	}
	return seen > 0
}

func normalizeHeaderKey(v string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(v, "\ufeff")))
}

// inferColumnType se basa exclusivamente en los valores debajo del titulo.
// El nombre de la columna nunca decide el tipo.
func inferColumnType(rows [][]string, headerIndex, col int) ValueType {
	numberSeen, textSeen := 0, 0
	limit := len(rows)
	if limit > headerIndex+101 {
		limit = headerIndex + 101
	}
	for i := headerIndex + 1; i < limit; i++ {
		if col >= len(rows[i]) {
			continue
		}
		v := strings.TrimSpace(rows[i][col])
		if v == "" {
			continue
		}
		if _, ok := parseNumber(v); ok {
			numberSeen++
		} else {
			textSeen++
		}
	}
	if numberSeen > 0 && textSeen == 0 {
		return ValueNumber
	}
	return ValueText
}

func makeMemoryValue(columnID, raw string, t ValueType) MemoryValue {
	v := MemoryValue{ColumnID: columnID, Raw: raw, Type: t}
	if t == ValueNumber {
		if n, ok := parseNumber(raw); ok {
			v.Number = n
		} else {
			v.Type = ValueText
		}
	}
	return v
}

func memoryRowEmpty(row []string) bool {
	return nonEmptyCount(row) == 0
}

func (t ValueType) String() string {
	switch t {
	case ValueText:
		return "TEXT"
	case ValueNumber:
		return "NUMBER"
	case ValueDate:
		return "DATE"
	default:
		return "EMPTY"
	}
}

func MemoryValueString(v MemoryValue) string {
	if v.Type == ValueNumber {
		return strconv.FormatFloat(v.Number, 'f', -1, 64)
	}
	return v.Raw
}

func memoryColumn(mw *MemoryWorkbook, sheetID, columnID string) *MemoryColumn {
	if mw == nil {
		return nil
	}
	for si := range mw.Sheets {
		if mw.Sheets[si].ID != sheetID {
			continue
		}
		for ci := range mw.Sheets[si].Columns {
			if mw.Sheets[si].Columns[ci].ID == columnID {
				return &mw.Sheets[si].Columns[ci]
			}
		}
	}
	return nil
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

// Compatibilidad con el codigo anterior. Ya no se usa para construir la
// memoria principal, pero queda disponible para filtros y pruebas existentes.
func headerScore(row []string) int {
	score := 0
	for _, v := range row {
		s := normalizeHeaderKey(v)
		for _, k := range []string{"factura", "cliente", "fecha", "cuit", "sku", "producto", "cantidad", "importe", "so"} {
			if s == k {
				score++
			}
		}
	}
	return score
}

func headerRowIndex(rows [][]string) int { return headerRowIndexStrict(rows) }

func normalizedHeader(v string, index int) string {
	v = strings.TrimSpace(strings.TrimPrefix(v, "\ufeff"))
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
		if len(names) > 0 {
			all = append(all, d.Sheets[names[0]]...)
		}
	}
	return all, nil
}

// --- Maestro externo GestionSO_Datos.csv ---

func locateMaster() string {
	c := LoadConfig()
	if c.MasterPath != "" {
		return c.MasterPath
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "GestionSO_Datos.csv")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "GestionSO_Datos.csv")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func openMasterCSV() (*os.File, error) {
	p := locateMaster()
	if p == "" {
		return nil, os.ErrNotExist
	}
	return os.Open(p)
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
	if p == "" {
		return nil, os.ErrNotExist
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	rows, e := r.ReadAll()
	if e != nil {
		return nil, e
	}
	md := &MasterData{Path: p, ByKey: map[string]MasterRow{}}
	if len(rows) == 0 {
		return md, nil
	}
	md.Headers = make([]string, len(rows[0]))
	for i, h := range rows[0] {
		md.Headers[i] = normalizedHeader(h, i)
	}
	keyIndex := -1
	for i, h := range md.Headers {
		if strings.EqualFold(strings.TrimSpace(h), "CLAVE") || strings.EqualFold(strings.TrimSpace(h), "SKU") {
			keyIndex = i
			break
		}
	}
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		mr := MasterRow{}
		for j, h := range md.Headers {
			if j < len(row) {
				mr[h] = strings.TrimSpace(row[j])
			} else {
				mr[h] = ""
			}
		}
		if keyIndex >= 0 && keyIndex < len(row) {
			key := strings.ToUpper(strings.TrimSpace(row[keyIndex]))
			if key != "" {
				md.ByKey[key] = mr
			}
		}
		md.Rows = append(md.Rows, mr)
	}
	return md, nil
}

// No se cachea el maestro de forma permanente: el archivo es externo y puede
// cambiar mientras el programa esta abierto. Cada calculo puede solicitar la
// version actual para que el resultado refleje el CSV vigente.
func cachedMaster() *MasterData {
	m, err := LoadMaster()
	if err != nil {
		logf("master CSV no disponible: %v", err)
		return nil
	}
	return m
}

func MasterBySKU(m *MasterData, sku string) MasterRow {
	if m == nil {
		return nil
	}
	key := strings.ToUpper(strings.TrimSpace(sku))
	if key == "" {
		return nil
	}
	if m.ByKey != nil {
		if r, ok := m.ByKey[key]; ok {
			return r
		}
	}
	for _, r := range m.Rows {
		if strings.EqualFold(strings.TrimSpace(r["CLAVE"]), sku) || strings.EqualFold(strings.TrimSpace(r["SKU"]), sku) {
			return r
		}
	}
	return nil
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
		if r["CLAVE"] == sku || r["SKU"] == sku {
			return r
		}
	}
	r := MasterRow{"CLAVE": sku, "SKU": sku}
	m.Rows = append(m.Rows, r)
	if m.ByKey == nil {
		m.ByKey = map[string]MasterRow{}
	}
	m.ByKey[strings.ToUpper(strings.TrimSpace(sku))] = r
	return r
}

func SetSO(r MasterRow, so string) { r["SO"] = so }
func SOState(r MasterRow) string { return r["SO"] }

func BuildLines(rows [][]string, source string) []Line {
	if len(rows) == 0 {
		return nil
	}
	hi := headerRowIndex(rows)
	if hi < 0 || hi >= len(rows) {
		return nil
	}
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
