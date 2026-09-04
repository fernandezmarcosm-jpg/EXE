//go:build windows

package main

import (
	"fmt"
	"strings"
	"unsafe"
)

const (
	columnViewSKU   = 2010
	columnViewDesc  = 2011
	columnViewPrice = 2012
	bsAutoCheckBox  = 0x00000003
	bmSetCheck      = 0x00F1
	bmGetCheck      = 0x00F0
	bstChecked      = 1
	bnClicked       = 0
)

var (
	columnSKU   uintptr
	columnDesc  uintptr
	columnPrice uintptr
)

// Las columnas son una prueba de visualizacion sobre el mismo XLSX que ya
// esta en memoria. No modifica el archivo Excel ni el CSV maestro.
func columnViewCreate(hwnd uintptr) {
	columnSKU = appMake(hwnd, "BUTTON", "SKU", WS_CHILD|WS_VISIBLE|WS_TABSTOP|bsAutoCheckBox, 10, 43, 90, 24, columnViewSKU)
	columnDesc = appMake(hwnd, "BUTTON", "DESCRIPCION", WS_CHILD|WS_VISIBLE|WS_TABSTOP|bsAutoCheckBox, 105, 43, 150, 24, columnViewDesc)
	columnPrice = appMake(hwnd, "BUTTON", "PRECIOUNIFC / UNIDADES_X_BULTO", WS_CHILD|WS_VISIBLE|WS_TABSTOP|bsAutoCheckBox, 260, 43, 285, 24, columnViewPrice)
	columnSetCheck(columnSKU, true)
	columnSetCheck(columnDesc, true)
	columnSetCheck(columnPrice, true)
	columnViewLayout(hwnd)
}

func columnViewLayout(hwnd uintptr) {
	if hwnd == 0 { return }
	var r appRect
	user32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	if w < 500 { w = 500 }
	if h < 300 { h = 300 }
	user32.NewProc("MoveWindow").Call(columnSKU, 10, 43, 90, 24, 1)
	user32.NewProc("MoveWindow").Call(columnDesc, 105, 43, 150, 24, 1)
	user32.NewProc("MoveWindow").Call(columnPrice, 260, 43, 285, 24, 1)
	user32.NewProc("MoveWindow").Call(appView, 10, 73, uintptr(w-20), uintptr(h-83), 1)
}

func columnSetCheck(hwnd uintptr, checked bool) {
	if hwnd == 0 { return }
	var v uintptr
	if checked { v = bstChecked }
	user32.NewProc("SendMessageW").Call(hwnd, bmSetCheck, v, 0)
}

func columnIsChecked(hwnd uintptr) bool {
	if hwnd == 0 { return false }
	r, _, _ := user32.NewProc("SendMessageW").Call(hwnd, bmGetCheck, 0, 0)
	return r == bstChecked
}

func columnViewHandlesCommand(wp uintptr) bool {
	id := int(wp & 0xffff)
	code := uint32((wp >> 16) & 0xffff)
	return code == bnClicked && (id == columnViewSKU || id == columnViewDesc || id == columnViewPrice)
}

func columnViewRefresh() {
	if appImportedWorkbook == nil || appView == 0 { return }
	appSetText(appView, renderSelectedColumnsForTest(appImportedWorkbook))
}

// columnHeaderRowIndexTest no utiliza el detector heuristico general. Los
// Excel de GestionSO contienen datos que incluyen "SO", "CLIENTE", etc.; el
// detector anterior podia elegir una fila de datos como encabezado y por eso
// se mostraban los nombres de columnas pero ninguna fila.
func columnHeaderRowIndexTest(rows [][]string) int {
	best := -1
	bestScore := -1
	for i, row := range rows {
		score := 0
		for _, v := range row {
			h := strings.ToLower(strings.TrimSpace(v))
			switch h {
			case "sku":
				score += 100
			case "descripción", "descripcion":
				score += 100
			case "factura", "fecha", "cliente":
				score += 20
			case "origen: preciounifc":
				score += 80
			}
		}
		if score > bestScore {
			bestScore = score
			best = i
		}
	}
	return best
}

func columnHeaderIndexFlexible(headers []string, wanted ...string) int {
	for i, h := range headers {
		n := strings.ToLower(strings.TrimSpace(h))
		for _, w := range wanted {
			if n == strings.ToLower(strings.TrimSpace(w)) { return i }
		}
	}
	return -1
}

