//go:build windows

package main

import "unsafe"

// openXLSXDialog keeps the existing XLSX picker on the UI thread, but moves
// merge/build work off the UI thread. The existing RECARGAR command is then
// sent back through the normal window procedure so all Win32 control updates
// remain on the UI thread.
func openXLSXDialog(owner uintptr) {
	panicGuard(func() {
		logf("openXLSXDialog start owner=%x", owner)
		files := pickMultipleXLSX(owner)
		logf("openXLSXDialog: picked %d files", len(files))
		if len(files) == 0 {
			return
		}

		go func(selected []string) {
			defer func() {
				if r := recover(); r != nil {
					logf("openXLSXDialog worker panic: %v", r)
				}
			}()

			rows, err := mergeXLSX(selected)
			if err != nil {
				logf("mergeXLSX error: %v", err)
				return
			}
			lines := BuildLines(rows, "xlsx")
			mainLines = lines
			currentView = append([]Line(nil), lines...)
			feedEngineFile(owner, selected[0])
			logf("openXLSXDialog worker done lines=%d", len(lines))

			// Send the existing RECARGAR command to the main window. SendMessageW
			// executes the command in the UI thread, avoiding cross-thread control
			// access while preserving the existing refresh/filter path.
			user32.NewProc("SendMessageW").Call(owner, WM_COMMAND, uintptr(ID_RECARGAR), 0)
		}(append([]string(nil), files...))
	})
}

// Keep unsafe referenced on Windows builds where the compiler evaluates the
// file independently before the main Win32 file is transformed by CI.
var _ = unsafe.Pointer(nil)
