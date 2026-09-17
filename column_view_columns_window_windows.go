//go:build windows
package main

import (
	"reflect"
	"syscall"
	"unsafe"
)

const (
	columnsWindowIDClear = 6501
	columnsWindowIDAll = 6502
	columnsWindowIDDeselect = 6503
	columnsWindowIDList = 6504
	columnsWindowLVMSetItemState = 0x102B
	columnsWindowLVNItemChanged = -101
	columnsWindowLVIFState = 8
	columnsWindowLVISStateImageMask = 0xF000
	columnsWindowLVISChecked = 0x2000
	columnsWindowLVSExCheckboxes = 0x00000004
)

type columnsWindowState struct { hwnd, list, parent uintptr; updating bool }
type columnsWindowNMListView struct { HwndFrom uintptr; IdFrom uintptr; Code int32; Item int32; SubItem int32; NewState uint32; OldState uint32; Changed uint32; PtX int32; PtY int32; LParam uintptr }
var columnsWindow columnsWindowState

func columnViewShowColumnsWindow(parent uintptr){if viewDataset==nil||columnsWindow.hwnd!=0{if columnsWindow.hwnd!=0{user32.NewProc("SetForegroundWindow").Call(columnsWindow.hwnd)};return};columnsWindow.parent=parent;cls:=appU16("GestionSOColumnsWin");wc:=appWndClass{CbSize:uint32(unsafe.Sizeof(appWndClass{})),LpfnWndProc:syscall.NewCallback(columnsWindowWndProc),HInstance:appHInstance,HCursor:loadArrowCursor(),HbrBackground:COLOR_WINDOW+1,LpszClassName:cls};user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)));columnsWindow.hwnd,_,_=user32.NewProc("CreateWindowExW").Call(0,reflect.ValueOf(cls).Pointer(),reflect.ValueOf(appU16("COLUMNAS")).Pointer(),WS_OVERLAPPEDWINDOW|WS_VISIBLE,0x80000000,0x80000000,640,600,parent,0,appHInstance,0);if columnsWindow.hwnd==0{columnsWindow.parent=0;return};appMake(columnsWindow.hwnd,"BUTTON","LIMPIAR FILTROS",WS_CHILD|WS_VISIBLE|WS_TABSTOP,10,10,150,30,columnsWindowIDClear);appMake(columnsWindow.hwnd,"BUTTON","MARCAR TODAS",WS_CHILD|WS_VISIBLE|WS_TABSTOP,170,10,130,30,columnsWindowIDAll);appMake(columnsWindow.hwnd,"BUTTON","DESMARCAR TODAS",WS_CHILD|WS_VISIBLE|WS_TABSTOP,310,10,150,30,columnsWindowIDDeselect);columnsWindow.list=appMake(columnsWindow.hwnd,"SysListView32","",WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|lvsReport|0x00200000,10,50,600,490,columnsWindowIDList);send:=user32.NewProc("SendMessageW");send.Call(columnsWindow.list,lvmSetExtended,0,lvsExGridlines|lvsExFullRowSelect|lvsExDoubleBuffer|columnsWindowLVSExCheckboxes);title:=appU16("Columna");col:=lvColumn{Mask:uint32(lvcfText),Fmt:int32(lvcfmtLeft),Cx:560,Text:title,TextMax:64,SubItem:0,Order:0};send.Call(columnsWindow.list,lvmInsertColumnW,0,uintptr(unsafe.Pointer(&col)));columnsWindowFill();appSetEnabled(parent,false);user32.NewProc("SetFocus").Call(columnsWindow.list)}
func columnsWindowFill(){if columnsWindow.list==0||viewDataset==nil{return};titleCounts:=map[string]int{};for _,c:=range viewDataset.Columns{titleCounts[datasetColumnDisplayTitle(c)]++};send:=user32.NewProc("SendMessageW");columnsWindow.updating=true;defer func(){columnsWindow.updating=false}();for i,c:=range viewDataset.Columns{title:=datasetColumnDisplayTitle(c);if titleCounts[title]>1{title+=" · "+c.ID};p:=appU16(title);item:=lvItem{Mask:uint32(lvifText),Item:int32(i),SubItem:0,Text:p,TextMax:int32(len([]rune(title))+1)};r,_,_:=send.Call(columnsWindow.list,lvmInsertItemW,0,uintptr(unsafe.Pointer(&item)));if int(r)<0{continue};state:=uint32(0);if c.Visible{state=columnsWindowLVISChecked};item.Mask=columnsWindowLVIFState;item.Item=int32(i);item.State=state;item.StateMask=columnsWindowLVISStateImageMask;send.Call(columnsWindow.list,columnsWindowLVMSetItemState,uintptr(i),uintptr(unsafe.Pointer(&item)))}}
func columnsWindowSyncChecks(){if columnsWindow.list==0||viewDataset==nil{return};send:=user32.NewProc("SendMessageW");columnsWindow.updating=true;defer func(){columnsWindow.updating=false}();for i,c:=range viewDataset.Columns{state:=uint32(0);if c.Visible{state=columnsWindowLVISChecked};item:=lvItem{Mask:columnsWindowLVIFState,Item:int32(i),State:state,StateMask:columnsWindowLVISStateImageMask};send.Call(columnsWindow.list,columnsWindowLVMSetItemState,uintptr(i),uintptr(unsafe.Pointer(&item)))}}
func columnsWindowWndProc(h uintptr,m uint32,w,l uintptr)uintptr{defer appRecover("columnsWindowWndProc");switch m{case WM_COMMAND:cmd:=int(w&0xffff);switch cmd{case columnsWindowIDClear:for _,fh:=range viewFilters{appSetEdit(fh,"")};columnViewBuildFilters();columnViewRefresh();return 0;case columnsWindowIDAll:if viewDataset!=nil{limit:=appSettings.MaxColumns;if limit<1{limit=20};for i:=range viewDataset.Columns{viewDataset.Columns[i].Visible=i<limit};columnViewRebuildOrderFromVisible();_=saveDatasetSettings(appSettings);columnViewBuildFilters();columnViewRefresh();columnsWindowSyncChecks()};return 0;case columnsWindowIDDeselect:if viewDataset!=nil{for i:=range viewDataset.Columns{viewDataset.Columns[i].Visible=false};columnViewRebuildOrderFromVisible();_=saveDatasetSettings(appSettings);columnViewBuildFilters();columnViewRefresh();columnsWindowSyncChecks()};return 0};case WM_NOTIFY:if columnsWindow.list!=0&&l!=0{nv:=(*columnsWindowNMListView)(unsafe.Pointer(l));if nv.HwndFrom==columnsWindow.list&&nv.Code==columnsWindowLVNItemChanged&&nv.Item>=0&&int(nv.Item)<len(viewDataset.Columns)&&nv.Changed&columnsWindowLVIFState!=0&&!columnsWindow.updating{checked:=nv.NewState&columnsWindowLVISStateImageMask==columnsWindowLVISChecked;if viewDataset.Columns[int(nv.Item)].Visible!=checked{if checked&&len(columnViewVisibleColumns())>=appSettings.MaxColumns{columnsWindowSyncChecks();return 0};viewDataset.Columns[int(nv.Item)].Visible=checked;columnViewRebuildOrderFromVisible();_=saveDatasetSettings(appSettings);columnViewBuildFilters();columnViewRefresh()};return 0}};case WM_CLOSE:closeColumnsWindow();return 0;case WM_DESTROY:return 0};r,_,_:=user32.NewProc("DefWindowProcW").Call(h,uintptr(m),w,l);return r}
func closeColumnsWindow(){h:=columnsWindow.hwnd;parent:=columnsWindow.parent;columnsWindow=columnsWindowState{};if h!=0{user32.NewProc("DestroyWindow").Call(h)};if parent!=0{appSetEnabled(parent,true);user32.NewProc("SetForegroundWindow").Call(parent)}}
