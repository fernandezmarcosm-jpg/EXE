//go:build windows
package main

import (
    "syscall"
    "time"
)

const (
    filterFixSetWindowLongPtr = ^uintptr(3)
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
        for i := 0; i < 1200; i++ {
            if appHwnd != 0 {
                filterFixInstall(appHwnd)
                filterFixRemoveControls(appHwnd)
            }
            time.Sleep(100 * time.Millisecond)
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
    appLog("DIAGNOSTICO: filtros deshabilitados: sin controles de filtro ni refresco automático")
}

// Los filtros se deshabilitan por completo por estabilidad: no se crean campos
// utilizables ni se permite que un cambio de filtro dispare recalculados.
// Se eliminan también si columnViewBuildFilters los recrea tras un cambio de columnas.
func filterFixRemoveControls(parent uintptr) {
    if parent == 0 { return }
    user32 := syscall.NewLazyDLL("user32.dll")
    find := user32.NewProc("FindWindowExW")
    getID := user32.NewProc("GetDlgCtrlID")
    destroy := user32.NewProc("DestroyWindow")
    prev := uintptr(0)
    removed := 0
    for i := 0; i < 1000; i++ {
        h, _, _ := find.Call(parent, prev, 0, 0)
        if h == 0 { break }
        prev = h
        id, _, _ := getID.Call(h)
        if int(id) >= filterBaseID {
            destroy.Call(h)
            removed++
            prev = 0
        }
    }
    if removed > 0 {
        appLog("DIAGNOSTICO: filtros eliminados=%d; valores importados quedan estancos", removed)
    }
}

func filterFixWndProc(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
    if msg == WM_COMMAND {
        id := int(w & 0xffff)
        notify := int((w >> 16) & 0xffff)
        if id >= filterBaseID {
            if notify == filterFixEnChange || notify == filterFixEnSetFocus || notify == filterFixEnKillFocus {
                return 0
            }
        }
    }
    return filterFixCallOld(hwnd, msg, w, l)
}

func filterFixCallOld(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
    if filterFixOldWndProc == 0 { return 0 }
    p := syscall.NewLazyDLL("user32.dll")
    r, _, _ := p.NewProc("CallWindowProcW").Call(filterFixOldWndProc, hwnd, uintptr(msg), w, l)
    return r
}
