//go:build windows

package main

import (
    "fmt"
    "math"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "syscall"
    "unsafe"
)

const (
    appIDOpen = 2001
    appIDFilter = 2002
    appIDClear = 2003
    appIDColumns = 2004
    appIDExport = 2005
    appIDStatus = 2006
    appIDGrid = 2007
    appIDSearch = 2008
    WM_APP_REFRESH = 0x8001
    WM_APP_COLUMNS = 0x8002
    BS_PUSHBUTTON = 0
    BS_CHECKBOX = 0x00000002
    BST_CHECKED = 1
    WS_OVERLAPPEDWINDOW = 0x00CF0000
    WS_VISIBLE = 0x10000000
    WS_CHILD = 0x40000000
    WS_TABSTOP = 0x00010000
    WS_BORDER = 0x00800000
    ES_AUTOHSCROLL = 0x0080
    LVS_REPORT = 0x0001
    LVS_SHOWSELALWAYS = 0x0008
    LVS_SINGLESEL = 0x0004
    LVS_EX_FULLROWSELECT = 0x0020
    LVM_FIRST = 0x1000
    LVM_DELETEALLITEMS = LVM_FIRST + 9
    LVM_INSERTITEMW = LVM_FIRST + 77
    LVM_SETITEMW = LVM_FIRST + 76
    LVM_INSERTCOLUMNW = LVM_FIRST + 97
    LVM_DELETECOLUMN = LVM_FIRST + 28
    LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
    LVCF_FMT = 0x0001
    LVCF_WIDTH = 0x0002
    LVCF_TEXT = 0x0004
    LVCFMT_LEFT = 0
    LVIF_TEXT = 0x0001
    WM_CREATE = 0x0001
    WM_DESTROY = 0x0002
    WM_SIZE = 0x0005
    WM_COMMAND = 0x0111
    WM_CLOSE = 0x0010
    EN_CHANGE = 0x0300
    OFN_EXPLORER = 0x00080000
    OFN_FILEMUSTEXIST = 0x00001000
    OFN_ALLOWMULTISELECT = 0x00000200
    OFN_HIDEREADONLY = 0x00000004
)

type appPoint struct { X, Y int32 }
type appMsg struct { Hwnd uintptr; Message uint32; WParam, LParam uintptr; Time uint32; Pt appPoint }
type appRect struct { Left, Top, Right, Bottom int32 }
type appWndClass struct { CbSize uint32; Style uint32; LpfnWndProc uintptr; CbClsExtra, CbWndExtra int32; HInstance, HIcon, HCursor, HbrBackground uintptr; LpszMenuName, LpszClassName *uint16; HIconSm uintptr }
type appLVC struct { Mask uint32; Fmt, Cx int32; Text uintptr; TextMax int32; SubItem, Image, Order int32 }
type appLVI struct { Mask uint32; Item, SubItem int32; State, StateMask uint32; Text uintptr; TextMax, Image int32; LParam uintptr }
type appOpenFile struct { LStructSize uint32; Pad0 uint32; HwndOwner, HInstance, Filter, CustomFilter uintptr; MaxCustom, FilterIndex uint32; File uintptr; MaxFile uint32; Pad1 uint32; FileTitle uintptr; MaxFileTitle uint32; Pad2 uint32; InitialDir, Title uintptr; Flags uint32; FileOffset, FileExtension uint16; DefExt, CustData, Hook, Template uintptr; Reserved uintptr; Reserved2, FlagsEx uint32 }
type appColumn struct { Name string; Width int; Visible bool }

var (
    appHwnd, appGrid, appSearch, appStatus, appColumnsWindow uintptr
    appLines []Line
    appView []Line
    appColumns []appColumn
    appMaster *MasterData
    appMu = make(chan struct{}, 1)
)

