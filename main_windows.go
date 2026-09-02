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

type lvcw struct { Mask uint32; Fmt int32; Cx int32; Text uintptr; TextMax int32; SubItem int32; Image int32; Order int32 }
type lvitemw struct { Mask uint32; Item int32; SubItem int32; State uint32; StateMask uint32; Text uintptr; TextMax int32; Image int32; LParam uintptr }

// OPENFILENAMEW minimal definition for GetOpenFileNameW usage
type OPENFILENAMEW struct {
	LStructSize    uint32
	HwndOwner      uintptr
	HInstance      uintptr
	lpstrFilter    uintptr
	lpstrCustomFilter uintptr
	nMaxCustFilter uint32
	nFilterIndex   uint32
	lpstrFile      uintptr
	nMaxFile       uint32
	lpstrFileTitle uintptr
	nMaxFileTitle  uint32
	lpstrInitialDir uintptr
	lpstrTitle     uintptr
	Flags          uint32
	nFileOffset    uint16
	nFileExtension uint16
	lpstrDefExt    uintptr
	lCustData      uintptr
	lpfnHook       uintptr
	lpTemplateName uintptr
}

var (
	hInstance uintptr
	hwndGrid uintptr
	hwndStatus uintptr
	hwndTotals uintptr
	hwndMode uintptr
	filterHandles = map[int]uintptr{}
)

const (
	WM_NOTIFY = 0x004E
	WM_INITDIALOG = 0x0110
	CBN_SELCHANGE = 1
	CB_ADDSTRING = 0x0143
	CB_SETCURSEL = 0x014E
	CB_GETCURSEL = 0x0147
	LVS_REPORT = 0x0001
	LVS_SINGLESEL = 0x0004
	LVS_SHOWSELALWAYS = 0x0008
	LVS_EX_FULLROWSELECT = 0x0020
	LVM_FIRST = 0x1000
	LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
	LVM_DELETEALLITEMS = LVM_FIRST + 9
	LVM_INSERTITEMW = LVM_FIRST + 77
	LVM_SETITEMW = LVM_FIRST + 76
	LVM_INSERTCOLUMNW = LVM_FIRST + 97
	LVCF_FMT = 0x0001
	LVCF_WIDTH = 0x0002
	LVCF_TEXT = 0x0004
	LVCFMT_LEFT = 0
	LVIF_TEXT = 0x0001
	BS_PUSHBUTTON = 0
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE = 0x10000000
	WS_CHILD = 0x40000000
	WS_TABSTOP = 0x00010000
	WS_BORDER = 0x00800000
	CBS_DROPDOWNLIST = 0x0003
	CW_USEDEFAULT = 0x80000000
	OFN_EXPLORER = 0x00080000
	OFN_FILEMUSTEXIST = 0x00001000
	OFN_ALLOWMULTISELECT = 0x00000200
	OFN_HIDEREADONLY = 0x00000004
	OFN_ENABLEHOOK = 0x00000020

	ID_TOMAR_EXCEL = 1007
	ID_FILTROS_CAB = 1008
	ID_SIMULADOR = 1009
	ID_RESALTAR = 1010
	ID_COLOR = 1011
	ID_DATOS_CSV = 1012
	ID_PAGE_SIZE = 1013
	ID_MODO = 1014
	ID_FILTRO_SO = 1101
	ID_FILTRO_ESTADO = 1102
	ID_FILTRO_SKU = 1103
	ID_FILTRO_SUMA = 1104
	ID_FILTRO_SDSRP2 = 1105
	ID_GRID = 1201
	ID_STATUS = 1202
	ID_TOTALS = 1203
)

var uiColumns = []ColumnDef{
	{Name: "SKU", Width: 90}, {Name: "Descripción", Width: 230}, {Name: "SUM (%) descuento", Width: 125}, {Name: "NETO PK", Width: 110},
	{Name: "UNIDADES", Width: 90}, {Name: "PALL", Width: 75}, {Name: "PK", Width: 75}, {Name: "NETO SO", Width: 110},
	{Name: "TN SO", Width: 95}, {Name: "CMG", Width: 90}, {Name: "PPP SO", Width: 100}, {Name: "ORIGEN", Width: 130},
}

