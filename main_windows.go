//go:build windows

package main

import (
	"os"
	"strings"
	"syscall"
	"unsafe"
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

type RECT struct { Left, Top, Right, Bottom int32 }

type lvcw struct {
	Mask      uint32
	Fmt       int32
	Cx        int32
	Text      uintptr
	TextMax   int32
	SubItem   int32
	Image     int32
	Order     int32
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

type openfilename struct {
	LStructSize uint32
	HwndOwner   uintptr
	HInstance   uintptr
	Filter      uintptr
	CustomFilter uintptr
	MaxCustFilter uint32
	FilterIndex uint32
	File        uintptr
	MaxFile     uint32
	FileTitle   uintptr
	MaxFileTitle uint32
	InitialDir  uintptr
	Title       uintptr
	Flags       uint32
	FileOffset  uint16
	FileExtension uint16
	DefExt      uintptr
	CustData    uintptr
	Hook        uintptr
	TemplateName uintptr
	Reserved    uintptr
	Reserved2   uint32
	FlagsEx     uint32
}

var (
	hInstance  uintptr
	hwndGrid   uintptr
	hwndStatus uintptr
	hwndTotals uintptr
	hwndMode   uintptr
	filterHandles = map[int]uintptr{}
)

const (
	WM_NOTIFY       = 0x004E
	CBN_SELCHANGE   = 1
	CB_ADDSTRING    = 0x0143
	CB_SETCURSEL    = 0x014E
	CB_GETCURSEL    = 0x0147
	CB_GETLBTEXTLEN = 0x0149
	CB_GETLBTEXT    = 0x0148
	LVS_REPORT      = 0x0001
	LVS_SINGLESEL   = 0x0004
	LVS_SHOWSELALWAYS = 0x0008
	LVS_EX_FULLROWSELECT = 0x0020
	LVM_FIRST       = 0x1000
	LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
	LVM_DELETEALLITEMS = LVM_FIRST + 9
	LVM_INSERTITEMW = LVM_FIRST + 77
	LVM_SETITEMW    = LVM_FIRST + 76
	LVM_INSERTCOLUMNW = LVM_FIRST + 97
	LVCF_FMT        = 0x0001
	LVCF_WIDTH      = 0x0002
	LVCF_TEXT       = 0x0004
	LVCFMT_LEFT     = 0
	LVIF_TEXT       = 0x0001
	BS_PUSHBUTTON   = 0
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE      = 0x10000000
	WS_CHILD        = 0x40000000
	WS_TABSTOP      = 0x00010000
	WS_BORDER       = 0x00800000
	CBS_DROPDOWNLIST = 0x0003
	CW_USEDEFAULT   = 0x80000000
	SW_SHOW         = 5
	OFN_EXPLORER    = 0x00080000
	OFN_FILEMUSTEXIST = 0x00001000
	OFN_ALLOWMULTISELECT = 0x00000200
	OFN_HIDEREADONLY = 0x00000004
	OFN_ENABLEHOOK  = 0x00000020

	ID_TOMAR_EXCEL = 1007
	ID_FILTROS_CAB = 1008
	ID_SIMULADOR   = 1009
	ID_RESALTAR    = 1010
	ID_COLOR       = 1011
	ID_DATOS_CSV   = 1012
	ID_PAGE_SIZE   = 1013
	ID_MODO        = 1014
	ID_FILTRO_SO   = 1101
	ID_FILTRO_ESTADO = 1102
	ID_FILTRO_SKU  = 1103
	ID_FILTRO_SUMA = 1104
	ID_FILTRO_SDSRP2 = 1105
	ID_GRID        = 1201
	ID_STATUS      = 1202
	ID_TOTALS      = 1203
)

var uiColumns = []ColumnDef{
	{Name: "SKU", Width: 90},
	{Name: "Descripción", Width: 230},
	{Name: "SUM (%) descuento", Width: 125},
	{Name: "NETO PK", Width: 110},
	{Name: "UNIDADES", Width: 90},
	{Name: "PALL", Width: 75},
	{Name: "PK", Width: 75},
	{Name: "NETO SO", Width: 110},
	{Name: "TN SO", Width: 95},
	{Name: "CMG", Width: 90},
	{Name: "PPP SO", Width: 100},
	{Name: "ORIGEN", Width: 130},
}

var toolbar = []struct { id int; text string }{
	{ID_ABRIR_XLSX, "ABRIR XLSX"},
	{ID_TOMAR_EXCEL, "TOMAR EXCEL ABIERTO"},
	{ID_RECARGAR, "RECARGAR"},
	{ID_COLUMNAS, "COLUMNAS..."},
	{ID_FILTROS_CAB, "FILTROS CABECERA..."},
	{ID_EXPORTAR_CSV, "EXPORTAR CSV"},
	{ID_SIMULADOR, "SIMULADOR"},
	{ID_RESALTAR, "RESALTAR..."},
	{ID_COLOR, "+/- COLOR..."},
	{ID_DATOS_CSV, "DATOS CSV..."},
}

var mainConfig configData
var mainLines []Line
var currentView []Line
var currentFilterCount int

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func u16z(s string) []uint16 { return syscall.StringToUTF16(s) }

func crearVentana() uintptr {
	hi, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	hInstance = hi
	className := syscall.StringToUTF16Ptr("GestionSO")
	hc, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	hb, _, _ := user32.NewProc("GetSysColorBrush").Call(15)
	wc := WNDCLASSEX{
		CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc: syscall.NewCallback(wndProc),
		HInstance: hInstance,
		HCursor: hc,
		HbrBackground: hb,
		LpszMenuName: nil,
		LpszClassName: className,
	}
	user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, _ := user32.NewProc("CreateWindowExW").Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Gestion SO V54 - SO RETENIDAS / CSV maestro"))),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		CW_USEDEFAULT, CW_USEDEFAULT, 1250, 780,
		0, 0, hInstance, 0,
	)
	if hwnd != 0 {
		mainConfig = LoadConfig()
		if mainConfig.Mode == "" { mainConfig.Mode = "MODO: SO RETENIDAS"; _ = SaveConfig(mainConfig) }
		user32.NewProc("SetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(u16(windowTitle()))))
		crearControles(hwnd)
		redimensionarControles(hwnd)
	}
	return hwnd
}

func windowTitle() string {
	mode := strings.TrimSpace(mainConfig.Mode)
	if strings.HasPrefix(mode, "MODO: ") { mode = strings.TrimSpace(strings.TrimPrefix(mode, "MODO: ")) }
	if mode == "" { mode = "SO RETENIDAS" }
	return "Gestion SO V54 - " + mode + " / CSV maestro"
}

func crearVentanaPrincipal() uintptr { return crearVentana() }

func crearControles(hwnd uintptr) {
	initLog()
	for _, b := range toolbar { crearBoton(hwnd, b.text, 0, 0, 100, 28, uintptr(b.id)) }

	inst := hInstance
	hwndMode, _, _ = user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("COMBOBOX"))), 0, WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST, 0, 0, 190, 180, hwnd, uintptr(ID_MODO), inst, 0)
	modes := []string{"MODO: FACTURAS PENDIENTES", "MODO: SO RETENIDAS", "MODO: FACTURAS"}
	selected := 0
	for i, mode := range modes { user32.NewProc("SendMessageW").Call(hwndMode, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(mode)))); if mode == mainConfig.Mode { selected = i } }
	user32.NewProc("SendMessageW").Call(hwndMode, CB_SETCURSEL, uintptr(selected), 0)

	for _, f := range []struct{id int; label string}{
		{ID_FILTRO_SO, "SO"}, {ID_FILTRO_ESTADO, "Estado"}, {ID_FILTRO_SKU, "SKU"}, {ID_FILTRO_SUMA, "SUMA DE"}, {ID_FILTRO_SDSRP2, "SDSRP2"},
	} {
		crearLabel(hwnd, f.label, 0, 0, 70, 20)
		h, _, _ := user32.NewProc("CreateWindowExW").Call(WS_BORDER, uintptr(unsafe.Pointer(u16("EDIT"))), 0, WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER, 0, 0, 120, 24, hwnd, uintptr(f.id), inst, 0)
		filterHandles[f.id] = h
	}
	crearBoton(hwnd, "FILTRAR", 0, 0, 85, 26, uintptr(ID_FILTRAR))
	crearBoton(hwnd, "LIMPIAR", 0, 0, 85, 26, uintptr(ID_LIMPIAR))

	hwndGrid, _, _ = user32.NewProc("CreateWindowExW").Call(WS_BORDER, uintptr(unsafe.Pointer(u16("SysListView32"))), 0, WS_CHILD|WS_VISIBLE|LVS_REPORT|LVS_SINGLESEL|LVS_SHOWSELALWAYS|WS_BORDER, 0, 0, 100, 100, hwnd, uintptr(ID_GRID), inst, 0)
	user32.NewProc("SendMessageW").Call(hwndGrid, LVM_SETEXTENDEDLISTVIEWSTYLE, LVS_EX_FULLROWSELECT, LVS_EX_FULLROWSELECT)
	for i, c := range uiColumns {
		title := c.Name
		if title == "CMG" { title = "CMG ▼" }
		t := u16(title)
		col := lvcw{Mask: LVCF_FMT|LVCF_WIDTH|LVCF_TEXT, Fmt: LVCFMT_LEFT, Cx: int32(c.Width), Text: uintptr(unsafe.Pointer(t)), TextMax: int32(len(title)+1), SubItem: int32(i)}
		user32.NewProc("SendMessageW").Call(hwndGrid, LVM_INSERTCOLUMNW, uintptr(i), uintptr(unsafe.Pointer(&col)))
	}

	hwndTotals, _, _ = user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16("BULTOS 0 | PALLETS 0 | TN 0 | UNIDADES 0\r\nNETO $ 0 | COSTO $ 0 | RESULTADO 0 | CMG 0"))), WS_CHILD|WS_VISIBLE, 0, 0, 700, 42, hwnd, uintptr(ID_TOTALS), inst, 0)
	hwndStatus, _, _ = user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16(BuildStatusBar(mainConfig.Mode, nil, 0, "Detalle de Descuentos Aplicados...")))), WS_CHILD|WS_VISIBLE, 0, 0, 700, 24, hwnd, uintptr(ID_STATUS), inst, 0)
}

