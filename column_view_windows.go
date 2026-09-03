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

func renderSelectedColumnsForTest(doc *xlsxDoc) string {
	if doc == nil || len(doc.Sheets) == 0 { return "Excel vacío o sin hojas legibles." }
	names := make([]string, 0, len(doc.Sheets))
	for name := range doc.Sheets { names = append(names, name) }
	first := names[0]
	rows := doc.Sheets[first]
	if len(rows) == 0 { return "Hoja sin datos." }

	hi := headerRowIndex(rows)
	if hi < 0 || hi >= len(rows) { return "No se encontró la fila de encabezados." }
	headers := uniqueHeaders(rows[hi])
	skuIdx := columnHeaderIndex(headers, "SKU")
	descIdx := columnHeaderIndex(headers, "MASTER_DESCRIPCION")
	if descIdx < 0 { descIdx = columnHeaderIndex(headers, "DESCRIPCION") }
	priceIdx := columnHeaderIndex(headers, "PRECIOUNIFC")
	unitsIdx := columnHeaderIndex(headers, "MASTER_UNIDADES_X_BULTO")
	if unitsIdx < 0 { unitsIdx = columnHeaderIndex(headers, "UNIDADES_X_BULTO") }

	var b strings.Builder
	b.WriteString("COLUMNAS A MOSTRAR\r\n")
	b.WriteString("Hoja: ")
	b.WriteString(first)
	b.WriteString("\r\n\r\n")
	if columnIsChecked(columnSKU) { b.WriteString("SKU\t") }
	if columnIsChecked(columnDesc) { b.WriteString("DESCRIPCION\t") }
	if columnIsChecked(columnPrice) { b.WriteString("PRECIOUNIFC / UNIDADES_X_BULTO\t") }
	b.WriteString("\r\n")

	end := len(rows)
	if end > hi+100 { end = hi + 100 }
	for i := hi + 1; i < end; i++ {
		shown := false
		if columnIsChecked(columnSKU) { b.WriteString(columnCellAt(rows[i], skuIdx)); b.WriteByte('\t'); shown = true }
		if columnIsChecked(columnDesc) { b.WriteString(columnCellAt(rows[i], descIdx)); b.WriteByte('\t'); shown = true }
		if columnIsChecked(columnPrice) {
			b.WriteString(columnPricePerBoxUnit(columnCellAt(rows[i], priceIdx), columnCellAt(rows[i], unitsIdx)))
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
	for i, h := range headers {
		if strings.EqualFold(strings.TrimSpace(h), wanted) { return i }
	}
	return -1
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
