//go:build windows

package main

import (
    "strconv"
    "strings"
    "time"
    "unicode/utf16"
    "unsafe"
)

// WM_APP + 2: reparación del ListView en el mismo hilo Win32 que posee la ventana.
// Se usa PostMessage desde el observador y la operación real se ejecuta en WndProc.
const wmAppRepairList uint32 = 0x8002

var repairListKey string

func init() {
    go func() {
        for {
            if appHwnd != 0 && viewDataset != nil {
                user32.NewProc("PostMessageW").Call(appHwnd, uintptr(wmAppRepairList), 0, 0)
            }
            time.Sleep(250 * time.Millisecond)
        }
    }()
}

func columnViewRepairConcrete() {
    if viewList == 0 || viewDataset == nil {
        return
    }

    records := safeFilterRecords(viewDataset, safeSnapshotFilters())
    visible := columnViewVisibleColumns()

    // La identidad del dataset y los filtros forman parte de la clave: así no
    // quedan filas viejas si dos importaciones tienen la misma cantidad de líneas.
    keyParts := []string{
        strconv.FormatUint(uint64(uintptr(unsafe.Pointer(viewDataset))), 10),
        strconv.Itoa(len(records)),
        strconv.Itoa(len(visible)),
    }
    for _, c := range visible {
        keyParts = append(keyParts, datasetColumnKey(c))
        if h := viewFilters[c.ID]; h != 0 {
            keyParts = append(keyParts, appGetEdit(h))
        }
    }
    key := strings.Join(keyParts, "|")

    getStyle := user32.NewProc("GetWindowLongPtrW")
    setStyle := user32.NewProc("SetWindowLongPtrW")
    gwlpStyle := ^uint(15) // -16, GWLP_STYLE en Win64.
    style, _, _ := getStyle.Call(viewList, uintptr(gwlpStyle))
    const lvsOwnerDataFix uintptr = 0x1000
    converted := style&lvsOwnerDataFix != 0
    if converted {
        // La vista virtual actual es la fuente del problema: el contenido depende
        // de LVN_GETDISPINFO. La pasamos a una vista normal y cargamos filas reales.
        setStyle.Call(viewList, uintptr(gwlpStyle), style&^lvsOwnerDataFix)
        repairListKey = ""
    }

    if !converted && key == repairListKey {
        return
    }

    send := user32.NewProc("SendMessageW")
    send.Call(viewList, lvmDeleteAll, 0, 0)

    // Las columnas se reconstruyen de forma determinista en cada repoblación.
    header, _, _ := send.Call(viewList, lvmGetHeader, 0, 0)
    hcount := uintptr(0)
    if header != 0 {
        hcount, _, _ = send.Call(header, hdmGetItemCount, 0, 0)
    }
    for hcount > 0 {
        if r, _, _ := send.Call(viewList, lvmDeleteColumn, 0, 0); r == 0 {
            break
        }
        hcount--
    }

    for i, c := range visible {
        width := c.Width
        if width < 120 {
            width = 120
        }
        if width > 420 {
            width = 420
        }
        fmtCol := int32(lvcfmtLeft)
        if c.Type == ValueNumber {
            fmtCol = lvcfmtRight
        }
        p := appU16(datasetColumnDisplayTitle(c))
        lc := lvColumn{
            Mask:    lvcfText,
            Fmt:     fmtCol,
            Cx:      int32(width),
            Text:    p,
            SubItem: int32(i),
        }
        send.Call(viewList, lvmInsertColumnW, uintptr(i), uintptr(unsafe.Pointer(&lc)))
    }

    for ri, r := range records {
        for ci, c := range visible {
            txt := datasetCellText(r, c)
            u16 := utf16.Encode([]rune(txt))
            p := appU16(txt)
            it := lvItem{
                Mask:    lvifText,
                Item:    int32(ri),
                SubItem: int32(ci),
                Text:    p,
                TextMax: int32(len(u16) + 1),
            }
            if ci == 0 {
                send.Call(viewList, lvmInsertItemW, 0, uintptr(unsafe.Pointer(&it)))
            } else {
                send.Call(viewList, lvmSetItemTextW, uintptr(ri), uintptr(unsafe.Pointer(&it)))
            }
        }
    }

    if appSettings.SubtotalEnabled && appSettings.SubtotalColumn != "" && len(records) > 0 {
        columnViewAddSubtotal(visible, records)
    }

    user32.NewProc("InvalidateRect").Call(viewList, 0, 1)
    user32.NewProc("UpdateWindow").Call(viewList)
    repairListKey = key
    appLog("DIAGNOSTICO: ListView concreto; filas=%d columnas=%d", len(records), len(visible))
}
