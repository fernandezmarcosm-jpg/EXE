package main

import (
    "syscall"
    "unsafe"
)

// --- Constantes de Windows ---
const (
    WS_OVERLAPPEDWINDOW = 0xCF0000
    WS_VISIBLE          = 0x10000000
    WS_CHILD            = 0x40000000
    WS_BORDER           = 0x00800000
    WS_TABSTOP          = 0x00010000
    WS_EX_CLIENTEDGE    = 0x00000200
    LVS_REPORT          = 0x0001
    LVS_SINGLESEL       = 0x0004
    LVS_SHOWSELALWAYS   = 0x0008
    CBS_DROPDOWN        = 0x0002
    CBS_HASSTRINGS      = 0x0200
    COLOR_BTNFACE       = 15
    IDC_ARROW           = 32512

    WM_CREATE   = 0x0001
    WM_DESTROY  = 0x0002
    WM_SIZE     = 0x0005
    WM_CLOSE    = 0x0010
    WM_COMMAND  = 0x0111
    WM_NOTIFY   = 0x004E

    BN_CLICKED = 0

    // IDs de controles
    ID_ABRIR_XLSX    = 1001
    ID_TOMAR_EXCEL   = 1002
    ID_COLUMNAS      = 1003
    ID_FILTROS       = 1004
    ID_EXPORTAR_CSV  = 1005
    ID_SIMULADOR     = 1006
    ID_RECARGAR      = 1007
    ID_RESALTAR      = 1008
    ID_COLOR         = 1009
    ID_DATOS_CSV     = 1010
    ID_FILTRAR       = 2001
    ID_LIMPIAR       = 2002
    ID_GRID          = 3001
    ID_STATUS        = 3002
)

// --- Estructuras de Windows ---
type WNDCLASSEX struct {
    CbSize        uint32
    Style         uint32
    LpfnWndProc   uintptr
    CbClsExtra    int32
    CbWndExtra    int32
    HInstance     uintptr
    HIcon         uintptr
    HCursor       uintptr
    HbrBackground uintptr
    LpszMenuName  *uint16
    LpszClassName *uint16
    HIconSm       uintptr
}

type RECT struct {
    Left, Top, Right, Bottom int32
}

type MSG struct {
    Hwnd    uintptr
    Message uint32
    WParam  uintptr
    LParam  uintptr
    Time    uint32
    Pt      struct{ X, Y int32 }
}

// --- Variables globales (variante inglesa) ---
var (
    g_hInstance   uintptr
    g_hwndMain    uintptr
    g_hwndGrid    uintptr
    g_hwndStatus  uintptr

    // Procs de DLLs
    pGetModuleHandleW *syscall.Proc
    pLoadCursorW      *syscall.Proc
    pGetSysColorBrush *syscall.Proc
    pRegisterClassExW *syscall.Proc
    pCreateWindowExW  *syscall.Proc
    pDefWindowProcW   *syscall.Proc
    pDestroyWindow    *syscall.Proc
    pPostQuitMessage  *syscall.Proc
    pGetClientRect    *syscall.Proc
    pMoveWindow       *syscall.Proc
    pSetWindowTextW   *syscall.Proc
    pGetWindowTextW   *syscall.Proc
    pSendMessageW     *syscall.Proc
    pMessageBoxW      *syscall.Proc
)

// --- Inicialización de procs ---
func initProcs() {
    kernel32 := syscall.NewLazyDLL("kernel32.dll")
    user32 := syscall.NewLazyDLL("user32.dll")

    pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
    pLoadCursorW = user32.NewProc("LoadCursorW")
    pGetSysColorBrush = user32.NewProc("GetSysColorBrush")
    pRegisterClassExW = user32.NewProc("RegisterClassExW")
    pCreateWindowExW = user32.NewProc("CreateWindowExW")
    pDefWindowProcW = user32.NewProc("DefWindowProcW")
    pDestroyWindow = user32.NewProc("DestroyWindow")
    pPostQuitMessage = user32.NewProc("PostQuitMessage")
    pGetClientRect = user32.NewProc("GetClientRect")
    pMoveWindow = user32.NewProc("MoveWindow")
    pSetWindowTextW = user32.NewProc("SetWindowTextW")
    pGetWindowTextW = user32.NewProc("GetWindowTextW")
    pSendMessageW = user32.NewProc("SendMessageW")
    pMessageBoxW = user32.NewProc("MessageBoxW")
}