func appU16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func appText(h uintptr) string { n, _, _ := user32.NewProc("GetWindowTextLengthW").Call(h); b := make([]uint16, n+1); user32.NewProc("GetWindowTextW").Call(h, uintptr(unsafe.Pointer(&b[0])), uintptr(n+1)); return syscall.UTF16ToString(b) }
func appSetText(h uintptr, s string) { p := appU16(s); user32.NewProc("SetWindowTextW").Call(h, uintptr(unsafe.Pointer(p))) }
func appMake(parent uintptr, cls, text string, style uint32, x,y,w,h int, id uintptr) uintptr { c:=appU16(cls); t:=appU16(text); r,_,_:=user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(c)),uintptr(unsafe.Pointer(t)),uintptr(style),uintptr(x),uintptr(y),uintptr(w),uintptr(h),parent,id,hInstance,0); return r }

func crearVentana() uintptr {
    hInstance,_,_ = kernel32.NewProc("GetModuleHandleW").Call(0)
    appLog("EVENTO: GetModuleHandleW => hInstance=0x%X", hInstance)
    cls:=appU16("GestionSOFunctional")
    wc:=appWndClass{CbSize:uint32(unsafe.Sizeof(appWndClass{})), LpfnWndProc:syscall.NewCallback(appWndProcLogged), HInstance:hInstance, HCursor:func() uintptr { r,_,_:=user32.NewProc("LoadCursorW").Call(0,32512); return r }(), LpszClassName:cls}
    rc,_,re:=user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
    appLog("EVENTO: RegisterClassExW rc=%d err=%v", rc, re)
    title:=appU16("GestionSO V57 - Analizador de Excel")
    h,_,e:=user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(cls)),uintptr(unsafe.Pointer(title)),WS_OVERLAPPEDWINDOW|WS_VISIBLE,0x80000000,0x80000000,1400,820,0,0,hInstance,0)
    appLog("EVENTO: CreateWindowExW hwnd=0x%X err=%v", h, e)
    return h
}

func appWndProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
    switch msg {
    case WM_CREATE:
        appHwnd=hwnd; appBuildControls(hwnd); return 0
    case WM_SIZE:
        appLayout(hwnd); return 0
    case WM_COMMAND:
        id:=int(wp&0xffff); code:=uint32((wp>>16)&0xffff)
        switch id {
        case appIDOpen: appLogRuntimeEvent("click ABRIR EXCEL"); appOpenXLSX(hwnd)
        case appIDFilter: appApplyFilter()
        case appIDClear: appSetText(appSearch, ""); appApplyFilter()
        case appIDColumns: appShowColumns()
        case appIDExport: appExportCSV(hwnd)
        case appIDSearch: if code==EN_CHANGE { appApplyFilter() }
        }
        return 0
    case WM_APP_REFRESH:
        appRefreshGrid(); return 0
    case WM_APP_COLUMNS:
        appRefreshGrid(); return 0
    case WM_CLOSE:
        user32.NewProc("DestroyWindow").Call(hwnd); return 0
    case WM_DESTROY:
        user32.NewProc("PostQuitMessage").Call(0); return 0
    }
    r,_,_:=user32.NewProc("DefWindowProcW").Call(hwnd,uintptr(msg),wp,lp)
    return r
}

func appBuildControls(hwnd uintptr) {
    appMake(hwnd,"BUTTON","ABRIR EXCEL",WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,10,10,120,30,appIDOpen)
    appMake(hwnd,"BUTTON","COLUMNAS...",WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,138,10,120,30,appIDColumns)
    appMake(hwnd,"BUTTON","EXPORTAR CSV",WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,266,10,120,30,appIDExport)
    appMake(hwnd,"STATIC","Filtro (ej.: SKU=PRE;Estado=RETENIDA;Cliente=EXXA):",WS_CHILD|WS_VISIBLE,400,14,380,22,0)
    appSearch=appMake(hwnd,"EDIT","",WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL,785,10,380,30,appIDSearch)
    appMake(hwnd,"BUTTON","LIMPIAR",WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,1173,10,90,30,appIDClear)
    appGrid=appMake(hwnd,"SysListView32","",WS_CHILD|WS_VISIBLE|LVS_REPORT|LVS_SINGLESEL|LVS_SHOWSELALWAYS|WS_BORDER,10,50,1300,680,appIDGrid)
    user32.NewProc("SendMessageW").Call(appGrid,LVM_SETEXTENDEDLISTVIEWSTYLE,LVS_EX_FULLROWSELECT,LVS_EX_FULLROWSELECT)
    appStatus=appMake(hwnd,"STATIC","Listo. Seleccione uno o varios XLSX.",WS_CHILD|WS_VISIBLE,10,740,1300,30,appIDStatus)
    appColumns=[]appColumn{}
}

