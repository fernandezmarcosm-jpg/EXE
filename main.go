//go:build windows

package main

import (
	"log"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
)

const (
	WM_CREATE  = 0x0001
	WM_DESTROY = 0x0002
	WM_CLOSE   = 0x0010
	WM_COMMAND = 0x0111
	WM_SIZE    = 0x0005
	BN_CLICKED = 0

	ID_ABRIR_XLSX   = 1001
	ID_RECARGAR     = 1002
	ID_COLUMNAS     = 1003
	ID_FILTRAR      = 1004
	ID_EXPORTAR_CSV = 1005
	ID_LIMPIAR      = 1006
)

type winPOINT struct { X, Y int32 }
type winMSG struct { Hwnd uintptr; Message uint32; WParam, LParam uintptr; Time uint32; Pt winPOINT }

func main() {
	console, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
	if console != 0 { user32.NewProc("ShowWindow").Call(console, 0) }
	comctl32.NewProc("InitCommonControls").Call()
	hwnd := crearVentana()
	if hwnd == 0 { log.Fatal("No se pudo crear la ventana principal") }
	var msg winMSG
	for {
		ret, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 { break }
		user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
		user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
	}
}
