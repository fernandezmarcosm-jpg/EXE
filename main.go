package main

import (
    "log"
    "os"
    "syscall"
    "unsafe"
)

var (
    user32   = syscall.NewLazyDLL("user32.dll")
    kernel32 = syscall.NewLazyDLL("kernel32.dll")
    comctl32 = syscall.NewLazyDLL("comctl32.dll")
)

const (
    WM_CLOSE   = 0x0010
    WM_COMMAND = 0x0111
    WM_SIZE    = 0x0005
    BN_CLICKED = 0
    
    ID_ABRIR_XLSX   = 1001
    ID_EXPORTAR_CSV = 1005
)

func main() {
    console := kernel32.NewProc("GetConsoleWindow").Call()
    if console != 0 {
        user32.NewProc("ShowWindow").Call(console, 0)
    }
    
    comctl32.NewProc("InitCommonControls").Call()
    
    hwnd := crearVentanaPrincipal()
    if hwnd == 0 {
        log.Fatal("No se pudo crear la ventana principal")
    }
    
    var msg syscall.MSG
    for {
        ret, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
        if ret == 0 {
            break
        }
        user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
        user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
    }
}
