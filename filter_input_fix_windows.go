//go:build windows
package main

import (
    "syscall"
    "time"
)

const (
    filterFixSetWindowLongPtr = -4
    filterFixEnChange         = 0x0300
    filterFixEnSetFocus       = 0x0100
    filterFixEnKillFocus      = 0x0200
)

var filterFixOldWndProc uintptr
var filterFixCallback uintptr
var filterFixInstalled bool

func init() {
    filterFixCallback = syscall.NewCallback(filterFixWndProc)
    go func() {
        for i := 0; i < 600 && !filterFixInstalled; i++ {
            if appHwnd != 0 {
                filterFixInstall(appHwnd)
                return
            }
            time.Sleep(50 * time.Millisecond)
        }
    }()
}

func filterFixInstall(hwnd uintptr) {
    if hwnd == 0 || filterFixInstalled { return }
    p := syscall.NewLazyDLL("user32.dll")
    proc := p.NewProc("SetWindowLongPtrW")
    old, _, _ := proc.Call(hwnd, filterFixSetWindowLongPtr, filterFixCallback)
    if old == 0 { return }
    filterFixOldWndProc = old
    filterFixInstalled = true
    appLog("DIAGNOSTICO: filtro: refresco durante edición deshabilitado")
}

func filterFixWndProc(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
    if msg == WM_COMMAND {
        id := int(w & 0xffff)
        notify := int((w >> 16) & 0xffff)
        if id >= filterBaseID {
            if notify == filterFixEnChange || notify == filterFixEnSetFocus {
                return 0
            }
            if notify == filterFixEnKillFocus {
                return filterFixCallOld(hwnd, msg, w, l)
            }
        }
    }
    return filterFixCallOld(hwnd, msg, w, l)
}

func filterFixCallOld(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
    if filterFixOldWndProc == 0 { return 0 }
    p := syscall.NewLazyDLL("user32.dll")
    return p.NewProc("CallWindowProcW").Call(filterFixOldWndProc, hwnd, uintptr(msg), w, l)
}
