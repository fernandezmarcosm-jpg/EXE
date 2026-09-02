//go:build windows

package main

func main() {
	registerClass()
	hwnd := createWindow()
	if hwnd != 0 {
		installMultiSelectButton(hwnd)
		msgLoop()
	}
}
