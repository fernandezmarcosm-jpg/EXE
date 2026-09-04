//go:build windows
package main

import (
    "strings"
    "sync"
    "syscall"
    "unsafe"
)

var (
    safeRenderMu sync.Mutex
    safeRenderGeneration uint64
    safeRenderRecords []DatasetRecord
    safeRenderVisible []DatasetColumn
    safeRenderIndex int
)

const safeRenderBatchSize = 10

type safeFilter struct { column DatasetColumn; text string }

func columnViewSetDatasetSafe(ds *MemoryDataset) {
    defer appRecover()
    if ds == nil { appLog("PANIC/ERROR: columnViewSetDatasetSafe recibió dataset nil"); return }
    appLog("DIAGNOSTICO: import_done -> destroy filters")
    columnViewDestroyFilters()
    viewDataset = ds
    appLog("DIAGNOSTICO: import_done -> build filters")
    columnViewBuildFilters()
    appLog("DIAGNOSTICO: import_done -> refresh safe")
    columnViewRefreshSafe()
    appLog("DIAGNOSTICO: import_done -> visual polish")
    appApplyVisualPolish(appHwnd)
    appLog("DIAGNOSTICO: import_done -> retorno WNDPROC")
}

func columnViewRefreshSafe() {
    safeRenderMu.Lock()
    safeRenderGeneration++
    generation := safeRenderGeneration
    safeRenderRecords = nil
    safeRenderVisible = nil
    safeRenderIndex = 0
    safeRenderMu.Unlock()

    if viewList == 0 { return }
    appLog("DIAGNOSTICO: refresh -> delete columns")
    columnViewDeleteColumns()
    user32.NewProc("SendMessageW").Call(viewList, lvmDeleteAll, 0, 0)
    if viewDataset == nil { return }

    visible := columnViewVisibleColumns()
    safeRenderMu.Lock()
    safeRenderVisible = append([]DatasetColumn(nil), visible...)
    safeRenderMu.Unlock()
    appLog("DIAGNOSTICO: refresh -> columnas visibles=%d", len(visible))

    for i, c := range visible {
        p := appU16(datasetColumnDisplayTitle(c))
        fmtCol := lvcfmtLeft
        if c.Type == ValueNumber { fmtCol = lvcfmtRight }
        width := c.Width
        if width < 120 { width = 120 }
        if width > 420 { width = 420 }
        lc := lvColumn{Mask: lvcfText, Fmt: int32(fmtCol), Cx: int32(width), Text: p, SubItem: int32(i)}
        user32.NewProc("SendMessageW").Call(viewList, lvmInsertColumnW, uintptr(i), uintptr(unsafe.Pointer(&lc)))
    }

    filters := safeSnapshotFilters()
    ds := viewDataset
    appLog("DIAGNOSTICO: refresh -> filtros=%d; iniciar filtrado en segundo plano; registros=%d", len(filters), len(ds.Records))
    go func(gen uint64, dataset *MemoryDataset, fs []safeFilter) {
        records := safeFilterRecords(dataset, fs)
        safeRenderMu.Lock()
        current := safeRenderGeneration
        if gen == current {
            safeRenderRecords = records
            safeRenderIndex = 0
        }
        safeRenderMu.Unlock()
        if gen != current || appHwnd == 0 { return }
        appLog("DIAGNOSTICO: filtrado terminado; registros visibles=%d", len(records))
        user32.NewProc("PostMessageW").Call(appHwnd, WM_APP_RENDER_BATCH, uintptr(gen), 0)
    }(generation, ds, filters)
}

func safeSnapshotFilters() []safeFilter {
    if viewDataset == nil { return nil }
    filters := make([]safeFilter, 0, len(viewFilters))
    for id, h := range viewFilters {
        text := strings.ToLower(strings.TrimSpace(appGetEdit(h)))
        if text == "" { continue }
        for _, c := range viewDataset.Columns {
            if c.ID == id {
                filters = append(filters, safeFilter{column: c, text: text})
                break
            }
        }
    }
    return filters
}

