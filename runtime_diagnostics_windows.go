//go:build windows

package main

import (
    "fmt"
    "time"
)

// Wrapper de diagnóstico alrededor del callback Win32. Un panic dentro del
// callback queda registrado sin llevarse por delante silenciosamente la UI.
func appWndProcLogged(hwnd uintptr, msg uint32, wp, lp uintptr) (ret uintptr) {
    start := time.Now()
    defer func() {
        elapsed := time.Since(start)
        if v := recover(); v != nil {
            appLogPanic(fmt.Sprintf("WndProc msg=0x%X hwnd=0x%X", msg, hwnd), v)
            ret = 0
            return
        }
        // Registrar sólo mensajes relevantes y callbacks anormalmente lentos.
        // Esto permite distinguir un crash de un bloqueo dentro de Win32.
        if msg == WM_COMMAND || msg == WM_CLOSE || msg == WM_DESTROY || elapsed > 500*time.Millisecond {
            appLog("WNDPROC salida msg=0x%X hwnd=0x%X wp=0x%X lp=0x%X duración=%s ret=0x%X", msg, hwnd, wp, lp, elapsed, ret)
        }
    }()

    if msg == WM_COMMAND {
        appLog("WM_COMMAND entrada hwnd=0x%X id=%d code=%d", hwnd, int(wp&0xffff), uint32((wp>>16)&0xffff))
    }
    if msg == 0x000F { // WM_PAINT
        appLog("WM_PAINT entrada hwnd=0x%X", hwnd)
    }
    if msg == 0x0100 || msg == 0x0101 { // WM_KEYDOWN / WM_KEYUP
        appLog("TECLADO msg=0x%X hwnd=0x%X vk=0x%X", msg, hwnd, wp&0xff)
    }
    if msg == 0x0201 || msg == 0x0202 { // WM_LBUTTONDOWN / UP
        appLog("RATON msg=0x%X hwnd=0x%X", msg, hwnd)
    }
    return appWndProc(hwnd, msg, wp, lp)
}

func appLogRuntimeEvent(name string) { appLog("EVENTO: %s", name) }
