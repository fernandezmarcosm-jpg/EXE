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
- `main_windows.go`: se dejó una única implementación coherente de la UI Win32. Los controles se crean desde `WM_CREATE`; `crearVentana()` no crea una segunda copia después de `CreateWindowExW`.
- `main_windows.go`: se almacenan handles de etiquetas y campos de filtro para que `WM_SIZE` reposicione ambos elementos.
- `main_windows.go`: `FILTRAR` conserva el resultado filtrado; `LIMPIAR` restaura explícitamente `currentView` desde `mainLines`.
- `main_windows.go`: `refreshGrid` escribe todas las celdas de las doce columnas y agrega filas informativas de subtotal por SO.
- `main_windows.go`: `feedEngineFile` quedó definido una sola vez y respaldado por el símbolo real; su contrato interno queda pendiente por falta del motor V54.
- `go.mod`: se eliminó `github.com/xuri/excelize/v2 v2.8.1`, que no es utilizado por el código reconstruido.
- `.github/workflows/build-exe.yml`: se corrigió el build para PowerShell/Windows y se configuró la generación de un ZIP con el ejecutable y documentación como artifact; el binario no se versiona.
- `functional_app_windows.go`: se eliminó la segunda definición de `appRecover()` sin argumentos. La única definición queda en `crashlog_windows.go` con firma `appRecover(where string)` y `appFinishImport` usa `defer appRecover("appFinishImport")`.
- `main.go`, `windows_safe_view.go` y `config_windows.go`: las llamadas a `appRecover` pasan explícitamente el nombre de la función contenedora.
- `column_view_windows.go`: se agregó el import `unicode/utf16` requerido por `appGetEdit`.
- `column_view_windows.go`: se restauró `configWndProc` mediante `config_windows.go`, eliminando el símbolo indefinido detectado por el build.
- `windows_safe_view.go`: la altura de fuente para `CreateFontW` se representa sin overflow de constante, mediante conversión a `int32` antes de pasarla a `uintptr`.

## Auditoría adicional de la revisión `main` recibida el 2026-09-03

### Hechos verificados en el código revisado

1. El ZIP recibido contiene una única `main_windows.go`; no se detectaron dos versiones del archivo dentro del árbol entregado.
2. La revisión auditada compila con `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` y pasa `go vet ./...`.
3. `main.go` carga `mainConfig` antes de crear la ventana.
4. `main_windows.go` concentra los IDs de controles en un único bloque y usa `crearVentana()`.
5. `LIMPIAR` restablece `currentView` a `mainLines`.
6. `BuildFilteredSortedViewByHeaders` aplica los filtros al campo indicado, no a cualquier columna.
7. La vista filtrada se ordena de forma estable por `SO`, con fallback a `factura` y `cliente`.
8. El parser XLSX respeta referencias de celda (`A1`, `C1`, etc.) y conserva columnas omitidas.
9. Las cadenas `inlineStr` con varios `<t>` se concatenan.
10. `mergeXLSX` selecciona la primera hoja por nombre ordenado para evitar resultados dependientes del orden de un `map`.
11. `parseNumber` admite formatos con coma y punto, incluyendo `1.234,56` y `1,234.56`.
12. La grilla genera una fila `SUBTOTAL SO <id>` después de cada grupo de SO.

### Inferencias controladas

- La fórmula exacta de subtotales no se recupera del binario. La reconstrucción suma los campos numéricos disponibles.
- El texto complementario de la fila subtotal (`RET`, estado, código y cliente) se deriva de las líneas del grupo; no se afirma equivalencia exacta.
- El modo seleccionado se persiste y actualiza el título, pero no se aplica una regla de filtrado comercial específica a cada modo porque esa condición no está demostrada.
- `TOMAR EXCEL ABIERTO` continúa como stub porque no está recuperado su contrato COM completo.
- `findDialogUnder` no devuelve una ventana arbitraria; queda pendiente porque su relación exacta con el diálogo V54 no está demostrada.
- El ajuste de anchos de toolbar y layout es aproximado y no pretende ser una reconstrucción pixel-perfect.

## Correcciones de estabilidad

### Cuelgue del handler `WM_APP_IMPORT_DONE` (0x8001)

**Hecho verificado por el log de runtime:** la consolidación XLSX termina en la goroutine de fondo en decenas/centenas de milisegundos y posteriormente se publica `WM_APP_IMPORT_DONE=0x8001`. El log muestra `MSG[...] antes Translate/Dispatch msg=0x8001` pero no muestra el correspondiente `después Dispatch`; al mismo tiempo continúan los `HEARTBEAT` cada 2 segundos. Esto demuestra que el proceso sigue vivo y que el hilo Win32 queda ocupado dentro del procesamiento del mensaje.

**Hecho verificado en el código:** `WM_APP_IMPORT_DONE` llama directamente a `appFinishImport`. Antes de entrar a la grilla, `appFinishImport` recupera el dataset pendiente, busca el botón `ABRIR EXCEL` mediante `findChildByID`, habilita el control y luego llama a `columnViewSetDatasetSafe`.

**Causa inmediata investigada:** el primer punto a bisectar antes de la grilla era `findChildByID`, porque enumeraba repetidamente hijos mediante `FindWindowExW` sin límite. Una enumeración Win32 defectuosa podía impedir el retorno del handler.

