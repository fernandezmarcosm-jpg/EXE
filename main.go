//go:build windows

package main

import (
    "syscall"
    "time"
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
    appLogInit()
    defer appRecover("main")
    appLog("EVENTO: inicio del programa")

    console, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
    if console != 0 {
        user32.NewProc("ShowWindow").Call(console, 0)
    }
    comctl32.NewProc("InitCommonControls").Call()
    appLog("EVENTO: common controls inicializados")

    hwnd := crearVentana()
    appLog("EVENTO: crearVentana => hwnd=0x%X", hwnd)
    if hwnd == 0 {
        appLog("ERROR: no se pudo crear la ventana principal")
        return
    }

    // Heartbeat independiente del hilo de UI. Si el log continúa avanzando
    // mientras la interfaz queda congelada, sabremos que el proceso vive y
    // que el bloqueo está dentro del hilo de mensajes/Win32.
    go func() {
        ticker := time.NewTicker(2 * time.Second)
        defer ticker.Stop()
        for range ticker.C {
            appLog("HEARTBEAT: proceso activo hwnd=0x%X", appHwnd)
        }
    }()

    var msg winMSG
    seq := uint64(0)
    for {
        ret, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
        if int32(ret) <= 0 {
            appLog("EVENTO: GetMessageW finalizó ret=%d", int32(ret))
            break
        }
        seq++
        logMessage := shouldLogMessage(msg.Message)
        var start time.Time
        if logMessage {
            start = time.Now()
            appLog("MSG[%d] antes Translate/Dispatch msg=0x%X hwnd=0x%X wp=0x%X lp=0x%X", seq, msg.Message, msg.Hwnd, msg.WParam, msg.LParam)
        }
        user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
        user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
        if logMessage {
            elapsed := time.Since(start)
            appLog("MSG[%d] después Dispatch msg=0x%X duración=%s", seq, msg.Message, elapsed)
            if elapsed > 500*time.Millisecond {
                appLog("DIAGNOSTICO: DispatchMessageW tardó %s para msg=0x%X", elapsed, msg.Message)
            }
        }
    }
    appLog("=== FIN GestionSO V57 ===")
}

func shouldLogMessage(msg uint32) bool {
    switch msg {
    case 0x0001, // WM_CREATE
        0x0002, // WM_DESTROY
        0x0005, // WM_SIZE
        0x000F, // WM_PAINT
        0x0010, // WM_CLOSE
        0x0111, // WM_COMMAND
        0x0100, // WM_KEYDOWN
        0x0101, // WM_KEYUP
        0x0201, // WM_LBUTTONDOWN
        0x0202, // WM_LBUTTONUP
        0x8001: // WM_APP_REFRESH
        return true
    default:
        return false
    }
}