var toolbar = []struct { id int; text string }{
	{ID_ABRIR_XLSX, "ABRIR XLSX"}, {ID_TOMAR_EXCEL, "TOMAR EXCEL ABIERTO"}, {ID_RECARGAR, "RECARGAR"}, {ID_COLUMNAS, "COLUMNAS..."},
	{ID_FILTROS_CAB, "FILTROS CABECERA..."}, {ID_EXPORTAR_CSV, "EXPORTAR CSV"}, {ID_SIMULADOR, "SIMULADOR"}, {ID_RESALTAR, "RESALTAR..."},
	{ID_COLOR, "+/- COLOR..."}, {ID_DATOS_CSV, "DATOS CSV..."},
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
	wc := WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), LpfnWndProc: syscall.NewCallback(wndProc), HInstance: hInstance, HCursor: hc, HbrBackground: hb, LpszMenuName: nil, LpszClassName: className}
	user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, _ := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Gestion SO V54 - SO RETENIDAS / CSV maestro"))), WS_OVERLAPPEDWINDOW|WS_VISIBLE, CW_USEDEFAULT, CW_USEDEFAULT, 1024, 640, 0, 0, hInstance, 0)
	if hwnd != 0 {
		mainConfig = LoadConfig()
		if mainConfig.Mode == "" { mainConfig.Mode = "MODO: SO RETENIDAS"; _ = SaveConfig(mainConfig) }
		setWindowText(hwnd, windowTitle())
		crearControles(hwnd)
		redimensionarControles(hwnd)
	}
	return hwnd
}

func windowTitle() string { mode := strings.TrimSpace(mainConfig.Mode); if strings.HasPrefix(mode, "MODO: ") { mode = strings.TrimSpace(strings.TrimPrefix(mode, "MODO: ")) }; if mode == "" { mode = "" }; return "Gestion SO V57 - " + mode }

func crearControles(hwnd uintptr) {
	initLog()
	for _, b := range toolbar { crearBoton(hwnd, b.text, 0, 0, 100, 28, uintptr(b.id)) }
	inst := hInstance
	var r1, r2 uintptr
	r1, r2, _ = user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("COMBOBOX"))), 0, WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST, 0, 0, 190, 180, hwnd, uintptr(ID_MODO), inst, 0)
	hwndMode = r1
	modes := []string{"MODO: FACTURAS PENDIENTES", "MODO: SO RETENIDAS", "MODO: FACTURAS"}
	selected := 0
	for i, mode := range modes { user32.NewProc("SendMessageW").Call(hwndMode, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(mode)))); if mode == mainConfig.Mode { selected = i } }
	user32.NewProc("SendMessageW").Call(hwndMode, CB_SETCURSEL, uintptr(selected), 0)
	for _, f := range []struct{id int; label string}{{ID_FILTRO_SO,"SO"},{ID_FILTRO_ESTADO,"Estado"},{ID_FILTRO_SKU,"SKU"},{ID_FILTRO_SUMA,"SUMA DE"},{ID_FILTRO_SDSRP2,"SDSRP2"}} {
		crearLabel(hwnd, f.label, 0, 0, 70, 20)
		h, _, _ := user32.NewProc("CreateWindowExW").Call(WS_BORDER, uintptr(unsafe.Pointer(u16("EDIT"))), 0, WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER, 0, 0, 120, 24, hwnd, uintptr(f.id), inst, 0)
		filterHandles[f.id] = h
	}
	crearBoton(hwnd, "FILTRAR", 0, 0, 85, 26, uintptr(ID_FILTRAR)); crearBoton(hwnd, "LIMPIAR", 0, 0, 85, 26, uintptr(ID_LIMPIAR))
	r1, r2, _ = user32.NewProc("CreateWindowExW").Call(WS_BORDER, uintptr(unsafe.Pointer(u16("SysListView32"))), 0, WS_CHILD|WS_VISIBLE|LVS_REPORT|LVS_SINGLESEL|LVS_SHOWSELALWAYS|WS_BORDER, 0, 120, 800, 400, hwnd, uintptr(ID_GRID), inst, 0)
	hwndGrid = r1
	user32.NewProc("SendMessageW").Call(hwndGrid, LVM_SETEXTENDEDLISTVIEWSTYLE, LVS_EX_FULLROWSELECT, LVS_EX_FULLROWSELECT)
	for i, c := range uiColumns { title:=c.Name; if title=="CMG" { title="CMG ▼" }; t:=u16(title); col:=lvcw{Mask:LVCF_FMT|LVCF_WIDTH|LVCF_TEXT,Fmt:LVCFMT_LEFT,Cx:int32(c.Width),Text:uintptr(unsafe.Pointer(t)),TextMax:int32(len(title)+1),SubItem:int32(i),Image:0,Order:0}; user32.NewProc("SendMessageW").Call(hwndGrid, LVM_INSERTCOLUMNW, uintptr(i), uintptr(unsafe.Pointer(&col))) }
	r1, r2, _ = user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16("BULTOS 0 | PALLETS 0 | TN 0 | UNIDADES 0\r\nNETO $ 0 | COSTO $ 0"))), WS_CHILD|WS_VISIBLE, 10, 520, 800, 40, hwnd, uintptr(ID_TOTALS), inst, 0)
	hwndTotals = r1
	r1, r2, _ = user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(u16("STATIC"))), uintptr(unsafe.Pointer(u16(BuildStatusBar(mainConfig.Mode,nil,0,"Detalle de Descuentos Aplicados...")))), WS_CHILD|WS_VISIBLE, 10, 560, 800, 24, hwnd, uintptr(ID_STATUS), inst, 0)
	hwndStatus = r1
}

