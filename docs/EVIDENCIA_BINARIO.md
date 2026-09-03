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
- `main.go`: la configuración se carga antes de `crearVentana()`, evitando que `WM_CREATE` inicialice el combo de modo con configuración todavía vacía.
- `main_windows.go`: `GetModuleHandleW`, `LoadCursorW` y `GetSysColorBrush` capturan sus tres retornos antes de inicializar `WNDCLASSEX`.
- `main_windows.go`: `LpszMenuName` quedó como `nil`, porque el campo es un puntero.
- `main_windows.go`: `wndProc` usa argumentos `uintptr`, compatibles con `syscall.NewCallback`.
- `main_windows.go`: `DefWindowProcW` captura `r, _, _` y devuelve únicamente `r`.
- `main_windows.go`: se eliminó la duplicación de constantes e IDs que hacía colisionar controles de la toolbar con IDs de la segunda versión de `main_windows.go`.
- `main_windows.go`: se dejó una única implementación coherente de la UI Win32. En particular, los controles se crean desde `WM_CREATE`; `crearVentana()` ya no crea una segunda copia de los controles después de `CreateWindowExW`.
- `main_windows.go`: se almacenan handles de etiquetas y campos de filtro para que `WM_SIZE` reposicione ambos elementos, no solo los `EDIT`.
- `main_windows.go`: `FILTRAR` conserva el resultado filtrado; la ruta anterior reconstruía la vista con filtros `nil` inmediatamente después de aplicar los filtros.
- `main_windows.go`: `refreshGrid` ahora escribe todas las celdas de las doce columnas, no solamente `SKU`.
- `main_windows.go`: `feedEngineFile` quedó definido una sola vez y respaldado por el símbolo real; su contrato interno queda pendiente por falta del motor V54.
- `go.mod`: se eliminó `github.com/xuri/excelize/v2 v2.8.1`, que no es utilizado por el código reconstruido.
- `.github/workflows/build-exe.yml`: el log real mostró que `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 ...` no puede escribirse como asignación POSIX dentro de PowerShell. Se corrigió definiendo esas variables a nivel de `job`; el comando de build quedó como `go build -ldflags "-H=windowsgui" ...`.
- Repositorio: se eliminó el `GestionSO-V57.zip` que había quedado accidentalmente versionado. `.gitignore` mantiene la exclusión de `*.exe` y `*.zip`.

## Auditoría adicional de main

**Hechos verificados en la versión previa del árbol:**

1. Había dos bloques de constantes con IDs repetidos (`ID_TOMAR_EXCEL`, `ID_SIMULADOR`, `ID_GRID`, etc.), lo que producía redeclaraciones y además provocaba colisiones de comandos.
2. `WM_CREATE` llamaba `crearControles` y `crearVentana` volvía a llamar `crearControles`, generando una doble inicialización de controles.
3. `applyHeaderFilters` calculaba una vista filtrada y luego `updateMainView` podía reemplazarla con una vista sin filtros.
4. `refreshGrid` insertaba únicamente el primer campo de cada línea; las otras once columnas no quedaban pobladas.
5. Las etiquetas de los filtros no tenían handles persistentes y no se reposicionaban en `WM_SIZE`.
6. La carga de configuración ocurría después de `CreateWindowExW`; como `WM_CREATE` se ejecuta durante esa llamada, el combo podía inicializarse con un modo distinto del persistido.

**Soluciones aplicadas:** se consolidaron IDs, se centralizó la creación de controles en `WM_CREATE`, se preservó la vista filtrada, se poblaron todas las celdas de la grilla, se guardaron los handles de etiquetas y se cargó la configuración antes de crear la ventana.

**Inferencia controlada:** ninguna de estas correcciones pretende afirmar que el fuente original tenía exactamente este código. Son correcciones de consistencia de la reconstrucción para que su comportamiento observable sea coherente con la evidencia disponible.

## Validación reproducible

El run de GitHub Actions disparado por el commit de auditoría `72a41b033f20aa877410af3d4ee6edefb6ba6589` terminó **SUCCESS**. El job `validar` terminó correctamente en los pasos `Build integrado` y `go vet`. Run: `33762560468`.

La reconstrucción local de referencia también fue comprobada con `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui"` y `go vet ./...`, antes de publicar la revisión equivalente.

## Persistencia y XLSX

`BuildLines` usa la fila que maximiza `headerScore` como encabezado y conserva esos nombres en `Line.Values`, con fallback `C<n>` solamente cuando un encabezado está vacío. `mergeXLSX` conserva la lectura de hojas y selecciona de forma determinista la primera hoja por nombre ordenado. Los XLSX originales no se modifican.

## Motor V54 y límite funcional

Aunque `main.feedEngineFile` existe como símbolo real y el binario contiene `GestionSO-V54-engine.exe`, el contrato interno no está verificado sin el motor real. `feedEngineFile` se mantiene parametrizado por `GESTIONSO_V54_ENGINE` y no se inventan argumentos.

Para una prueba end-to-end real se necesitan `GestionSO-V54-engine.exe`, `GestionSO_Datos.csv` y XLSX de datos reales/prueba compatibles. Hasta disponer de ellos, quedan pendientes las fórmulas de negocio reales, subtotales exactos y el flujo completo con V54.

## Binarios

No se commitean `.exe` ni `.zip`. El ejecutable de entrega se publica exclusivamente como artifact de GitHub Actions.