func renderSelectedColumnsForTest(doc *xlsxDoc) string {
	if doc == nil || len(doc.Sheets) == 0 { return "Excel vacío o sin hojas legibles." }
	names := make([]string, 0, len(doc.Sheets))
	for name := range doc.Sheets { names = append(names, name) }
	first := names[0]
	rows := doc.Sheets[first]
	if len(rows) == 0 { return "Hoja sin datos." }

	hi := columnHeaderRowIndexTest(rows)
	if hi < 0 || hi >= len(rows) { return "No se encontró la fila de encabezados." }
	headers := uniqueHeaders(rows[hi])
	skuIdx := columnHeaderIndexFlexible(headers, "SKU")
	descIdx := columnHeaderIndexFlexible(headers, "MASTER_DESCRIPCION", "DESCRIPCION", "Descripción", "ORIGEN: DESCRIPCION ITEM")
	priceIdx := columnHeaderIndexFlexible(headers, "PRECIOUNIFC", "ORIGEN: PRECIOUNIFC")
	unitsIdx := columnHeaderIndexFlexible(headers, "MASTER_UNIDADES_X_BULTO", "UNIDADES_X_BULTO")

	master := cachedMaster()

	var b strings.Builder
	b.WriteString("COLUMNAS A MOSTRAR\r\n")
	b.WriteString("Hoja: ")
	b.WriteString(first)
	b.WriteString("\r\n\r\n")
	if columnIsChecked(columnSKU) { b.WriteString("SKU\t") }
	if columnIsChecked(columnDesc) { b.WriteString("DESCRIPCION\t") }
	if columnIsChecked(columnPrice) { b.WriteString("PRECIOUNIFC / UNIDADES_X_BULTO\t") }
	b.WriteString("\r\n")

	// Se muestran las filas reales del XLSX. Los valores del maestro se usan
	// directamente por SKU como respaldo, por lo que la prueba funciona incluso
	// si una versión del XLSX todavía no trae las columnas MASTER_* anexadas.
	end := len(rows)
	if end > hi+100 { end = hi + 100 }
	for i := hi + 1; i < end; i++ {
		if len(rows[i]) == 0 { continue }
		sku := columnCellAt(rows[i], skuIdx)
		if sku == "" { continue }

		masterRow := MasterBySKU(master, sku)
		desc := columnCellAt(rows[i], descIdx)
		if desc == "" && masterRow != nil { desc = masterRow["DESCRIPCION"] }

		price := columnCellAt(rows[i], priceIdx)
		units := columnCellAt(rows[i], unitsIdx)
		if masterRow != nil {
			if units == "" { units = masterRow["UNIDADES_X_BULTO"] }
			// El Excel de origen usa ORIGEN: PRECIOUNIFC. Si no está presente,
			// usamos el precio unitario de lista del maestro como respaldo.
			if price == "" { price = masterRow["PRECIO_LISTA_UNITARIO"] }
		}

		shown := false
		if columnIsChecked(columnSKU) { b.WriteString(sku); b.WriteByte('\t'); shown = true }
		if columnIsChecked(columnDesc) { b.WriteString(desc); b.WriteByte('\t'); shown = true }
		if columnIsChecked(columnPrice) {
			b.WriteString(columnPricePerBoxUnit(price, units))
			b.WriteByte('\t')
			shown = true
		}
		if shown { b.WriteString("\r\n") }
	}

	totalRows, totalCells := workbookSize(doc)
	b.WriteString(fmt.Sprintf("\r\nTOTAL EN MEMORIA: %d fila(s), %d celda(s), %d hoja(s).", totalRows, totalCells, len(doc.Sheets)))
	return b.String()
}

func columnHeaderIndex(headers []string, wanted string) int {
	return columnHeaderIndexFlexible(headers, wanted)
}

func columnCellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) { return "" }
	return cleanCell(row[idx])
}

func columnPricePerBoxUnit(priceText, unitsText string) string {
	price, okPrice := parseNumber(priceText)
	units, okUnits := parseNumber(unitsText)
	if !okPrice || !okUnits || units == 0 { return "" }
	return fmt.Sprintf("$ %.2f", price/units)
}
