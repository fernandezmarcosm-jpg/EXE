//go:build windows
package main
import "time"

// Wrapper de diagnóstico alrededor del callback Win32. La creación de controles
// pertenece exclusivamente a appWndProc; este wrapper nunca duplica la UI.
func appWndProcLogged(hwnd uintptr,msg uint32,wp,lp uintptr)(ret uintptr){start:=time.Now();defer func(){if v:=recover();v!=nil{appLogPanic("WndProc",v);ret=0};if msg==WM_COMMAND||msg==WM_CLOSE||msg==WM_DESTROY||time.Since(start)>500*time.Millisecond{appLog("WNDPROC msg=0x%X hwnd=0x%X duración=%s",msg,hwnd,time.Since(start))}}();return appWndProc(hwnd,msg,wp,lp)}
func appLogRuntimeEvent(name string){appLog("EVENTO: %s",name)}
