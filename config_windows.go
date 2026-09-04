//go:build windows

package main

import (
    "strings"
    "syscall"
    "unsafe"
)

// configWndProc handles only the controls created by datasetShowConfig.
// Settings are persisted through the existing DatasetSettings serializer.
func configWndProc(h uintptr, m uint32, w, l uintptr) uintptr {
    defer appRecover("configWndProc")
    switch m {
    case WM_COMMAND:
        cmd := int(w & 0xffff)
        switch cmd {
        case configIDOK:
            s := appSettings
            s.Decimals = strconvSafe(appGetEdit(configEdits[configIDDecimals]), s.Decimals)
            s.FontSize = strconvSafe(appGetEdit(configEdits[configIDFont]), s.FontSize)
            s.SOColumn = strconvSafe(appGetEdit(configEdits[configIDSOColumn]), s.SOColumn)
            s.JoinExcelColumn = strings.TrimSpace(appGetEdit(configEdits[configIDJoin]))
            s.FormulaTitle = strings.TrimSpace(appGetEdit(configEdits[configIDFormulaTitle]))
            s.Formula = strings.TrimSpace(appGetEdit(configEdits[configIDFormula]))
            subtotalText := strings.TrimSpace(appGetEdit(configEdits[configIDSubtotal]))
            s.SubtotalColumns = nil
            if subtotalText != "" {
                for _, part := range strings.Split(subtotalText, ";") {
                    if v := strings.TrimSpace(part); v != "" {
                        s.SubtotalColumns = append(s.SubtotalColumns, v)
                    }
                }
            }
            if len(s.SubtotalColumns) > 0 {
                s.SubtotalColumn = s.SubtotalColumns[0]
            } else {
                s.SubtotalColumn = ""
            }
            s.MaxColumns = strconvSafe(appGetEdit(configEdits[configIDMaxColumns]), s.MaxColumns)
            check := appGetCheck(configEdits[configIDSubtotalCheck])
            s.SubtotalEnabled = check
            datasetSettingsNormalize(&s)
            appSettings = s
            if err := saveDatasetSettings(appSettings); err != nil {
                appLog("ERROR: no se pudo guardar configuración: %v", err)
            }
            closeConfigWindow()
            appApplySettings()
            return 0
        case configIDCancel:
            closeConfigWindow()
            return 0
        case configIDNames:
            // The names editor is not implemented in this reconstruction yet.
            appLog("EVENTO: EDITAR NOMBRES solicitado; editor pendiente")
            return 0
        }
    case WM_CLOSE, WM_DESTROY:
        closeConfigWindow()
        return 0
    }
    r, _, _ := user32.NewProc("DefWindowProcW").Call(h, uintptr(m), w, l)
    return r
}

func appGetCheck(h uintptr) bool {
    if h == 0 {
        return false
    }
    v, _, _ := user32.NewProc("SendMessageW").Call(h, bmGetCheck, 0, 0)
    return v == bstChecked
}

func closeConfigWindow() {
    h := configHwnd
    configHwnd = 0
    configEdits = map[int]uintptr{}
    if h != 0 {
        user32.NewProc("DestroyWindow").Call(h)
    }
    if configParent != 0 {
        appSetEnabled(configParent, true)
        configParent = 0
    }
    _ = unsafe.Pointer(nil)
    _ = syscall.Handle(0)
}
