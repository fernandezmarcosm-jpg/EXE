//go:build windows

package main

import (
	"testing"
	"unsafe"
)

func TestOPENFILENAMEWWin64Size(t *testing.T) {
	const want = uintptr(144)
	if got := unsafe.Sizeof(OPENFILENAMEW{}); got != want {
		t.Fatalf("OPENFILENAMEW size = %d, want %d", got, want)
	}
}
