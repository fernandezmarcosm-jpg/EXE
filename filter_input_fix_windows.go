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
    filterFixWmKeyDown        = 0x0100
    filterFixVkReturn         = 0x000D
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
    appLog("DIAGNOSTICO: filtro: refresco automático deshabilitado; aplicar con ENTER")
}

func filterFixWndProc(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
    if msg == filterFixWmKeyDown && w == filterFixVkReturn {
        for _, h := range viewFilters {
            if h != 0 && filterFixIsEditFocused(h) {
                filterFixApplyFocusedFilter(h)
                return 0
            }
        }
    }
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

func filterFixApplyFocusedFilter(editHwnd uintptr) {
    for id, h := range viewFilters {
        if h == editHwnd {
            columnViewHandleFilterChange(uintptr(filterBaseID))
            appLog("DIAGNOSTICO: filtro aplicado explícitamente con ENTER; columna=%s", id)
            return
        }
    }
}

func filterFixIsEditFocused(target uintptr) bool {
    p := syscall.NewLazyDLL("user32.dll")
    focused, _, _ := p.NewProc("GetFocus").Call()
    return focused == target
}

func filterFixCallOld(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
    if filterFixOldWndProc == 0 { return 0 }
    p := syscall.NewLazyDLL("user32.dll")
    r, _, _ := p.NewProc("CallWindowProcW").Call(filterFixOldWndProc, hwnd, uintptr(msg), w, l)
    return r
}