**Corrección aplicada:** `findChildByID` ahora tiene un máximo de 1000 iteraciones y verifica que el handle devuelto avance respecto del anterior. `appFinishImport` registra cada frontera crítica: entrada, Lock/Unlock, `findChildByID`, validación de error/dataset y entrada/salida de `columnViewSetDatasetSafe`.

**Corrección defensiva adicional:** si la importación termina sin error pero `appPendingDataset` es `nil`, `appFinishImport` muestra un error en el estado y retorna sin tocar la grilla, evitando una desreferencia de `ds.Records`.

**Separación entre hecho e inferencia:** el log verifica el bloqueo dentro del procesamiento de `WM_APP_IMPORT_DONE`, pero el log anterior por sí solo no demuestra que `findChildByID` sea definitivamente la causa raíz. La instrumentación y el límite de enumeración se agregaron precisamente para distinguir esa hipótesis de cualquier bloqueo posterior.

**Filtrado/renderizado:** la implementación de `windows_safe_view.go` mantiene el filtrado pesado fuera del hilo UI y programa el llenado de la grilla mediante mensajes `WM_APP_RENDER_BATCH`, en lotes pequeños, para que el hilo de ventana pueda volver al loop de mensajes entre lotes.

### Blindaje de bucles Win32 y render

**Hecho verificado en el código revisado:** `columnViewDeleteColumns` era un `for` sin cota que repetía `SendMessageW(LVM_DELETECOLUMN, índice 0)` hasta que la API devolviera cero. Ese patrón podía dejar atrapado al hilo UI si el control no progresaba correctamente.

**Corrección aplicada:** ahora se obtiene el `HWND` del header con `LVM_GETHEADER` y el número real de columnas con `HDM_GETITEMCOUNT`. Se intenta borrar exactamente esa cantidad, siempre desde el índice 0, con un límite duro adicional de 512 iteraciones. Se registran las columnas detectadas y las efectivamente borradas. Si no se obtiene el header, el borrado se omite en lugar de entrar en un bucle potencialmente infinito.

**Resto de bucles Win32 revisados:** `findChildByID` tiene límite de 1000 y corte si el handle no avanza. Los demás recorridos encontrados en `column_view_windows.go`, `windows_safe_view.go` y la ruta de importación son recorridos sobre slices/mapas o tienen límites derivados de cantidades finitas; no se dejó otro `for` abierto esperando indefinidamente una respuesta de `SendMessageW`/Win32.

**Ruta NO-safe:** `columnViewRefresh` también pasa por la nueva `columnViewDeleteColumns`, por lo que edición de nombres, columnas y cualquier refresco tradicional queda protegido contra el mismo cuelgue.

**Fuente:** `appApplyVisualPolish` usa ahora una altura negativa válida (`-14`) para `CreateFontW`, en lugar de la representación incorrecta `^uint32(8)`.

**Recuperación:** `appFinishImport`, `columnViewSetDatasetSafe` y el render por lotes quedan protegidos con recuperación de pánico y registro mediante `appLog`, para evitar que un fallo inesperado derribe el proceso completo.

### Validación

La modificación de estabilidad debe validarse mediante GitHub Actions con `go test ./...`, `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...`, `go vet ./...` y el workflow de generación del EXE. El éxito de compilación no se considera prueba de resolución del cuelgue de runtime: la confirmación final requiere que el log de ejecución muestre el retorno de `DispatchMessageW` para `0x8001` y, posteriormente, las etapas `import_done` y `render batch`.

## Pruebas agregadas

`core_test.go` cubre:

- números con separadores regionales;
- preservación de columnas dispersas en XLSX;
- aplicación de filtros sobre la cabecera indicada.

Estas pruebas validan la reconstrucción implementada; no constituyen una prueba de equivalencia con el binario original.

## Validación reproducible

En el entorno de trabajo de esta auditoría se ejecutó:

- `go test ./...` → **PASS**.
- `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` → **PASS**.
- `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go vet ./...` → **PASS**.
- `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o GestionSO-V57.exe ./...` → **PASS**; PE32+ Windows x64 GUI generado para verificación local.

El ejecutable local de verificación **no se incorpora al repositorio**.

## Persistencia y XLSX

`BuildLines` usa la fila que maximiza `headerScore` como encabezado y conserva esos nombres en `Line.Values`, con fallback `C<n>` solamente cuando un encabezado está vacío. `mergeXLSX` conserva la lectura de hojas y selecciona de forma determinista la primera hoja por nombre ordenado. Los XLSX originales no se modifican.

## Motor V54 y límite funcional

Aunque `main.feedEngineFile` existe como símbolo real y el binario contiene `GestionSO-V54-engine.exe`, el contrato interno no está verificado sin el motor real. `feedEngineFile` se mantiene parametrizado por `GESTIONSO_V54_ENGINE` y no se inventan argumentos.

Para una prueba end-to-end real se necesitan `GestionSO-V54-engine.exe`, `GestionSO_Datos.csv` y XLSX de datos reales/prueba compatibles. Hasta disponer de ellos, quedan pendientes las fórmulas de negocio reales, subtotales exactos y el flujo completo con V54.

## Binarios

No se commitean `.exe` ni `.zip`. El ejecutable de entrega se publica exclusivamente como artifact de GitHub Actions.
