# Reescritura de UI a Go + Wails

## Objetivo

Reemplazar la capa Win32/`SysListView32` de GestionSO V57 por una UI Wails (HTML/CSS/TypeScript) sin cambiar la lógica de negocio de importación, consolidación, enriquecimiento, fórmulas ni persistencia.

## Rama de trabajo

- Base: `main` al iniciar la reescritura.
- Rama: `rewrite-wails`.
- `main` no se modifica durante esta fase de trabajo.

## Estado

### Fase 0 — Preparación

- [x] Crear `rewrite-wails` desde `main`.
- [x] Crear este documento como punto de seguimiento.
- [x] Mantener la rama `main` intacta durante la reescritura.

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
- [x] Artifact `GestionSO-V57.exe`.
- [x] Sin binarios commiteados.
- [x] Actualizar README con el nuevo stack y comandos.
- [ ] Ejecutar CI verde en `rewrite-wails`.
- [ ] Validar el EXE físicamente en Windows con el caso real de varios XLSX.
- [ ] Revisión final y merge a `main`.

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

No se debe hacer merge a `main` hasta que el workflow de `rewrite-wails` sea verde y se haya comprobado que el EXE:

1. importa uno o varios XLSX;
2. conserva todas las filas esperadas;
3. apila esquemas iguales entre archivos sin crear `_2`/`_3` espurios;
4. conserva duplicados físicos reales con IDs independientes;
5. permite ocultar todas las columnas sin headers fantasma;
6. muestra datos de cada columna en todas las filas;
7. guarda y recupera la selección de columnas entre ejecuciones.
