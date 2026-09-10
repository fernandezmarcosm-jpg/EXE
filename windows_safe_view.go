//go:build windows
package main

import ("strings"; "sync"; "syscall"; "unsafe")

const swShow = 5
const swpNoActivate uintptr = 0x0010
const swpShowWindow uintptr = 0x0040
const wmSetFont uintptr = 0x0030
const wmSetText uintptr = 0x000C
const esMultiline uintptr = 0x0004
const esAutoVScroll uintptr = 0x0040
const esAutoHScroll uintptr = 0x0080
const esReadOnly uintptr = 0x0800
const WS_VSCROLL uint32 = 0x00200000
const WS_HSCROLL uint32 = 0x00100000

var safeRenderMu sync.Mutex
type safeFilter struct{column DatasetColumn; text string}

func columnViewSetDatasetSafe(ds *MemoryDataset) {
	defer appRecover("columnViewSetDatasetSafe")
	if ds == nil { return }
	columnViewDestroyFilters()
	if viewList != 0 { user32.NewProc("DestroyWindow").Call(viewList); viewList = 0 }
	style := WS_CHILD | WS_VISIBLE | WS_BORDER | WS_VSCROLL | WS_HSCROLL | esMultiline | esAutoVScroll | esAutoHScroll | esReadOnly
	viewList = appMake(appHwnd, "EDIT", "", style, 0, 0, 100, 100, appIDView)
	if viewList == 0 { appLog("DIAGNOSTICO: ERROR CreateWindowExW EDIT para datos"); return }
	viewDataset = ds
	columnViewRefreshSafe()
	columnViewForceDisplay()
	appLog("DATOS: vista directa creada; filas=%d columnas=%d", len(ds.Records), len(ds.Columns))
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
	appLog("DIAGNOSTICO: visualizacion directa finalizada; hwndView=0x%X rect=%dx%d", viewList, w, h)
}

func columnViewRefreshSafe() {
	defer appRecover("columnViewRefreshSafe")
	if viewList == 0 || viewDataset == nil { return }
	safeRenderMu.Lock()
	defer safeRenderMu.Unlock()
	visible := columnViewVisibleColumns()
	if len(visible) == 0 { visible = viewDataset.Columns }
	records := safeFilterRecords(viewDataset, safeSnapshotFilters())
	var b strings.Builder
	for i, c := range visible { if i > 0 { b.WriteString("\t") }; b.WriteString(datasetColumnDisplayTitle(c)) }
	b.WriteString("\r\n")
	for ri, r := range records {
		for ci, c := range visible { if ci > 0 { b.WriteString("\t") }; b.WriteString(datasetCellText(r, c)) }
		if ri+1 < len(records) { b.WriteString("\r\n") }
	}
	text := b.String()
	p := appU16(text)
	user32.NewProc("SendMessageW").Call(viewList, wmSetText, 0, uintptr(unsafe.Pointer(p)))
	gdi := syscall.NewLazyDLL("gdi32.dll")
	createFont := gdi.NewProc("CreateFontW")
	size := appSettings.FontSize
	if size < 8 || size > 32 { size = 10 }
	face := appU16("Segoe UI")
	font, _, _ := createFont.Call(uintptr(int32(-size)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(face)))
	if font != 0 { user32.NewProc("SendMessageW").Call(viewList, wmSetFont, font, 1) }
	appLog("DATOS: texto enviado a vista directa; filas=%d columnas=%d caracteres=%d", len(records), len(visible), len([]rune(text)))
	for ri, r := range records { for ci, c := range visible { appLog("DATOS: fila=%d columna=%d titulo=%q valor=%q", ri, ci, datasetColumnDisplayTitle(c), datasetCellText(r, c)) } }
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
	for _, b := range buttons { h:=uintptr(0); switch b.id { case appIDOpen:h=appOpenButton; case appIDColumns:h=appColumnsButton; case appIDConfig:h=appConfigButton }; if h==0 { h=findChildByID(parent,"BUTTON",b.id) }; if h==0 { continue }; setTheme.Call(h,uintptr(unsafe.Pointer(explorer)),0); if font!=0 { user32.NewProc("SendMessageW").Call(h,wmSetFont,font,1) }; user32.NewProc("MoveWindow").Call(h,b.x,7,b.w,30,1) }
	status:=appStatus
	if status==0 { status=findChildByID(parent,"STATIC",appIDStatus) }
	if status!=0 { if font!=0 { user32.NewProc("SendMessageW").Call(status,wmSetFont,font,1) }; user32.NewProc("MoveWindow").Call(status,410,11,uintptr(maxInt(240,currentClientWidth()-430)),22,1) }
}