func appLayout(hwnd uintptr) {
    var r appRect; user32.NewProc("GetClientRect").Call(hwnd,uintptr(unsafe.Pointer(&r))); w:=int(r.Right-r.Left); h:=int(r.Bottom-r.Top); if w<900 {w=900}; if h<500 {h=500}
    user32.NewProc("MoveWindow").Call(appSearch,uintptr(785),uintptr(10),uintptr(max(220,w-935)),uintptr(30),1)
    user32.NewProc("MoveWindow").Call(appGrid,uintptr(10),uintptr(50),uintptr(w-20),uintptr(h-90),1)
    user32.NewProc("MoveWindow").Call(appStatus,uintptr(10),uintptr(h-32),uintptr(w-20),uintptr(25),1)
}
func max(a,b int) int { if a>b{return a}; return b }

func appOpenXLSX(owner uintptr) {
    appLog("EVENTO: appOpenXLSX iniciado owner=0x%X", owner)
    files:=appPickXLSX(owner)
    appLog("EVENTO: appPickXLSX devolvió %d archivo(s)", len(files))
    if len(files)==0{return}
    appSetText(appStatus,fmt.Sprintf("Procesando %d archivo(s)...",len(files)))
    go func(selected []string) {
        defer appRecover("procesamiento XLSX")
        lock:=appMu; lock<-struct{}{}; defer func(){<-lock}()
        appLog("EVENTO: mergeXLSX inicia con %d archivo(s)", len(selected))
        rows,err:=mergeXLSX(selected); if err!=nil { appLog("ERROR mergeXLSX: %v",err); appSetTextSafe(appStatus,"ERROR: "+err.Error()); return }
        appLog("EVENTO: mergeXLSX terminó filas=%d", len(rows))
        masterPath:=appFindMaster(); appLog("EVENTO: maestro=%s",masterPath); m,err:=LoadMaster(masterPath); if err!=nil { appLog("ERROR LoadMaster: %v",err); appSetTextSafe(appStatus,"ERROR maestro: "+err.Error()); return }
        lines:=appBuildCalculatedLines(rows,m); appLines=lines; appView=append([]Line(nil),lines...); appMaster=m
        appLog("EVENTO: cálculos terminados líneas=%d", len(lines))
        if len(appColumns)==0 { appInitColumns(lines) }
        user32.NewProc("PostMessageW").Call(appHwnd,WM_APP_REFRESH,0,0)
    }(append([]string(nil),files...))
}

func appSetTextSafe(h uintptr,s string) { if h!=0 { appSetText(h,s) } }

func appFindMaster() string {
    exe,_:=os.Executable(); dir:=filepath.Dir(exe)
    candidates:=[]string{filepath.Join(dir,"datos","GestionSO_Datos.csv"),filepath.Join(dir,"GestionSO_Datos.csv"),filepath.Join("acceso chatgpt","GestionSO_Datos.csv")}
    for _,p:=range candidates {if _,e:=os.Stat(p);e==nil{return p}}
    return candidates[0]
}

func appInitColumns(lines []Line) {
    preferred:=[]string{"SKU","Descripción","SUMA DE DESCUENTOS","NETO PK","UNIDADES","PALL","PK","NETO SO","TN SO","CMG","PPP SO","ORIGEN: GENERADOR","SO","Estado","Ejecutivo","Cliente","Factura","Tipo Doc","Documento","Provincia","Ciudad"}
    seen:=map[string]bool{}; appColumns=nil
    add:=func(n string,v bool){k:=strings.ToLower(n);if seen[k]{return};seen[k]=true;appColumns=append(appColumns,appColumn{Name:n,Width:115,Visible:v})}
    for _,n:=range preferred {add(n,true)}
    all:=map[string]bool{}
    for _,l:=range lines {for k:=range l.Values {all[k]=true}}
    rest:=make([]string,0,len(all));for k:=range all {rest=append(rest,k)};sort.Strings(rest)
    for _,k:=range rest {add(k,false)}
}

