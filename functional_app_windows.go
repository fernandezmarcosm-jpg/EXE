//go:build windows

package main

import (
    "fmt"
    "path/filepath"
    "strings"
    "sync"
    "syscall"
    "time"
    "unsafe"
)

const (
    appIDOpen   = 2001
    appIDStatus = 2006
    appIDView   = 2007

    WM_CREATE  = 0x0001
    WM_SIZE    = 0x0005
    WM_CLOSE   = 0x0010
    WM_DESTROY = 0x0002
    WM_COMMAND = 0x0111
    WM_APP_IMPORT_DONE = 0x8001

    WS_OVERLAPPEDWINDOW = 0x00CF0000
    WS_VISIBLE          = 0x10000000
    WS_CHILD            = 0x40000000
    WS_TABSTOP          = 0x00010000
    WS_BORDER            = 0x00800000
    WS_VSCROLL           = 0x00200000
    WS_HSCROLL           = 0x00100000

    BS_PUSHBUTTON = 0

    ES_MULTILINE   = 0x0004
    ES_AUTOVSCROLL = 0x0040
    ES_AUTOHSCROLL = 0x0080
    ES_READONLY    = 0x0800

    ofnExplorer      = 0x00080000
    ofnPathMustExist = 0x00000800
    ofnFileMustExist = 0x00001000
    ofnHideReadOnly  = 0x00000004
)

type appRect struct { Left, Top, Right, Bottom int32 }

type appWndClass struct {
    CbSize uint32
    Style uint32
    LpfnWndProc uintptr
    CbClsExtra, CbWndExtra int32
    HInstance, HIcon, HCursor, HbrBackground uintptr
    LpszMenuName, LpszClassName *uint16
    HIconSm uintptr
}

type appOpenFile struct {
    LStructSize uint32
    _ uint32
    HwndOwner uintptr
    HInstance uintptr
    Filter uintptr
    CustomFilter uintptr
    MaxCustom uint32
    FilterIndex uint32
    File uintptr
    MaxFile uint32
    _ uint32
    FileTitle uintptr
    MaxFileTitle uint32
    _ uint32
    InitialDir uintptr
    Title uintptr
    Flags uint32
    FileOffset uint16
    FileExtension uint16
    DefExt uintptr
    CustData uintptr
    Hook uintptr
    Template uintptr
    Reserved uintptr
    Reserved2 uint32
    FlagsEx uint32
}

var (
    appHwnd uintptr
    appHInstance uintptr
    appStatus uintptr
    appView uintptr

    // Unico almacenamiento del XLSX ya interpretado.
    appImportedWorkbook *xlsxDoc
    appImportedPath string

    // Resultado del lector en segundo plano. La interfaz lo recoge mediante
    // WM_APP_IMPORT_DONE, evitando bloquear el hilo de mensajes.
    appImportMu sync.Mutex
    appPendingWorkbook *xlsxDoc
    appPendingPath string
    appPendingErr error
)

func appU16(s string) *uint16 {
    p, _ := syscall.UTF16PtrFromString(s)
    return p
}

func appSetText(hwnd uintptr, text string) {
    if hwnd == 0 { return }
    p := appU16(text)
    user32.NewProc("SetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(p)))
}

func appSetEnabled(hwnd uintptr, enabled bool) {
    if hwnd == 0 { return }
    var v uintptr
    if enabled { v = 1 }
    user32.NewProc("EnableWindow").Call(hwnd, v)
}

func appMake(parent uintptr, className, text string, style uint32, x, y, w, h int, id uintptr) uintptr {
    cls := appU16(className)
    txt := appU16(text)
    hwnd, _, _ := user32.NewProc("CreateWindowExW").Call(
        0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(txt)), uintptr(style),
        uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, id, appHInstance, 0,
    )
    return hwnd
}

func crearVentana() uintptr {
    appHInstance, _, _ = kernel32.NewProc("GetModuleHandleW").Call(0)
    cls := appU16("GestionSOExcelImporter")
    wc := appWndClass{
        CbSize: uint32(unsafe.Sizeof(appWndClass{})),
        LpfnWndProc: syscall.NewCallback(appWndProcLogged),
        HInstance: appHInstance,
        HCursor: loadArrowCursor(),
        LpszClassName: cls,
    }
    user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))

    title := appU16("GestionSO V57 - Importar Excel")
    hwnd, _, _ := user32.NewProc("CreateWindowExW").Call(
        0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(title)), WS_OVERLAPPEDWINDOW|WS_VISIBLE,
        0x80000000, 0x80000000, 1200, 750, 0, 0, appHInstance, 0,
    )
    return hwnd
}

func loadArrowCursor() uintptr {
    cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
    return cursor
}

func appWndProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
    switch msg {
    case WM_CREATE:
        appHwnd = hwnd
        appBuildControls(hwnd)
        appLog("EVENTO: controles de interfaz creados")
        return 0

    case WM_SIZE:
        appLayout(hwnd)
        return 0

    case WM_COMMAND:
        if int(wp&0xffff) == appIDOpen {
            appOpenXLSX(hwnd)
            return 0
        }
        // Los controles de columnas trabajan sobre appImportedWorkbook, que
        // es la misma estructura MemoryWorkbook construida al importar.
        if columnViewHandlesCommand(wp) {
            columnViewRefresh()
            return 0
        }
        return 0

    case WM_APP_IMPORT_DONE:
        appFinishImport(hwnd)
        return 0

    case WM_CLOSE:
        user32.NewProc("DestroyWindow").Call(hwnd)
        return 0

    case WM_DESTROY:
        appLog("EVENTO: WM_DESTROY hwnd=0x%X", hwnd)
        user32.NewProc("PostQuitMessage").Call(0)
        return 0
    }

    r, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, uintptr(msg), wp, lp)
    return r
}

func appBuildControls(hwnd uintptr) {
    appMake(hwnd, "BUTTON", "ABRIR EXCEL", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON, 10, 10, 130, 32, appIDOpen)
    appStatus = appMake(hwnd, "STATIC", "Seleccione un archivo XLSX.", WS_CHILD|WS_VISIBLE, 155, 15, 1000, 24, appIDStatus)
    appView = appMake(hwnd, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|WS_HSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_AUTOHSCROLL|ES_READONLY, 10, 73, 1160, 612, appIDView)

    // Los selectores de columnas se crean una sola vez. No contienen datos:
    // solamente controlan la presentacion de la informacion que ya esta en memoria.
    columnViewCreate(hwnd)
}

func appLayout(hwnd uintptr) {
    var r appRect
    user32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&r)))
    w := int(r.Right - r.Left)
    h := int(r.Bottom - r.Top)
    if w < 500 { w = 500 }
    if h < 250 { h = 250 }
    user32.NewProc("MoveWindow").Call(appStatus, 155, 15, uintptr(maxInt(250, w-170)), 24, 1)
    columnViewLayout(hwnd)
}

func maxInt(a, b int) int {
    if a > b { return a }
    return b
}

func appPickXLSX(owner uintptr) string {
    f1, _ := syscall.UTF16FromString("Archivos Excel (*.xlsx)")
    f2, _ := syscall.UTF16FromString("*.xlsx")
    f3, _ := syscall.UTF16FromString("Todos los archivos (*.*)")
    f4, _ := syscall.UTF16FromString("*.*")
    filter := make([]uint16, 0, len(f1)+len(f2)+len(f3)+len(f4)+2)
    filter = append(filter, f1...)
    filter = append(filter, f2...)
    filter = append(filter, f3...)
    filter = append(filter, f4...)
    filter = append(filter, 0, 0)

    buffer := make([]uint16, 32768)
    title := appU16("Seleccionar archivo Excel")
    defExt := appU16("xlsx")
    ofn := appOpenFile{
        LStructSize: uint32(unsafe.Sizeof(appOpenFile{})),
        HwndOwner: owner,
        Filter: uintptr(unsafe.Pointer(&filter[0])),
        FilterIndex: 1,
        File: uintptr(unsafe.Pointer(&buffer[0])),
        MaxFile: uint32(len(buffer)),
        Title: uintptr(unsafe.Pointer(title)),
        Flags: ofnExplorer|ofnFileMustExist|ofnPathMustExist|ofnHideReadOnly,
        DefExt: uintptr(unsafe.Pointer(defExt)),
    }

    appLog("EVENTO: abrir selector XLSX")
    ret, _, _ := comdlg32.NewProc("GetOpenFileNameW").Call(uintptr(unsafe.Pointer(&ofn)))
    if ret == 0 {
        appLog("EVENTO: selector XLSX cancelado")
        return ""
    }
    path := strings.TrimSpace(syscall.UTF16ToString(buffer))
    if path != "" { appLog("EVENTO: selector devolvio archivo=%s", path) }
    return path
}

