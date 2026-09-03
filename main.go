//go:build windows

package main

import (
	"encoding/csv"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
)

type winPOINT struct{ X, Y int32 }
type winMSG struct {
	Hwnd uintptr
	Message uint32
	WParam, LParam uintptr
	Time uint32
	Pt winPOINT
}

// Punto de entrada único de la reconstrucción Win32.
// La aplicación arranca con la evidencia de datos disponible en el paquete,
// sin modificar los CSV originales y sin depender del motor V54 para mostrar
// la información recuperada.
func main() {
	console, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
	if console != 0 {
		user32.NewProc("ShowWindow").Call(console, 0)
	}
	comctl32.NewProc("InitCommonControls").Call()

	mainConfig = LoadConfig()
	if mainConfig.Mode == "" {
		mainConfig.Mode = "MODO: SO RETENIDAS"
	}
	if mainConfig.MasterPath == "" {
		if p := bundledDataPath("GestionSO_Datos.csv"); p != "" {
			mainConfig.MasterPath = p
		}
	}
	_ = SaveConfig(mainConfig)

	// La evidencia aportada incluye un export operativo de SO. Se usa como
	// dataset inicial para que el programa sea funcional desde el arranque.
	// El XLSX continúa disponible mediante ABRIR XLSX y reemplaza esta vista.
	mainLines = loadBundledExport()
	currentView = append([]Line(nil), mainLines...)

	hwnd := crearVentana()
	if hwnd == 0 {
		log.Fatal("No se pudo crear la ventana principal")
	}
	updateMainView(hwnd)

	var msg winMSG
	for {
		ret, _, _ := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
		user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// bundledDataPath busca primero junto al ejecutable y luego en la estructura
// del repositorio. Esto permite ejecutar tanto el binario empaquetado como una
// compilación local desde GitHub Desktop.
func bundledDataPath(name string) string {
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, name),
			filepath.Join(dir, "acceso chatgpt", name),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, name),
			filepath.Join(wd, "acceso chatgpt", name),
		)
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func loadBundledExport() []Line {
	path := bundledDataPath("archivo exportado.csv")
	if path == "" {
		logf("startup dataset not found: archivo exportado.csv")
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		logf("startup dataset open error: %v", err)
		return nil
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = false
	rows, err := r.ReadAll()
	if err != nil {
		logf("startup dataset parse error: %v", err)
		return nil
	}
	if len(rows) == 0 {
		return nil
	}
	// El export puede conservar BOM UTF-8 en la primera cabecera.
	rows[0][0] = strings.TrimPrefix(rows[0][0], "\ufeff")
	lines := BuildLines(rows, path)
	logf("startup dataset loaded path=%q rows=%d lines=%d", path, len(rows), len(lines))
	return lines
}
