# GestionSO V57 — reconstrucción

Este repositorio contiene una **reconstrucción/reimplementación** de GestionSO V57 a partir de evidencia observable en el binario `GestionSO-V57.exe` y de los datos de trabajo recuperados. **No es el fuente original** y no se afirma equivalencia interna con el programa original.

## Estado actual

- **Reconstrucción V57:** código fuente Go/Win32 consolidado en una única implementación.
- **Datos base recuperados:** maestro `GestionSO_Datos.csv`, export operativo `archivo exportado.csv` y workbook `Archivo a importar.xlsx`.
- **Arranque funcional:** el ejecutable carga `archivo exportado.csv` automáticamente cuando está junto al ejecutable o dentro de `acceso chatgpt`.
- **ABRIR XLSX:** permite reemplazar la vista inicial cargando el/los XLSX seleccionados.
- **Auditoría 2026-09-03:** se corrigieron inconsistencias de UI, parseo XLSX, filtros por cabecera, ordenamiento, números regionales y generación de subtotales.
- **Corrección 2026-09-03:** reparado el ABI Win64 de `OPENFILENAMEW`, causa concreta identificada para que `GetOpenFileNameW` no abriera correctamente el selector desde `ABRIR XLSX`.
- **Tests:** `go test ./...` verde para las pruebas existentes de parseo XLSX, filtros y números.
- **Build objetivo:** `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` verde en la revisión auditada.
- **go vet:** `go vet ./...` verde en la revisión auditada.
- **Build del ejecutable:** GitHub Actions lo genera en Windows x64 como artifact, sin versionar binarios.

La reconstrucción usa la evidencia recuperada para reproducir la superficie observable del programa. La evidencia verificada y las inferencias están separadas en `docs/EVIDENCIA_BINARIO.md`.

## Datos recuperados utilizados

### `acceso chatgpt/GestionSO_Datos.csv`

Maestro de productos con campos como `SKU`, `DESCRIPCION`, unidades por bulto, precio de lista unitario, costo unitario, bultos por pallet, kg por bulto, estado y base de operación. Es la referencia maestra de productos recuperada.

### `acceso chatgpt/archivo exportado.csv`

Export operativo detallado de órdenes de venta. Contiene `SKU`, descripción, descuentos, cantidades, pallets, `NETO SO`, `TN SO`, `CMG`, `PPP SO`, `ORIGEN`, `SO`, retención, estado, ejecutivo, cliente, factura, documento, provincia, ciudad, ajustes, flete, precio, costo y otros campos de origen. Es la base que ahora alimenta la vista inicial del ejecutable.

### `acceso chatgpt/Archivo a importar.xlsx`

Workbook de entrada recuperado. Se conserva como dato externo del flujo `ABRIR XLSX`; el parser XLSX reconstruido respeta referencias de celda, cadenas compartidas y `inlineStr`.

### `acceso chatgpt/Imagen programa.png`

Captura de referencia visual del programa real. La evidencia visual está transcripta y separada de las inferencias en `docs/EVIDENCIA_BINARIO.md`.

## UI reconstruida — referencia V54

La pantalla de referencia usada es el programa real en modo `SO RETENIDAS`. Se reproduce:

- Título `Gestion SO V54 - SO RETENIDAS / CSV maestro`, derivado de `configData.Mode`. El ejecutable reconstruido sigue siendo V57; la mención V54 es deliberada porque corresponde a la captura de referencia.
- Barra superior: `ABRIR XLSX`, `TOMAR EXCEL ABIERTO`, `RECARGAR`, `COLUMNAS...`, `FILTROS CABECERA...`, `EXPORTAR CSV`, `SIMULADOR`, `RESALTAR...`, `+/- COLOR...`, `DATOS CSV...` y combo de modo.
- Filtros: `SO`, `Estado`, `SKU`, `SUMA DE`, `SDSRP2`, con `FILTRAR` y `LIMPIAR`.
- Grilla real `SysListView32` en modo reporte.
- Columnas, en orden exacto: `SKU`, `Descripción`, `SUM (%) descuento`, `NETO PK`, `UNIDADES`, `PALL`, `PK`, `NETO SO`, `TN SO`, `CMG`, `PPP SO`, `ORIGEN`. `CMG` lleva el indicador visual `▼`.
- Filas informativas `SUBTOTAL SO ...` generadas por grupo de SO.
- Barra de estado con el formato `MODO: ... | RETENIDAS n | LIBERADAS n | SO n | LINEAS n | n filtros | Detalle de Descuentos Aplicados... | CSV`.

