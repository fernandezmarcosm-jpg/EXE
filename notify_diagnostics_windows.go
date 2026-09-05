//go:build windows
package main

func columnViewLogNotify(l uintptr) {
 if l==0 { return }
 h:=columnViewReadNMHeader(l)
 appLog("DIAGNOSTICO: WM_NOTIFY hWndFrom=0x%X idFrom=0x%X h.Code=%d",h.HwndFrom,h.IDFrom,h.Code)
 if h.Code==nmDblClk { n:=columnViewReadNMItemActivate(l); appLog("DIAGNOSTICO: nmDblClk item=%d subItem=%d",n.Item,n.SubItem) }
 if h.Code==hdnItemDblClickW || h.Code==hdnEndDrag { n:=columnViewReadNMHeaderNotify(l); appLog("DIAGNOSTICO: header notify code=%d item=%d",n.Hdr.Code,n.Item) }
}
