//go:build windows

package main

import (
    "fmt"
    "os"
    "path/filepath"
    "runtime/debug"
    "sync"
    "time"
)

var appLogMu sync.Mutex
var appLogFile *os.File
var appLogEnabled = true

func appLogPath() string {
    if exe, err := os.Executable(); err == nil {
        return filepath.Join(filepath.Dir(exe), "gestionso_debug.log")
    }
    return filepath.Join(os.TempDir(), "gestionso_debug.log")
}

func appLogInit() {
    path := appLogPath()
    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        fallback := filepath.Join(os.TempDir(), "gestionso_debug.log")
        f, err = os.OpenFile(fallback, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil { return }
        path = fallback
    }
    appLogFile = f
    appLog("=== INICIO GestionSO V57 | log=%s ===", path)
}

func appLog(format string, args ...interface{}) {
    if !appLogEnabled { return }
    appLogMu.Lock()
    defer appLogMu.Unlock()
    if appLogFile == nil { return }
    fmt.Fprintf(appLogFile, "%s | %s\r\n", time.Now().Format("2006-01-02 15:04:05.000"), fmt.Sprintf(format, args...))
    _ = appLogFile.Sync()
}

func logf(format string, args ...interface{}) {
    appLog(format, args...)
}

func appLogPanic(where string, v interface{}) {
    appLog("PANIC en %s: %v\n%s", where, v, debug.Stack())
}

func appRecover(where string) {
    if v := recover(); v != nil {
        appLogPanic(where, v)
    }
}
