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
    HInstance     syscall.Handle
    HIcon         syscall.Handle
    HCursor       syscall.Handle
    HbrBackground syscall.Handle
    LpszMenuName  *uint16
    LpszClassName *uint16
    HIconSm       syscall.Handle
}

type RECT struct {
    Left, Top, Right, Bottom int32
}

var (
    hInstance  syscall.Handle
    hwndGrid   syscall.Handle
    hwndStatus syscall.Handle
)

func crearVentanaPrincipal() syscall.Handle {
    hInstance = kernel32.NewProc("GetModuleHandleW").Call(0)
    
    className := syscall.StringToUTF16Ptr("GestionSO")
    
    wc := WNDCLASSEX{
        CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
        Style:         0,
        LpfnWndProc:   syscall.NewCallback(wndProc),
        CbClsExtra:    0,
        CbWndExtra:    0,
        HInstance:     syscall.Handle(hInstance),
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
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Gestion SO V57"))),
        0xCF0000|0x10000000,
        100, 100, 800, 500,
        0, 0, hInstance, 0,
    )
    
    return syscall.Handle(hwnd)
}

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
    switch msg {
    case WM_CREATE:
        crearControles(hwnd)
        return 0
    case WM_CLOSE:
        user32.NewProc("DestroyWindow").Call(uintptr(hwnd))
        return 0
    case WM_DESTROY:
        user32.NewProc("PostQuitMessage").Call(0)
        return 0
    }
    return user32.NewProc("DefWindowProcW").Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
}

func crearControles(hwnd syscall.Handle) {
    var rect RECT
    user32.NewProc("GetClientRect").Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
    
    ancho := int(rect.Right - rect.Left)
    alto := int(rect.Bottom - rect.Top)
    
    // Botón ABRIR XLSX
    user32.NewProc("CreateWindowExW").Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("BUTTON"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("ABRIR XLSX"))),
        0x40000000|0x10000000,
        uintptr(10), uintptr(10), uintptr(100), uintptr(28),
        uintptr(hwnd),
        uintptr(ID_ABRIR_XLSX),
        hInstance,
        0,
    )
    
    // Botón EXPORTAR CSV
    user32.NewProc("CreateWindowExW").Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("BUTTON"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("EXPORTAR CSV"))),
        0x40000000|0x10000000,
        uintptr(120), uintptr(10), uintptr(100), uintptr(28),
        uintptr(hwnd),
        uintptr(ID_EXPORTAR_CSV),
        hInstance,
        0,
    )
    
    // Status bar
    y := alto - 30
    hwndStatus, _, _ = user32.NewProc("CreateWindowExW").Call(
        0,
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("STATIC"))),
        uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Listo"))),
        0x40000000|0x10000000|0x00800000,
        uintptr(10), uintptr(y),
        uintptr(ancho-20), uintptr(30),
        uintptr(hwnd),
        uintptr(3002),
        hInstance,
        0,
    )
}
