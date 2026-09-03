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
    LVS_REPORT = 0x0001
    LVS_SHOWSELALWAYS = 0x0008
    LVS_EX_FULLROWSELECT = 0x0020
    LVM_FIRST = 0x1000
    LVM_DELETEALLITEMS = LVM_FIRST + 9
    LVM_INSERTITEMW = LVM_FIRST + 77
    LVM_INSERTCOLUMNW = LVM_FIRST + 97
    LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
    LVCF_FMT = 0x0001
    LVCF_WIDTH = 0x0002
    LVCF_TEXT = 0x0004
    LVCFMT_LEFT = 0
    LVIF_TEXT = 0x0001
    WM_CREATE = 0x0001
    WM_SIZE = 0x0005
    WM_COMMAND = 0x0111
    WM_CLOSE = 0x0010
    WM_DESTROY = 0x0002
)

type appPoint struct{ X, Y int32 }
type appRect struct{ Left, Top, Right, Bottom int32 }
type appWndClass struct { CbSize uint32; Style uint32; LpfnWndProc uintptr; CbClsExtra, CbWndExtra int32; HInstance, HIcon, HCursor, HbrBackground uintptr; LpszMenuName, LpszClassName *uint16; HIconSm uintptr }
type appLVC struct { Mask uint32; Fmt, Cx int32; Text uintptr; TextMax int32; SubItem, Image, Order int32 }
type appLVI struct { Mask uint32; Item, SubItem int32; State, StateMask uint32; Text uintptr; TextMax, Image int32; LParam uintptr }
type appOpenFile struct { LStructSize uint32; Pad0 uint32; HwndOwner, HInstance, Filter, CustomFilter uintptr; MaxCustom, FilterIndex uint32; File uintptr; MaxFile uint32; Pad1 uint32; FileTitle uintptr; MaxFileTitle uint32; Pad2 uint32; InitialDir, Title uintptr; Flags uint32; FileOffset, FileExtension uint16; DefExt, CustData, Hook, Template uintptr; Reserved uintptr; Reserved2, FlagsEx uint32 }

type appRow struct { Values []string }

var (
    appHwnd, appGrid, appStatus uintptr
    appHInstance uintptr
    appHeaders []string
    appRows []appRow
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
        appHwnd=hwnd; appBuildControls(hwnd); return 0
    case WM_SIZE:
        appLayout(hwnd); return 0
    case WM_COMMAND:
        if int(wp&0xffff)==appIDOpen { appOpenXLSX(hwnd) }
        return 0
    case WM_APP_REFRESH:
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
    appMake(hwnd,"BUTTON","ABRIR EXCEL",WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,10,10,130,32,appIDOpen)
    appStatus=appMake(hwnd,"STATIC","Listo. Seleccione un archivo XLSX para comenzar.",WS_CHILD|WS_VISIBLE,155,15,1100,24,appIDStatus)
    appGrid=appMake(hwnd,"SysListView32","",WS_CHILD|WS_VISIBLE|LVS_REPORT|LVS_SHOWSELALWAYS|WS_BORDER,10,55,1360,700,appIDGrid)
    user32.NewProc("SendMessageW").Call(appGrid,LVM_SETEXTENDEDLISTVIEWSTYLE,LVS_EX_FULLROWSELECT,LVS_EX_FULLROWSELECT)
}

func appLayout(hwnd uintptr) {
    var r appRect; user32.NewProc("GetClientRect").Call(hwnd,uintptr(unsafe.Pointer(&r)))
    w:=int(r.Right-r.Left); h:=int(r.Bottom-r.Top); if w<600 {w=600}; if h<300 {h=300}
    user32.NewProc("MoveWindow").Call(appStatus,155,15,uintptr(maxInt(300,w-170)),24,1)
    user32.NewProc("MoveWindow").Call(appGrid,10,55,uintptr(w-20),uintptr(h-95),1)
}
func maxInt(a,b int) int { if a>b{return a}; return b }

