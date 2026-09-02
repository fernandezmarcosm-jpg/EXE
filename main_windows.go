//go:build windows

// RECONSTRUCCION DE GestionSO V57.
// Este archivo no es el fuente original. Los nombres de funciones se conservan
// cuando estan respaldados por simbolos observados en el binario V57; la
// implementacion Win32 es una reimplementacion documentada.

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	WM_CREATE = 0x0001
	WM_DESTROY = 0x0002
	WM_COMMAND = 0x0111
	WM_NOTIFY = 0x004E
	WM_INITDIALOG = 0x0110
	BN_CLICKED = 0
	SW_SHOW = 5
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE = 0x10000000
	WS_CHILD = 0x40000000
	WS_TABSTOP = 0x00010000
	BS_PUSHBUTTON = 0x00000000
	OFN_EXPLORER = 0x00080000
	OFN_FILEMUSTEXIST = 0x00001000
	OFN_ALLOWMULTISELECT = 0x00000200
	OFN_HIDEREADONLY = 0x00000004
	CW_USEDEFAULT = 0x80000000
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	pRegisterClassExW = user32.NewProc("RegisterClassExW")
	pCreateWindowExW = user32.NewProc("CreateWindowExW")
	pDefWindowProcW = user32.NewProc("DefWindowProcW")
	pShowWindow = user32.NewProc("ShowWindow")
	pUpdateWindow = user32.NewProc("UpdateWindow")
	pGetMessageW = user32.NewProc("GetMessageW")
	pTranslateMessage = user32.NewProc("TranslateMessage")
	pDispatchMessageW = user32.NewProc("DispatchMessageW")
	pPostQuitMessage = user32.NewProc("PostQuitMessage")
	pGetDlgItem = user32.NewProc("GetDlgItem")
	pFindWindowW = user32.NewProc("FindWindowW")
	pEnumWindows = user32.NewProc("EnumWindows")
	pEnumChildWindows = user32.NewProc("EnumChildWindows")
	pGetWindowTextW = user32.NewProc("GetWindowTextW")
	pGetWindowTextLenW = user32.NewProc("GetWindowTextLengthW")
	pSetWindowTextW = user32.NewProc("SetWindowTextW")
	pGetClassNameW = user32.NewProc("GetClassNameW")
	pGetClientRect = user32.NewProc("GetClientRect")
	pMoveWindow = user32.NewProc("MoveWindow")
	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	pGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type msg struct { Hwnd uintptr; Message uint32; WParam, LParam uintptr; Time uint32; Pt point }
type wndClassEx struct { CbSize uint32; Style uint32; LpfnWndProc uintptr; CbClsExtra int32; CbWndExtra int32; HInstance, HIcon, HCursor, HbrBackground, LpszMenuName, LpszClassName, HIconSm uintptr }
type openfilename struct { LStructSize uint32; HwndOwner uintptr; HInstance uintptr; LpstrFilter uintptr; LpstrCustomFilter uintptr; NMaxCustFilter uint32; NFilterIndex uint32; LpstrFile uintptr; NMaxFile uint32; LpstrFileTitle uintptr; NMaxFileTitle uint32; LpstrInitialDir uintptr; LpstrTitle uintptr; Flags uint32; NFileOffset uint16; NFileExtension uint16; LpstrDefExt uintptr; LCustData uintptr; LpfnHook uintptr; LpTemplateName uintptr; PvReserved uintptr; DwReserved uint32; FlagsEx uint32 }

var className = "GestionSO_V57_Reconstruction"
var wndProcCB = syscall.NewCallback(mainWndProc)
var hookProcCB = syscall.NewCallback(multiSelectHook)
var selectedFiles []string

func u16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func u16z(s string) []uint16 { return syscall.StringToUTF16(s) }

// multiSZ construye el buffer doble-NUL usado por OPENFILENAME multi-select.
func multiSZ(items []string) []uint16 { var out []uint16; for _, item := range items { out = append(out, syscall.StringToUTF16(item)...); out = append(out, 0) }; out = append(out, 0); return out }