func crearBoton(hwnd uintptr, texto string, x,y,ancho,alto int,id uintptr) { user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(u16("BUTTON"))),uintptr(unsafe.Pointer(u16(texto))),WS_CHILD|WS_VISIBLE|BS_PUSHBUTTON,uintptr(x),uintptr(y),uintptr(ancho),uintptr(alto),hwnd,id,hInstance,0) }
func crearLabel(hwnd uintptr,texto string,x,y,ancho,alto int) { user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(u16("STATIC"))),uintptr(unsafe.Pointer(u16(texto))),WS_CHILD|WS_VISIBLE,uintptr(x),uintptr(y),uintptr(ancho),uintptr(alto),hwnd,0,hInstance,0) }

func redimensionarControles(hwnd uintptr) {
	var r RECT; user32.NewProc("GetClientRect").Call(hwnd,uintptr(unsafe.Pointer(&r))); w,h:=int(r.Right-r.Left),int(r.Bottom-r.Top); if w<900 {w=900}; if h<500 {h=500}
	x:=10; for _,b:=range toolbar {bw:=len([]rune(b.text))*8+22;if bw<85{bw=85};if bw>160{bw=160};if ctl:=getDlgItem(hwnd,b.id);ctl!=0{user32.NewProc("MoveWindow").Call(ctl,uintptr(x),8,uintptr(bw),uintptr(28),1)};x+=bw+6}
	labels:=[]struct{id,x,width int}{{ID_FILTRO_SO,10,105},{ID_FILTRO_ESTADO,125,135},{ID_FILTRO_SKU,270,125},{ID_FILTRO_SUMA,405,135},{ID_FILTRO_SDSRP2,550,135}};for _,f:=range labels{if e:=filterHandles[f.id];e!=0{user32.NewProc("MoveWindow").Call(e,uintptr(f.x),38,uintptr(f.width),uintptr(24),1)}}
	gridY:=78;statusH:=26;totalsH:=42;gridH:=h-gridY-statusH-totalsH-12;if gridH<120{gridH=120};if hwndGrid!=0{user32.NewProc("MoveWindow").Call(hwndGrid,10,uintptr(gridY),uintptr(w-20),uintptr(gridH),1)}
	user32.NewProc("MoveWindow").Call(hwndTotals,10,uintptr(h-statusH-totalsH-6),uintptr(w-20),uintptr(totalsH),1)
	user32.NewProc("MoveWindow").Call(hwndStatus,10,uintptr(h-statusH-2),uintptr(w-20),uintptr(statusH),1)
}

