//go:build windows
package main

import (
    "syscall"
    "unsafe"
)

func columnViewSetDatasetSafe(ds *MemoryDataset) {
    columnViewDestroyFilters()
    viewDataset = ds
    columnViewBuildFilters()
    columnViewRefreshSafe()
    appApplyVisualPolish(appHwnd)
}

func columnViewRefreshSafe() {
    if viewList == 0 { return }
    columnViewDeleteColumns()
    user32.NewProc("SendMessageW").Call(viewList, lvmDeleteAll, 0, 0)
    if viewDataset == nil { return }

    visible := columnViewVisibleColumns()
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

    records := columnViewFilteredRecords()
    for ri, r := range records {
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
    if appSettings.SubtotalEnabled && appSettings.SubtotalColumn != "" {
        columnViewAddSubtotal(visible, records)
    }
    columnViewLayoutFilters(currentClientWidth())
    columnViewApplyFont()
}

func appApplyVisualPolish(parent uintptr) {
    if parent == 0 { return }
    gdi := syscall.NewLazyDLL("gdi32.dll")
    createFont := gdi.NewProc("CreateFontW")
    face := appU16("Segoe UI")
    font, _, _ := createFont.Call(uintptr(-9), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(face)))
    if font == 0 { return }
    for _, id := range []uintptr{appIDOpen, appIDColumns, appIDConfig} {
        h := findChildByID(parent, "BUTTON", id)
        if h != 0 { user32.NewProc("SendMessageW").Call(h, WM_SETFONT, font, 1) }
    }
    status := findChildByID(parent, "STATIC", appIDStatus)
    if status != 0 {
        user32.NewProc("SendMessageW").Call(status, WM_SETFONT, font, 1)
    }
}
