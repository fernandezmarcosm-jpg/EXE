//go:build windows
package main

import ("strings"; "sync"; "syscall"; "unsafe")

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

func columnViewSetDatasetSafe(ds *MemoryDataset) {
	defer appRecover("columnViewSetDatasetSafe")
	if ds == nil { return }
	columnViewDestroyFilters()
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
	invalidate := user32.NewProc("InvalidateRect")
	update := user32.NewProc("UpdateWindow")
	show.Call(viewList, swShow)
	var r appRect
	user32.NewProc("GetClientRect").Call(appHwnd, uintptr(unsafe.Pointer(&r)))
	w := maxInt(300, int(r.Right-r.Left)-20)
	h := maxInt(150, int(r.Bottom-r.Top)-94)
	move.Call(viewList, 0, 10, 84, uintptr(w), uintptr(h), swpNoActivate|swpShowWindow)
	invalidate.Call(viewList, 0, 1)
	update.Call(viewList)
	appLog("DIAGNOSTICO: visualizacion tabular finalizada; hwndView=0x%X rect=%dx%d", viewList, w, h)
}

func safeDisplayColumns(ds *MemoryDataset) []DatasetColumn {
	if ds == nil { return nil }
	allHidden := len(ds.Columns) > 0
	for _, c := range ds.Columns { if c.Visible { allHidden = false; break } }
	if allHidden { return []DatasetColumn{} }
	current := columnViewVisibleColumns()
	hasData := func(c DatasetColumn) bool {
		for _, r := range ds.Records {
			if strings.TrimSpace(datasetCellText(r, c)) != "" { return true }
		}
		return false
	}
	out := make([]DatasetColumn, 0)
	for _, c := range current {
		if hasData(c) { out = append(out, c) }
	}
	if len(out) > 0 { return out }
	limit := appSettings.MaxColumns
	if limit < 1 { limit = 20 }
	appendData := func(c DatasetColumn) {
		if len(out) >= limit || !hasData(c) { return }
		for _, x := range out { if x.ID == c.ID { return } }
		out = append(out, c)
	}
	for _, c := range ds.Columns { if c.Source == "XLSX" { appendData(c) } }
	for _, c := range ds.Columns { if c.Source != "XLSX" { appendData(c) } }
	if len(out) > 0 { return out }
	for _, c := range ds.Columns {
		if len(out) >= limit { break }
		out = append(out, c)
	}
	return out
}

func columnViewRefreshSafe() {
	defer appRecover("columnViewRefreshSafe")
	if viewList == 0 || viewDataset == nil { return }
	safeRenderMu.Lock()
	defer safeRenderMu.Unlock()
	visible := safeDisplayColumns(viewDataset)
	records := safeFilterRecords(viewDataset, safeSnapshotFilters())
	send := user32.NewProc("SendMessageW")
	columnViewDeleteColumns()
	for i, c := range visible {
		title := appU16(datasetColumnDisplayTitle(c))
		width := c.Width
		if width < 80 { width = 120 }
		if width > 500 { width = 500 }
		col := lvColumn{Mask:uint32(lvcfText), Fmt:int32(lvcfmtLeft), Cx:int32(width), Text:title, TextMax:int32(len([]rune(datasetColumnDisplayTitle(c)))+1), SubItem:int32(i), Order:int32(i)}
		send.Call(viewList,lvmInsertColumnW,uintptr(i),uintptr(unsafe.Pointer(&col)))
	}
	count := len(records)
	if appSettings.SubtotalEnabled && appSettings.SubtotalColumn != "" && len(visible)>0 { count++ }
	send.Call(viewList,lvmSetItemCountEx,uintptr(count),0)
	columnViewAutoFit(visible)
	columnViewLayoutFilters(currentClientWidth())
	columnViewApplyFont()
	send.Call(viewList,lvmSetExtended,0,lvsExGridlines|lvsExFullRowSelect|lvsExDoubleBuffer|lvsExHeaderDragDrop)
	user32.NewProc("InvalidateRect").Call(viewList,0,1)
	user32.NewProc("UpdateWindow").Call(viewList)
	appLog("DATOS: tabla actualizada; filas=%d columnas=%d",len(records),len(visible))
	for ri,r:=range records{for ci,c:=range visible{appLog("DATOS: fila=%d columna=%d titulo=%q valor=%q",ri,ci,datasetColumnDisplayTitle(c),datasetCellText(r,c))}}
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
	filters := make([]safeFilter, 0, len(viewFilters))
	for id, h := range viewFilters { text := strings.ToLower(strings.TrimSpace(appGetEdit(h))); if text == "" { continue }; for _, c := range viewDataset.Columns { if c.ID == id { filters = append(filters, safeFilter{column:c, text:text}); break } } }
	return filters
}

func safeFilterRecords(ds *MemoryDataset, filters []safeFilter) []DatasetRecord {
	if ds == nil { return nil }
	if len(filters) == 0 { return append([]DatasetRecord(nil), ds.Records...) }
	out := make([]DatasetRecord, 0, len(ds.Records))
	for _, r := range ds.Records { ok := true; for _, f := range filters { if !strings.Contains(strings.ToLower(datasetCellText(r, f.column)), f.text) { ok = false; break } }; if ok { out = append(out, r) } }
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