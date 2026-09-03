# GestionSO V57 — reconstrucción

Este repositorio contiene una **reconstrucción/reimplementación** de GestionSO V57 a partir de evidencia observable en el binario `GestionSO-V57.exe` disponible durante el análisis. **No es el fuente original** y no se afirma equivalencia interna con el programa original.

## Estado

- **Reconstrucción V57:** código fuente Go/Win32 consolidado en una única implementación.
- **Auditoría 2026-09-03:** se corrigieron inconsistencias de UI, parseo XLSX, filtros por cabecera, ordenamiento, números regionales y generación de subtotales.
- **Corrección 2026-09-03:** reparado el ABI Win64 de `OPENFILENAMEW`, causa concreta identificada para que `GetOpenFileNameW` no abriera correctamente el selector desde `ABRIR XLSX`.
- **Tests:** `go test ./...` verde para las pruebas de parseo XLSX, filtros y números.
- **Build objetivo:** `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` verde en la revisión auditada.
- **go vet:** `go vet ./...` verde en la revisión auditada.
- **Build del ejecutable:** GitHub Actions lo genera en Windows x64 como artifact, sin versionar binarios.

El código fuente actual es una reconstrucción/reimplementación, no el fuente original del binario. La evidencia verificada y las inferencias están separadas en `docs/EVIDENCIA_BINARIO.md`.

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

## Descarga del ejecutable

El ejecutable se genera exclusivamente como artifact de GitHub Actions; no se commitean `.exe` ni `.zip`.

- Run que generó el artifact actual: https://github.com/fernandezmarcosm-jpg/EXE/actions/runs/33764014134
- Artifact: `GestionSO-V57` (artifact ID `9896720748`), generado desde el commit `1cb44122046e6a4f65ac4d2633ce67319589e4fc`.
- Descarga: abrir el run y entrar en **Artifacts → GestionSO-V57**.
- El ZIP incluye `GestionSO-V57.exe`, `README.md`, `docs/EVIDENCIA_BINARIO.md` y `LEEME.txt`.
- Requiere Windows x64.
- El flujo end-to-end puede requerir `GestionSO-V54-engine.exe`, configurado mediante `GESTIONSO_V54_ENGINE`; el motor no está incluido.
- `GestionSO_Datos.csv` y los XLSX son datos externos y no se incluyen.

## Compilación

```text
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o dist/GestionSO-V57.exe ./...
go vet ./...
go test ./...
```

La validación permanente usa GitHub Actions. El workflow de build usa un runner Windows para producir el PE32+ x64 con subsistema GUI.

## Límites funcionales

Esta es una reconstrucción. Para validar end-to-end el flujo `ABRIR XLSX` → motor se requieren `GestionSO-V54-engine.exe`, `GestionSO_Datos.csv` y XLSX de datos reales/de prueba compatibles. Esos componentes externos no están incluidos.

`feedEngineFile` no se modifica más allá de la parametrización existente mediante `GESTIONSO_V54_ENGINE` y logging.

## Trazabilidad

La separación entre evidencia verificada e inferencia está documentada en [`docs/EVIDENCIA_BINARIO.md`](docs/EVIDENCIA_BINARIO.md), incluyendo las correcciones de compilación, la auditoría del `main` y la discrepancia V54/V57.
