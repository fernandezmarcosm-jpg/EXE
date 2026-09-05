//go:build windows
package main

import ("reflect"; "syscall"; "unsafe")

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