func crearBoton(hwnd uintptr, texto string, x, y, ancho, alto int, id uintptr) {
	user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("BUTTON"))), uintptr(unsafe.Pointer(u16(texto))), WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto), hwnd, id, hInstance, 0)
}
func crearLabel(hwnd uintptr, texto string, x, y, ancho, alto int) {
	user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16(texto))), WS_CHILD|WS_VISIBLE, uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto), hwnd, 0, hInstance, 0)
}

func redimensionarControles(hwnd uintptr) {
	var r RECT
	user32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w, h := int(r.Right-r.Left), int(r.Bottom-r.Top)
	if w < 900 { w = 900 }
	if h < 500 { h = 500 }

	x := 10
	for _, b := range toolbar {
		bw := len([]rune(b.text))*8 + 22
		if bw < 85 { bw = 85 }
		if bw > 160 { bw = 160 }
		if ctl := getDlgItem(hwnd, b.id); ctl != 0 { user32.NewProc("MoveWindow").Call(ctl, uintptr(x), 8, uintptr(bw), 28, 1) }
		x += bw + 5
	}
	if hwndMode != 0 { user32.NewProc("MoveWindow").Call(hwndMode, uintptr(x), 8, 190, 28, 1) }

	labels := []struct{id int; x int; width int}{{ID_FILTRO_SO,10,105},{ID_FILTRO_ESTADO,125,135},{ID_FILTRO_SKU,270,125},{ID_FILTRO_SUMA,405,135},{ID_FILTRO_SDSRP2,550,135}}
	for _, f := range labels {
		if e := filterHandles[f.id]; e != 0 { user32.NewProc("MoveWindow").Call(e, uintptr(f.x+55), 45, uintptr(f.width), 24, 1) }
	}
	// Labels were created without IDs; their position is intentionally approximate.
	if b := getDlgItem(hwnd, ID_FILTRAR); b != 0 { user32.NewProc("MoveWindow").Call(b, 695, 44, 85, 26, 1) }
	if b := getDlgItem(hwnd, ID_LIMPIAR); b != 0 { user32.NewProc("MoveWindow").Call(b, 785, 44, 85, 26, 1) }

	gridY := 78
	statusH := 26
	totalsH := 42
	gridH := h-gridY-statusH-totalsH-12
	if gridH < 120 { gridH = 120 }
	if hwndGrid != 0 { user32.NewProc("MoveWindow").Call(hwndGrid, 10, uintptr(gridY), uintptr(w-20), uintptr(gridH), 1) }
	if hwndTotals != 0 { user32.NewProc("MoveWindow").Call(hwndTotals, 10, uintptr(gridY+gridH+3), uintptr(w-20), uintptr(totalsH), 1) }
	if hwndStatus != 0 { user32.NewProc("MoveWindow").Call(hwndStatus, 10, uintptr(h-statusH), uintptr(w-20), uintptr(statusH), 1) }
}