func appBuildCalculatedLines(rows [][]string,m *MasterData) []Line {
    if len(rows)==0{return nil}; hi:=headerRowIndex(rows); if hi<0||hi>=len(rows){return nil}; headers:=uniqueHeaders(rows[hi]); out:=make([]Line,0,len(rows)-hi-1)
    masterBySKU:=map[string]MasterRow{}; if m!=nil {for _,r:=range m.Rows {if s:=strings.TrimSpace(r["CLAVE"]);s!="" {masterBySKU[strings.ToUpper(s)]=r}}}
    for i:=hi+1;i<len(rows);i++ {
        vals:=map[string]string{}; empty:=true
        for j,v:=range rows[i] {if j<len(headers){vals[headers[j]]=v;if strings.TrimSpace(v)!=""{empty=false}}}; if empty{continue}
        sku:=firstVal(vals,"SKU","ORIGEN: ITEM"); master:=masterBySKU[strings.ToUpper(strings.TrimSpace(sku))]
        qty:=number(firstVal(vals,"ORIGEN: CANTIDAD","CANTIDAD","PK")); units:=number(master["UNIDADES_X_BULTO"]); bpp:=number(master["BULTOS_X_PALLET"]); kgB:=number(master["KG_X_BULTO"])
        if units>0 {vals["UNIDADES"]=fmtNum(qty*units)} else if vals["UNIDADES"]=="" {vals["UNIDADES"]=fmtNum(qty)}
        if bpp>0 {vals["PALL"]=fmtNum(qty/bpp)}
        vals["PK"]=fmtNum(qty)
        kg:=number(firstVal(vals,"ORIGEN: KG","KG")); if kg<=0&&kgB>0 {kg=qty*kgB}; if kg>0 {vals["TN SO"]=fmtNum(kg/1000)}
        netPK:=number(firstVal(vals,"NETO PK","ORIGEN: PRECIOUNIFC")); cost:=number(firstVal(vals,"COSTO","COSTO_UNITARIO")); if netPK>0 {vals["NETO PK"]=fmtNum(netPK); vals["PPP SO"]=fmtNum(netPK); vals["NETO SO"]=fmtNum(netPK*qty); if cost>0 {vals["CMG"]=fmtPct((netPK-cost)/cost*100)}}
        if vals["Descripción"]=="" {vals["Descripción"]=firstVal(vals,"ORIGEN: DESCRIPCION ITEM","DESCRIPCION")}
        if vals["SKU"]=="" {vals["SKU"]=sku}
        if vals["Estado"]=="" {vals["Estado"]=firstVal(vals,"ESTADO","ORIGEN: ESTADO")}
        if vals["Cliente"]=="" {vals["Cliente"]=firstVal(vals,"ORIGEN: CLIENTE","CLIENTE")}
        if vals["SO"]=="" {vals["SO"]=firstVal(vals,"ORIGEN: SDDOCO","ORIGEN: SO","SO")}
        if vals["ORIGEN: GENERADOR"]=="" {vals["ORIGEN: GENERADOR"]=firstVal(vals,"ORIGEN","ORIGEN: GENERADOR")}
        if disc:=firstVal(vals,"SUMA DE DESCUENTOS","DTOCNL","ORIGEN: DTOCNL");disc!="" {vals["SUMA DE DESCUENTOS"]=disc}
        out=append(out,Line{Values:vals,Source:"xlsx",RowNumber:i+1})
    }
    return out
}

