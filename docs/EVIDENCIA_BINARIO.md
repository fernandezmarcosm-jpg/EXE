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
- La reconstrucción mantiene el botón local `ABRIR XLSX` ya existente en la Entrega 1 para poder compilar/probar el núcleo sin el V54 externo. Esto es una adaptación de prueba, no una afirmación de que el V57 original creara ese botón de esa forma.
- `repositionOverlay` queda sin uso efectivo porque el README de V57 indica que el botón visible ya no debe ocultarse ni reemplazarse por un overlay.
- Las firmas y estructuras de configuración que no son recuperables literalmente se implementan de forma conservadora y están marcadas en `core.go` como inferencia.
- La escritura de una hoja XLSX completa no se inventa: `buildMergedSheet` conserva la operación observada, pero el contrato exacto del archivo de salida y su entrega al motor V54 queda pendiente de validación con el motor real.

## Persistencia

El ZIP analizado contiene únicamente el EXE y el LEEME. No incluye SQLite/Access ni `GestionSO_Datos.csv`. El LEEME indica expresamente que `GestionSO_Datos.csv` no está incluido.

## Entrega 2

Se agregó `core.go` con tipos, logging defensivo, lectura XLSX ZIP/XML, merge, configuración, CSV/maestro, vistas/proceso y stubs explícitos para opciones/simulador donde la evidencia disponible no permite reconstruir la lógica interna con honestidad.

## Validación integrada de compilación

El workflow permanente `Validar Go` ejecutó en GitHub Actions, sobre el commit `dd27561a28bd1cbdd0eefd9eb8c84d9ece57f24e`:

- `CGO_ENABLED=0`
- `GOOS=windows`
- `GOARCH=amd64`
- Go `1.23.2`
- `go build ./...` → **success**, sin errores de compilación.
- `go vet ./...` → **success**, sin salida de errores.

Run verificado: https://github.com/fernandezmarcosm-jpg/EXE/actions/runs/33654572489

No fue necesario corregir símbolos/tipos duplicados, firmas incompatibles, imports faltantes/no usados ni redefiniciones de `initLog`/`logf` en esa validación.

## Build de entrega

Se agregó el workflow permanente `.github/workflows/build-exe.yml`, que compila como Windows GUI mediante:

`CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o dist/GestionSO-V57.exe ./...`

El workflow empaqueta el `.exe` junto con `README.md`, este documento y `LEEME.txt` en `GestionSO-V57.zip`, y lo publica como artifact `GestionSO-V57`. Los binarios no se commitean al repositorio.

El resultado de este build de entrega debe considerarse validado únicamente cuando su propio run de Actions termine en verde y su artifact aparezca disponible.

## Validación funcional / end-to-end

La compilación verde no valida el flujo funcional completo. Para validar `ABRIR XLSX` de extremo a extremo se requiere `GestionSO-V54-engine.exe`, `GestionSO_Datos.csv` y archivos XLSX de prueba compatibles. Esos materiales no forman parte del material analizado ni del repositorio.

## Limitación principal

Aunque `main.feedEngineFile` existe como símbolo real, **su contrato interno con `GestionSO-V54-engine.exe` no está verificado** porque ese ejecutable no forma parte del material analizado. No se afirma compatibilidad funcional hasta disponer de ese motor y poder probar el flujo completo.
