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

// Esta capa solamente controla la presentacion. Nunca vuelve al XLSX crudo:
// todos los valores salen de appImportedWorkbook.Memory.
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
	if hwnd == 0 {
		return
	}
	var r appRect
	user32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	if w < 500 {
		w = 500
	}
	if h < 300 {
		h = 300
	}
	user32.NewProc("MoveWindow").Call(columnSKU, 10, 43, 90, 24, 1)
	user32.NewProc("MoveWindow").Call(columnDesc, 105, 43, 150, 24, 1)
	user32.NewProc("MoveWindow").Call(columnPrice, 260, 43, 285, 24, 1)
	user32.NewProc("MoveWindow").Call(appView, 10, 73, uintptr(w-20), uintptr(h-83), 1)
}

func columnSetCheck(hwnd uintptr, checked bool) {
	if hwnd == 0 {
		return
	}
	var v uintptr
	if checked {
		v = bstChecked
	}
	user32.NewProc("SendMessageW").Call(hwnd, bmSetCheck, v, 0)
}

func columnIsChecked(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	r, _, _ := user32.NewProc("SendMessageW").Call(hwnd, bmGetCheck, 0, 0)
	return r == bstChecked
}

func columnViewHandlesCommand(wp uintptr) bool {
	id := int(wp & 0xffff)
	code := uint32((wp >> 16) & 0xffff)
	return code == bnClicked && (id == columnViewSKU || id == columnViewDesc || id == columnViewPrice)
}

func columnViewRefresh() {
	if appImportedWorkbook == nil || appImportedWorkbook.Memory == nil || appView == 0 {
		return
	}
	appSetText(appView, renderSelectedMemoryPreview(appImportedWorkbook.Memory))
}

// renderSelectedMemoryPreview trabaja exclusivamente con ColumnID/Row.Values.
// El titulo es solamente metadato de presentacion; nunca identifica el dato.
func renderSelectedMemoryPreview(mw *MemoryWorkbook) string {
	if mw == nil || len(mw.Sheets) == 0 {
		return "Excel vacío o sin datos en memoria."
	}
	s := &mw.Sheets[0]
	var b strings.Builder
	b.WriteString("DATOS CARGADOS EN MEMORIA\r\n")
	b.WriteString("Hoja: ")
	b.WriteString(s.Name)
	b.WriteString("\r\n\r\n")

	selected := make([]MemoryColumn, 0, len(s.Columns))
	for _, c := range s.Columns {
		show := c.Visible
		if c.ID == findMemoryColumnID(s, "SKU") {
			show = columnIsChecked(columnSKU)
		} else if c.ID == findMemoryColumnID(s, "DESCRIPCION") {
			show = columnIsChecked(columnDesc)
		} else if c.ID == findMemoryColumnID(s, "PRECIOUNIFC") {
			show = columnIsChecked(columnPrice)
		}
		if show {
			selected = append(selected, c)
		}
	}

	for i, c := range selected {
		if i > 0 {
			b.WriteByte('\t')
		}
		b.WriteString(c.Title)
	}
	b.WriteString("\r\n")

	rowLimit := len(s.Rows)
	if rowLimit > 40 {
		rowLimit = 40
	}
	for i := 0; i < rowLimit; i++ {
		r := s.Rows[i]
		for ci, c := range selected {
			if ci > 0 {
				b.WriteByte('\t')
			}
			v, ok := r.Values[c.ID]
			if !ok {
				continue
			}
			b.WriteString(cleanCell(MemoryValueString(v)))
		}
		b.WriteString("\r\n")
	}

	b.WriteString(fmt.Sprintf("\r\nTOTAL EN MEMORIA: %d fila(s), %d valor(es), %d hoja(s).", mw.TotalRows, mw.TotalValues, len(mw.Sheets)))
	return b.String()
}

func findMemoryColumnID(s *MemorySheet, title string) string {
	if s == nil {
		return ""
	}
	for _, c := range s.Columns {
		if strings.EqualFold(strings.TrimSpace(c.Title), title) || strings.Contains(strings.ToLower(c.Title), strings.ToLower(title)) {
			return c.ID
		}
	}
	return ""
}
