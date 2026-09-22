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

## Cruce multi-base CSV físico — 2026-09-21

La aplicación admite múltiples tablas CSV auxiliares físicas. Cada archivo .csv adicional encontrado junto al ejecutable (con fallback al directorio de trabajo) se interpreta como una base independiente: la primera columna es la clave y las columnas restantes son atributos. El encabezado de esa primera columna se compara contra el encabezado físico de una columna del XLSX importado; el alias configurado en COLUMNAS no participa del cruce. Se toleran variantes como Nº CLIENTE, N° CLIENTE, N CLIENTE, NRO CLIENTE y CLIENTE.

Las claves se normalizan en ambos lados. Para valores numéricos se eliminan espacios y separadores de miles y se descartan decimales cero (80003285, 80003285.00 y 80003285,00 producen la misma clave; también se contemplan formatos es-AR como 23.961,00). Los atributos se agregan con IDs estables LOOKUP:<BASE>:<ATRIBUTO>, inicialmente ocultos y administrables desde COLUMNAS. GestionSO_Datos.csv conserva su esquema CSV:* y su fallback embebido.

### Convención definitiva de archivos lookup auxiliares

- Los CSV auxiliares deben colocarse junto al `.exe`; si no están allí, también se busca en el directorio de trabajo (cwd).
- No deben llamarse `GestionSO_Datos.csv`; ese nombre queda reservado para la base maestra `CSV:*` existente.
- En cada CSV auxiliar, la **primera columna es la clave** y las columnas siguientes son atributos que pueden incorporarse al dataset.
- El encabezado de la primera columna debe coincidir con el **header físico** de una columna del XLSX importado. El alias configurado en `COLUMNAS` no participa del cruce. Se toleran las variantes de numeración de cliente documentadas por los tests, como `N CLIENTE`, `NRO CLIENTE` y `Nº CLIENTE`.
- Las claves se normalizan en ambos lados, incluyendo valores numéricos con decimales cero y separadores de miles.
- Las columnas generadas tienen IDs `LOOKUP:<BASE>:<ATRIBUTO>`, se crean **ocultas** inicialmente y deben activarse desde el panel `COLUMNAS` para mostrarlas en la grilla.

### Fix directo de valores y diagnóstico

- El cruce real está implementado directamente en `dataset.go`; no depende de scripts que modifiquen el código durante CI.
- La coincidencia de encabezados prioriza primero `normalizeHeader` exacto y solo después las variantes equivalentes (`Nº CLIENTE`, `N CLIENTE`, `NRO CLIENTE`, etc.).
- `normalizeJoinKey` canonicaliza claves numéricas para que `80003285`, `80003285.00`, `80003285,00` y `80.003.285` puedan coincidir sin alterar claves alfanuméricas.
- El diagnóstico `[LOOKUP-DBG]` registra una vez por base el encabezado de la base, la columna XLSX seleccionada, algunas claves normalizadas del CSV y el primer valor crudo/normalizado observado en el XLSX.
- `lookup_value_test.go` cubre el caso `Nº CLIENTE;ATRIBUTO` con `80003285.00` y el caso con separadores de miles.


## Logging a disco eliminado — 2026-09-22

La aplicación ya no crea ni redirige el logger estándar hacia `GestionSO_log.txt` ni hacia otro archivo de diagnóstico. El diagnóstico de lookup que permanezca usa únicamente el logger estándar (stderr) y no se escribe a disco. La barra de estado conserva los avisos de `matched_columns`, `enriched_rows` y errores de interpretación de fechas sin mostrar ninguna ruta de log.

## Normalización canónica central de valores — 2026-09-22

La política de datos establece que MemoryValue.Raw es el valor canónico de trabajo y no una copia de la representación visual de Excel/CSV. La normalización ocurre al crear el MemoryValue mediante makeMemoryValue; cuando un valor es numérico y Number es válido, Raw se guarda con strconv.FormatFloat(..., 'f', -1, 64). Así se eliminan notación científica, separadores regionales y ceros sobrantes. Los valores no numéricos conservan su texto recortado.