func getDlgItem(hwnd uintptr, id int) uintptr { h, _, _ := user32.NewProc("GetDlgItem").Call(hwnd, uintptr(id)); return h }
func windowText(hwnd uintptr) string { if hwnd == 0 { return "" }; n, _, _ := user32.NewProc("GetWindowTextLengthW").Call(hwnd); b := make([]uint16, n+1); user32.NewProc("GetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(&b[0])), n+1); return syscall.UTF16ToString(b) }
func setWindowText(hwnd uintptr, text string) { if hwnd != 0 { user32.NewProc("SetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(u16(text)))) } }

func handleCommand(hwnd, wParam, lParam uintptr) uintptr {
	id := int(wParam & 0xffff)
	notify := uint16((wParam >> 16) & 0xffff)
	switch id {
	case ID_ABRIR_XLSX:
		if notify == BN_CLICKED { openXLSXDialog(hwnd) }
	case ID_RECARGAR:
		if notify == BN_CLICKED { updateMainView(hwnd); logf("UI stub: RECARGAR") }
	case ID_COLUMNAS:
		if notify == BN_CLICKED { logf("UI stub: COLUMNAS...") }
	case ID_FILTRAR:
		if notify == BN_CLICKED { applyHeaderFilters(hwnd) }
	case ID_LIMPIAR:
		if notify == BN_CLICKED { clearHeaderFilters(hwnd) }
	case ID_EXPORTAR_CSV:
		if notify == BN_CLICKED { logf("UI stub: EXPORTAR CSV") }
	case ID_TOMAR_EXCEL:
		if notify == BN_CLICKED { logf("UI stub: TOMAR EXCEL ABIERTO (COM pendiente)") }
	case ID_FILTROS_CAB:
		if notify == BN_CLICKED { logf("UI stub: FILTROS CABECERA...") }
	case ID_SIMULADOR:
		if notify == BN_CLICKED { openSimulator(hwnd); logf("UI stub: SIMULADOR") }
	case ID_RESALTAR:
		if notify == BN_CLICKED { logf("UI stub: RESALTAR...") }
	case ID_COLOR:
		if notify == BN_CLICKED { logf("UI stub: +/- COLOR...") }
	case ID_DATOS_CSV:
		if notify == BN_CLICKED { logf("UI stub: DATOS CSV...") }
	case ID_MODO:
		if notify == CBN_SELCHANGE { saveSelectedMode(hwnd) }
	}
	return 0
}

func saveSelectedMode(hwnd uintptr) {
	if hwndMode == 0 { return }
	idx, _, _ := user32.NewProc("SendMessageW").Call(hwndMode, CB_GETCURSEL, 0, 0)
	modes := []string{"MODO: FACTURAS PENDIENTES", "MODO: SO RETENIDAS", "MODO: FACTURAS"}
	if int(idx) < 0 || int(idx) >= len(modes) { return }
	mainConfig.Mode = modes[int(idx)]
	if err := SaveConfig(mainConfig); err != nil { logf("SaveConfig Mode error: %v", err) }
	setWindowText(hwnd, windowTitle())
	updateMainView(hwnd)
}

func applyHeaderFilters(hwnd uintptr) {
	filters := map[string]string{}
	for _, id := range []int{ID_FILTRO_SO, ID_FILTRO_ESTADO, ID_FILTRO_SKU, ID_FILTRO_SUMA, ID_FILTRO_SDSRP2} {
		v := strings.TrimSpace(windowText(filterHandles[id]))
		if v != "" { filters[strings.TrimSpace(filterName(id))] = v; currentFilterCount++ }
	}
	currentView = BuildFilteredSortedViewByHeaders(mainLines, filters)
	refreshGrid(currentView)
	updateStatus(hwnd, currentView)
}
func filterName(id int) string { switch id { case ID_FILTRO_SO:return "SO"; case ID_FILTRO_ESTADO:return "Estado"; case ID_FILTRO_SKU:return "SKU"; case ID_FILTRO_SUMA:return "SUMA DE"; case ID_FILTRO_SDSRP2:return "SDSRP2" }; return "" }
func clearHeaderFilters(hwnd uintptr) { for _, id := range []int{ID_FILTRO_SO,ID_FILTRO_ESTADO,ID_FILTRO_SKU,ID_FILTRO_SUMA,ID_FILTRO_SDSRP2} { setWindowText(filterHandles[id], "") }; currentFilterCount=0; currentView=BuildFilteredSortedViewByHeaders(mainLines,nil); refreshGrid(currentView); updateStatus(hwnd,currentView) }

func updateMainView(hwnd uintptr) {
	currentView = BuildFilteredSortedViewByHeaders(mainLines, nil)
	refreshGrid(currentView)
	updateStatus(hwnd, currentView)
	setWindowText(hwnd, windowTitle())
}

func resolveUIValue(l Line, name string) string {
	aliases := map[string][]string{
		"SKU": {"sku"}, "Descripción": {"descrip", "descripcion", "producto"}, "SUM (%) descuento": {"sum", "descuento", "% descuento"},
		"NETO PK": {"neto pk"}, "UNIDADES": {"unidades", "unidad", "cantidad"}, "PALL": {"pall", "pallet"}, "PK": {"pk"},
		"NETO SO": {"neto so"}, "TN SO": {"tn so", "tn", "tonelada"}, "CMG": {"cmg", "margen"}, "PPP SO": {"ppp so", "ppp"}, "ORIGEN": {"origen"},
	}
	for _, a := range aliases[name] { for k,v := range l.Values { lk:=strings.ToLower(strings.TrimSpace(k)); if lk==a || strings.Contains(lk,a) { return v } } }
	return ""
}

func refreshGrid(lines []Line) {
	if hwndGrid == 0 { return }
	user32.NewProc("SendMessageW").Call(hwndGrid, LVM_DELETEALLITEMS, 0, 0)
	for _, l := range lines {
		row := insertGridRow(resolveUIValue(l, uiColumns[0].Name))
		for i := 1; i < len(uiColumns); i++ { setGridCell(row, i, resolveUIValue(l, uiColumns[i].Name)) }
	}
	for _, s := range CalculateSOSubtotals(lines) {
		row := insertGridRow(FormatSOSubtotal(s))
		for i := 1; i < len(uiColumns); i++ { setGridCell(row, i, "") }
	}
}
func insertGridRow(text string) int { t:=u16(text);it:=lvitemw{Mask:LVIF_TEXT,Item:0,SubItem:0,Text:uintptr(unsafe.Pointer(t)),TextMax:int32(len(text)+1)};r,_,_:=user32.NewProc("SendMessageW").Call(hwndGrid,LVM_INSERTITEMW,0,uintptr(unsafe.Pointer(&it)));return int(r) }
func setGridCell(row,col int,text string) { t:=u16(text);it:=lvitemw{Mask:LVIF_TEXT,Item:int32(row),SubItem:int32(col),Text:uintptr(unsafe.Pointer(t)),TextMax:int32(len(text)+1)};user32.NewProc("SendMessageW").Call(hwndGrid,LVM_SETITEMW,0,uintptr(unsafe.Pointer(&it))) }

func updateStatus(hwnd uintptr, lines []Line) { setWindowText(hwndStatus, BuildStatusBar(mainConfig.Mode, lines, currentFilterCount, "Detalle de Descuentos Aplicados...")) }

func openXLSXDialog(owner uintptr) {
	files := pickMultipleXLSX(owner)
	if len(files)==0 { return }
	selectedFiles := files
	rows, err := mergeXLSX(files)
	if err != nil { logf("mergeXLSX error: %v",err); return }
	mainLines = BuildLines(rows, strings.Join(files,";"))
	currentFilterCount=0
	currentView=BuildFilteredSortedViewByHeaders(mainLines,nil)
	refreshGrid(currentView)
	updateStatus(owner,currentView)
	for _, f := range selectedFiles { feedEngineFile(owner,f) }
}

func pickMultipleXLSX(owner uintptr) []string {
	buf:=make([]uint16,32768)
	filter:=u16z("Archivos XLSX (*.xlsx)\x00*.xlsx\x00Todos los archivos (*.*)\x00*.*\x00\x00")
	title:=u16("ABRIR XLSX")
	of:=openfilename{LStructSize:uint32(unsafe.Sizeof(openfilename{})),HwndOwner:owner,Filter:uintptr(unsafe.Pointer(&filter[0])),File:uintptr(unsafe.Pointer(&buf[0])),MaxFile:uint32(len(buf)),Title:uintptr(unsafe.Pointer(title)),Flags:OFN_EXPLORER|OFN_ALLOWMULTISELECT|OFN_FILEMUSTEXIST|OFN_HIDEREADONLY|OFN_ENABLEHOOK,Hook:syscall.NewCallback(multiSelectHook)}
	r,_,_:=user32.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(&of)))
	if r==0{return nil};return parseMultiSelectBuffer(buf)
}
func parseMultiSelectBuffer(buf []uint16) []string { parts:=[]string{};start:=0;for start<len(buf){end:=start;for end<len(buf)&&buf[end]!=0{end++};if end==start{break};parts=append(parts,syscall.UTF16ToString(buf[start:end]));start=end+1};if len(parts)<=1{return parts};dir:=parts[0];out:=make([]string,0,len(parts)-1);for _,name:=range parts[1:]{if strings.Contains(name,`\`){out=append(out,name)}else{out=append(out,dir+`\`+name)}};return out }
func multiSelectHook(hwnd,msg,wParam,lParam uintptr) uintptr { if msg==WM_INITDIALOG { logf("multi-select dialog hook hwnd=%x",hwnd) }; return 0 }

func findWindowByTitles(titles []string) uintptr { for _,t:=range titles { h,_,_:=user32.NewProc("FindWindowW").Call(0,uintptr(unsafe.Pointer(u16(t))));if h!=0{return h} };return 0 }
func enumTopWindows(fn func(uintptr)bool){cb:=syscall.NewCallback(func(hwnd,lParam uintptr)uintptr{if fn(hwnd){return 1};return 0});user32.NewProc("EnumWindows").Call(cb,0)}
func enumChildren(hwnd uintptr,fn func(uintptr)bool){cb:=syscall.NewCallback(func(child,lParam uintptr)uintptr{if fn(child){return 1};return 0});user32.NewProc("EnumChildWindows").Call(hwnd,cb,0)}
func findChildByText(hwnd uintptr,text string)uintptr{var found uintptr;enumChildren(hwnd,func(c uintptr)bool{if windowText(c)==text{found=c;return false};return true});return found}
func findFirstEdit(hwnd uintptr)uintptr{var found uintptr;enumChildren(hwnd,func(c uintptr)bool{if strings.EqualFold(getClassName(c),"EDIT"){found=c;return false};return true});return found}
func findDialogUnder(hwnd uintptr)uintptr{var found uintptr;enumTopWindows(func(w uintptr)bool{if w!=hwnd{found=w;return false};return true});return found}
func getClassName(hwnd uintptr)string{b:=make([]uint16,256);n,_,_:=user32.NewProc("GetClassNameW").Call(hwnd,uintptr(unsafe.Pointer(&b[0])),uintptr(len(b)));return syscall.UTF16ToString(b[:n])}
func repositionOverlay(hwnd uintptr){_ = hwnd}

func wndProc(hwnd,msg,wParam,lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE: crearControles(hwnd); return 0
	case WM_SIZE: redimensionarControles(hwnd); return 0
	case WM_COMMAND: return handleCommand(hwnd,wParam,lParam)
	case WM_NOTIFY: return handleNotify(hwnd,wParam,lParam)
	case WM_CLOSE: user32.NewProc("DestroyWindow").Call(hwnd); return 0
	case WM_DESTROY: user32.NewProc("PostQuitMessage").Call(0); return 0
	}
	ret,_,_:=user32.NewProc("DefWindowProcW").Call(hwnd,msg,wParam,lParam);return ret
}
func handleNotify(hwnd,wParam,lParam uintptr) uintptr { return handleMainNotify(hwnd,wParam,lParam) }
func handleMainNotify(hwnd,wParam,lParam uintptr) uintptr { _=hwnd;_=wParam;_=lParam;return 0 }

// HECHO VERIFICADO: el binario contiene main.feedEngineFile.
// INFERENCIA: el contrato interno no es recuperable sin GestionSO-V54-engine.exe;
// solo se conserva la parametrizacion documentada por GESTIONSO_V54_ENGINE.
func feedEngineFile(owner uintptr,file string){engine:=os.Getenv("GESTIONSO_V54_ENGINE");if engine==""{logf("feedEngineFile owner=%x file=%q engine=not-configured",owner,file);return};logf("feedEngineFile owner=%x file=%q engine=%q",owner,file,engine)}
