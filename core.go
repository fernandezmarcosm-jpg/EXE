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
	"sync"
)

type MasterRow map[string]string

type MasterData struct {
	Headers []string
	Rows    []MasterRow
	Path    string
	ByKey   map[string]MasterRow
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

// INFERENCIA: la captura de referencia muestra SO RETENIDAS como pantalla
// inicial. Las tres opciones de modo siguen siendo las strings verificadas.
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

// ReadXLSX es el punto unico por el que el programa incorpora un XLSX a memoria.
// Despues de leerlo, se integra contra GestionSO_Datos.csv en memoria usando
// SKU del XLSX <-> CLAVE del maestro. Si el maestro no esta disponible, el
// XLSX sigue funcionando exactamente como antes.
func ReadXLSX(path string) (*xlsxDoc, error) {
	d, err := readXLSXDoc(path)
	if err != nil {
		return nil, err
	}
	integrateMasterIntoWorkbook(d)
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
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "GestionSO_Datos.csv")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	candidate := filepath.Join("acceso chatgpt", "GestionSO_Datos.csv")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return filepath.Join(os.TempDir(), "GestionSO_Datos.csv")
}

func openMasterCSV() (*os.File, error) {
	return os.Open(locateMaster())
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

var masterOnce sync.Once
var masterModel *MasterData

func cachedMaster() *MasterData {
	masterOnce.Do(func() {
		m, err := LoadMaster()
		if err != nil {
			logf("master CSV no disponible: %v", err)
			return
		}
		masterModel = m
		logf("master CSV cargado en memoria; archivo=%s filas=%d", m.Path, len(m.Rows))
	})
	return masterModel
}

// integrateMasterIntoWorkbook conserva todas las columnas originales del XLSX
// y agrega al final una pequeña proyeccion del maestro para cada SKU encontrado.
// Esto deja el dato realmente integrado dentro del mismo objeto que ya guarda
// el programa en memoria, sin introducir una base de datos ni persistencia extra.
func integrateMasterIntoWorkbook(doc *xlsxDoc) {
	if doc == nil {
		return
	}
	master := cachedMaster()
	if master == nil || len(master.Rows) == 0 {
		return
	}
	for sheetName, rows := range doc.Sheets {
		if len(rows) == 0 {
			continue
		}
		hi := headerRowIndex(rows)
		if hi < 0 || hi >= len(rows) {
			continue
		}
		headers := uniqueHeaders(rows[hi])
		skuCol := -1
		for i, h := range headers {
			if strings.EqualFold(strings.TrimSpace(h), "SKU") {
				skuCol = i
				break
			}
		}
		if skuCol < 0 {
			continue
		}
		const (
			masterClave = iota
			masterEstado
			masterUnidades
			masterPrecio
			masterCosto
			masterDescripcion
		)
		added := []string{"MASTER_CLAVE", "MASTER_ESTADO", "MASTER_UNIDADES_X_BULTO", "MASTER_PRECIO_LISTA_UNITARIO", "MASTER_COSTO_UNITARIO", "MASTER_DESCRIPCION"}
		for i := range rows {
			if i == hi {
				rows[i] = append(rows[i], added...)
				continue
			}
			sku := ""
			if skuCol < len(rows[i]) {
				sku = strings.TrimSpace(rows[i][skuCol])
			}
			r := MasterBySKU(master, sku)
			values := make([]string, len(added))
			if r != nil {
				values[masterClave] = r["CLAVE"]
				values[masterEstado] = r["ESTADO"]
				values[masterUnidades] = r["UNIDADES_X_BULTO"]
				values[masterPrecio] = r["PRECIO_LISTA_UNITARIO"]
				values[masterCosto] = r["COSTO_UNITARIO"]
				values[masterDescripcion] = r["DESCRIPCION"]
			}
			rows[i] = append(rows[i], values...)
		}
		doc.Sheets[sheetName] = rows
	}
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
