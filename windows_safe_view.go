//go:build windows
package main

import (
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

// Compatibilidad para callers antiguos: la implementación real vive en
// column_view_windows.go y mantiene un único caché de renderizado.
func columnViewSetDatasetSafe(ds *MemoryDataset) {
	defer appRecover("columnViewSetDatasetSafe")
	if ds == nil { return }
	viewDataset = ds
	columnViewBuildFilters()
	columnViewRefresh()
	columnViewForceDisplay()
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
	appLog("DIAGNOSTICO: visualizacion tabular finalizada; hwndView=0x%X rect=%dx%d", viewList, w, h)
}

func columnViewRefreshSafe() {
	columnViewRefresh()
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
	for _, b := range buttons {
		h := uintptr(0)
		switch b.id { case appIDOpen: h=appOpenButton; case appIDColumns: h=appColumnsButton; case appIDConfig: h=appConfigButton }
		if h == 0 { h=findChildByID(parent,"BUTTON",b.id) }
		if h == 0 { continue }
		setTheme.Call(h,uintptr(unsafe.Pointer(explorer)),0)
		if font != 0 { user32.NewProc("SendMessageW").Call(h,wmSetFont,font,1) }
		user32.NewProc("MoveWindow").Call(h,b.x,7,b.w,30,1)
		user32.NewProc("SetWindowPos").Call(h,0,b.x,7,b.w,30,swpNoActivate|swpShowWindow)
	}
	status := appStatus
	if status == 0 { status=findChildByID(parent,"STATIC",appIDStatus) }
	if status != 0 {
		if font != 0 { user32.NewProc("SendMessageW").Call(status,wmSetFont,font,1) }
		user32.NewProc("MoveWindow").Call(status,410,11,uintptr(maxInt(240,currentClientWidth()-430)),22,1)
		user32.NewProc("SetWindowPos").Call(status,0,410,11,uintptr(maxInt(240,currentClientWidth()-430)),22,swpNoActivate|swpShowWindow)
	}
}
