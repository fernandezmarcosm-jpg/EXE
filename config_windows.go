//go:build windows
package main

import (
	"strings"
	"unicode/utf16"
	"unsafe"
)

const configWM_TIMER uint32 = 0x0113
const configLayoutTimer uintptr = 7311

const (
	LB_GETCURSEL = 0x0188
	LB_GETTEXT = 0x0189
	LBN_SELCHANGE = 1
	LBN_DBLCLK = 2
	LB_SETITEMHEIGHT = 0x01A0
)

func configLayoutButtons(h uintptr) {
	defer appRecover("configLayoutButtons")
	if h == 0 { return }
	var r appRect
	user32.NewProc("GetClientRect").Call(h, uintptr(unsafe.Pointer(&r)))
	w := int(r.Right-r.Left)
	hh := int(r.Bottom-r.Top)
	if w < 620 { w = 620 }
	if hh < 520 { hh = 520 }

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

	buttonY := hh - 50
	if buttonY < 450 { buttonY = 450 }
	gap := 10
	cancelW, okW, namesW := 110, 110, 150
	cancelX := w - 20 - cancelW
	okX := cancelX - gap - okW
	namesX := okX - gap - namesW
	if namesX < 20 {
		namesX = 20
		okX = namesX + namesW + gap
		cancelX = okX + okW + gap
	}
	place := func(b uintptr, x, bw int) {
		if b == 0 { return }
		move.Call(b, uintptr(x), uintptr(buttonY), uintptr(bw), 32, 1)
		setPos.Call(b, 0, uintptr(x), uintptr(buttonY), uintptr(bw), 32, swpNoActivate|swpShowWindow)
		show.Call(b, swShow)
	}
	place(names, namesX, namesW)
	place(ok, okX, okW)
	place(cancel, cancelX, cancelW)
}

func configLayoutFormulaSelector(h uintptr) {
	defer appRecover("configLayoutFormulaSelector")
	if h == 0 || configFormulaSelector == 0 { return }
	var r appRect
	user32.NewProc("GetClientRect").Call(h, uintptr(unsafe.Pointer(&r)))
	w := int(r.Right-r.Left)
	hh := int(r.Bottom-r.Top)
	if w < 700 { w = 700 }
	if hh < 520 { hh = 520 }

	buttonY := hh - 50
	if buttonY < 450 { buttonY = 450 }
	listY := 306
	listBottom := buttonY - 42
	listH := listBottom - listY
	if listH < 115 { listH = 115 }

	user32.NewProc("MoveWindow").Call(configFormulaSelector, 325, uintptr(listY), 360, uintptr(listH), 1)
	check := findChildByID(h, "BUTTON", configIDSubtotalCheck)
	if check != 0 {
		cy := listY + listH + 8
		if cy+26 > buttonY-5 { cy = buttonY-32 }
		user32.NewProc("MoveWindow").Call(check, 20, uintptr(cy), 200, 26, 1)
	}

	// GWLP_STYLE is -16. Convert it to its 64-bit uintptr representation
	// without a compile-time negative-constant overflow.
	const gwlpStyle uintptr = ^uintptr(15)
	const LBS_NOINTEGRALHEIGHT uintptr = 0x0100
	getStyle := user32.NewProc("GetWindowLongPtrW")
	setStyle := user32.NewProc("SetWindowLongPtrW")
	style, _, _ := getStyle.Call(configFormulaSelector, gwlpStyle)
	if style&LBS_NOINTEGRALHEIGHT == 0 {
		setStyle.Call(configFormulaSelector, gwlpStyle, style|LBS_NOINTEGRALHEIGHT)
	}
}

func configInsertSelectedFormulaToken() {
	if configFormulaSelector == 0 { return }
	send := user32.NewProc("SendMessageW")
	sel, _, _ := send.Call(configFormulaSelector, LB_GETCURSEL, 0, 0)
	if int(sel) < 0 { return }

	buf := make([]uint16, 512)
	n, _, _ := send.Call(configFormulaSelector, LB_GETTEXT, sel, uintptr(unsafe.Pointer(&buf[0])))
	if int(n) <= 0 { return }

	token := string(utf16.Decode(buf[:n]))
	if h, ok := configEdits[configIDFormula]; ok && h != 0 {
		current := appGetEdit(h)
		appSetEdit(h, current+token)
		appLog("DIAGNOSTICO: formula selector sel=%d token=%s formula=%s", int(sel), token, current+token)
	}
}

