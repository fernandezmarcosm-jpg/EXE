package main

import (
    "syscall"
    "unsafe"
)

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

var (
    hInstance  uintptr
    hwndGrid   uintptr
    hwndStatus uintptr
)

// crearVentana - FUNCIÓN PRINCIPAL QUE ESTABA FALTANDO
func crearVentana() uintptr {
    hInstance = kernel32.NewProc("GetModuleHandleW").Call(0)

    className := syscall.StringToUTF16Ptr("GestionSO")

    wc := WNDCLASSEX{
        CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
        Style:         0,
        LpfnWndProc:   syscall.NewCallback(wndProc),
        CbClsExtra:    0,
        CbWndExtra:    0,
        HInstance:     hInstance,
        HIcon:         0,
        HCursor:       user32.NewProc("LoadCursorW").Call(0, 32512),
        HbrBackground: user32.NewProc("GetSysColorBrush").Call(15),
        LpszMenuName:  0,
        LpszClassName: className,
        HIconSm:       0,
    }

    user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))

    hwnd, _, _ := user32.NewProc("CreateWindowExW").Call(
        0,
        uintptr(unsafe.Pointer(className)),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Gestion SO V57 - SO RETENIDAS"))),
        0xCF0000|0x10000000,
        100, 100, 1000, 600,
        0, 0, hInstance, 0,
    )

    return hwnd
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
    switch msg {
    case WM_CREATE:
        crearControles(hwnd)
        return 0

    case WM_SIZE:
        redimensionarControles(hwnd)
        return 0

    case WM_COMMAND:
        comando := uint16(wParam & 0xFFFF)
        notificacion := uint16((wParam >> 16) & 0xFFFF)

        if notificacion == BN_CLICKED {
            switch comando {
            case ID_ABRIR_XLSX:
                mostrarInfo("Info", "ABRIR XLSX")
                return 0
            case ID_EXPORTAR_CSV:
                mostrarInfo("Info", "EXPORTAR CSV")
                return 0
            case ID_RECARGAR:
                mostrarInfo("Info", "RECARGAR")
                return 0
            case ID_COLUMNAS:
                mostrarInfo("Info", "COLUMNAS")
                return 0
            case ID_FILTRAR:
                mostrarInfo("Info", "FILTRAR")
                return 0
            case ID_LIMPIAR:
                mostrarInfo("Info", "LIMPIAR")
                return 0
            }
        }
        return 0

    case WM_CLOSE:
        user32.NewProc("DestroyWindow").Call(hwnd)
        return 0

    case WM_DESTROY:
        user32.NewProc("PostQuitMessage").Call(0)
        return 0
    }

    return user32.NewProc("DefWindowProcW").Call(hwnd, uintptr(msg), wParam, lParam)
}

func crearControles(hwnd uintptr) {
    var rect RECT
    user32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&rect)))

    ancho := int(rect.Right - rect.Left)
    alto := int(rect.Bottom - rect.Top)

    y := 10
    x := 10

    crearBoton(hwnd, "ABRIR XLSX", x, y, 100, 28, ID_ABRIR_XLSX)
    x += 110
    crearBoton(hwnd, "RECARGAR", x, y, 80, 28, ID_RECARGAR)
    x += 90
    crearBoton(hwnd, "COLUMNAS...", x, y, 90, 28, ID_COLUMNAS)
    x += 100
    crearBoton(hwnd, "EXPORTAR CSV", x, y, 100, 28, ID_EXPORTAR_CSV)

    y = 48

    crearLabel(hwnd, "SO", 10, y+5, 20, 20)
    crearInput(hwnd, 35, y, 80, 24)

    crearLabel(hwnd, "SKU", 125, y+5, 30, 20)
    crearInput(hwnd, 160, y, 80, 24)

    crearBoton(hwnd, "FILTRAR", 250, y, 70, 28, ID_FILTRAR)
    crearBoton(hwnd, "LIMPIAR", 330, y, 70, 28, ID_LIMPIAR)

    y = 82
    altoGrid := alto - y - 40

    hwndGrid, _, _ = user32.NewProc("CreateWindowExW").Call(
        0x00000200,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SysListView32"))),
        0,
        0x40000000|0x10000000|0x0001|0x00800000,
        uintptr(10), uintptr(y),
        uintptr(ancho-20), uintptr(altoGrid),
        hwnd,
        uintptr(3001),
        hInstance,
        0,
    )

    y = alto - 30
    hwndStatus, _, _ = user32.NewProc("CreateWindowExW").Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("STATIC"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("MODO: SO RETENIDAS | LINEAS: 0"))),
        0x40000000|0x10000000|0x00800000,
        uintptr(10), uintptr(y),
        uintptr(ancho-20), uintptr(30),
        hwnd,
        uintptr(3002),
        hInstance,
        0,
    )
}

func crearBoton(hwnd uintptr, texto string, x, y, ancho, alto int, id uintptr) {
    user32.NewProc("CreateWindowExW").Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("BUTTON"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(texto))),
        0x40000000|0x10000000|0x00010000,
        uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto),
        hwnd,
        id,
        hInstance,
        0,
    )
}

func crearLabel(hwnd uintptr, texto string, x, y, ancho, alto int) {
    user32.NewProc("CreateWindowExW").Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("STATIC"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(texto))),
        0x40000000|0x10000000,
        uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto),
        hwnd,
        0,
        hInstance,
        0,
    )
}

func crearInput(hwnd uintptr, x, y, ancho, alto int) {
    user32.NewProc("CreateWindowExW").Call(
        0x00000200,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("EDIT"))),
        0,
        0x40000000|0x10000000|0x00010000,
        uintptr(x), uintptr(y), uintptr(ancho), uintptr(alto),
        hwnd,
        0,
        hInstance,
        0,
    )
}

func redimensionarControles(hwnd uintptr) {
    var rect RECT
    user32.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&rect)))

    ancho := int(rect.Right - rect.Left)
    alto := int(rect.Bottom - rect.Top)

    if hwndGrid != 0 {
        user32.NewProc("MoveWindow").Call(
            hwndGrid,
            10, 82,
            uintptr(ancho-20), uintptr(alto-82-40),
            1,
        )
    }

    if hwndStatus != 0 {
        user32.NewProc("MoveWindow").Call(
            hwndStatus,
            10, uintptr(alto-30),
            uintptr(ancho-20), 30,
            1,
        )
    }
}

func mostrarInfo(titulo, mensaje string) {
    user32.NewProc("MessageBoxW").Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(mensaje))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(titulo))),
        0x40,
    )
}