func getDlgItem(hwnd uintptr,id int)uintptr{h,_,_:=user32.NewProc("GetDlgItem").Call(hwnd,uintptr(id));return h}
func windowText(hwnd uintptr)string{if hwnd==0{return ""};n,_,_:=user32.NewProc("GetWindowTextLengthW").Call(hwnd);b:=make([]uint16,n+1);user32.NewProc("GetWindowTextW").Call(hwnd,uintptr(unsafe.Pointer(&b[0])),uintptr(len(b)));return syscall.UTF16ToString(b[:n])}
func setWindowText(hwnd uintptr,text string){if hwnd!=0{user32.NewProc("SetWindowTextW").Call(hwnd,uintptr(unsafe.Pointer(u16(text))))}}

// handleCommand is kept as original but WM_COMMAND will be wrapped in wndProc to catch panics
func handleCommand(hwnd,wParam,lParam uintptr)uintptr{_ = lParam;id:=int(wParam&0xffff);notify:=uint16((wParam>>16)&0xffff);switch id{case ID_ABRIR_XLSX:if notify==BN_CLICKED{openXLSXDialog(hwnd)};case ID_FILTRAR:if notify==BN_CLICKED{applyHeaderFilters(hwnd)};case ID_LIMPIAR:if notify==BN_CLICKED{clearHeaderFilters(hwnd);updateMainView(hwnd)};case ID_RECARGAR:if notify==BN_CLICKED{mainLines=nil;updateMainView(hwnd)};default:}
	return 0}

func saveSelectedMode(hwnd uintptr){if hwndMode==0{return};idx,_,_:=user32.NewProc("SendMessageW").Call(hwndMode,CB_GETCURSEL,0,0);modes:=[]string{"MODO: FACTURAS PENDIENTES","MODO: SO RETENIDAS","MODO: FACTURAS"};if int(idx)>=0&&int(idx)<len(modes){mainConfig.Mode=modes[int(idx)];_ = SaveConfig(mainConfig);setWindowText(hwnd,windowTitle())}}
func applyHeaderFilters(hwnd uintptr){filters:=map[string]string{};currentFilterCount=0;for _,id:=range []int{ID_FILTRO_SO,ID_FILTRO_ESTADO,ID_FILTRO_SKU,ID_FILTRO_SUMA,ID_FILTRO_SDSRP2}{v:=strings.TrimSpace(windowText(filterHandles[id]));if v!=""{filters[filterName(id)]=v;currentFilterCount++}};mainLines=BuildLines(mainLines,"");currentView=BuildFilteredSortedViewByHeaders(mainLines,filters);updateMainView(hwnd)}
func filterName(id int)string{switch id{case ID_FILTRO_SO:return "SO";case ID_FILTRO_ESTADO:return "Estado";case ID_FILTRO_SKU:return "SKU";case ID_FILTRO_SUMA:return "SUMA DE";case ID_FILTRO_SDSRP2:return "SDSRP2"};return ""}
func clearHeaderFilters(hwnd uintptr){for _,id:=range []int{ID_FILTRO_SO,ID_FILTRO_ESTADO,ID_FILTRO_SKU,ID_FILTRO_SUMA,ID_FILTRO_SDSRP2}{setWindowText(filterHandles[id],"")};currentFilterCount=0}
func updateMainView(hwnd uintptr){currentView=BuildFilteredSortedViewByHeaders(mainLines,nil);refreshGrid(currentView);updateStatus(hwnd,currentView);setWindowText(hwnd,windowTitle())}