func configWndProc(h uintptr,m uint32,w,l uintptr)uintptr{
	defer appRecover("configWndProc")
	if m==wmEraseBkgnd{return 1}
	if m==wmCtlColorStatic{b,_,_:=user32.NewProc("GetSysColorBrush").Call(5);return b}
	if m==WM_CREATE{user32.NewProc("SetTimer").Call(h,configLayoutTimer,50,0);return 0}
	if m==configWM_TIMER{if w==configLayoutTimer{user32.NewProc("KillTimer").Call(h,configLayoutTimer);configLayoutButtons(h);configLayoutFormulaSelector(h);return 0}}
	if m==WM_SIZE{configLayoutButtons(h);configLayoutFormulaSelector(h);return 0}
	if m==WM_COMMAND{
		cmd:=int(w&0xffff)
		notify:=int((w>>16)&0xffff)
		configLayoutButtons(h)
		configLayoutFormulaSelector(h)
		switch cmd{
		case configIDFormulaColumns:
			// LBS_NOTIFY sends WM_COMMAND with LBN_SELCHANGE and LBN_DBLCLK.
			// A normal selection change only changes the highlighted row. A
			// double click inserts exactly once, avoiding duplicate tokens.
			if notify==LBN_DBLCLK { configInsertSelectedFormulaToken() }
			return 0
		case configIDOK:
			s:=appSettings
			s.Decimals=strconvSafe(appGetEdit(configEdits[configIDDecimals]),s.Decimals)
			s.FontSize=strconvSafe(appGetEdit(configEdits[configIDFont]),s.FontSize)
			s.SOColumn=strconvSafe(appGetEdit(configEdits[configIDSOColumn]),s.SOColumn)
			s.JoinExcelColumn=strings.TrimSpace(appGetEdit(configEdits[configIDJoin]))
			s.FormulaTitle=strings.TrimSpace(appGetEdit(configEdits[configIDFormulaTitle]))
			s.Formula=strings.TrimSpace(appGetEdit(configEdits[configIDFormula]))
			subtotalText:=strings.TrimSpace(appGetEdit(configEdits[configIDSubtotal]))
			s.SubtotalColumns=nil
			if subtotalText!=""{for _,part:=range strings.Split(subtotalText,";"){if v:=strings.TrimSpace(part);v!=""{s.SubtotalColumns=append(s.SubtotalColumns,v)}}}
			if len(s.SubtotalColumns)>0{s.SubtotalColumn=s.SubtotalColumns[0]}else{s.SubtotalColumn=""}
			s.MaxColumns=strconvSafe(appGetEdit(configEdits[configIDMaxColumns]),s.MaxColumns)
			s.SubtotalEnabled=appGetCheck(configEdits[configIDSubtotalCheck])
			datasetSettingsNormalize(&s)
			appSettings=s
			appLog("DIAGNOSTICO: GUARDAR config FormulaTitle=%q Formula=%q dataset=%t",appSettings.FormulaTitle,appSettings.Formula,appImportedDataset!=nil)
			if err:=saveDatasetSettings(appSettings);err!=nil{appLog("ERROR: no se pudo guardar configuración: %v",err)}
			closeConfigWindow()
			appApplySettings()
			appLog("DIAGNOSTICO: appApplySettings ejecutado; dataset=%t",appImportedDataset!=nil)
			return 0
		case configIDCancel:
			closeConfigWindow()
			return 0
		case configIDNames:
			appLog("DIAGNOSTICO: EDITAR NOMBRES pulsado; viewDataset=%t",viewDataset!=nil)
			columnViewNames()
			appLog("DIAGNOSTICO: columnViewNames retorno namesHwnd=0x%X",namesHwnd)
			return 0
		}
	}
	if m==WM_CLOSE||m==WM_DESTROY{user32.NewProc("KillTimer").Call(h,configLayoutTimer);closeConfigWindow();return 0}
	r,_,_:=user32.NewProc("DefWindowProcW").Call(h,uintptr(m),w,l)
	return r
}

func appGetCheck(h uintptr)bool{if h==0{return false};v,_,_:=user32.NewProc("SendMessageW").Call(h,bmGetCheck,0,0);return v==bstChecked}
func closeConfigWindow(){h:=configHwnd;configHwnd=0;configEdits=map[int]uintptr{};configFormulaSelector=0;if h!=0{user32.NewProc("KillTimer").Call(h,configLayoutTimer);user32.NewProc("DestroyWindow").Call(h)};if configParent!=0{appSetEnabled(configParent,true);configParent=0}}