// --- Funciones de ventana (inglesas) ---
func registerClass() {
    if g_hInstance == 0 {
        r, _, _ := pGetModuleHandleW.Call(0)
        g_hInstance = r
    }

    className := syscall.StringToUTF16Ptr("GestionSOClass")

    wc := WNDCLASSEX{
        CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
        Style:         0,
        LpfnWndProc:   syscall.NewCallback(wndProcIngles),
        CbClsExtra:    0,
        CbWndExtra:    0,
        HInstance:     g_hInstance,
        HIcon:         0,
        HCursor:       pLoadCursorW.Call(0, IDC_ARROW),
        HbrBackground: pGetSysColorBrush.Call(COLOR_BTNFACE),
        LpszMenuName:  0,
        LpszClassName: className,
        HIconSm:       0,
    }

    pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
}

func createWindow() uintptr {
    className := syscall.StringToUTF16Ptr("GestionSOClass")
    title := syscall.StringToUTF16Ptr("Gestion SO V57 - SO RETENIDAS")

    hwnd, _, _ := pCreateWindowExW.Call(
        0,
        uintptr(unsafe.Pointer(className)),
        uintptr(unsafe.Pointer(title)),
        WS_OVERLAPPEDWINDOW|WS_VISIBLE,
        100, 100, 1000, 700,
        0, 0, g_hInstance, 0,
    )

    g_hwndMain = hwnd
    return hwnd
}

func installMultiSelectButton() {
    // No se usa en esta versión
}

func msgLoop() {
    var msg MSG
    for {
        ret, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
        if ret == 0 {
            break
        }
        pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
        pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
    }
}

// --- wndProc (ingles) ---
func wndProcIngles(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
    switch msg {
    case WM_CREATE:
        crearControlesIngles(hwnd)
        return 0

    case WM_SIZE:
        redimensionarControlesIngles(hwnd)
        return 0

    case WM_COMMAND:
        comando := uint16(wParam & 0xFFFF)
        notificacion := uint16((wParam >> 16) & 0xFFFF)
        if notificacion == BN_CLICKED {
            switch comando {
            case ID_ABRIR_XLSX:
                mostrarInfoIngles("Info", "ABRIR XLSX")
                return 0
            case ID_EXPORTAR_CSV:
                mostrarInfoIngles("Info", "EXPORTAR CSV")
                return 0
            case ID_RECARGAR:
                mostrarInfoIngles("Info", "RECARGAR")
                return 0
            case ID_COLUMNAS:
                mostrarInfoIngles("Info", "COLUMNAS")
                return 0
            case ID_FILTRAR:
                mostrarInfoIngles("Info", "FILTRAR")
                return 0
            case ID_LIMPIAR:
                mostrarInfoIngles("Info", "LIMPIAR")
                return 0
            }
        }
        return 0

    case WM_CLOSE:
        pDestroyWindow.Call(hwnd)
        return 0

    case WM_DESTROY:
        pPostQuitMessage.Call(0)
        return 0
    }

    ret, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
    return ret
}