func registerClass() { inst, _, _ := pGetModuleHandleW.Call(0); wc := wndClassEx{CbSize:uint32(unsafe.Sizeof(wndClassEx{})), LpfnWndProc:wndProcCB, HInstance:inst, LpszClassName:uintptr(unsafe.Pointer(u16(className)))}; pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))) }

func createWindow() uintptr { title:=u16("GestionSO V57 - Reconstruccion"); inst,_,_:=pGetModuleHandleW.Call(0); hwnd,_,_:=pCreateWindowExW.Call(0,uintptr(unsafe.Pointer(u16(className))),uintptr(unsafe.Pointer(title)),WS_OVERLAPPEDWINDOW|WS_VISIBLE,CW_USEDEFAULT,CW_USEDEFAULT,900,600,0,0,inst,0); if hwnd!=0 { createMainControls(hwnd); pShowWindow.Call(hwnd,SW_SHOW); pUpdateWindow.Call(hwnd) }; return hwnd }

func createMainControls(hwnd uintptr) { inst,_,_:=pGetModuleHandleW.Call(0); pCreateWindowExW.Call(0,uintptr(unsafe.Pointer(u16("BUTTON"))),uintptr(unsafe.Pointer(u16("ABRIR XLSX"))),WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON,20,20,160,32,hwnd,1001,inst,0) }
func layoutMain(hwnd uintptr) { var r rect; pGetClientRect.Call(hwnd,uintptr(unsafe.Pointer(&r))); btn:=getDlgItem(hwnd,1001); if btn!=0 { pMoveWindow.Call(btn,20,20,160,32,1) } }
func handleCommand(hwnd,wParam,lParam uintptr) uintptr { id:=uint16(wParam&0xffff); notify:=uint16((wParam>>16)&0xffff); if id==1001 && notify==BN_CLICKED { openXLSXDialog(hwnd); return 0 }; return 0 }
func handleNotify(hwnd,wParam,lParam uintptr) uintptr { return handleMainNotify(hwnd,wParam,lParam) }
func handleMainNotify(hwnd,wParam,lParam uintptr) uintptr { return 0 }

func mainWndProc(hwnd uintptr,msgID uint32,wParam,lParam uintptr) uintptr { switch msgID { case WM_CREATE: installMultiSelectButton(hwnd); return 0; case WM_COMMAND: return handleCommand(hwnd,wParam,lParam); case WM_NOTIFY: return handleNotify(hwnd,wParam,lParam); case 0x0005: layoutMain(hwnd); return 0; case WM_DESTROY: pPostQuitMessage.Call(0); return 0 }; ret,_,_:=pDefWindowProcW.Call(hwnd,uintptr(msgID),wParam,lParam); return ret }
func msgLoop() { var m msg; for { r,_,_:=pGetMessageW.Call(uintptr(unsafe.Pointer(&m)),0,0,0); if int32(r)<=0{return}; pTranslateMessage.Call(uintptr(unsafe.Pointer(&m))); pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m))) } }

func installMultiSelectButton(hwnd uintptr) { logf("ABRIR XLSX multi-select hook installed hwnd=%x",hwnd); _=findWindowByTitles([]string{"Gestion SO V54","GestionSO V54"}) }
func openXLSXDialog(owner uintptr) { files:=pickMultipleXLSX(owner); if len(files)==0{return}; selectedFiles=files; for _,f:=range files { feedEngineFile(owner,f) } }