func appOpenXLSX(owner uintptr) {
    path := appPickXLSX(owner)
    if path == "" { return }

    appSetEnabled(findChildByID(owner, "BUTTON", appIDOpen), false)
    appSetText(appStatus, "Leyendo Excel...")
    appSetText(appView, "Leyendo archivo XLSX...\r\nLa interfaz no se bloquea durante la lectura.")
    appLog("EVENTO: iniciar lectura XLSX en segundo plano; archivo=%s", filepath.Base(path))

    go func() {
        start := time.Now()
        workbook, err := ReadXLSX(path)
        elapsed := time.Since(start)

        appImportMu.Lock()
        appPendingWorkbook = workbook
        appPendingPath = path
        appPendingErr = err
        appImportMu.Unlock()

        appLog("EVENTO: lectura XLSX finalizada; duración=%s", elapsed)
        if err != nil {
            appLog("ERROR importando XLSX: %v", err)
        } else {
            rows, cells := workbookSize(workbook)
            appLog("EVENTO: XLSX preparado; hojas=%d filas=%d celdas=%d", len(workbook.Sheets), rows, cells)
        }
        user32.NewProc("PostMessageW").Call(owner, WM_APP_IMPORT_DONE, 0, 0)
    }()
}

func findChildByID(parent uintptr, className string, id uintptr) uintptr {
    cls := appU16(className)
    child, _, _ := user32.NewProc("FindWindowExW").Call(parent, 0, uintptr(unsafe.Pointer(cls)), 0)
    for child != 0 {
        got, _, _ := user32.NewProc("GetDlgCtrlID").Call(child)
        if got == id { return child }
        child, _, _ = user32.NewProc("FindWindowExW").Call(parent, child, uintptr(unsafe.Pointer(cls)), 0)
    }
    return 0
}

func appFinishImport(hwnd uintptr) {
    appImportMu.Lock()
    workbook := appPendingWorkbook
    path := appPendingPath
    err := appPendingErr
    appPendingWorkbook = nil
    appPendingPath = ""
    appPendingErr = nil
    appImportMu.Unlock()

    appSetEnabled(findChildByID(hwnd, "BUTTON", appIDOpen), true)

    if err != nil {
        appSetText(appStatus, "ERROR al leer XLSX")
        appSetText(appView, "No se pudo leer el archivo XLSX.\r\n\r\n"+err.Error())
        return
    }

    rows, cells := workbookSize(workbook)
    appImportedWorkbook = workbook
    appImportedPath = path

    status := fmt.Sprintf("EN MEMORIA: %d hoja(s) | %d fila(s) | %d celda(s) | %s", len(workbook.Sheets), rows, cells, filepath.Base(path))
    appSetText(appStatus, status)
    appSetText(appView, renderMemoryWorkbookPreview(workbook))
    appLog("EVENTO: XLSX almacenado en memoria y visualizado; archivo=%s hojas=%d filas=%d celdas=%d", filepath.Base(path), len(workbook.Sheets), rows, cells)
}

func workbookSize(doc *xlsxDoc) (rows, cells int) {
    if doc == nil { return 0, 0 }
    for _, sheetRows := range doc.Sheets {
        rows += len(sheetRows)
        for _, row := range sheetRows { cells += len(row) }
    }
    return rows, cells
}

func renderMemoryWorkbookPreview(doc *xlsxDoc) string {
    if doc == nil || doc.Memory == nil || len(doc.Memory.Sheets) == 0 { return "Excel vacío o sin filas de datos debajo de los títulos." }
    const maxRows = 40
    const maxCols = 20
    s := &doc.Memory.Sheets[0]
    var b strings.Builder
    b.WriteString("DATOS CARGADOS EN MEMORIA\r\n")
    b.WriteString("Hoja: ")
    b.WriteString(s.Name)
    b.WriteString("\r\n\r\n")
    colLimit := len(s.Columns)
    if colLimit > maxCols { colLimit = maxCols }
    for i := 0; i < colLimit; i++ {
        if i > 0 { b.WriteByte('\t') }
        b.WriteString(s.Columns[i].Title)
    }
    b.WriteString("\r\n")
    rowLimit := len(s.Rows)
    if rowLimit > maxRows { rowLimit = maxRows }
    for i := 0; i < rowLimit; i++ {
        r := s.Rows[i]
        for ci := 0; ci < colLimit; ci++ {
            if ci > 0 { b.WriteByte('\t') }
            colID := s.Columns[ci].ID
            if v, ok := r.Values[colID]; ok { b.WriteString(cleanCell(MemoryValueString(v))) }
        }
        b.WriteString("\r\n")
    }
    if len(s.Rows) > maxRows || len(s.Columns) > maxCols {
        b.WriteString("\r\n[Vista previa limitada. El contenido completo permanece en memoria.]\r\n")
    }
    b.WriteString(fmt.Sprintf("\r\nTOTAL EN MEMORIA: %d fila(s), %d valor(es), %d hoja(s).", doc.Memory.TotalRows, doc.Memory.TotalValues, len(doc.Memory.Sheets)))
    return b.String()
}

func maxColumns(rows [][]string) int {
    max := 0
    for _, row := range rows {
        if len(row) > max { max = len(row) }
    }
    return max
}

func cleanCell(s string) string {
    return strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(s)
}
