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
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	hInstance uintptr
)

type winPOINT struct{ X, Y int32 }
type winMSG struct {
	Hwnd uintptr
	Message uint32
	WParam, LParam uintptr
	Time uint32
	Pt winPOINT
}

// Punto de entrada único de la reconstrucción Win32.
func main() {
	console, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
	if console != 0 {
		user32.NewProc("ShowWindow").Call(console, 0)
	}
	comctl32.NewProc("InitCommonControls").Call()

	hwnd := crearVentana()
	if hwnd == 0 {
		log.Fatal("No se pudo crear la ventana principal")
	}

	var msg winMSG
	for {
		ret, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
		user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
	}
}
