# Reescritura de UI a Go + Wails

## Objetivo

Reemplazar la capa Win32/`SysListView32` de GestionSO V57 por una UI Wails (HTML/CSS/TypeScript) sin cambiar la lógica de negocio de importación, consolidación, enriquecimiento, fórmulas ni persistencia.

## Rama de trabajo

- Base: `main` al iniciar la reescritura.
- Rama de trabajo: `rewrite-wails`.
- PR: #7.
- Merge final a `main`: `297b00999e62e424c745b493c7f81a2bfa705f4d`.

## Estado final

La reescritura fue completada y mergeada a `main`.

### Fase 0 — Preparación

- [x] Crear `rewrite-wails` desde `main`.
- [x] Crear este documento como punto de seguimiento.
- [x] Mantener `main` intacta durante la implementación de la nueva UI.

### Fase 1 — Lógica de negocio sin Win32

- [x] Conservar `core.go`, `dataset.go`, `xlsx_dates.go` y `xlsx_xml_fix.go`.
- [x] Quitar el build tag Windows de `xlsx_xml_fix.go`.
- [x] Mantener `BuildMemoryDataset(docs, settings)` y la reutilización de IDs entre archivos.
- [x] Mover helpers de datos usados por tests a archivos independientes de Windows.
- [x] Trasladar `TestDuplicatedPhysicalColumns` a la suite multiplataforma.
- [x] Retirar la capa Win32 de la aplicación.

### Fase 2 — Backend Wails

- [x] Inicializar Wails v2.
- [x] Crear `App` con `ImportXLSX`, `GetSettings`, `SaveSettings` y `SetVisibleColumns`.
- [x] Usar selector nativo multiarchivo para `*.xlsx`.
- [x] Exponer `DatasetDTO` con columnas, filas y contadores.
- [x] Mantener el procesamiento en las funciones existentes de `core.go`/`dataset.go`.

### Fase 3 — Frontend

- [x] Botón `ABRIR EXCEL`.
- [x] Panel `COLUMNAS` sin menú modal Win32.
- [x] Scroll vertical propio del panel de columnas.
- [x] Marcar/desmarcar columnas y persistir inmediatamente.
- [x] `MARCAR TODAS`, `DESMARCAR TODAS` y `LIMPIAR FILTROS`.
- [x] Scroll vertical/horizontal nativo de la tabla.
- [x] Filtros por columna.
- [x] `content-visibility` para reducir el coste de renderizado de tablas grandes.
- [x] Sin columnas fantasma al dejar todas ocultas.

### Fase 4 — CI, build y limpieza

- [x] Workflow Windows para `go vet`, `go test` y `wails build`.
- [x] CI verde en `rewrite-wails`.
- [x] Artifact `GestionSO-V57.exe` generado correctamente.
- [x] Sin binarios commiteados.
- [x] Actualizar README con el nuevo stack y comandos.
- [x] Revisión final y merge a `main`.
- [ ] Validación física en la PC Windows con el caso real de varios XLSX.

## Validación CI

Última ejecución verde del workflow Wails:

- `go vet ./...`: PASS.
- `go test ./...`: PASS.
- `wails build`: PASS.
- Artifact `GestionSO-V57-Windows`: generado correctamente.

## Archivos conservados

- `core.go`: lector XLSX y estructuras de memoria.
- `dataset.go`: dataset, join con CSV, fórmulas y settings.
- `xlsx_dates.go`: fechas seriales de Excel.
- `xlsx_xml_fix.go`: parser XML de worksheets.
- `core_helpers_restored.go`: helpers de datos existentes.
- Tests de lógica de negocio existentes.

## Archivos retirados de la UI Win32

Se elimina la familia de archivos de ventanas, controles, `SysListView32`, `user32`, diálogos Win32 y message loop que formaban la UI anterior. La nueva entrada es `main.go` + `app.go` y el frontend Wails.

## Milestone de aceptación

La validación automática confirmó compilación y tests. Queda como último paso manual ejecutar el EXE en Windows y comprobar con el caso real:

1. importar uno o varios XLSX;
2. conservar todas las filas esperadas;
3. apilar esquemas iguales entre archivos sin crear `_2`/`_3` espurios;
4. conservar duplicados físicos reales con IDs independientes;
5. ocultar todas las columnas sin headers fantasma;
6. mostrar datos de cada columna en todas las filas;
7. guardar y recuperar la selección de columnas entre ejecuciones.


## Evolución posterior — Subtotales y presentación

- [x] Subtotal por conteo de valores únicos (`conteo_unico`), incluyendo columnas de texto y TOTAL GENERAL.
- [x] Control de frontend para seleccionar `Contador (únicos)` en cualquier columna distinta de la agrupación.

- [x] Piso de ancho de columna reducido a 20 px en backend y frontend.
- [x] Alineación por columna persistente (`left`/`center`/`right`) con default izquierdo para texto y derecho para numéricas/calculadas.

- [x] Alias de título por columna persistente, sin modificar IDs ni claves internas; alias vacío restaura el título físico.
