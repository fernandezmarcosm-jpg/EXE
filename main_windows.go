//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// RECONSTRUCCION: capa Win32 de GestionSO V57. NO es el fuente original.
// HECHO VERIFICADO: los simbolos Win32, el hook del selector XLSX y los strings
// de UI indicados en docs/EVIDENCIA_BINARIO.md existen en la evidencia disponible.
// INFERENCIA: layout exacto y comportamiento de negocio no recuperable.

const (
	ID_ABRIR_XLSX    = 1001
	ID_TOMAR_EXCEL   = 1002
	ID_COLUMNAS      = 1003
	ID_FILTROS_CAB   = 1004
	ID_EXPORTAR_CSV  = 1005
	ID_SIMULADOR     = 1006
	ID_RECARGAR      = 1007
	ID_RESALTAR      = 1008
	ID_COLOR         = 1009
	ID_DATOS_CSV     = 1010
	ID_MODO          = 1011
	ID_FILTRO_SO     = 1101
	ID_FILTRO_ESTADO = 1102
	ID_FILTRO_SKU    = 1103
	ID_FILTRO_SUMA   = 1104
	ID_FILTRO_SDSRP2 = 1105
	ID_FILTRAR       = 1106
	ID_LIMPIAR       = 1107
	ID_GRID           = 1201
	ID_TOTALS         = 1202
	ID_STATUS         = 1203
	ID_PAGE_SIZE      = 1204

	BN_CLICKED    = 0
	WM_CREATE     = 0x0001
	WM_DESTROY    = 0x0002
	WM_SIZE       = 0x0005
	WM_CLOSE      = 0x0010
	WM_COMMAND    = 0x0111
	WM_NOTIFY     = 0x004E
	WM_INITDIALOG = 0x0110

	CBN_SELCHANGE = 1
	CB_ADDSTRING  = 0x0143
	CB_SETCURSEL  = 0x014E
	CB_GETCURSEL  = 0x0147

	LVS_REPORT                   = 0x0001
	LVS_SINGLESEL                = 0x0004
	LVS_SHOWSELALWAYS            = 0x0008
	LVS_EX_FULLROWSELECT         = 0x0020
	LVM_FIRST                    = 0x1000
	LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
	LVM_DELETEALLITEMS           = LVM_FIRST + 9
	LVM_INSERTITEMW              = LVM_FIRST + 77
	LVM_SETITEMW                 = LVM_FIRST + 76
	LVM_INSERTCOLUMNW             = LVM_FIRST + 97
	LVCF_FMT                     = 0x0001
	LVCF_WIDTH                   = 0x0002
	LVCF_TEXT                    = 0x0004
	LVCFMT_LEFT                  = 0
	LVIF_TEXT                    = 0x0001

	BS_PUSHBUTTON       = 0
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	WS_BORDER           = 0x00800000
	CBS_DROPDOWNLIST    = 0x0003
	CW_USEDEFAULT       = 0x80000000

	OFN_EXPLORER         = 0x00080000
	OFN_FILEMUSTEXIST    = 0x00001000
	OFN_ALLOWMULTISELECT = 0x00000200
	OFN_HIDEREADONLY     = 0x00000004
	OFN_ENABLEHOOK       = 0x00000020
)

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type RECT struct{ Left, Top, Right, Bottom int32 }
type lvcw struct {
	Mask    uint32
	Fmt     int32
	Cx      int32
	Text    uintptr
	TextMax int32
	SubItem int32
	Image   int32
	Order   int32
}
type lvitemw struct {
	Mask      uint32
	Item      int32
	SubItem   int32
	State     uint32
	StateMask uint32
	Text      uintptr
	TextMax   int32
	Image     int32
	LParam    uintptr
}
type OPENFILENAMEW struct {
	LStructSize       uint32
	HwndOwner         uintptr
	HInstance         uintptr
	lpstrFilter       uintptr
	lpstrCustomFilter uintptr
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         uintptr
	nMaxFile           uint32
	lpstrFileTitle     uintptr
	nMaxFileTitle      uint32
	lpstrInitialDir    uintptr
	lpstrTitle         uintptr
	Flags              uint32
	nFileOffset        uint16
	nFileExtension     uint16
	lpstrDefExt        uintptr
	lCustData          uintptr
	lpfnHook           uintptr
	lpTemplateName     uintptr
}

