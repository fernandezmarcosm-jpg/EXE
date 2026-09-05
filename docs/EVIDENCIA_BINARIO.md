# Evidencia del binario — GestionSO V57

## Alcance

Trazabilidad de la reconstrucción. Se separan hechos verificados de inferencias. La referencia visual disponible corresponde a V54; el ejecutable reconstruido es V57.

## Hechos verificados

- El binario analizado es PE32+, x86-64, Windows GUI y no es .NET.
- El binario contiene APIs Win32 y símbolos de lectura XLSX, consolidación y vista.
- El binario contiene la referencia `GestionSO-V54-engine.exe`, pero ese motor y datos oficiales de prueba no están disponibles en este repositorio.
- La reconstrucción usa `SysListView32` y conserva el enfoque Win32 directo.

## Correcciones de esta revisión — 2026-09-05

### 1. Compilación / `appRecover`

**Verificado:** `crashlog_windows.go` tiene una sola definición `func appRecover(where string)`. `functional_app_windows.go` conserva solo el comentario y todos los call sites revisados pasan contexto. La búsqueda de `appRecover()` sin argumentos devolvió 0 coincidencias.

### 2. Dataset desconectado

**Verificado:** `functional_app_windows.go`, `appFinishImport` asigna `appImportedDataset` y `appImportedPaths` antes de renderizar. `appApplySettings` recalcula y refresca el dataset ya cargado, sin reinicio.

### 3. Grilla usable

**Verificado:** `windows_safe_view.go`, `columnViewRenderBatch` renderiza por lotes y ejecuta `columnViewAutoFit` al terminar. `column_view_windows.go`, `columnViewRefresh` asigna `SubItem` explícitamente para cada celda.

### 4. Filtros

**Verificado:** `column_view_windows.go`, `columnViewLayoutFilters` usa `LVM_GETCOLUMNWIDTH`; `columnViewBuildFilters` destruye los EDIT anteriores antes de reconstruirlos.

### 5. Columnas repetidas

**Verificado:** `dataset.go`, `BuildMemoryDataset` y `columnViewVisibleColumns` deduplican por `datasetColumnKey` (`Source|Title`).

### 6. Fondos negros

**Verificado:** `functional_app_windows.go`, `crearVentana` usa `COLOR_WINDOW+1` y `appWndProc` maneja `WM_ERASEBKGND`/`WM_CTLCOLORSTATIC`. `datasetShowConfig`, `configWndProc` y los diálogos de edición aplican el mismo fondo de sistema.

### 7. Fechas seriales XLSX

**Verificado:** `xlsx_dates.go` abre `xl/styles.xml`, lee `cellXfs`/`numFmtId`, detecta formatos de fecha y convierte seriales con base 1899-12-30. `functional_app_windows.go` llama `decorateXLSXDates` después de `ReadXLSX` y antes de construir el dataset.

### 8. Doble clic en título

**Verificado:** `column_view_windows.go`, `columnViewHandleNotify` procesa `HDN_ITEMDBLCLICKW`. `columnViewBeginColumnEdit` permite nombre, tipo, decimales y negativos; `columnEditWndProc` persiste mediante `saveDatasetSettings`.

### 9. Tipo natural / preferencia

**Verificado:** `dataset.go` conserva el tipo de cada columna y aplica `ColumnTypes` persistido. El diálogo puede sobrescribirlo.

### 10. Edición de celda

**Verificado:** `columnViewHandleNotify` procesa `NM_DBLCLK`; `columnViewBeginCellEdit` abre el EDIT y `parseEditedValue` valida según texto/número/fecha. El cambio queda en `MemoryDataset` y no modifica el XLSX original.

### 11. Orden por arrastre

**Verificado:** `columnViewCreate` mantiene `LVS_EX_HEADERDRAGDROP`; `columnViewHandleNotify` procesa `HDN_ENDDRAG`; `columnViewSaveOrder` persiste `ColumnOrder` mediante `LVM_GETCOLUMNORDERARRAY`.

### 12. Nombres

**Verificado:** `columnViewNames` lista las columnas XLSX/CSV y `namesWndProc` persiste `ColumnTitles` con `saveDatasetSettings`.

### 13. Fórmulas + CSV

**Verificado:** `dataset.go` incorpora todos los encabezados del CSV al dataset, `evaluateFormula` acepta `[COLUMNA]` y `[CSV:COLUMNA]`, con `+ - * /` y paréntesis. `applyDatasetFormula` crea/actualiza la columna calculada y recalcula valores.

### 14. Selector de columnas

**Verificado:** `datasetShowConfig` agrega un ComboBox con las columnas disponibles y `configWndProc` inserta el token seleccionado en la fórmula.

### 15. Persistencia

**Verificado:** `dataset.go`, `datasetSettingsPath` usa `os.UserConfigDir()`. En Windows la ruta efectiva es `%LOCALAPPDATA%\\GestionSO V57\\dataset.txt`. Se persisten títulos, orden, visibilidad, tipos, decimales, porcentajes, negativos, fórmulas, columnas calculadas, subtotales y fuente.

### 16. Fuente

**Verificado:** `DatasetSettings.FontSize` se carga/persiste; `columnViewApplyFont` aplica `Segoe UI` a la grilla y `appApplyVisualPolish` aplica el tamaño configurado a controles principales.

## Sintaxis de fórmula de la reconstrucción

- `[COLUMNA]`: referencia por título a una columna XLSX.
- `[CSV:COLUMNA]`: referencia explícita a una columna del CSV maestro.
- Operadores: `+`, `-`, `*`, `/`, paréntesis.

Esto documenta el parser de la reconstrucción; **no se afirma que sea la sintaxis original del motor V54**.

## CI / validación

La validación se ejecuta en Windows con `CGO_ENABLED=0`, `GOOS=windows`, `GOARCH=amd64`.

La primera corrida de esta revisión detectó errores de sintaxis en `column_view_windows.go`. La siguiente corrida detectó y corrigió: overflow de constante para `-1`, conversiones `int32/int` y ausencia de `xlsx_dates.go` en la rama. **La rama no se debe fusionar hasta que una corrida posterior confirme simultáneamente Test, Vet y Build en verde.**

## Inferencias y límites

- La fórmula exacta de subtotales de V54 no está recuperada; cualquier cálculo reconstruido es inferencia.
- El layout de diálogos y medidas es una implementación funcional, no una afirmación pixel-perfect.
- La detección de fecha por `styles.xml` sigue la semántica estándar de Excel; no prueba que sea el algoritmo interno del V54.
- No puede verificarse el flujo end-to-end real con `GestionSO-V54-engine.exe` porque faltan el ejecutable y los datos oficiales de prueba.

## Regla de honestidad

No se presenta como recuperada ninguna lógica de negocio que no esté demostrada por el binario, código existente o datos oficiales.

## Binarios

No se commitean `.exe` ni `.zip`. El ejecutable de entrega se publica exclusivamente como artifact de GitHub Actions.