func appOpenXLSX(owner uintptr) {
    files:=appPickXLSX(owner); if len(files)==0 { return }
    appSetText(appStatus,fmt.Sprintf("Leyendo %d archivo(s)...",len(files)))
    go func(selected []string) {
        defer appRecover("importacion XLSX")
        headers,rows,source,err:=loadPreviewXLSX(selected[0])
        if err!=nil { appLog("ERROR importando XLSX: %v",err); appSetText(appStatus,"ERROR: "+err.Error()); return }
        appHeaders=headers; appRows=rows; appSource=source
        user32.NewProc("PostMessageW").Call(appHwnd,WM_APP_REFRESH,0,0)
    }(append([]string(nil),files...))
}

func loadPreviewXLSX(path string) ([]string,[]appRow,string,error) {
    doc,err:=ReadXLSX(path); if err!=nil{return nil,nil,"",err}
    if len(doc.Sheets)==0{return nil,nil,"",fmt.Errorf("el Excel no contiene hojas legibles")}
    names:=make([]string,0,len(doc.Sheets)); for n:=range doc.Sheets {names=append(names,n)}; sort.Strings(names)
    rows:=doc.Sheets[names[0]]; if len(rows)==0{return nil,nil,path,nil}
    headers:=makeHeaders(rows[0],maxPreviewColumns(rows)); out:=make([]appRow,0,len(rows)-1)
    for i:=1;i<len(rows);i++ { vals:=make([]string,len(headers)); for j:=range vals {if j<len(rows[i]){vals[j]=rows[i][j]}}; out=append(out,appRow{Values:vals}) }
    return headers,out,path,nil
}
func maxPreviewColumns(rows [][]string) int {n:=0;for _,r:=range rows{if len(r)>n{n=len(r)}};if n>80{n=80};return n}
func makeHeaders(row []string,n int) []string {out:=make([]string,n);seen:=map[string]int{};for i:=0;i<n;i++{name:="Columna "+fmt.Sprint(i+1);if i<len(row)&&strings.TrimSpace(row[i])!=""{name=strings.TrimSpace(row[i])};base:=name;seen[base]++;if seen[base]>1{name=fmt.Sprintf("%s (%d)",base,seen[base])};out[i]=name};return out}

func appRefreshGrid() {
    if appGrid==0{return}; send:=user32.NewProc("SendMessageW"); send.Call(appGrid,LVM_DELETEALLITEMS,0,0)
    for i,h:=range appHeaders {text:=appU16(h);c:=appLVC{Mask:LVCF_TEXT|LVCF_WIDTH|LVCF_FMT,Fmt:LVCFMT_LEFT,Cx:140,Text:uintptr(unsafe.Pointer(text)),TextMax:int32(len([]rune(h))+1)};send.Call(appGrid,LVM_INSERTCOLUMNW,uintptr(i),uintptr(unsafe.Pointer(&c)))}
    for i,row:=range appRows {if i>=5000{break};first:="";if len(row.Values)>0{first=row.Values[0]};p:=appU16(first);item:=appLVI{Mask:LVIF_TEXT,Item:int32(i),SubItem:0,Text:uintptr(unsafe.Pointer(p)),TextMax:int32(len([]rune(first))+1)};send.Call(appGrid,LVM_INSERTITEMW,0,uintptr(unsafe.Pointer(&item)));for j:=1;j<len(appHeaders);j++{v:="";if j<len(row.Values){v=row.Values[j]};pv:=appU16(v);item2:=appLVI{Mask:LVIF_TEXT,Item:int32(i),SubItem:int32(j),Text:uintptr(unsafe.Pointer(pv)),TextMax:int32(len([]rune(v))+1)};send.Call(appGrid,LVM_FIRST+46,0,uintptr(unsafe.Pointer(&item2)))}}
    appSetText(appStatus,fmt.Sprintf("%d fila(s) mostradas — %s",len(appRows),filepath.Base(appSource)))
}
