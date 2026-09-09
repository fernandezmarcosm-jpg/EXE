//go:build windows
package main

import ("reflect"; "syscall"; "unsafe")

// LVN_GETDISPINFOW is handled in column_view_windows.go.
// Stable script trigger for final handler correction.
func columnViewCopyNotify(dst interface{}, src uintptr, size uintptr) {
	k := syscall.NewLazyDLL("kernel32.dll")
	k.NewProc("RtlMoveMemory").Call(reflect.ValueOf(dst).Pointer(), src, size)
}

func columnViewReadNMHeader(src uintptr) nmhdr {
	var v nmhdr
	columnViewCopyNotify(&v, src, unsafe.Sizeof(v))
	return v
}

func columnViewReadNMItemActivate(src uintptr) nmItemActivate {
	var v nmItemActivate
	columnViewCopyNotify(&v, src, unsafe.Sizeof(v))
	return v
}

func columnViewReadNMHeaderNotify(src uintptr) nmheader {
	var v nmheader
	columnViewCopyNotify(&v, src, unsafe.Sizeof(v))
	return v
}

func columnViewReadNMLVDispInfo(src uintptr) nmlvDispInfo {
	var v nmlvDispInfo
	columnViewCopyNotify(&v, src, unsafe.Sizeof(v))
	return v
}