func firstVal(m map[string]string,names ...string) string { for _,n:=range names {if v:=m[n];strings.TrimSpace(v)!="" {return v}; for k,v:=range m {if strings.EqualFold(strings.TrimSpace(k),strings.TrimSpace(n))&&strings.TrimSpace(v)!=""{return v}}};return "" }
func number(s string) float64 {s=strings.TrimSpace(s);if s==""||strings.EqualFold(s,"#N/D")||strings.EqualFold(s,"SIN REF"){return 0};s=strings.ReplaceAll(s,"%",""); if strings.Contains(s,",")&&strings.Contains(s,".") {if strings.LastIndex(s,",")>strings.LastIndex(s,"."){s=strings.ReplaceAll(s,".","");s=strings.ReplaceAll(s,",",".")}} else if strings.Contains(s,","){s=strings.ReplaceAll(s,",",".")};v,_:=strconv.ParseFloat(s,64);return v}
func fmtNum(v float64) string {if math.Abs(v-math.Round(v))<1e-9{return fmt.Sprintf("%.0f",v)};return strconv.FormatFloat(v,'f',3,64)}

func appRefreshGrid() {
    if appGrid==0{return}
    send:=user32.NewProc("SendMessageW")
    for {r,_,_:=send.Call(appGrid,LVM_DELETECOLUMN,0,0);if r==0{break}}
    send.Call(appGrid,LVM_DELETEALLITEMS,0,0)
    visibleIndex:=0
    for _,c:=range appColumns {if !c.Visible{continue}; t:=appU16(c.Name); col:=appLVC{Mask:LVCF_FMT|LVCF_WIDTH|LVCF_TEXT,Fmt:LVCFMT_LEFT,Cx:int32(c.Width),Text:uintptr(unsafe.Pointer(t)),TextMax:int32(len(syscall.StringToUTF16(c.Name))),SubItem:int32(visibleIndex)};send.Call(appGrid,LVM_INSERTCOLUMNW,uintptr(visibleIndex),uintptr(unsafe.Pointer(&col)));visibleIndex++}
    for ri,l:=range appView {visibleIndex=0;for _,c:=range appColumns {if !c.Visible{continue}; s:=l.Values[c.Name]; if s=="" {for k,v:=range l.Values {if strings.EqualFold(k,c.Name){s=v;break}}}; p:=appU16(s);if visibleIndex==0 {it:=appLVI{Mask:LVIF_TEXT,Item:int32(ri),SubItem:0,Text:uintptr(unsafe.Pointer(p)),TextMax:int32(len(syscall.StringToUTF16(s)))};send.Call(appGrid,LVM_INSERTITEMW,0,uintptr(unsafe.Pointer(&it)))} else {it:=appLVI{Mask:LVIF_TEXT,Item:int32(ri),SubItem:int32(visibleIndex),Text:uintptr(unsafe.Pointer(p)),TextMax:int32(len(syscall.StringToUTF16(s)))};send.Call(appGrid,LVM_SETITEMW,0,uintptr(unsafe.Pointer(&it)))};visibleIndex++}}
    appSetText(appStatus,fmt.Sprintf("Filas: %d | Columnas disponibles: %d | Visibles: %d",len(appView),len(appColumns),appVisibleCount()))
}
func appVisibleCount() int {n:=0;for _,c:=range appColumns{if c.Visible{n++}};return n}

func appApplyFilter() {
    q:=strings.TrimSpace(appText(appSearch)); if q=="" {appView=append([]Line(nil),appLines...);appRefreshGrid();return}
    terms:=strings.Split(q,";"); appView=nil
    for _,l:=range appLines {ok:=true;for _,term:=range terms {term=strings.TrimSpace(term);if term==""{continue};parts:=strings.SplitN(term,"=",2);if len(parts)!=2{parts=[]string{"",""};parts[1]=term};key:=strings.TrimSpace(parts[0]);want:=strings.ToLower(strings.TrimSpace(parts[1]));got:="";if key=="" {for _,v:=range l.Values {if strings.Contains(strings.ToLower(v),want){got=v;break}}} else {for k,v:=range l.Values {if strings.EqualFold(strings.TrimSpace(k),key){got=v;break}}};if !strings.Contains(strings.ToLower(got),want){ok=false;break}};if ok{appView=append(appView,l)}}
    appRefreshGrid()
}

