//go:build windows

package main

import (
    "fmt"
    "path/filepath"
    "sort"
    "strings"
    "syscall"
    "unsafe"
)

const (
    appIDOpen = 2001
    appIDStatus = 2006
    appIDGrid = 2007
    WM_APP_REFRESH = 0x8001
    BS_PUSHBUTTON = 0
    WS_OVERLAPPEDWINDOW = 0x00CF0000
    WS_VISIBLE = 0x10000000
    WS_CHILD = 0x40000000
    WS_TABSTOP = 0x00010000
    WS_BORDER = 0x00800000
    WS_VSCROLL = 0x00200000
    WS_HSCROLL = 0x00100000
    ES_MULTILINE = 0x0004
    ES_AUTOVSCROLL = 0x0040
    ES_AUTOHSCROLL = 0x0080
    ES_READONLY = 0x0800
    ES_WANTRETURN = 0x1000
    WM_CREATE = 0x0001
    WM_SIZE = 0x0005
    WM_COMMAND = 0x0111
    WM_CLOSE = 0x0010
    WM_DESTROY = 0x0002
)

type appPoint struct{ X, Y int32 }
type appRect struct{ Left, Top, Right, Bottom int32 }
type appWndClass struct { CbSize uint32; Style uint32; LpfnWndProc uintptr; CbClsExtra, CbWndExtra int32; HInstance, HIcon, HCursor, HbrBackground uintptr; LpszMenuName, LpszClassName *uint16; HIconSm uintptr }
type appOpenFile struct { LStructSize uint32; Pad0 uint32; HwndOwner, HInstance, Filter, CustomFilter uintptr; MaxCustom, FilterIndex uint32; File uintptr; MaxFile uint32; Pad1 uint32; FileTitle uintptr; MaxFileTitle uint32; Pad2 uint32; InitialDir, Title uintptr; Flags uint32; FileOffset, FileExtension uint16; DefExt, CustData, Hook, Template uintptr; Reserved uintptr; Reserved2, FlagsEx uint32 }

var (
    appHwnd, appView, appStatus uintptr
    appHInstance uintptr
    appStatusText string
    appViewText string
    appSource string
)

func appU16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func appSetText(h uintptr, s string) { if h != 0 { p:=appU16(s); user32.NewProc("SetWindowTextW").Call(h, uintptr(unsafe.Pointer(p))) } }
func appMake(parent uintptr, cls, text string, style uint32, x,y,w,h int, id uintptr) uintptr { c:=appU16(cls); t:=appU16(text); r,_,_:=user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(c)),uintptr(unsafe.Pointer(t)),uintptr(style),uintptr(x),uintptr(y),uintptr(w),uintptr(h),parent,id,appHInstance,0); return r }

func crearVentana() uintptr {
    appHInstance,_,_ = kernel32.NewProc("GetModuleHandleW").Call(0)
    cls:=appU16("GestionSOExcelImporter")
    wc:=appWndClass{CbSize:uint32(unsafe.Sizeof(appWndClass{})),LpfnWndProc:syscall.NewCallback(appWndProcLogged),HInstance:appHInstance,HCursor:func() uintptr { r,_,_:=user32.NewProc("LoadCursorW").Call(0,32512); return r }(),LpszClassName:cls}
    user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
    title:=appU16("GestionSO V57 - Importar Excel")
    h,_,_:=user32.NewProc("CreateWindowExW").Call(0,uintptr(unsafe.Pointer(cls)),uintptr(unsafe.Pointer(title)),WS_OVERLAPPEDWINDOW|WS_VISIBLE,0x80000000,0x80000000,1400,820,0,0,appHInstance,0)
    return h
}

func appWndProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
    switch msg {
    case WM_CREATE:
        appHwnd=hwnd
        appBuildControls(hwnd)
        return 0
    case WM_SIZE:
        appLayout(hwnd)
        return 0
    case WM_COMMAND:
        if int(wp&0xffff)==appIDOpen { appOpenXLSX(hwnd) }
        return 0
    case WM_APP_REFRESH:
        appRefreshView()
        return 0
    case WM_CLOSE:
        user32.NewProc("DestroyWindow").Call(hwnd)
        return 0
    case WM_DESTROY:
        user32.NewProc("PostQuitMessage").Call(0)
        return 0
    }
    r,_,_:=user32.NewProc("DefWindowProcW").Call(hwnd,uintptr(msg),wp,lp)
    return r
}

func appBuildControls(hwnd uintptr) {
    appMake(hwnd,"BUTTON","ABRIR EXCEL",WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,10,10,130,32,appIDOpen)
    appStatus=appMake(hwnd,"STATIC","Listo. Seleccione un archivo XLSX para comenzar.",WS_CHILD|WS_VISIBLE,155,15,1100,24,appIDStatus)
    appView=appMake(hwnd,"EDIT","",WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|WS_HSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_AUTOHSCROLL|ES_READONLY|ES_WANTRETURN,10,55,1360,700,appIDGrid)
}

func appLayout(hwnd uintptr) {
    var r appRect
    user32.NewProc("GetClientRect").Call(hwnd,uintptr(unsafe.Pointer(&r)))
    w:=int(r.Right-r.Left); h:=int(r.Bottom-r.Top)
    if w<600 {w=600}; if h<300 {h=300}
    user32.NewProc("MoveWindow").Call(appStatus,155,15,uintptr(maxInt(300,w-170)),24,1)
    user32.NewProc("MoveWindow").Call(appView,10,55,uintptr(w-20),uintptr(h-65),1)
}
func maxInt(a,b int) int { if a>b{return a}; return b }