var (
	hInstance          uintptr
	hwndMain           uintptr
	hwndGrid           uintptr
	hwndStatus         uintptr
	hwndTotals         uintptr
	hwndMode           uintptr
	filterHandles      = map[int]uintptr{}
	filterLabels       = map[int]uintptr{}
	mainConfig         configData
	mainLines          []Line
	currentView        []Line
	currentFilterCount int
)

var uiColumns = []ColumnDef{
	{Name: "SKU", Width: 90}, {Name: "Descripción", Width: 230}, {Name: "SUM (%) descuento", Width: 125}, {Name: "NETO PK", Width: 110},
	{Name: "UNIDADES", Width: 90}, {Name: "PALL", Width: 75}, {Name: "PK", Width: 75}, {Name: "NETO SO", Width: 110},
	{Name: "TN SO", Width: 95}, {Name: "CMG", Width: 90}, {Name: "PPP SO", Width: 100}, {Name: "ORIGEN", Width: 130},
}

var toolbar = []struct {
	id   int
	text string
}{
	{ID_ABRIR_XLSX, "ABRIR XLSX"}, {ID_TOMAR_EXCEL, "TOMAR EXCEL ABIERTO"}, {ID_RECARGAR, "RECARGAR"}, {ID_COLUMNAS, "COLUMNAS..."},
	{ID_FILTROS_CAB, "FILTROS CABECERA..."}, {ID_EXPORTAR_CSV, "EXPORTAR CSV"}, {ID_SIMULADOR, "SIMULADOR"}, {ID_RESALTAR, "RESALTAR..."},
	{ID_COLOR, "+/- COLOR..."}, {ID_DATOS_CSV, "DATOS CSV..."},
}

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func u16z(s string) []uint16 { return syscall.StringToUTF16(s) }

func crearVentana() uintptr {
	hi, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	hInstance = hi
	className := syscall.StringToUTF16Ptr("GestionSO")
	title := syscall.StringToUTF16Ptr(windowTitle())
	hc, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	hb, _, _ := user32.NewProc("GetSysColorBrush").Call(0, 15)
	wc := WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), LpfnWndProc: syscall.NewCallback(wndProc), HInstance: hInstance, HCursor: hc, HbrBackground: hb, LpszMenuName: nil, LpszClassName: className}
	user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, _ := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), WS_OVERLAPPEDWINDOW|WS_VISIBLE, CW_USEDEFAULT, CW_USEDEFAULT, 1200, 720, 0, 0, hInstance, 0)
	if hwnd == 0 {
		return 0
	}
	hwndMain = hwnd
	setWindowText(hwnd, windowTitle())
	return hwnd
}

func windowTitle() string {
	mode := strings.TrimSpace(strings.TrimPrefix(mainConfig.Mode, "MODO: "))
	if mode == "" {
		mode = "SO RETENIDAS"
	}
	return "Gestion SO V54 - " + mode + " / CSV maestro"
}

