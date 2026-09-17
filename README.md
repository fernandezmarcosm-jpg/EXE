# GestionSO V57

Aplicación de escritorio Windows para importar uno o varios XLSX, consolidarlos con el maestro `GestionSO_Datos.csv`, mostrar el dataset en una tabla, seleccionar columnas y persistir esa selección entre ejecuciones.

## Stack

- Go 1.23.2
- Wails v2
- Frontend HTML/CSS/TypeScript + Vite
- Parser XLSX y lógica de dataset en Go puro

La reescritura reemplaza la UI Win32/`SysListView32` anterior. La lógica de negocio existente se conserva: lectura XLSX, fechas, unión con CSV maestro, fórmulas, formatos y persistencia de settings.

## Funcionalidad

- `ABRIR EXCEL`: selector nativo para uno o varios `.xlsx`.
- Consolidación de todos los archivos seleccionados mediante `BuildMemoryDataset`.
- Reutilización de IDs lógicos cuando el mismo esquema aparece en varios archivos.
- Duplicados físicos dentro de una misma hoja conservados como `_2`, `_3`, etc.
- Panel `COLUMNAS` con scroll vertical independiente.
- `MARCAR TODAS`, `DESMARCAR TODAS` y `LIMPIAR FILTROS`.
- Filtro de texto por cada columna visible.
- Scroll vertical y horizontal nativo de la tabla.
- Persistencia de columnas visibles por ID lógico.
- Contadores de filas, duplicadas, filas del CSV y enriquecidas.
- `content-visibility` en filas para reducir el coste de renderizado de datasets grandes.

## Desarrollo

Desde la raíz del repositorio:

```text
wails dev
```

El comando genera/actualiza los bindings de Wails y ejecuta el frontend en modo desarrollo.

## Build Windows

```text
wails build
```

El resultado es:

```text
build/bin/GestionSO-V57.exe
```

No se versionan ejecutables ni artefactos de build.

## Tests y validación

```text
go vet ./...
go test ./...
```

El workflow `.github/workflows/build.yml` ejecuta estas validaciones en Windows y después `wails build`, publicando `GestionSO-V57.exe` como artifact.

## Lógica preservada

Los archivos principales de negocio son `core.go`, `dataset.go`, `xlsx_dates.go`, `xlsx_xml_fix.go` y `core_helpers_restored.go`. `xlsx_xml_fix.go` es ahora multiplataforma porque no requiere ninguna API de Windows.

La identidad de las columnas sigue separando ID lógico y título visible. `BuildMemoryDataset(docs, settings)` mantiene la reutilización entre archivos y solo genera sufijos para duplicados físicos dentro de una misma hoja.

## Seguimiento

El plan, los milestones y los criterios de aceptación están en `docs/REESCRITURA.md`. El merge a `main` queda condicionado a CI verde y validación física del EXE con el caso real de varios XLSX.