- **Número / entero / moneda / porcentaje / decimal:** Type=ValueNumber, Number contiene el float64 y Raw contiene el número canónico sin formato de presentación. Moneda, porcentaje, separador de miles y cantidad de decimales se aplican únicamente en datasetValueText; no modifican Raw.
- **Fecha:** Type=ValueDate y Raw se guarda en ISO (2006-01-02 o 2006-01-02 15:04:05). Las fechas detectadas por el estilo de Excel se canonizan en decorateXLSXDates; una columna configurada como fecha también pasa por canonicalizeMemoryValue. formatDatasetDate convierte ese valor interno a la presentación visible.
- **Texto:** Type=ValueText y Raw conserva el contenido textual recortado; no se aplican conversiones numéricas.
- **LOOKUP / join:** las claves XLSX y CSV pasan por normalizeJoinKey, con fallback strconv.ParseFloat para notación científica. Por lo tanto 8.0003285E7, 80003285, 80003285.00 y 80.003.285 convergen en la misma clave cuando representan el mismo entero.
- **Filtros y subtotales:** consumen MemoryValue.Number para operaciones numéricas y Raw para identidad/texto; el formateo visible queda separado de la representación canónica. Esto evita que una diferencia de formato visual vuelva a alterar un cruce, filtro o cálculo.

Los tests cubren notación científica, separadores ,/. , miles, fechas, porcentajes y el enriquecimiento end-to-end de un valor XLSX científico contra una clave CSV convencional.


## Cruce por rango de fechas en CSV auxiliares — 2026-09-22
La convención fue restaurada después de la pérdida accidental de esta lógica en el commit `57af8324`: el motor actual conserva el cruce exacto y agrega el modo por vigencia sin reintroducir logging físico.


Los CSV auxiliares también pueden representar rangos de vigencia. Se detectan cuando las dos primeras columnas se llaman **DESDE/HASTA** o **FECHA DESDE/FECHA HASTA**. La **tercera columna es el header físico de la fecha del XLSX** contra la que se evalúa el rango; las columnas desde la cuarta en adelante son atributos enriquecidos.

Las fechas de DESDE/HASTA y de la fila XLSX aceptan ISO (yyyy-mm-dd, con hora opcional), dd/mm/aaaa, variantes equivalentes con / o -, y serial numérico de Excel. El intervalo es inclusivo en ambos extremos. Si existen rangos superpuestos, se utiliza el primero después de ordenar por fecha DESDE ascendente.

Ejemplo: DESDE;HASTA;FECHA;EJERCICIO con 01/01/2026;31/12/2026;FECHA;Ejercicio 2026 cruza contra la columna física FECHA del XLSX. La base conserva el modo exacto anterior cuando no tiene encabezados DESDE/HASTA.

## Ventana REPORTE

La barra principal incorpora **REPORTE**, una ventana tipo tabla dinámica sobre el dataset ya enriquecido. El reporte reutiliza las filas que quedan luego de los filtros de texto, filtros por valores y criterios activos de la grilla.

- **FILAS (niveles):** permite seleccionar uno o varios campos y su orden, por ejemplo Cadena → Año → Mes. Están disponibles las columnas XLSX, CSV/LOOKUP, calculadas y cualquier columna derivada de período presente en el dataset.
- **MEDIDA:** selector editable de cualquier columna numérica disponible, incluidos los campos calculados y campos de volumen/kg/toneladas cuando existan.
- **OPERACIÓN:** Suma, Promedio, Conteo y Promedio ponderado.
- **PROMEDIO PONDERADO:** requiere una segunda columna numérica como **COLUMNA PESO** y calcula suma(medida × peso) / suma(peso).
- **RESULTADO:** muestra las combinaciones multinivel seleccionadas, la medida agregada y la cantidad de filas de cada grupo, con **TOTAL GENERAL** al pie.

El cálculo del primer entregable se realiza en frontend para reutilizar directamente filteredRows() y mantener consistencia con los filtros activos de la grilla. La lógica de pivot y agregación está en frontend/src/report_pivot.ts y cuenta con prueba unitaria ejecutada por el workflow.

La medida y la columna peso no quedan fijadas a una columna física: ambas son configurables desde la ventana REPORTE. No se asume una columna por defecto de volumen/kg/toneladas; se utiliza la primera columna numérica disponible si el usuario todavía no eligió otra.