func crearControles(hwnd uintptr) {
	initLog()
	inst := hInstance
	for _, b := range toolbar {
		crearBoton(hwnd, b.text, 0, 0, 100, 28, uintptr(b.id))
	}
	combo, _, _ := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("COMBOBOX"))), 0, WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST, 0, 0, 190, 180, hwnd, uintptr(ID_MODO), inst, 0)
	hwndMode = combo
	modes := []string{"MODO: FACTURAS PENDIENTES", "MODO: SO RETENIDAS", "MODO: FACTURAS"}
	selected := 0
	for i, mode := range modes {
		user32.NewProc("SendMessageW").Call(hwndMode, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(mode))))
		if mode == mainConfig.Mode {
			selected = i
		}
	}
	user32.NewProc("SendMessageW").Call(hwndMode, CB_SETCURSEL, uintptr(selected), 0)
	filters := []struct {
		id    int
		label string
	}{{ID_FILTRO_SO, "SO"}, {ID_FILTRO_ESTADO, "Estado"}, {ID_FILTRO_SKU, "SKU"}, {ID_FILTRO_SUMA, "SUMA DE"}, {ID_FILTRO_SDSRP2, "SDSRP2"}}
	for _, f := range filters {
		filterLabels[f.id] = crearLabel(hwnd, f.label, 0, 0, 70, 20)
		h, _, _ := user32.NewProc("CreateWindowExW").Call(WS_BORDER, uintptr(unsafe.Pointer(u16("EDIT"))), 0, WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER, 0, 0, 120, 24, hwnd, uintptr(f.id), inst, 0)
		filterHandles[f.id] = h
	}
	crearBoton(hwnd, "FILTRAR", 0, 0, 85, 26, uintptr(ID_FILTRAR))
	crearBoton(hwnd, "LIMPIAR", 0, 0, 85, 26, uintptr(ID_LIMPIAR))
	grid, _, _ := user32.NewProc("CreateWindowExW").Call(WS_BORDER, uintptr(unsafe.Pointer(u16("SysListView32"))), 0, WS_CHILD|WS_VISIBLE|LVS_REPORT|LVS_SINGLESEL|LVS_SHOWSELALWAYS|WS_BORDER, 0, 120, 800, 400, hwnd, uintptr(ID_GRID), inst, 0)
	hwndGrid = grid
	user32.NewProc("SendMessageW").Call(hwndGrid, LVM_SETEXTENDEDLISTVIEWSTYLE, LVS_EX_FULLROWSELECT, LVS_EX_FULLROWSELECT)
	for i, c := range uiColumns {
		title := c.Name
		if title == "CMG" {
			title = "CMG ▼"
		}
		t := u16(title)
		col := lvcw{Mask: LVCF_FMT | LVCF_WIDTH | LVCF_TEXT, Fmt: LVCFMT_LEFT, Cx: int32(c.Width), Text: uintptr(unsafe.Pointer(t)), TextMax: int32(len(syscall.StringToUTF16(title))), SubItem: int32(i)}
		user32.NewProc("SendMessageW").Call(hwndGrid, LVM_INSERTCOLUMNW, uintptr(i), uintptr(unsafe.Pointer(&col)))
	}
	totalsText := u16("BULTOS 0 | PALLETS 0 | TN 0 | UNIDADES 0\r\nNETO $ 0 | COSTO $ 0 | RESULTADO 0 | CMG 0")
	totals, _, _ := user32.NewProc("CreateWindowExW").Call(WS_BORDER, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(totalsText)), WS_CHILD|WS_VISIBLE|WS_BORDER, 0, 0, 400, 48, hwnd, uintptr(ID_TOTALS), inst, 0)
	hwndTotals = totals
	statusText := u16(BuildStatusBar(mainConfig.Mode, nil, 0, "Detalle de Descuentos Aplicados..."))
	status, _, _ := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(statusText)), WS_CHILD|WS_VISIBLE, 0, 0, 400, 28, hwnd, uintptr(ID_STATUS), inst, 0)
	hwndStatus = status
}

func crearBoton(hwnd uintptr, texto string, x, y, ancho, alto int, id uintptr) uintptr {
	h, _, _ := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("BUTTON"))), uintptr(unsafe.Pointer(u16(texto))), WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto), hwnd, id, hInstance, 0)
	return h
}
func crearLabel(hwnd uintptr, texto string, x, y, ancho, alto int) uintptr {
	h, _, _ := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16(texto))), WS_CHILD|WS_VISIBLE, uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto), hwnd, 0, hInstance, 0)
	return h
}

