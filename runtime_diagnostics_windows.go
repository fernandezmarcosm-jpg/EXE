//go:build windows

package main

import "fmt"

// Wrapper de diagnóstico alrededor del callback Win32. Un panic dentro del
// callback queda registrado sin llevarse por delante silenciosamente la UI.
func appWndProcLogged(hwnd uintptr, msg uint32, wp, lp uintptr) (ret uintptr) {
    defer func() {
        if v := recover(); v != nil {
            appLogPanic(fmt.Sprintf("WndProc msg=0x%X hwnd=0x%X", msg, hwnd), v)
            ret = 0
        }
    }()
    if msg == WM_COMMAND {
        appLog("WM_COMMAND hwnd=0x%X id=%d code=%d", hwnd, int(wp&0xffff), uint32((wp>>16)&0xffff))
    }
    return appWndProc(hwnd, msg, wp, lp)
}

// Inicialización de diagnóstico temprana. El wrapper se instala modificando
// el callback de clase mediante un pequeño reemplazo en crearVentana en la
// próxima compilación; mientras tanto registra el arranque y errores de alto
// nivel.
func appLogRuntimeEvent(name string) { appLog("EVENTO: %s", name) }