func pickMultipleXLSX(owner uintptr) []string { buf:=make([]uint16,32768); filter:=u16z("Archivos XLSX (*.xlsx)\x00*.xlsx\x00Todos los archivos (*.*)\x00*.*\x00\x00"); title:=u16("ABRIR XLSX"); of:=openfilename{LStructSize:uint32(unsafe.Sizeof(openfilename{})),HwndOwner:owner,LpstrFilter:uintptr(unsafe.Pointer(&filter[0])),LpstrFile:uintptr(unsafe.Pointer(&buf[0])),NMaxFile:uint32(len(buf)),LpstrTitle:uintptr(unsafe.Pointer(title)),Flags:OFN_EXPLORER|OFN_ALLOWMULTISELECT|OFN_FILEMUSTEXIST|OFN_HIDEREADONLY,LpfnHook:hookProcCB}; r,_,_:=pGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&of))); if r==0{return nil}; return parseMultiSelectBuffer(buf) }
func parseMultiSelectBuffer(buf []uint16) []string { parts:=make([]string,0,8); start:=0; for start<len(buf){ end:=start; for end<len(buf)&&buf[end]!=0{end++}; if end==start{break}; s:=syscall.UTF16ToString(buf[start:end]); if s!=""{parts=append(parts,s)}; start=end+1 }; if len(parts)<=1{return parts}; dir:=parts[0]; out:=make([]string,0,len(parts)-1); for _,name:=range parts[1:] { if strings.Contains(name,`\`){out=append(out,name)}else{out=append(out,dir+`\`+name)} }; return out }
func repositionOverlay(hwnd uintptr) { _=hwnd }

func findWindowByTitles(titles []string) uintptr { for _,title:=range titles { if h:=findWindowByTitle(title); h!=0{return h} }; return 0 }
func findWindowByTitle(title string) uintptr { h,_,_:=pFindWindowW.Call(0,uintptr(unsafe.Pointer(u16(title)))); return h }
func enumTopWindows(fn func(uintptr)bool) { cb:=syscall.NewCallback(func(hwnd,lParam uintptr)uintptr{if fn(hwnd){return 1};return 0}); pEnumWindows.Call(cb,0) }
func enumChildren(hwnd uintptr,fn func(uintptr)bool) { cb:=syscall.NewCallback(func(child,lParam uintptr)uintptr{if fn(child){return 1};return 0}); pEnumChildWindows.Call(hwnd,cb,0) }
func findChildByText(hwnd uintptr,text string) uintptr { var found uintptr; enumChildren(hwnd,func(child uintptr)bool{if windowText(child)==text{found=child;return false};return true}); return found }
func findFirstEdit(hwnd uintptr) uintptr { var found uintptr; enumChildren(hwnd,func(child uintptr)bool{if strings.EqualFold(getClassName(child),"EDIT"){found=child;return false};return true}); return found }
func findDialogUnder(hwnd uintptr) uintptr { var found uintptr; enumTopWindows(func(w uintptr)bool{if w!=hwnd{found=w;return false};return true}); return found }
func windowText(hwnd uintptr) string { if hwnd==0{return ""}; n,_,_:=pGetWindowTextLenW.Call(hwnd); buf:=make([]uint16,n+1); pGetWindowTextW.Call(hwnd,uintptr(unsafe.Pointer(&buf[0])),n+1); return syscall.UTF16ToString(buf) }
func getClassName(hwnd uintptr) string { buf:=make([]uint16,256); n,_,_:=pGetClassNameW.Call(hwnd,uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf))); return syscall.UTF16ToString(buf[:n]) }
func setWindowText(hwnd uintptr,text string) { pSetWindowTextW.Call(hwnd,uintptr(unsafe.Pointer(u16(text)))) }
func getDlgItem(hwnd uintptr,id int) uintptr { h,_,_:=pGetDlgItem.Call(hwnd,uintptr(id)); return h }

func multiSelectHook(hwnd,msgID,wParam,lParam uintptr) uintptr { if msgID==WM_INITDIALOG { logf("multi-select dialog hook hwnd=%x",hwnd) }; return 0 }

// El simbolo feedEngineFile EXISTE en el binario real. El contrato interno del
// motor V54 no puede verificarse sin GestionSO-V54-engine.exe.
func feedEngineFile(owner uintptr,file string) { engine:=os.Getenv("GESTIONSO_V54_ENGINE"); if engine=="" { logf("feedEngineFile owner=%x file=%q engine=not-configured",owner,file); return }; logf("feedEngineFile owner=%x file=%q engine=%q",owner,file,engine) }

var logFile *os.File
var logger *log.Logger
func initLog() { if logger!=nil{return}; path:=os.TempDir()+`\\GestionSO-V57-debug.log`; f,err:=os.OpenFile(path,os.O_CREATE|os.O_APPEND|os.O_WRONLY,0644); if err==nil {logFile=f;logger=log.New(f,"",log.LstdFlags)} }
func logf(format string,args ...interface{}) { initLog(); if logger!=nil{logger.Printf(format,args...)}else{_=fmt.Sprintf(format,args...)} }