func redimensionarControles(hwnd uintptr) {
	var r RECT
	user32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right-r.Left), int(r.Bottom-r.Top)
	if w < 900 {
		w = 900
	}
	if h < 500 {
		h = 500
	}
	// INFERENCIA: reservar espacio al selector de modo y compactar la barra
	// evita que los botones se superpongan en la ventana inicial.
	modeW := 180
	gap := 4
	available := w - 16 - modeW - gap
	if available < 400 {
		available = 400
	}
	widths := make([]int, len(toolbar))
	remaining := available - gap*(len(toolbar)-1)
	for i, b := range toolbar {
		bw := len([]rune(b.text))*7 + 20
		if bw < 78 {
			bw = 78
		}
		if bw > 150 {
			bw = 150
		}
		widths[i] = bw
		remaining -= bw
	}
	if remaining < 0 {
		for i := range widths {
			if remaining >= 0 {
				break
			}
			cut := widths[i] - 70
			if cut > -remaining {
				cut = -remaining
			}
			if cut > 0 {
				widths[i] -= cut
				remaining += cut
			}
		}
	}
	x := 8
	for i, b := range toolbar {
		bw := widths[i]
		if ctl := getDlgItem(hwnd, b.id); ctl != 0 {
			user32.NewProc("MoveWindow").Call(ctl, uintptr(x), 6, uintptr(bw), 28, 1)
		}
		x += bw + gap
	}
	if hwndMode != 0 {
		user32.NewProc("MoveWindow").Call(hwndMode, uintptr(w-modeW-8), 6, uintptr(modeW), 28, 1)
	}
	labels := []struct{ id, x, width int }{{ID_FILTRO_SO, 8, 105}, {ID_FILTRO_ESTADO, 120, 135}, {ID_FILTRO_SKU, 263, 125}, {ID_FILTRO_SUMA, 398, 135}, {ID_FILTRO_SDSRP2, 543, 135}}
	for _, f := range labels {
		if lh := filterLabels[f.id]; lh != 0 {
			user32.NewProc("MoveWindow").Call(lh, uintptr(f.x), 42, 65, 20, 1)
		}
		if eh := filterHandles[f.id]; eh != 0 {
			user32.NewProc("MoveWindow").Call(eh, uintptr(f.x), 60, uintptr(f.width), 24, 1)
		}
	}
	if b := getDlgItem(hwnd, ID_FILTRAR); b != 0 {
		user32.NewProc("MoveWindow").Call(b, 690, 58, 85, 27, 1)
	}
	if b := getDlgItem(hwnd, ID_LIMPIAR); b != 0 {
		user32.NewProc("MoveWindow").Call(b, 780, 58, 85, 27, 1)
	}
	gridY, statusH, totalsH := 92, 28, 48
	gridH := h - gridY - statusH - totalsH - 8
	if gridH < 100 {
		gridH = 100
	}
	if hwndGrid != 0 {
		user32.NewProc("MoveWindow").Call(hwndGrid, 8, uintptr(gridY), uintptr(w-16), uintptr(gridH), 1)
	}
	if hwndTotals != 0 {
		user32.NewProc("MoveWindow").Call(hwndTotals, 8, uintptr(h-statusH-totalsH), uintptr(w-16), uintptr(totalsH), 1)
	}
	if hwndStatus != 0 {
		user32.NewProc("MoveWindow").Call(hwndStatus, 8, uintptr(h-statusH), uintptr(w-16), uintptr(statusH), 1)
	}
}