// --- Funciones de creación de controles (ingles) ---
func crearControlesIngles(hwnd uintptr) {
    var rect RECT
    pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
    ancho := int(rect.Right - rect.Left)
    alto := int(rect.Bottom - rect.Top)

    y := 10
    x := 10

    crearBotonIngles(hwnd, "ABRIR XLSX", x, y, 100, 28, ID_ABRIR_XLSX)
    x += 110
    crearBotonIngles(hwnd, "TOMAR EXCEL ABIERTO", x, y, 140, 28, ID_TOMAR_EXCEL)
    x += 150
    crearBotonIngles(hwnd, "RECARGAR", x, y, 80, 28, ID_RECARGAR)
    x += 90
    crearBotonIngles(hwnd, "COLUMNAS...", x, y, 90, 28, ID_COLUMNAS)
    x += 100
    crearBotonIngles(hwnd, "FILTROS CABECERA...", x, y, 130, 28, ID_FILTROS)
    x += 140
    crearBotonIngles(hwnd, "EXPORTAR CSV", x, y, 100, 28, ID_EXPORTAR_CSV)
    x += 110
    crearBotonIngles(hwnd, "SIMULADOR", x, y, 80, 28, ID_SIMULADOR)
    x += 90
    crearBotonIngles(hwnd, "RESALTAR...", x, y, 80, 28, ID_RESALTAR)
    x += 90
    crearBotonIngles(hwnd, "+/- COLOR...", x, y, 90, 28, ID_COLOR)
    x += 100
    crearBotonIngles(hwnd, "DATOS CSV...", x, y, 90, 28, ID_DATOS_CSV)

    // Combo de registros
    crearComboIngles(hwnd, ancho-60, y, 50, 100, 0)

    y = 48
    crearLabelIngles(hwnd, "SO", 10, y+5, 20, 20)
    crearInputIngles(hwnd, 35, y, 80, 24)

    crearLabelIngles(hwnd, "Estado", 125, y+5, 45, 20)
    crearInputIngles(hwnd, 175, y, 80, 24)

    crearLabelIngles(hwnd, "SKU", 265, y+5, 30, 20)
    crearInputIngles(hwnd, 300, y, 80, 24)

    crearLabelIngles(hwnd, "SUMA DE", 390, y+5, 55, 20)
    crearInputIngles(hwnd, 450, y, 80, 24)

    crearLabelIngles(hwnd, "SDSRP2", 540, y+5, 45, 20)
    crearInputIngles(hwnd, 590, y, 80, 24)

    crearBotonIngles(hwnd, "FILTRAR", 680, y, 70, 28, ID_FILTRAR)
    crearBotonIngles(hwnd, "LIMPIAR", 760, y, 70, 28, ID_LIMPIAR)

    // Grid
    y = 82
    altoGrid := alto - y - 40

    g_hwndGrid, _, _ = pCreateWindowExW.Call(
        WS_EX_CLIENTEDGE,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SysListView32"))),
        0,
        WS_CHILD|WS_VISIBLE|LVS_REPORT|LVS_SINGLESEL|WS_BORDER,
        uintptr(10), uintptr(y),
        uintptr(ancho-20), uintptr(altoGrid),
        hwnd,
        ID_GRID,
        g_hInstance,
        0,
    )

    // Status bar
    y = alto - 30
    g_hwndStatus, _, _ = pCreateWindowExW.Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("STATIC"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("MODO: SO RETENIDAS | RETENIDAS 0 | LIBERADAS 0 | SO 0 | LINEAS 0 | 0 filtros | Detalle de Descuentos Aplicados... | CSV"))),
        WS_CHILD|WS_VISIBLE|WS_BORDER,
        uintptr(10), uintptr(y),
        uintptr(ancho-20), uintptr(30),
        hwnd,
        ID_STATUS,
        g_hInstance,
        0,
    )
}

func redimensionarControlesIngles(hwnd uintptr) {
    var rect RECT
    pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
    ancho := int(rect.Right - rect.Left)
    alto := int(rect.Bottom - rect.Top)

    if g_hwndGrid != 0 {
        pMoveWindow.Call(g_hwndGrid, 10, 82, uintptr(ancho-20), uintptr(alto-82-40), 1)
    }
    if g_hwndStatus != 0 {
        pMoveWindow.Call(g_hwndStatus, 10, uintptr(alto-30), uintptr(ancho-20), 30, 1)
    }
}

// --- Funciones auxiliares de creación de controles (ingles) ---
func crearBotonIngles(hwnd uintptr, texto string, x, y, ancho, alto int, id uintptr) {
    pCreateWindowExW.Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("BUTTON"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(texto))),
        WS_CHILD|WS_VISIBLE|WS_TABSTOP,
        uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto),
        hwnd,
        id,
        g_hInstance,
        0,
    )
}

func crearLabelIngles(hwnd uintptr, texto string, x, y, ancho, alto int) {
    pCreateWindowExW.Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("STATIC"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(texto))),
        WS_CHILD|WS_VISIBLE,
        uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto),
        hwnd,
        0,
        g_hInstance,
        0,
    )
}

func crearInputIngles(hwnd uintptr, x, y, ancho, alto int) {
    pCreateWindowExW.Call(
        WS_EX_CLIENTEDGE,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("EDIT"))),
        0,
        WS_CHILD|WS_VISIBLE|WS_TABSTOP,
        uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto),
        hwnd,
        0,
        g_hInstance,
        0,
    )
}

func crearComboIngles(hwnd uintptr, x, y, ancho, alto int, id uintptr) {
    pCreateWindowExW.Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("COMBOBOX"))),
        0,
        WS_CHILD|WS_VISIBLE|CBS_DROPDOWN|CBS_HASSTRINGS|WS_TABSTOP,
        uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto),
        hwnd,
        id,
        g_hInstance,
        0,
    )
}

func mostrarInfoIngles(titulo, mensaje string) {
    pMessageBoxW.Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(mensaje))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(titulo))),
        0x40, // MB_ICONINFORMATION
    )
}

// --- Agregar los procs que faltaban ---
var (
    pGetMessageW      *syscall.Proc
    pTranslateMessage *syscall.Proc
    pDispatchMessageW *syscall.Proc
)

func init() {
    user32 := syscall.NewLazyDLL("user32.dll")
    pGetMessageW = user32.NewProc("GetMessageW")
    pTranslateMessage = user32.NewProc("TranslateMessage")
    pDispatchMessageW = user32.NewProc("DispatchMessageW")
}