func appPickXLSX(owner uintptr) []string {
    const (
        ofnExplorer      = 0x00080000
        ofnFileMustExist = 0x00001000
        ofnPathMustExist = 0x00000800
        ofnHideReadOnly  = 0x00000004
    )
    f1, _ := syscall.UTF16FromString("Archivos Excel (*.xlsx)")
    f2, _ := syscall.UTF16FromString("*.xlsx")
    f3, _ := syscall.UTF16FromString("Todos los archivos (*.*)")
    f4, _ := syscall.UTF16FromString("*.*")
    filter := make([]uint16, 0, len(f1)+len(f2)+len(f3)+len(f4)+1)
    filter = append(filter, f1...); filter = append(filter, f2...); filter = append(filter, f3...); filter = append(filter, f4...); filter = append(filter, 0)
    buffer := make([]uint16, 32768)
    title := appU16("Seleccionar archivo Excel")
    defExt := appU16("xlsx")
    ofn := appOpenFile{LStructSize:uint32(unsafe.Sizeof(appOpenFile{})),HwndOwner:owner,Filter:uintptr(unsafe.Pointer(&filter[0])),FilterIndex:1,File:uintptr(unsafe.Pointer(&buffer[0])),MaxFile:uint32(len(buffer)),Title:uintptr(unsafe.Pointer(title)),Flags:ofnExplorer|ofnFileMustExist|ofnPathMustExist|ofnHideReadOnly,DefExt:uintptr(unsafe.Pointer(defExt))}
    ret,_,_:=comdlg32.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(&ofn)))
    if ret==0{return nil}
    n:=0; for n<len(buffer)&&buffer[n]!=0{n++}
    if n==0{return nil}
    path:=syscall.UTF16ToString(buffer[:n])
    if strings.TrimSpace(path)==""{return nil}
    return []string{path}
}

func appOpenXLSX(owner uintptr) {
    files:=appPickXLSX(owner)
    if len(files)==0{return}
    appStatusText=fmt.Sprintf("Leyendo %s...",filepath.Base(files[0]))
    appSetText(appStatus,appStatusText)
    go func(path string) {
        defer appRecover("importacion XLSX")
        headers,rows,source,err:=loadPreviewXLSX(path)
        if err!=nil {
            appStatusText="ERROR: "+err.Error()
            appViewText=""
            appSource=""
            user32.NewProc("PostMessageW").Call(appHwnd,WM_APP_REFRESH,0,0)
            return
        }
        appStatusText=fmt.Sprintf("%d fila(s) cargadas — %s",len(rows),filepath.Base(source))
        appViewText=renderPreview(headers,rows)
        appSource=source
        user32.NewProc("PostMessageW").Call(appHwnd,WM_APP_REFRESH,0,0)
    }(files[0])
}

func loadPreviewXLSX(path string) ([]string,[]appRow,string,error) {
    doc,err:=ReadXLSX(path); if err!=nil{return nil,nil,"",err}
    if len(doc.Sheets)==0{return nil,nil,"",fmt.Errorf("el Excel no contiene hojas legibles")}
    names:=make([]string,0,len(doc.Sheets)); for n:=range doc.Sheets {names=append(names,n)}; sort.Strings(names)
    rows:=doc.Sheets[names[0]]
    if len(rows)==0{return nil,nil,"",fmt.Errorf("la hoja %q no contiene filas de datos",filepath.Base(names[0]))}
    n:=maxPreviewColumns(rows)
    headers:=makeHeaders(rows[0],n)
    out:=make([]appRow,0,len(rows)-1)
    for i:=1;i<len(rows);i++ { vals:=make([]string,n); for j:=0;j<n;j++ {if j<len(rows[i]){vals[j]=rows[i][j]}}; out=append(out,appRow{Values:vals}) }
    if len(out)>5000 {out=out[:5000]}
    return headers,out,path,nil
}
func maxPreviewColumns(rows [][]string) int {n:=0;for _,r:=range rows{if len(r)>n{n=len(r)}};if n>80{n=80};return n}
func makeHeaders(row []string,n int) []string {out:=make([]string,n);seen:=map[string]int{};for i:=0;i<n;i++{name:="Columna "+fmt.Sprint(i+1);if i<len(row)&&strings.TrimSpace(row[i])!=""{name=strings.TrimSpace(row[i])};base:=name;seen[base]++;if seen[base]>1{name=fmt.Sprintf("%s (%d)",base,seen[base])};out[i]=name};return out}

func renderPreview(headers []string, rows []appRow) string {
    var b strings.Builder
    for i,h:=range headers {if i>0{b.WriteByte('\t')};b.WriteString(cleanCell(h))}
    b.WriteByte('\r');b.WriteByte('\n')
    for _,row:=range rows {
        for i:=range headers {if i>0{b.WriteByte('\t')}; if i<len(row.Values){b.WriteString(cleanCell(row.Values[i]))}}
        b.WriteByte('\r');b.WriteByte('\n')
    }
    return b.String()
}
func cleanCell(s string) string {return strings.NewReplacer("\r"," ","\n"," ","\t"," ").Replace(s)}

func appRefreshView() {
    appSetText(appStatus,appStatusText)
    appSetText(appView,appViewText)
}
