//go:build windows
package main

import "strings"

const configWM_TIMER uint32 = 0x0113
const configLayoutTimer uintptr = 7311

func configLayoutButtons(h uintptr) {
	defer appRecover("configLayoutButtons")
	if h == 0 { return }
	var r appRect
	user32.NewProc("GetClientRect").Call(h, uintptr(unsafe.Pointer(&r)))
	w := int(r.Right-r.Left)
	hh := int(r.Bottom-r.Top)
	if w < 620 { w = 620 }
	if hh < 470 { hh = 470 }
	find := func(id uintptr) uintptr { return findChildByID(h, "BUTTON", id) }
	names := find(configIDNames)
	ok := find(configIDOK)
	cancel := find(configIDCancel)
	move := user32.NewProc("MoveWindow")
	show := user32.NewProc("ShowWindow")
	setPos := user32.NewProc("SetWindowPos")
	const swShow = 5
	const swpNoActivate uintptr = 0x0010
	const swpShowWindow uintptr = 0x0040
	buttonY := hh - 55
	if buttonY < 405 { buttonY = 405 }
	cancelX := w - 130
	okX := cancelX - 120
	namesX := okX - 160
	if namesX < 20 { namesX = 20; okX = 180; cancelX = 300 }
	place := func(b uintptr, x, bw int) {
		if b == 0 { return }
		move.Call(b, uintptr(x), uintptr(buttonY), uintptr(bw), 32, 1)
		setPos.Call(b, 0, uintptr(x), uintptr(buttonY), uintptr(bw), 32, swpNoActivate|swpShowWindow)
		show.Call(b, swShow)
	}
	place(names, namesX, 150)
	place(ok, okX, 110)
	place(cancel, cancelX, 110)
}

func configWndProc(h uintptr,m uint32,w,l uintptr)uintptr{defer appRecover("configWndProc");if m==wmEraseBkgnd{return 1};if m==wmCtlColorStatic{b,_,_:=user32.NewProc("GetSysColorBrush").Call(5);return b};if m==WM_CREATE{user32.NewProc("SetTimer").Call(h,configLayoutTimer,50,0);return 0};if m==configWM_TIMER{if w==configLayoutTimer{user32.NewProc("KillTimer").Call(h,configLayoutTimer);configLayoutButtons(h);return 0}};if m==WM_SIZE{configLayoutButtons(h);return 0};if m==WM_COMMAND{configLayoutButtons(h);cmd:=int(w&0xffff);switch cmd{case configIDFormulaColumns:if configFormulaSelector!=0{sel,_,_:=user32.NewProc("SendMessageW").Call(configFormulaSelector,cbGetCurSel,0,0);if int(sel)>=0&&viewDataset!=nil&&int(sel)<len(viewDataset.Columns){token:=formulaTokenName(viewDataset.Columns[int(sel)]);old:=appGetEdit(configEdits[configIDFormula]);appSetEdit(configEdits[configIDFormula],old+token);appLog("DIAGNOSTICO: formula selector sel=%d token=%s formula=%s",int(sel),token,old+token)}};return 0;case configIDOK:s:=appSettings;s.Decimals=strconvSafe(appGetEdit(configEdits[configIDDecimals]),s.Decimals);s.FontSize=strconvSafe(appGetEdit(configEdits[configIDFont]),s.FontSize);s.SOColumn=strconvSafe(appGetEdit(configEdits[configIDSOColumn]),s.SOColumn);s.JoinExcelColumn=strings.TrimSpace(appGetEdit(configEdits[configIDJoin]));s.FormulaTitle=strings.TrimSpace(appGetEdit(configEdits[configIDFormulaTitle]));s.Formula=strings.TrimSpace(appGetEdit(configEdits[configIDFormula]));subtotalText:=strings.TrimSpace(appGetEdit(configEdits[configIDSubtotal]));s.SubtotalColumns=nil;if subtotalText!=""{for _,part:=range strings.Split(subtotalText,";"){if v:=strings.TrimSpace(part);v!=""{s.SubtotalColumns=append(s.SubtotalColumns,v)}}};if len(s.SubtotalColumns)>0{s.SubtotalColumn=s.SubtotalColumns[0]}else{s.SubtotalColumn=""};s.MaxColumns=strconvSafe(appGetEdit(configEdits[configIDMaxColumns]),s.MaxColumns);s.SubtotalEnabled=appGetCheck(configEdits[configIDSubtotalCheck]);datasetSettingsNormalize(&s);appSettings=s;appLog("DIAGNOSTICO: GUARDAR config FormulaTitle=%q Formula=%q dataset=%t",appSettings.FormulaTitle,appSettings.Formula,appImportedDataset!=nil);if err:=saveDatasetSettings(appSettings);err!=nil{appLog("ERROR: no se pudo guardar configuración: %v",err)};closeConfigWindow();appApplySettings();appLog("DIAGNOSTICO: appApplySettings ejecutado; dataset=%t",appImportedDataset!=nil);return 0;case configIDCancel:closeConfigWindow();return 0;case configIDNames:appLog("DIAGNOSTICO: EDITAR NOMBRES pulsado; viewDataset=%t",viewDataset!=nil);columnViewNames();appLog("DIAGNOSTICO: columnViewNames retorno namesHwnd=0x%X",namesHwnd);return 0}};if m==WM_CLOSE||m==WM_DESTROY{user32.NewProc("KillTimer").Call(h,configLayoutTimer);closeConfigWindow();return 0};r,_,_:=user32.NewProc("DefWindowProcW").Call(h,uintptr(m),w,l);return r}
func appGetCheck(h uintptr)bool{if h==0{return false};v,_,_:=user32.NewProc("SendMessageW").Call(h,bmGetCheck,0,0);return v==bstChecked}
func closeConfigWindow(){h:=configHwnd;configHwnd=0;configEdits=map[int]uintptr{};configFormulaSelector=0;if h!=0{user32.NewProc("KillTimer").Call(h,configLayoutTimer);user32.NewProc("DestroyWindow").Call(h)};if configParent!=0{appSetEnabled(configParent,true);configParent=0}}