func resolveUIValue(l Line,name string)string{aliases:=map[string][]string{"SKU":{"sku"},"Descripción":{"descrip","descripcion","producto"},"SUM (%) descuento":{"sum","descuento","% descuento"},"NETO PK":{"neto pk"}};for k,v:=range l.Values{if strings.EqualFold(strings.TrimSpace(k),name){return v}};for alt:=range aliases{_ = alt};for k,v:=range l.Values{if strings.EqualFold(k,name){return v}};return ""}
func refreshGrid(lines []Line){if hwndGrid==0{return};user32.NewProc("SendMessageW").Call(hwndGrid,LVM_DELETEALLITEMS,0,0);for i,l:=range lines{insertGridRow(resolveUIValue(l,uiColumns[0].Name))}}
func insertGridRow(text string)int{t:=u16(text);it:=lvitemw{Mask:LVIF_TEXT,Item:0,SubItem:0,Text:uintptr(unsafe.Pointer(t)),TextMax:int32(len(text)+1)};r,_,_:=user32.NewProc("SendMessageW").Call(hwndGrid,LVM_INSERTITEMW,0,uintptr(unsafe.Pointer(&it)));return int(r)}
func setGridCell(row,col int,text string){t:=u16(text);it:=lvitemw{Mask:LVIF_TEXT,Item:int32(row),SubItem:int32(col),Text:uintptr(unsafe.Pointer(t)),TextMax:int32(len(text)+1)};user32.NewProc("SendMessageW").Call(hwndGrid,LVM_SETITEMW,0,uintptr(unsafe.Pointer(&it)))}
func updateStatus(hwnd uintptr,lines []Line){_ = hwnd;setWindowText(hwndStatus,BuildStatusBar(mainConfig.Mode,lines,currentFilterCount,"Detalle de Descuentos Aplicados..."))}

func openXLSXDialog(owner uintptr){
	// Wrap the call with panicGuard and logging to avoid crashing the process
	panicGuard(func(){
		logf("openXLSXDialog start owner=%x", owner)
		files := pickMultipleXLSX(owner)
		logf("openXLSXDialog: picked %d files", len(files))
		if len(files) == 0 { logf("openXLSXDialog: no files selected or cancelled"); return }
		rows, err := mergeXLSX(files)
		if err != nil { logf("mergeXLSX error: %v", err); return }
		mainLines = BuildLines(rows, "xlsx")
		updateMainView(owner)
		logf("openXLSXDialog end")
	})
}

func pickMultipleXLSX(owner uintptr)[]string{
	// Prepare buffer and filter
	buf := make([]uint16, 32768)
	filter := u16z("Archivos XLSX (*.xlsx)\x00*.xlsx\x00Todos los archivos (*.*)\x00*.*\x00\x00")
	title := u16("ABRIR XLSX")
	ofn := OPENFILENAMEW{}
	of := &ofn
	of.LStructSize = uint32(unsafe.Sizeof(*of))
	of.HwndOwner = owner
	of.lpstrFilter = uintptr(unsafe.Pointer(&filter[0]))
	of.lpstrFile = uintptr(unsafe.Pointer(&buf[0]))
	of.nMaxFile = uint32(len(buf))
	of.lpstrTitle = uintptr(unsafe.Pointer(title))
	of.Flags = OFN_EXPLORER | OFN_FILEMUSTEXIST | OFN_ALLOWMULTISELECT | OFN_HIDEREADONLY | OFN_ENABLEHOOK
	of.lpfnHook = syscall.NewCallback(multiSelectHook)

	ret, _, _ := comdlg32.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(of)))
	if ret == 0 {
		// Dialog failed or cancelled; log extended error if any
		errCode, _, _ := comdlg32.NewProc("CommDlgExtendedError").Call()
		if errCode != 0 { logf("GetOpenFileNameW failed with code %d", errCode) }
		return nil
	}
	// parse the returned buffer; it may contain multiple "+"-separated paths or a single path
	return parseMultiSelectBuffer(buf)
}

func parseMultiSelectBuffer(buf []uint16)[]string{parts:=[]string{};i:=0;if buf[0]==0{return parts};
	// If single filename, it's null-terminated and next char is 0
	// If multiple selected, buffer: Dir\0File1\0File2\0\0
	// Find first null-terminated string
	first := syscall.UTF16ToString(buf)
	// check if there are additional zeros after first string
	endFirst := len(first)
	// search for second zero after endFirst
	idx := endFirst+1
	if idx < len(buf) && buf[idx] != 0 {
		// multiple
		// get directory
		dir := first
		for i = idx; i < len(buf); {
			if buf[i] == 0 { break }
			j := i
			for j < len(buf) && buf[j] != 0 { j++ }
			parts = append(parts, filepathJoin(dir, syscall.UTF16ToString(buf[i:j])))
			i = j + 1
		}
		return parts
	}
	// single file
	parts = append(parts, first)
	return parts
}

