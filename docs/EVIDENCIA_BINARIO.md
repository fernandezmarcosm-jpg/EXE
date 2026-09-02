# Evidencia del binario — GestionSO V57

## Alcance

Documento de trazabilidad de la reconstrucción. Se separan hechos observados del binario de inferencias necesarias para producir código compilable y una UI visualmente próxima a la captura real de Gestion SO V54.

## Hechos verificados del binario

- Archivo analizado: `GestionSO-V57.exe`.
- PE32+, x86-64, Windows GUI, Go `1.23.2`, `GOOS=windows`, `GOARCH=amd64`, `CGO_ENABLED=0`.
- No hay CLR Runtime Header; no es .NET.
- Existen símbolos `main.registerClass`, `main.createWindow`, `main.createMainControls`, `main.layoutMain`, `main.mainWndProc`, `main.handleCommand`, `main.handleNotify`, `main.handleMainNotify`.
- Existen `main.installMultiSelectButton`, `main.openXLSXDialog`, `main.pickMultipleXLSX`, `main.repositionOverlay`.
- Existe realmente `main.feedEngineFile`.
- Existen `main.ReadXLSX`, `main.readXLSXDoc`, `main.readZipEntry`, `main.decodeSheet`, `main.parseSharedStrings`, `main.parseRows`, `main.normalizeRows`, `main.normalizeRowXML`, `main.buildMergedSheet`, `main.rewriteRowNumber`, `main.xmlEscape`, `main.hashRow`, `main.colFromRef`, `main.headerScore`, `main.mergeXLSX`.
- Existen `BuildLines`, `GroupLines`, `BuildFilteredSortedView`, `FilterValue`, `DisplayValue`, `availableColumns` y símbolos de ordenamiento/vista.
- El binario contiene `ABRIR XLSX`, `*.xlsx`, `GestionSO-V54-engine.exe`, `GestionSO-debug.log`, `GestionSO-V57-debug.log` y las tres cadenas de modo exactas: `MODO: FACTURAS PENDIENTES`, `MODO: SO RETENIDAS`, `MODO: FACTURAS`.
- El binario contiene los formatos exactos `BULTOS %s | PALLETS %s | TN %s | UNIDADES %s` y `NETO $ %s | COSTO $ %s | RESULTADO %s | CMG %s`.
- El binario contiene APIs Win32 incluyendo `GetOpenFileNameW`, `SendMessageW`, `ShowWindow`, `GetDlgItem`, `FindWindowW`, `EnumWindows`, `EnumChildWindows`, `GetWindowTextW`, `SetWindowTextW`, `GetClassNameW`, `GetWindowRect`, `SetWindowLongPtrW`, `CallWindowProcW`, `GetSystemMetrics`, `TranslateMessage`, `DispatchMessageW`.
- El binario contiene una referencia a automatización COM/PowerShell de Excel mediante `Runtime.InteropServices.Marshal` y `GetActiveObject('Excel...`.

## Evidencia visual (captura del programa en funcionamiento)

La siguiente lista registra como **hecho verificado visualmente** lo observado en la captura real aportada para el modo `SO RETENIDAS`. La captura es evidencia visual de V54; el binario analizado es V57, por lo que se documenta explícitamente la discrepancia de nombre.

### Hechos verificados por la captura