func appShowColumns() {
    if appColumnsWindow!=0 {user32.NewProc("SetForegroundWindow").Call(appColumnsWindow);return}
    cls:=appU16("GestionSOColumns"); wc:=appWndClass{CbSize:uint32(unsafe.Sizeof(appWndClass{})),LpfnWndProc:syscall.NewCallback(appColumnsProc),HInstance:hInstance,HCursor:func()uintptr{r,_,_:=user32.NewProc("LoadCursorW").Call(0,32512);return r}(),LpszClassName:cls};user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)));title:=appU16("Columnas visibles");h,_,_:=user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(cls)),uintptr(unsafe.Pointer(title)),WS_OVERLAPPEDWINDOW|WS_VISIBLE,100,80,520,650,appHwnd,0,hInstance,0);appColumnsWindow=h
}
var appChecks=map[int]uintptr{}
func appColumnsProc(hwnd uintptr,msg uint32,wp,lp uintptr)uintptr {switch msg {case WM_CREATE: y:=10;appChecks=map[int]uintptr{};for i,c:=range appColumns {if i>55{break};h:=appMake(hwnd,"BUTTON",c.Name,WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_CHECKBOX,10,y,470,24,uintptr(3000+i));if c.Visible{user32.NewProc("SendMessageW").Call(h,0x00F1,BST_CHECKED,0)};appChecks[i]=h;y+=25};appMake(hwnd,"BUTTON","APLICAR",WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,10,570,100,30,3999);case WM_COMMAND:if int(wp&0xffff)==3999 {for i,h:=range appChecks {r,_,_:=user32.NewProc("SendMessageW").Call(h,0x00F0,0,0);appColumns[i].Visible=(r==BST_CHECKED)};user32.NewProc("DestroyWindow").Call(hwnd)};case WM_DESTROY:appColumnsWindow=0;user32.NewProc("PostMessageW").Call(appHwnd,WM_APP_COLUMNS,0,0)};r,_,_:=user32.NewProc("DefWindowProcW").Call(hwnd,uintptr(msg),wp,lp);return r}

func appPickXLSX(owner uintptr) []string {
    appLog("EVENTO: preparando diálogo GetOpenFileNameW")
    buf:=make([]uint16,32768); filter:=syscall.StringToUTF16("Excel (*.xlsx)\x00*.xlsx\x00Todos (*.*)\x00*.*\x00\x00");of:=appOpenFile{LStructSize:uint32(unsafe.Sizeof(appOpenFile{})),HwndOwner:owner,Filter:uintptr(unsafe.Pointer(&filter[0])),File:uintptr(unsafe.Pointer(&buf[0])),MaxFile:uint32(len(buf)),Flags:OFN_EXPLORER|OFN_FILEMUSTEXIST|OFN_ALLOWMULTISELECT|OFN_HIDEREADONLY};appLog("EVENTO: OPENFILENAME size=%d buffer=%d",unsafe.Sizeof(of),len(buf));r,_,e:=comdlg32.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(&of)));appLog("EVENTO: GetOpenFileNameW retorno=%d err=%v",r,e);if r==0{return nil};all:=syscall.UTF16ToString(buf);parts:=strings.Split(all,"\x00");if len(parts)<2{return []string{all}};dir:=parts[0];if len(parts)==2{return []string{dir}};out:=[]string{};for _,n:=range parts[1:] {if n==""{break};if filepath.IsAbs(n){out=append(out,n)}else{out=append(out,filepath.Join(dir,n))}};return out
}

func appExportCSV(owner uintptr) {
    if len(appView)==0{return}; path:=filepath.Join(os.TempDir(),"GestionSO-V57-resultado.csv");f,e:=os.Create(path);if e!=nil{return};defer f.Close();cols:=[]string{};for _,c:=range appColumns{if c.Visible{cols=append(cols,c.Name)}};f.WriteString(strings.Join(cols,";")+"\r\n");for _,l:=range appView{row:=make([]string,len(cols));for i,c:=range cols{s:=l.Values[c];s=strings.ReplaceAll(strings.ReplaceAll(s,";",","),"\r"," ");s=strings.ReplaceAll(s,"\n"," ");row[i]=s};f.WriteString(strings.Join(row,";")+"\r\n")};appSetText(appStatus,"CSV exportado: "+path)
}
