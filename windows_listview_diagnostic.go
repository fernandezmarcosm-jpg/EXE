//go:build windows

package main

import (
    "fmt"
    "syscall"
    "unsafe"
)

const (
    diagGWChild         = 5
    diagGWHWndNext      = 2
    diagGWHWndPrev      = 3
    diagGWLStyle        = -16
    diagGWLExStyle      = -20
    diagLVMGetItemTextW = 0x1073
    diagLVIFText        = 0x0001
)

type diagRect struct { Left, Top, Right, Bottom int32 }
type diagLVItem struct {
    Mask uint32
    Item, SubItem int32
    State, StateMask uint32
    Text *uint16
    TextMax int32
    Image, Param int32
}

func diagCall(proc *syscall.LazyProc, args ...uintptr) (uintptr, uintptr, syscall.Errno) { return proc.Call(args...) }

func diagClass(hwnd uintptr) string {
    if hwnd == 0 { return "<0>" }
    buf := make([]uint16, 256)
    n, _, _ := diagCall(user32.NewProc("GetClassNameW"), hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
    if n == 0 { return "<unknown>" }
    return syscall.UTF16ToString(buf[:n])
}

func diagText(hwnd uintptr) string {
    if hwnd == 0 { return "" }
    buf := make([]uint16, 512)
    n, _, _ := diagCall(user32.NewProc("GetWindowTextW"), hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
    if n == 0 { return "" }
    return syscall.UTF16ToString(buf[:n])
}

func diagRectOf(hwnd uintptr) (diagRect, bool) {
    var r diagRect
    ok, _, _ := diagCall(user32.NewProc("GetWindowRect"), hwnd, uintptr(unsafe.Pointer(&r)))
    return r, ok != 0
}

func diagStyle(hwnd uintptr) (uintptr, uintptr) {
    s, _, _ := diagCall(user32.NewProc("GetWindowLongPtrW"), hwnd, uintptr(diagGWLStyle))
    e, _, _ := diagCall(user32.NewProc("GetWindowLongPtrW"), hwnd, uintptr(diagGWLExStyle))
    return s, e
}

func boolNum(v bool) int { if v { return 1 }; return 0 }
func isWindowVisible(hwnd uintptr) bool { v, _, _ := diagCall(user32.NewProc("IsWindowVisible"), hwnd); return v != 0 }
func isWindowEnabled(hwnd uintptr) bool { v, _, _ := diagCall(user32.NewProc("IsWindowEnabled"), hwnd); return v != 0 }
func isWindowFocus(hwnd uintptr) bool { f, _, _ := diagCall(user32.NewProc("GetFocus")); return f == hwnd }

func diagRectString(r diagRect, ok bool) string {
    if !ok { return "<error>" }
    return fmt.Sprintf("%d,%d-%d,%d (%dx%d)", r.Left, r.Top, r.Right, r.Bottom, r.Right-r.Left, r.Bottom-r.Top)
}

func diagLogWindow(label string, hwnd uintptr) {
    if hwnd == 0 { appLog("LISTVIEW %s HWND=0", label); return }
    parent, _, _ := diagCall(user32.NewProc("GetParent"), hwnd)
    prev, _, _ := diagCall(user32.NewProc("GetWindow"), hwnd, diagGWHWndPrev)
    next, _, _ := diagCall(user32.NewProc("GetWindow"), hwnd, diagGWHWndNext)
    style, exstyle := diagStyle(hwnd)
    r, rok := diagRectOf(hwnd)
    appLog("LISTVIEW %s HWND=0x%X CLASS=%q TEXT=%q PARENT=0x%X PREV=0x%X NEXT=0x%X STYLE=0x%X EXSTYLE=0x%X VISIBLE=%d ENABLED=%d FOCUS=%d RECT=%s",
        label, hwnd, diagClass(hwnd), diagText(hwnd), parent, prev, next, style, exstyle,
        boolNum(isWindowVisible(hwnd)), boolNum(isWindowEnabled(hwnd)), boolNum(isWindowFocus(hwnd)), diagRectString(r, rok))
}

func diagLogChildren(parent uintptr, target uintptr) {
    appLog("LISTVIEW CHILDREN_BEGIN parent=0x%X target=0x%X", parent, target)
    child, _, _ := diagCall(user32.NewProc("GetWindow"), parent, diagGWChild)
    for i := 0; child != 0 && i < 256; i++ {
        r, _ := diagRectOf(child); style, ex := diagStyle(child)
        appLog("LISTVIEW CHILD[%d] HWND=0x%X CLASS=%q TEXT=%q ID=%d VISIBLE=%d STYLE=0x%X EXSTYLE=0x%X RECT=%s ZPREV=0x%X ZNEXT=0x%X TARGET=%d",
            i, child, diagClass(child), diagText(child), windowID(child), boolNum(isWindowVisible(child)), style, ex,
            diagRectString(r, true), prevWindow(child), nextWindow(child), boolNum(child == target))
        child, _, _ = diagCall(user32.NewProc("GetWindow"), child, diagGWHWndNext)
    }
    appLog("LISTVIEW CHILDREN_END")
}

func windowID(hwnd uintptr) uintptr { v, _, _ := diagCall(user32.NewProc("GetDlgCtrlID"), hwnd); return v }
func prevWindow(hwnd uintptr) uintptr { v, _, _ := diagCall(user32.NewProc("GetWindow"), hwnd, diagGWHWndPrev); return v }
func nextWindow(hwnd uintptr) uintptr { v, _, _ := diagCall(user32.NewProc("GetWindow"), hwnd, diagGWHWndNext); return v }

func diagReadCell(row, col int) string {
    if viewList == 0 || row < 0 || col < 0 { return "" }
    buf := make([]uint16, 1024)
    it := diagLVItem{Mask: diagLVIFText, Item: int32(row), SubItem: int32(col), Text: (*uint16)(unsafe.Pointer(&buf[0])), TextMax: int32(len(buf))}
    ret, _, _ := diagCall(user32.NewProc("SendMessageW"), viewList, diagLVMGetItemTextW, uintptr(row), uintptr(unsafe.Pointer(&it)))
    if ret == 0 { return "" }
    return syscall.UTF16ToString(buf)
}

func columnViewDeepDiagnostic(stage string) {
    defer appRecover("columnViewDeepDiagnostic")
    if viewList == 0 { appLog("LISTVIEW DEEP stage=%s HWND=0", stage); return }
    appLog("LISTVIEW DEEP_BEGIN stage=%s", stage)
    diagLogWindow("LIST", viewList)
    diagLogWindow("PARENT", appHwnd)
    header, _, _ := diagCall(user32.NewProc("SendMessageW"), viewList, lvmGetHeader, 0, 0)
    diagLogWindow("HEADER", header)
    count, _, _ := diagCall(user32.NewProc("SendMessageW"), viewList, lvmGetItemCount, 0, 0)
    hcount, _, _ := diagCall(user32.NewProc("SendMessageW"), header, hdmGetItemCount, 0, 0)
    appLog("LISTVIEW COUNTS rows=%d columns=%d headerColumns=%d", count, hcount, hcount)
    for i := 0; i < int(count) && i < 64; i++ {
        for c := 0; c < int(hcount) && c < 32; c++ { appLog("LISTVIEW CELL row=%d col=%d text=%q", i, c, diagReadCell(i, c)) }
    }
    diagLogChildren(appHwnd, viewList)
    appLog("LISTVIEW DEEP_END stage=%s", stage)
}
