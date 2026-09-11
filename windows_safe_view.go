//go:build windows
package main

import (
    "strings"
    "sync"
    "syscall"
    "unsafe"
)

const swShow = 5
const swpNoActivate uintptr = 0x0010
const swpShowWindow uintptr = 0x0040
const wmSetFont uintptr = 0x0030
const wmSetText uintptr = 0x000C
const esMultiline uint32 = 0x0004
const esAutoVScroll uint32 = 0x0040
const esAutoHScroll uint32 = 0x0080
const esReadOnly uint32 = 0x0800
const WS_VSCROLL uint32 = 0x00200000
const WS_HSCROLL uint32 = 0x00100000

var safeRenderMu sync.Mutex
type safeFilter struct{column DatasetColumn; text string}

// Caché estable: durante el pintado de SysListView32 no se vuelven a leer
// controles Win32 ni se recalcula el dataset.
var (
    viewCacheRecords []DatasetRecord
    viewCacheColumns []DatasetColumn
    viewCacheFilters []safeFilter
    safeRefreshActive bool
)

func columnViewSetDatasetSafe(ds *MemoryDataset) {
    defer appRecover("columnViewSetDatasetSafe")
    if ds == nil { return }
    columnViewDestroyFilters()
    viewCacheRecords = nil
    viewCacheColumns = nil
    viewCacheFilters = nil
    safeRefreshActive = false
    if viewList != 0 { user32.NewProc("DestroyWindow").Call(viewList); viewList = 0 }
    style := uint32(WS_CHILD | WS_VISIBLE | WS_BORDER | WS_TABSTOP | lvsReport | lvsOwnerData)
    viewList = appMake(appHwnd, "SysListView32", "", style, 0, 0, 100, 100, appIDView)
    if viewList == 0 { appLog("DIAGNOSTICO: ERROR CreateWindowExW SysListView32 para datos"); return }
    viewDataset = ds
    user32.NewProc("SendMessageW").Call(viewList, lvmSetExtended, 0, lvsExGridlines|lvsExFullRowSelect|lvsExDoubleBuffer|lvsExHeaderDragDrop)
    columnViewBuildFilters()
    columnViewRefreshSafe()
    columnViewForceDisplay()
    appLog("DATOS: vista tabular creada; filas=%d columnas=%d", len(ds.Records), len(ds.Columns))
}

func columnViewForceDisplay() {
    defer appRecover("columnViewForceDisplay")
    if viewList == 0 { return }
    show := user32.NewProc("ShowWindow")
    move := user32.NewProc("SetWindowPos")
    show.Call(viewList, swShow)
    var r appRect
    user32.NewProc("GetClientRect").Call(appHwnd, uintptr(unsafe.Pointer(&r)))
    w := maxInt(300, int(r.Right-r.Left)-20)
    h := maxInt(150, int(r.Bottom-r.Top)-94)
    move.Call(viewList, 0, 10, 84, uintptr(w), uintptr(h), swpNoActivate|swpShowWindow)
    // No UpdateWindow: forzar un pintado síncrono aquí duplicaba el trabajo
    // inmediatamente después de LVM_SETITEMCOUNTEX.
    appLog("DIAGNOSTICO: visualizacion tabular finalizada; hwndView=0x%X rect=%dx%d", viewList, w, h)
}

func safeDisplayColumns(ds *MemoryDataset) []DatasetColumn {
    if ds == nil { return nil }
    allHidden := len(ds.Columns) > 0
    for _, c := range ds.Columns {
        if c.Visible { allHidden = false; break }
    }
    if allHidden { return []DatasetColumn{} }
    if !safeRefreshActive && viewCacheColumns != nil { return viewCacheColumns }
    return columnViewVisibleColumns()
}