func getDlgItem(hwnd uintptr, id int) uintptr {
	h, _, _ := user32.NewProc("GetDlgItem").Call(hwnd, uintptr(id))
	return h
}
func windowText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := user32.NewProc("GetWindowTextLengthW").Call(hwnd)
	b := make([]uint16, int(n)+1)
	user32.NewProc("GetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(&b[0])), n+1)
	return syscall.UTF16ToString(b)
}
func setWindowText(hwnd uintptr, text string) {
	if hwnd != 0 {
		user32.NewProc("SetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(u16(text))))
	}
}

func handleCommand(hwnd, wParam, lParam uintptr) uintptr {
	_ = lParam
	id := int(wParam & 0xffff)
	notify := uint16((wParam >> 16) & 0xffff)
	if notify != BN_CLICKED && id != ID_MODO {
		return 0
	}
	switch id {
	case ID_ABRIR_XLSX:
		openXLSXDialog(hwnd)
	case ID_RECARGAR:
		reloadMainView(hwnd)
	case ID_EXPORTAR_CSV:
		exportCurrentView()
	case ID_FILTRAR:
		applyHeaderFilters()
		updateMainView(hwnd)
	case ID_LIMPIAR:
		clearHeaderFilters()
		updateMainView(hwnd)
	case ID_MODO:
		if notify == CBN_SELCHANGE {
			saveSelectedMode()
			currentView = BuildFilteredSortedView(mainLines, "")
			updateMainView(hwnd)
		}
	case ID_TOMAR_EXCEL, ID_COLUMNAS, ID_FILTROS_CAB, ID_SIMULADOR, ID_RESALTAR, ID_COLOR, ID_DATOS_CSV:
		logf("UI stub button id=%d", id)
	}
	return 0
}
func saveSelectedMode() {
	if hwndMode == 0 {
		return
	}
	idx, _, _ := user32.NewProc("SendMessageW").Call(hwndMode, CB_GETCURSEL, 0, 0)
	modes := []string{"MODO: FACTURAS PENDIENTES", "MODO: SO RETENIDAS", "MODO: FACTURAS"}
	if int(idx) >= 0 && int(idx) < len(modes) {
		mainConfig.Mode = modes[int(idx)]
		_ = SaveConfig(mainConfig)
	}
}
func applyHeaderFilters() {
	filters := map[string]string{}
	currentFilterCount = 0
	for _, id := range []int{ID_FILTRO_SO, ID_FILTRO_ESTADO, ID_FILTRO_SKU, ID_FILTRO_SUMA, ID_FILTRO_SDSRP2} {
		v := strings.TrimSpace(windowText(filterHandles[id]))
		if v != "" {
			filters[filterName(id)] = v
			currentFilterCount++
		}
	}
	currentView = BuildFilteredSortedViewByHeaders(mainLines, filters)
}
func filterName(id int) string {
	switch id {
	case ID_FILTRO_SO:
		return "SO"
	case ID_FILTRO_ESTADO:
		return "Estado"
	case ID_FILTRO_SKU:
		return "SKU"
	case ID_FILTRO_SUMA:
		return "SUMA DE"
	case ID_FILTRO_SDSRP2:
		return "SDSRP2"
	}
	return ""
}
func clearHeaderFilters() {
	for _, id := range []int{ID_FILTRO_SO, ID_FILTRO_ESTADO, ID_FILTRO_SKU, ID_FILTRO_SUMA, ID_FILTRO_SDSRP2} {
		setWindowText(filterHandles[id], "")
	}
	currentFilterCount = 0
	currentView = append([]Line(nil), mainLines...)
}
func updateMainView(hwnd uintptr) {
	if currentView == nil {
		currentView = append([]Line(nil), mainLines...)
	}
	refreshGrid(currentView)
	updateTotals(currentView)
	updateStatus(hwnd, currentView)
	setWindowText(hwnd, windowTitle())
}
func reloadMainView(hwnd uintptr) {
	logf("RECARGAR")
	applyHeaderFilters()
	updateMainView(hwnd)
}
func exportCurrentView() {
	path := filepathJoin(os.TempDir(), "GestionSO-export.csv")
	if err := exportVisible(currentView, path); err != nil {
		logf("exportVisible error: %v", err)
		return
	}
	logf("EXPORTAR CSV path=%q lines=%d", path, len(currentView))
}

func resolveUIValue(l Line, name string) string {
	if v := fieldValue(l, name); v != "" {
		return v
	}
	aliases := map[string][]string{
		"SKU":               {"sku"},
		"Descripción":       {"descrip", "descripcion", "producto"},
		"SUM (%) descuento": {"sum", "descuento", "% descuento"},
		"NETO PK":           {"neto pk", "neto_pk", "netopk"},
		"UNIDADES":          {"unidades", "unidad", "cantidad"},
		"PALL":              {"pall", "pallets", "pallet"},
		"PK":                {"pk"},
		"NETO SO":           {"neto so", "neto_so", "netoso"},
		"TN SO":             {"tn so", "tn", "tonelada"},
		"CMG":               {"cmg", "margen"},
		"PPP SO":            {"ppp so", "ppp", "precio promedio"},
		"ORIGEN":            {"origen"},
	}
	for _, a := range aliases[name] {
		if v := findAnyValue(l, a); v != "" {
			return v
		}
	}
	return ""
}

func refreshGrid(lines []Line) {
	if hwndGrid == 0 {
		return
	}
	user32.NewProc("SendMessageW").Call(hwndGrid, LVM_DELETEALLITEMS, 0, 0)
	renderLines := appendGridSubtotals(lines)
	for rowIndex, l := range renderLines {
		values := make([]string, len(uiColumns))
		for colIndex, c := range uiColumns {
			values[colIndex] = resolveUIValue(l, c.Name)
		}
		first := u16(values[0])
		item := lvitemw{Mask: LVIF_TEXT, Item: int32(rowIndex), SubItem: 0, Text: uintptr(unsafe.Pointer(first)), TextMax: int32(len(syscall.StringToUTF16(values[0])))}
		user32.NewProc("SendMessageW").Call(hwndGrid, LVM_INSERTITEMW, 0, uintptr(unsafe.Pointer(&item)))
		for colIndex := 1; colIndex < len(values); colIndex++ {
			setGridCell(rowIndex, colIndex, values[colIndex])
		}
	}
}

// INFERENCIA: la captura muestra una fila SUBTOTAL SO después de cada grupo.
// Los campos numéricos se obtienen de CalculateSOSubtotals; no se afirma
// reproducir la fórmula comercial del binario original.
func appendGridSubtotals(lines []Line) []Line {
	if len(lines) == 0 {
		return nil
	}
	out := make([]Line, 0, len(lines)+len(GroupLines(lines)))
	var group []Line
	currentSO := ""
	flush := func() {
		if len(group) == 0 {
			return
		}
		out = append(out, group...)
		if currentSO != "" {
			out = append(out, makeSubtotalLine(currentSO, group))
		}
	}
	for _, l := range lines {
		so := strings.TrimSpace(fieldValue(l, "SO"))
		if so == "" {
			if len(group) > 0 {
				flush()
				group = nil
				currentSO = ""
			}
			out = append(out, l)
			continue
		}
		if currentSO != "" && so != currentSO {
			flush()
			group = nil
		}
		currentSO = so
		group = append(group, l)
	}
	flush()
	return out
}

func makeSubtotalLine(so string, group []Line) Line {
	values := map[string]string{"SKU": fmt.Sprintf("SUBTOTAL SO %s", so)}
	if len(group) == 0 {
		return Line{Values: values, Source: "subtotal", RowNumber: -1}
	}
	first := group[0]
	ret := 0
	estado := ""
	for _, l := range group {
		state := strings.ToUpper(strings.TrimSpace(fieldValue(l, "Estado")))
		if state == "RETENIDA" || state == "RETENIDAS" {
			ret++
		}
		if estado == "" {
			estado = fieldValue(l, "Estado")
		}
	}
	cod := findAnyValue(first, "cod")
	clienteKey := findFieldKey(first, "cliente")
	clienteValue := fieldValue(first, clienteKey)
	pct := resolveUIValue(first, "SUM (%) descuento")
	values["Descripción"] = fmt.Sprintf("RET %d | %s | %s | %s", ret, estado, cod, clienteValue)
	values["SUM (%) descuento"] = pct
	sums := CalculateSOSubtotals(group)
	for _, field := range []string{"BULTOS", "PALL", "PK", "UNIDADES", "NETO PK", "NETO SO", "TN SO", "CMG", "PPP SO", "RESULTADO"} {
		if v, ok := sums[so][field]; ok {
			values[field] = displayNumber(v)
		}
	}
	return Line{Values: values, Source: "subtotal", RowNumber: -1}
}

func setGridCell(row, col int, text string) {
	t := u16(text)
	item := lvitemw{Mask: LVIF_TEXT, Item: int32(row), SubItem: int32(col), Text: uintptr(unsafe.Pointer(t)), TextMax: int32(len(syscall.StringToUTF16(text)))}
	user32.NewProc("SendMessageW").Call(hwndGrid, LVM_SETITEMW, 0, uintptr(unsafe.Pointer(&item)))
}

func updateTotals(lines []Line) {
	if hwndTotals == 0 {
		return
	}
	var bultos, pallets, tn, unidades, neto, costo, resultado, cmg float64
	for _, l := range lines {
		bultos += numericByNames(l, "bultos", "bulto")
		pallets += numericByNames(l, "pall", "pallet")
		tn += numericByNames(l, "tn so", "tn")
		unidades += numericByNames(l, "unidades", "unidad", "cantidad")
		neto += numericByNames(l, "neto so", "neto_so", "netoso")
		costo += numericByNames(l, "costo")
		resultado += numericByNames(l, "resultado")
		cmg += numericByNames(l, "cmg", "margen")
	}
	setWindowText(hwndTotals, fmt.Sprintf("BULTOS %s | PALLETS %s | TN %s | UNIDADES %s\r\nNETO $ %s | COSTO $ %s | RESULTADO %s | CMG %s", displayNumber(bultos), displayNumber(pallets), displayNumber(tn), displayNumber(unidades), displayNumber(neto), displayNumber(costo), displayNumber(resultado), displayNumber(cmg)))
}

func displayNumber(v float64) string {
	if v == 0 {
		return "0"
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
}

func updateStatus(hwnd uintptr, lines []Line) {
	_ = hwnd
	setWindowText(hwndStatus, BuildStatusBar(mainConfig.Mode, lines, currentFilterCount, "Detalle de Descuentos Aplicados..."))
}

func openXLSXDialog(owner uintptr) {
	panicGuard(func() {
		logf("openXLSXDialog start owner=%x", owner)
		files := pickMultipleXLSX(owner)
		logf("openXLSXDialog: picked %d files", len(files))
		if len(files) == 0 {
			return
		}
		rows, err := mergeXLSX(files)
		if err != nil {
			logf("mergeXLSX error: %v", err)
			return
		}
		mainLines = BuildLines(rows, "xlsx")
		currentView = append([]Line(nil), mainLines...)
		updateMainView(owner)
		feedEngineFile(owner, files[0])
		logf("openXLSXDialog end lines=%d", len(mainLines))
	})
}

func pickMultipleXLSX(owner uintptr) []string {
	buf := make([]uint16, 32768)
	filter := u16z("Archivos XLSX (*.xlsx)\x00*.xlsx\x00Todos los archivos (*.*)\x00*.*\x00\x00")
	ofn := OPENFILENAMEW{LStructSize: uint32(unsafe.Sizeof(OPENFILENAMEW{})), HwndOwner: owner, lpstrFilter: uintptr(unsafe.Pointer(&filter[0])), lpstrFile: uintptr(unsafe.Pointer(&buf[0])), nMaxFile: uint32(len(buf)), lpstrTitle: uintptr(unsafe.Pointer(u16("ABRIR XLSX"))), Flags: OFN_EXPLORER | OFN_FILEMUSTEXIST | OFN_ALLOWMULTISELECT | OFN_HIDEREADONLY | OFN_ENABLEHOOK, lpfnHook: syscall.NewCallback(multiSelectHook)}
	ret, _, _ := comdlg32.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		errCode, _, _ := comdlg32.NewProc("CommDlgExtendedError").Call()
		if errCode != 0 {
			logf("GetOpenFileNameW failed code=%d", errCode)
		}
		return nil
	}
	return parseMultiSelectBuffer(buf)
}

func parseMultiSelectBuffer(buf []uint16) []string {
	if len(buf) == 0 || buf[0] == 0 {
		return nil
	}
	first := syscall.UTF16ToString(buf)
	end := len(first)
	if end+1 >= len(buf) || buf[end+1] == 0 {
		return []string{first}
	}
	dir := first
	result := []string{}
	i := end + 1
	for i < len(buf) && buf[i] != 0 {
		j := i
		for j < len(buf) && buf[j] != 0 {
			j++
		}
		name := syscall.UTF16ToString(buf[i:j])
		if name != "" {
			result = append(result, filepathJoin(dir, name))
		}
		i = j + 1
	}
	return result
}

func filepathJoin(a, b string) string {
	if strings.HasSuffix(a, "\\") || strings.HasSuffix(a, "/") {
		return a + b
	}
	return a + "\\" + b
}

func multiSelectHook(hwnd, msg, wParam, lParam uintptr) uintptr {
	_ = wParam
	_ = lParam
	if msg == WM_INITDIALOG {
		logf("ABRIR XLSX multi-select hook installed hwnd=%x", hwnd)
	}
	return 0
}

func findWindowByTitles(titles []string) uintptr {
	for _, t := range titles {
		h, _, _ := user32.NewProc("FindWindowW").Call(0, uintptr(unsafe.Pointer(u16(t))))
		if h != 0 {
			return h
		}
	}
	return 0
}

func enumTopWindows(fn func(uintptr) bool) {
	cb := syscall.NewCallback(func(hwnd, lParam uintptr) uintptr {
		_ = lParam
		if fn(hwnd) {
			return 1
		}
		return 0
	})
	user32.NewProc("EnumWindows").Call(cb, 0)
}

func enumChildren(hwnd uintptr, fn func(uintptr) bool) {
	cb := syscall.NewCallback(func(child, lParam uintptr) uintptr {
		_ = lParam
		if fn(child) {
			return 1
		}
		return 0
	})
	user32.NewProc("EnumChildWindows").Call(hwnd, cb, 0)
}

func findChildByText(hwnd uintptr, text string) uintptr {
	var found uintptr
	enumChildren(hwnd, func(c uintptr) bool {
		if windowText(c) == text {
			found = c
			return false
		}
		return true
	})
	return found
}

func findFirstEdit(hwnd uintptr) uintptr {
	var found uintptr
	enumChildren(hwnd, func(c uintptr) bool {
		if strings.EqualFold(getClassName(c), "EDIT") {
			found = c
			return false
		}
		return true
	})
	return found
}

func findDialogUnder(hwnd uintptr) uintptr {
	// INFERENCIA/PENDIENTE: no se puede recuperar con certeza la relación
	// entre el owner y el diálogo del selector V54. No devolver una ventana
	// arbitraria evita actuar sobre una ventana ajena.
	_ = hwnd
	return 0
}

func getClassName(hwnd uintptr) string {
	b := make([]uint16, 256)
	n, _, _ := user32.NewProc("GetClassNameW").Call(hwnd, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(b[:n])
}

func repositionOverlay(hwnd uintptr) { _ = hwnd }

func wndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		crearControles(hwnd)
		redimensionarControles(hwnd)
		return 0
	case WM_SIZE:
		redimensionarControles(hwnd)
		return 0
	case WM_COMMAND:
		panicGuard(func() { handleCommand(hwnd, wParam, lParam) })
		return 0
	case WM_NOTIFY:
		return handleNotify(hwnd, wParam, lParam)
	case WM_CLOSE:
		user32.NewProc("DestroyWindow").Call(hwnd)
		return 0
	case WM_DESTROY:
		user32.NewProc("PostQuitMessage").Call(0)
		return 0
	}
	r, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, msg, wParam, lParam)
	return r
}

func handleNotify(hwnd, wParam, lParam uintptr) uintptr {
	return handleMainNotify(hwnd, wParam, lParam)
}

func handleMainNotify(hwnd, wParam, lParam uintptr) uintptr {
	_ = hwnd
	_ = wParam
	_ = lParam
	return 0
}

// HECHO VERIFICADO: el binario contiene main.feedEngineFile.
// INFERENCIA: el contrato interno no es recuperable sin GestionSO-V54-engine.exe.
func feedEngineFile(owner uintptr, file string) {
	engine := os.Getenv("GESTIONSO_V54_ENGINE")
	if engine == "" {
		logf("feedEngineFile owner=%x file=%q engine=not-configured", owner, file)
		return
	}
	logf("feedEngineFile owner=%x file=%q engine=%s", owner, file, engine)
}