func filepathJoin(a,b string) string { if strings.HasSuffix(a, "\\") { return a + b }; return a + "\\" + b }

func multiSelectHook(hwnd,msg,wParam,lParam uintptr)uintptr{_ = wParam; _ = lParam; if msg == WM_INITDIALOG { logf("multi-select dialog hook hwnd=%x", hwnd) }; return 0 }

func findWindowByTitles(titles []string)uintptr{for _,t:=range titles{h,_,_:=user32.NewProc("FindWindowW").Call(0,uintptr(unsafe.Pointer(u16(t))));if h!=0{return h}};return 0}
func enumTopWindows(fn func(uintptr)bool){cb:=syscall.NewCallback(func(hwnd,lParam uintptr)uintptr{_=lParam;if fn(hwnd){return 1};return 0});user32.NewProc("EnumWindows").Call(cb,0)}
func enumChildren(hwnd uintptr,fn func(uintptr)bool){cb:=syscall.NewCallback(func(child,lParam uintptr)uintptr{_=lParam;if fn(child){return 1};return 0});user32.NewProc("EnumChildWindows").Call(hwnd,cb,0)}
func findChildByText(hwnd uintptr,text string)uintptr{var found uintptr;enumChildren(hwnd,func(c uintptr)bool{if windowText(c)==text{found=c;return false};return true});return found}
func findFirstEdit(hwnd uintptr)uintptr{var found uintptr;enumChildren(hwnd,func(c uintptr)bool{if strings.EqualFold(getClassName(c),"EDIT"){found=c;return false};return true});return found}
func findDialogUnder(hwnd uintptr)uintptr{var found uintptr;enumTopWindows(func(w uintptr)bool{if w!=hwnd{found=w;return false};return true});return found}
func getClassName(hwnd uintptr)string{b:=make([]uint16,256);n,_,_:=user32.NewProc("GetClassNameW").Call(hwnd,uintptr(unsafe.Pointer(&b[0])),uintptr(len(b)));return syscall.UTF16ToString(b[:n])}
func repositionOverlay(hwnd uintptr){_=hwnd}

func wndProc(hwnd,msg,wParam,lParam uintptr)uintptr{switch msg{case WM_CREATE:crearControles(hwnd);return 0;case WM_SIZE:redimensionarControles(hwnd);return 0;case WM_COMMAND:
		// Protect all WM_COMMAND handlers so panics don't kill the process and log start/end
		panicGuard(func(){
			logf("WM_COMMAND start hwnd=%x wParam=%x lParam=%x", hwnd, wParam, lParam)
			handleCommand(hwnd,wParam,lParam)
			logf("WM_COMMAND end hwnd=%x", hwnd)
		})
		return 0;case WM_NOTIFY:return handleNotify(hwnd,wParam,lParam);case WM_INITDIALOG:return 0;case WM_CLOSE:user32.NewProc("DestroyWindow").Call(hwnd);return 0;case WM_DESTROY:user32.NewProc("PostQuitMessage").Call(0);return 0;default:
		r,_,_:=user32.NewProc("DefWindowProcW").Call(hwnd,msg,wParam,lParam)
		return r
	}}

func handleNotify(hwnd,wParam,lParam uintptr)uintptr{return handleMainNotify(hwnd,wParam,lParam)}
func handleMainNotify(hwnd,wParam,lParam uintptr)uintptr{_=hwnd;_=wParam;_=lParam;return 0}

// HECHO VERIFICADO: el binario contiene main.feedEngineFile.
// INFERENCIA: su contrato interno no es recuperable sin GestionSO-V54-engine.exe.
func feedEngineFile(owner uintptr,file string){engine:=os.Getenv("GESTIONSO_V54_ENGINE");if engine==""{logf("feedEngineFile owner=%x file=%q engine=not-configured",owner,file);return};logf("feedEngineFile would call %s with %s",engine,file)}
