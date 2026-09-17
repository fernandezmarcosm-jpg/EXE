//go:build windows
package main

import (
    "fmt"
    "reflect"
)

// columnViewShowMenuMejorado conserva el menú existente y, cuando cambia la
// selección de columnas, reconstruye el dataset desde los archivos importados.
// Esto evita que el ListView acumule grupos de columnas físicos al cambiar la vista.
func columnViewShowMenuMejorado(hwnd uintptr) {
    before := append([]string(nil), appSettings.VisibleColumns...)

    columnViewShowMenu(hwnd)

    after := append([]string(nil), appSettings.VisibleColumns...)
    if reflect.DeepEqual(before, after) {
        return
    }
    if len(appImportedPaths) == 0 {
        return
    }

    docs := make([]*xlsxDoc, 0, len(appImportedPaths))
    for _, p := range appImportedPaths {
        d, err := ReadXLSX(p)
        if err != nil {
            appLog("WARN: recarga tras COLUMNAS falló: ReadXLSX(%q): %v", p, err)
            return
        }
        decorateXLSXDates(d, p)
        docs = append(docs, d)
    }

    ds, err := BuildMemoryDataset(docs, appSettings)
    if err != nil {
        appLog("WARN: recarga tras COLUMNAS falló: BuildMemoryDataset: %v", err)
        return
    }
    applyDatasetFormula(ds, appSettings)
    applySavedColumnVisibility(ds)
    appImportedDataset = ds
    columnViewSetDatasetSafe(ds)
    appLog("EVENTO: dataset recargado tras COLUMNAS; columnas_visibles=%d", len(columnViewVisibleColumns()))
}

var _ = fmt.Sprintf