func safeFilterRecords(ds *MemoryDataset, filters []safeFilter) []DatasetRecord {
    if ds == nil { return nil }
    if len(filters) == 0 { return append([]DatasetRecord(nil), ds.Records...) }
    out := make([]DatasetRecord, 0, len(ds.Records))
    for _, r := range ds.Records {
        ok := true
        for _, f := range filters {
            if !strings.Contains(strings.ToLower(datasetCellText(r, f.column)), f.text) { ok = false; break }
        }
        if ok { out = append(out, r) }
    }
    return out
}

func columnViewRenderBatch(generation uint64) {
    defer appRecover()
    if viewList == 0 { return }
    safeRenderMu.Lock()
    if generation != safeRenderGeneration { safeRenderMu.Unlock(); return }
    records := safeRenderRecords
    visible := append([]DatasetColumn(nil), safeRenderVisible...)
    start := safeRenderIndex
    end := start + safeRenderBatchSize
    if end > len(records) { end = len(records) }
    safeRenderMu.Unlock()

    if start == end {
        columnViewLayoutFilters(currentClientWidth())
        columnViewApplyFont()
        return
    }

    appLog("DIAGNOSTICO: render batch %d-%d de %d", start+1, end, len(records))
    for ri := start; ri < end; ri++ {
        r := records[ri]
        for ci, c := range visible {
            txt := datasetCellText(r, c)
            p := appU16(txt)
            it := lvItem{Mask: lvifText, Item: int32(ri), SubItem: int32(ci), Text: p, TextMax: int32(len([]rune(txt)) + 1)}
            if ci == 0 {
                user32.NewProc("SendMessageW").Call(viewList, lvmInsertItemW, 0, uintptr(unsafe.Pointer(&it)))
            } else {
                user32.NewProc("SendMessageW").Call(viewList, lvmSetItemTextW, uintptr(ri), uintptr(unsafe.Pointer(&it)))
            }
        }
    }

    safeRenderMu.Lock()
    if generation == safeRenderGeneration { safeRenderIndex = end }
    next := safeRenderIndex < len(safeRenderRecords)
    safeRenderMu.Unlock()
    if next {
        user32.NewProc("PostMessageW").Call(appHwnd, WM_APP_RENDER_BATCH, uintptr(generation), 0)
        return
    }

    if appSettings.SubtotalEnabled && appSettings.SubtotalColumn != "" { columnViewAddSubtotal(visible, records) }
    columnViewLayoutFilters(currentClientWidth())
    columnViewApplyFont()
    appLog("DIAGNOSTICO: render completo; registros=%d", len(records))
}

func appApplyVisualPolish(parent uintptr) {
    defer appRecover()
    if parent == 0 { return }
    theme := syscall.NewLazyDLL("uxtheme.dll")
    setTheme := theme.NewProc("SetWindowTheme")
    gdi := syscall.NewLazyDLL("gdi32.dll")
    createFont := gdi.NewProc("CreateFontW")
    face := appU16("Segoe UI")
    fontHeight := uint32(int32(-14))
    font, _, _ := createFont.Call(uintptr(fontHeight), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(face)))
    explorer := appU16("Explorer")
    buttons := []struct{ id, x, w uintptr }{{appIDOpen, 12, 125}, {appIDColumns, 145, 105}, {appIDConfig, 258, 135}}
    for _, b := range buttons {
        h := uintptr(0)
        switch b.id { case appIDOpen: h = appOpenButton; case appIDColumns: h = appColumnsButton; case appIDConfig: h = appConfigButton }
        if h == 0 { h = findChildByID(parent, "BUTTON", b.id) }
        if h == 0 { continue }
        setTheme.Call(h, uintptr(unsafe.Pointer(explorer)), 0)
        if font != 0 { user32.NewProc("SendMessageW").Call(h, WM_SETFONT, font, 1) }
        user32.NewProc("MoveWindow").Call(h, b.x, 7, b.w, 30, 1)
    }
    status := appStatus
    if status == 0 { status = findChildByID(parent, "STATIC", appIDStatus) }
    if status != 0 {
        if font != 0 { user32.NewProc("SendMessageW").Call(status, WM_SETFONT, font, 1) }
        user32.NewProc("MoveWindow").Call(status, 410, 11, uintptr(maxInt(240, currentClientWidth()-430)), 22, 1)
    }
}