- Título: `Gestion SO V54 - SO RETENIDAS / CSV maestro`.
- Barra superior con `ABRIR XLSX`, `TOMAR EXCEL ABIERTO`, `RECARGAR`, `COLUMNAS...`, `FILTROS CABECERA...`, `EXPORTAR CSV`, `SIMULADOR`, `RESALTAR...`, `+/- COLOR...`, `DATOS CSV...` y un combo con valor `10`.
- Fila de filtros con etiquetas `SO`, `Estado`, `SKU`, `SUMA DE`, `SDSRP2` y botones `FILTRAR` y `LIMPIAR`.
- Grilla con columnas, en este orden: `SKU`, `Descripción`, `SUM (%) descuento`, `NETO PK`, `UNIDADES`, `PALL`, `PK`, `NETO SO`, `TN SO`, `CMG`, `PPP SO`, `ORIGEN`.
- `CMG` aparece como columna ordenada con indicador `▼`.
- Existen filas de subtotal por grupo con prefijo `SUBTOTAL SO` y datos de retención/estado/código/cliente y métricas de unidades, pallets, PK, neto SO, TN SO, CMG y PPP.
- Barra de estado con formato observado: `MODO: SO RETENIDAS | RETENIDAS <n> | LIBERADAS <n> | SO <n> | LINEAS <n> | <n> filtros | Detalle de Descuentos Aplicados... | CSV`.

### Inferencias de implementación

- El título se genera desde `configData.Mode`, quitando el prefijo `MODO: ` y formando `Gestion SO V54 - <modo> / CSV maestro`. La discrepancia es intencional: el ejecutable reconstruido es V57, pero la captura de referencia muestra V54.
- Los botones sin comportamiento recuperable se muestran como controles Win32 y sus handlers registran un stub en `%TEMP%\\GestionSO-V57-debug.log`; no se inventa lógica para `TOMAR EXCEL ABIERTO`, `COLUMNAS...`, `FILTROS CABECERA...`, `SIMULADOR`, `RESALTAR...`, `+/- COLOR...` o `DATOS CSV...`.
- `TOMAR EXCEL ABIERTO` queda expresamente pendiente porque la evidencia solo demuestra la referencia COM/Excel, no su protocolo completo.
- La grilla usa `SysListView32` en modo reporte y `ColumnDef` con las doce columnas observadas. El mapeo busca nombres reales de encabezado mediante coincidencias conservadoras; si no encuentra un campo, deja vacío.
- Los filtros `SO`, `Estado`, `SKU`, `SUMA DE` y `SDSRP2` se conectan a `BuildFilteredSortedViewByHeaders`. `LIMPIAR` vacía los campos y restaura la vista.
- El ordenamiento mantiene el campo real `SO` como primera clave preferida, con fallback a `factura` y `cliente`.
- Las filas de subtotal se agregan por grupo de `SO` mediante `CalculateSOSubtotals` y se presentan como fila informativa de la grilla.
- La barra de estado se construye mediante `BuildStatusBar`; `RETENIDAS`, `LIBERADAS`, cantidad de `SO` y cantidad de líneas se derivan conservadoramente de los datos cargados.
- Los cálculos de subtotales y conteos **no son afirmados como las fórmulas originales**. No se dispone de evidencia suficiente para recuperar las fórmulas exactas de negocio.
- El layout, tamaños, espaciados, fuentes y métricas de pixel son aproximaciones visuales; los símbolos y strings no permiten reconstruirlos literalmente.

## Mapeo de columnas

La grilla fija conserva el orden observado y resuelve valores contra encabezados reales detectados por `BuildLines`. Alias conservadores usados: `SKU→sku`; `Descripción→descrip/descripcion/producto`; `SUM (%) descuento→sum/descuento`; `NETO PK→neto pk`; `UNIDADES→unidades/unidad/cantidad`; `PALL→pall/pallet`; `PK→pk`; `NETO SO→neto so`; `TN SO→tn so/tn/tonelada`; `CMG→cmg/margen`; `PPP SO→ppp so/ppp`; `ORIGEN→origen`.

## Correcciones de compilación

Estas correcciones son **hechos verificados en el árbol fuente y/o en los logs de GitHub Actions** y fueron realizadas para eliminar inconsistencias entre las versiones mezcladas de `main.go` y `main_windows.go`:

- `main.go`: se eliminó el import no utilizado `os`.
- `main.go`: `GetConsoleWindow` y `GetMessageW` capturan los tres retornos de `syscall.Proc.Call` (`ret, _, _`).
- `main.go`: se unificó el punto de entrada con `crearVentana()`.
- `main.go`: se agregó `//go:build windows` y una estructura `winMSG` compatible con la API Win32, porque el log real de compilación mostró `./main.go:44:21: undefined: syscall.MSG`.
- `main_windows.go`: `GetModuleHandleW`, `LoadCursorW` y `GetSysColorBrush` capturan sus tres retornos antes de inicializar `WNDCLASSEX`.
- `main_windows.go`: `LpszMenuName` quedó como `nil`, porque el campo es un puntero.
- `main_windows.go`: `wndProc` usa argumentos `uintptr`, compatibles con `syscall.NewCallback`.
- `main_windows.go`: `DefWindowProcW` captura `r, _, _` y devuelve únicamente `r`.
- `main_windows.go`: se consolidó una única implementación de la UI Win32, selector múltiple XLSX, filtros, grilla, subtotales y barra de estado.
- `main_windows.go`: se agregó `WM_INITDIALOG` para el hook del selector múltiple.
- `main_windows.go`: se dejó una única implementación de `feedEngineFile`, respaldada por el símbolo real; su contrato interno queda pendiente por falta del motor V54.
- `go.mod`: se eliminó `github.com/xuri/excelize/v2 v2.8.1`, que no es utilizado por el código reconstruido.
- `.github/workflows/build-exe.yml`: el log real mostró que `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 ...` no puede escribirse como asignación POSIX dentro de PowerShell. Se corrigió definiendo esas variables a nivel de `job`; el comando de build quedó como `go build -ldflags "-H=windowsgui" ...`.
- Repositorio: se eliminó el `GestionSO-V57.zip` que había quedado accidentalmente versionado. `.gitignore` mantiene la exclusión de `*.exe` y `*.zip`.

## Validación reproducible

**Build integrado en verde:** run `Validar Go` **33695199005**, asociado al código reconstruido ya consolidado; `go build ./...` terminó correctamente y `go vet ./...` también terminó correctamente.

**Build del ejecutable en verde:** run `Build GestionSO V57` **33695331006**, sobre `main` en el commit `4c5b893afd3139eb221a96c9b49d8da5548f477c`. Los pasos `Build GestionSO V57`, `go vet` y `Upload artifact` terminaron correctamente.

El artifact `GestionSO-V57` fue generado con ID `9871569226`, tamaño `1.962.292` bytes y SHA-256 del ZIP de artifact `893d26511e9e41245e7444f4d784645527a09700fdd09f4d7953229f113cb656`.

La reconstrucción generada dentro del artifact es un **PE32+ Windows x86-64 GUI**. El ejecutable extraído del artifact tiene `3.302.912` bytes. Esta diferencia de tamaño respecto del binario original no implica equivalencia ni diferencia funcional por sí sola.

## Persistencia y XLSX

`BuildLines` usa la fila que maximiza `headerScore` como encabezado y conserva esos nombres en `Line.Values`, con fallback `C<n>` solamente cuando un encabezado está vacío. `mergeXLSX` conserva la lectura de hojas y eliminación de encabezado duplicado observada en la reconstrucción previa. Los XLSX originales no se modifican.

## Motor V54 y límite funcional

Aunque `main.feedEngineFile` existe como símbolo real y el binario contiene `GestionSO-V54-engine.exe`, el contrato interno no está verificado sin el motor real. `feedEngineFile` se mantiene parametrizado por `GESTIONSO_V54_ENGINE` y no se inventan argumentos.

Para una prueba end-to-end real se necesitan `GestionSO-V54-engine.exe`, `GestionSO_Datos.csv` y XLSX de datos reales/prueba compatibles. Hasta disponer de ellos, quedan pendientes las fórmulas de negocio reales, subtotales exactos y el flujo completo con V54.

## Binarios

No se commitean `.exe` ni `.zip`. El ejecutable de entrega se publica exclusivamente como artifact de GitHub Actions.