Los nombres y elementos anteriores son **hechos verificados por la captura y/o strings/símbolos del binario**. El layout exacto en píxeles, fórmulas de negocio y conteos originales son **inferencias** donde no existe evidencia suficiente.

## Mapeo de datos

`BuildLines` detecta la fila de encabezados mediante `headerScore` y utiliza los nombres reales en `Line.Values`. La grilla fija resuelve cada columna contra esos nombres con alias conservadores. El parser XLSX respeta referencias de celda, por lo que una columna omitida no desplaza las siguientes.

Los filtros de cabecera se aplican al campo indicado y la vista se ordena de forma estable por `SO`, con fallback a `factura` y `cliente`. Los subtotales se agrupan por `SO` y suman campos numéricos reconocibles; esto es una inferencia y no una afirmación sobre las fórmulas comerciales originales.

## Botones pendientes

Los controles cuyo comportamiento interno no es recuperable (`TOMAR EXCEL ABIERTO`, `COLUMNAS...`, `FILTROS CABECERA...`, `SIMULADOR`, `RESALTAR...`, `+/- COLOR...` y `DATOS CSV...`) permanecen como stubs documentados y registran su activación en `%TEMP%\\GestionSO-V57-debug.log`. `TOMAR EXCEL ABIERTO` queda pendiente de validar contra la automatización COM/Excel real.

`RECARGAR` y `EXPORTAR CSV` tienen implementación conservadora dentro de la reconstrucción: actualizan/exportan la vista disponible, sin afirmar que reproduzcan el flujo interno original.

## Paquete de entrega

GitHub Actions genera el ejecutable exclusivamente como artifact; no se commitean `.exe` ni `.zip`.

El ZIP contiene:

- `GestionSO-V57.exe`
- `LEEME.txt`
- `README.md`
- `docs/EVIDENCIA_BINARIO.md`
- `datos/GestionSO_Datos.csv`
- `datos/archivo exportado.csv`
- `datos/Archivo a importar.xlsx`

Requiere Windows x64.

El flujo end-to-end con el motor legado puede requerir `GestionSO-V54-engine.exe`, configurado mediante `GESTIONSO_V54_ENGINE`. El motor no se inventa, no se sustituye y no está incluido porque no forma parte de la evidencia entregada.

## Compilación

```text
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o dist/GestionSO-V57.exe ./...
go vet ./...
go test ./...
```

La validación permanente usa GitHub Actions. El workflow de build usa un runner Windows para producir el PE32+ x64 con subsistema GUI.

## Límites funcionales

Esta es una reconstrucción. La vista inicial ya no queda vacía: usa el `archivo exportado.csv` recuperado. `ABRIR XLSX` permite cargar el workbook recuperado u otros XLSX compatibles.

La automatización `TOMAR EXCEL ABIERTO` y el contrato interno de `GestionSO-V54-engine.exe` siguen pendientes porque no existe evidencia suficiente para reconstruirlos sin inventar comportamiento.

## Trazabilidad

La separación entre evidencia verificada e inferencia está documentada en [`docs/EVIDENCIA_BINARIO.md`](docs/EVIDENCIA_BINARIO.md), incluyendo las correcciones de compilación, la auditoría del `main`, la discrepancia V54/V57 y la integración de los datos recuperados.

Build trigger: ejecución definitiva del paquete Windows x64.
