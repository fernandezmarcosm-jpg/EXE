//go:build windows

package main

// The old regression test checks the Win64 ABI size of OPENFILENAMEW.
// Keep the public test name while the functional app uses the same layout.
type OPENFILENAMEW = appOpenFile