func columnViewRefreshSafe() {
    defer appRecover("columnViewRefreshSafe")
    if viewList == 0 || viewDataset == nil { return }
    safeRenderMu.Lock()
    defer safeRenderMu.Unlock()

    safeRefreshActive = true
    defer func(){ safeRefreshActive = false }()

    // Un único snapshot por refresh explícito.
    viewCacheFilters = safeSnapshotFilters()
    viewCacheRecords = safeFilterRecords(viewDataset, viewCacheFilters)
    viewCacheColumns = safeDisplayColumns(viewDataset)

    send := user32.NewProc("SendMessageW")
    columnViewDeleteColumns()
    for i, c := range viewCacheColumns {
        title := datasetColumnDisplayTitle(c)
        titlePtr := appU16(title)
        width := c.Width
        if width < 80 { width = 120 }
        if width > 500 { width = 500 }
        col := lvColumn{
            Mask: uint32(lvcfText), Fmt: int32(lvcfmtLeft),
            Cx: int32(width), Text: titlePtr,
            TextMax: int32(len([]rune(title)) + 1),
            SubItem: int32(i), Order: int32(i),
        }
        send.Call(viewList, lvmInsertColumnW, uintptr(i), uintptr(unsafe.Pointer(&col)))
    }

    count := len(viewCacheRecords)
    if appSettings.SubtotalEnabled && appSettings.SubtotalColumn != "" && len(viewCacheColumns) > 0 { count++ }
    send.Call(viewList, lvmSetItemCountEx, uintptr(count), 0)

    // Evitar LVSCW_AUTOSIZE en cada importación/refresco: con LVS_OWNERDATA
    // puede provocar consultas de muchas celdas y volver el primer render muy lento.
    // Conservamos los anchos ya definidos por DatasetColumn.
    columnViewLayoutFilters(currentClientWidth())
    columnViewApplyFont()
    send.Call(viewList, lvmSetExtended, 0, lvsExGridlines|lvsExFullRowSelect|lvsExDoubleBuffer|lvsExHeaderDragDrop)
    // Un solo invalidado asíncrono; Windows pintará cuando corresponda.
    user32.NewProc("InvalidateRect").Call(viewList, 0, 1)

    appLog("DATOS: tabla actualizada; filas=%d columnas=%d", len(viewCacheRecords), len(viewCacheColumns))
}

func safeSubtotalCellText(visible []DatasetColumn, records []DatasetRecord, ci int) string {
    if ci < 0 || ci >= len(visible) { return "" }
    c := visible[ci]
    var sum float64
    found := false
    for _, r := range records { if v, ok := r.Values[c.ID]; ok && v.Type == ValueNumber { sum += v.Number; found = true } }
    if !found { return "" }
    if datasetColumnIsPercent(c) { return formatDatasetNumber(sum*100, datasetColumnDecimals(c)) + "%" }
    return formatDatasetNumber(sum, datasetColumnDecimals(c))
}

func safeSnapshotFilters() []safeFilter {
    if viewDataset == nil { return nil }
    if !safeRefreshActive && viewCacheFilters != nil { return viewCacheFilters }
    filters := make([]safeFilter, 0, len(viewFilters))
    for id, h := range viewFilters {
        text := strings.ToLower(strings.TrimSpace(appGetEdit(h)))
        if text == "" { continue }
        for _, c := range viewDataset.Columns {
            if c.ID == id { filters = append(filters, safeFilter{column:c, text:text}); break }
        }
    }
    return filters
}

func safeFilterRecords(ds *MemoryDataset, filters []safeFilter) []DatasetRecord {
    if ds == nil { return nil }
    if !safeRefreshActive && viewCacheRecords != nil { return viewCacheRecords }
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

func appApplyVisualPolish(parent uintptr) {
    defer appRecover("appApplyVisualPolish")
    if parent == 0 { return }
    theme := syscall.NewLazyDLL("uxtheme.dll")
    setTheme := theme.NewProc("SetWindowTheme")
    gdi := syscall.NewLazyDLL("gdi32.dll")
    createFont := gdi.NewProc("CreateFontW")
    face := appU16("Segoe UI")
    size := appSettings.FontSize
    if size < 8 || size > 32 { size = 10 }
    font, _, _ := createFont.Call(uintptr(int32(-size)), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(face)))
    explorer := appU16("Explorer")
    buttons := []struct{id,x,w uintptr}{{appIDOpen,12,125},{appIDColumns,145,105},{appIDConfig,258,135}}
    for _, b := range buttons { h:=uintptr(0); switch b.id { case appIDOpen:h=appOpenButton; case appIDColumns:h=appColumnsButton; case appIDConfig:h=appConfigButton }; if h==0 { h=findChildByID(parent,"BUTTON",b.id) }; if h==0 { continue }; setTheme.Call(h,uintptr(unsafe.Pointer(explorer)),0); if font!=0 { user32.NewProc("SendMessageW").Call(h,wmSetFont,font,1) }; user32.NewProc("MoveWindow").Call(h,b.x,7,b.w,30,1); user32.NewProc("SetWindowPos").Call(h,0,b.x,7,b.w,30,swpNoActivate|swpShowWindow) }
    status:=appStatus
    if status==0 { status=findChildByID(parent,"STATIC",appIDStatus) }
    if status!=0 { if font!=0 { user32.NewProc("SendMessageW").Call(status,wmSetFont,font,1) }; user32.NewProc("MoveWindow").Call(status,410,11,uintptr(maxInt(240,currentClientWidth()-430)),22,1); user32.NewProc("SetWindowPos").Call(status,0,410,11,uintptr(maxInt(240,currentClientWidth()-430)),22,swpNoActivate|swpShowWindow) }
}
