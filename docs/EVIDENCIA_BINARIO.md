# Evidencia del binario — GestionSO V57

## Alcance

Documento de trazabilidad de la reconstrucción. Se separan hechos observados del binario de inferencias necesarias para producir código compilable.

## Hechos verificados

- Archivo analizado: `GestionSO-V57.exe`.
- PE: PE32+, x86-64, Windows GUI.
- Build Go: `go1.23.2`.
- `GOARCH=amd64`.
- `GOOS=windows`.
- `CGO_ENABLED=0`.
- Buildmode: `exe`.
- `-trimpath=true`.
- No hay CLR Runtime Header; no es .NET.
- El binario contiene símbolos/nombres `main.registerClass`, `main.createWindow`, `main.createMainControls`, `main.layoutMain`, `main.mainWndProc`, `main.handleCommand`, `main.handleNotify`, `main.handleMainNotify`.
- El binario contiene `main.installMultiSelectButton`, `main.openXLSXDialog`, `main.pickMultipleXLSX`, `main.repositionOverlay`.
- El binario contiene `main.multiSZ`, `main.u16`, `main.u16z`.
- El binario contiene `main.findWindowByTitles`, `main.enumTopWindows`, `main.enumChildren`, `main.findChildByText`, `main.findFirstEdit`, `main.findDialogUnder`, `main.windowText`, `main.getClassName`, `main.setWindowText`.
- **El símbolo `main.feedEngineFile` existe realmente.**
- El binario contiene `main.ReadXLSX`, `main.readXLSXDoc`, `main.readZipEntry`, `main.decodeSheet`, `main.parseSharedStrings`, `main.parseRows`, `main.normalizeRows`, `main.normalizeRowXML`, `main.buildMergedSheet`, `main.rewriteRowNumber`, `main.xmlEscape`, `main.hashRow`, `main.colFromRef`, `main.headerScore`, `main.mergeXLSX`.
- El binario contiene los nombres de archivos fuente `gestionso/core.go` y `gestionso/main_windows.go`.
- El binario contiene `ABRIR XLSX`.
- El binario contiene `*.xlsx`.
- El binario contiene `ABRIR XLSX multi-select hook installed hwnd=%x`.
- El binario contiene `GestionSO-V54-engine.exe`.
- El binario contiene `GestionSO-debug.log` y `GestionSO-V57-debug.log`.
- El binario contiene APIs/referencias Win32 incluyendo `GetOpenFileNameW`, `SendMessageW`, `ShowWindow`, `GetDlgItem`, `FindWindowW`, `EnumWindows`, `EnumChildWindows`, `GetWindowTextW`, `SetWindowTextW`, `GetClassNameW`, `GetWindowRect`, `SetWindowLongPtrW`, `CallWindowProcW`, `GetSystemMetrics`, `TranslateMessage`, `DispatchMessageW`.
- El binario contiene `archive/zip`, `encoding/xml`, `encoding/csv` y `encoding/json`.
- El binario contiene cadenas de modo: `MODO: FACTURAS PENDIENTES`, `MODO: SO RETENIDAS`, `MODO: FACTURAS`.
- El binario contiene formatos `BULTOS %s | PALLETS %s | TN %s | UNIDADES %s` y `NETO $ %s | COSTO $ %s | RESULTADO %s | CMG %s`.
- El binario contiene una referencia a automatización COM/PowerShell de Excel mediante `Runtime.InteropServices.Marshal` y `GetActiveObject('Excel...`.

## Inferencias de implementación

- El selector múltiple se reimplementa usando `OPENFILENAME` + `GetOpenFileNameW` y `OFN_ALLOWMULTISELECT`.
- El hook se conecta a `OPENFILENAME.lpfnHook`; su comportamiento exacto original no puede recuperarse literalmente del símbolo.
- El V54 externo se representa mediante la variable de entorno `GESTIONSO_V54_ENGINE`; no se inventa el contrato interno del motor.
- La reconstrucción crea un botón local `ABRIR XLSX` para poder compilar y probar el núcleo sin el V54 externo. Esto es una adaptación de prueba, no una afirmación de que el V57 original creara ese botón de esa forma.
- `repositionOverlay` queda sin uso efectivo porque el README de V57 indica que el botón visible ya no debe ocultarse ni reemplazarse por un overlay.

## Limitaciones

Sin `GestionSO-V54-engine.exe`, `GestionSO_Datos.csv` y el entorno original de ejecución no es posible afirmar que la reconstrucción reproduzca el flujo completo de producción.
