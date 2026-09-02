# GestionSO V57 — reconstrucción

Este repositorio contiene una **reconstrucción/reimplementación** de GestionSO V57 a partir de evidencia observable en el binario `GestionSO-V57.exe` disponible durante el análisis. **No es el código fuente original** y no debe presentarse como tal.

## Estado

- **Entrega 1:** reconstrucción Win32, botón `ABRIR XLSX`, selector múltiple y trazabilidad del hook.
- **Entrega 2:** núcleo XLSX, configuración/persistencia CSV, vistas/proceso y stubs explícitos.
- **Fidelidad UI:** pantalla de referencia real **V54 / SO RETENIDAS** replicada en una única implementación Win32: barra superior, filtros, grilla `SysListView32`, columnas fijas, subtotales informativos y barra de estado.
- **Build integrado:** verde con Go 1.23.2, `CGO_ENABLED=0`, `GOOS=windows`, `GOARCH=amd64`.
- **go vet:** verde en el workflow de validación.
- **Build del ejecutable:** verde en Windows x64; artifact `GestionSO-V57` generado correctamente.

El código fuente actual de `main` está consolidado en una única implementación coherente. El binario analizado es Go 1.23.2, Windows x86-64. El símbolo `main.feedEngineFile` existe realmente; su contrato interno no puede recuperarse sin el motor V54.

## UI reconstruida — referencia V54

La pantalla de referencia usada es el programa real en modo `SO RETENIDAS`. Se reproduce:

- Título `Gestion SO V54 - SO RETENIDAS / CSV maestro`, derivado de `configData.Mode`. El ejecutable reconstruido sigue siendo V57; la mención V54 es deliberada porque corresponde a la captura de referencia.
- Barra superior: `ABRIR XLSX`, `TOMAR EXCEL ABIERTO`, `RECARGAR`, `COLUMNAS...`, `FILTROS CABECERA...`, `EXPORTAR CSV`, `SIMULADOR`, `RESALTAR...`, `+/- COLOR...`, `DATOS CSV...` y combo de modo.
- Filtros: `SO`, `Estado`, `SKU`, `SUMA DE`, `SDSRP2`, con `FILTRAR` y `LIMPIAR`.
- Grilla real `SysListView32` en modo reporte.
- Columnas, en orden exacto: `SKU`, `Descripción`, `SUM (%) descuento`, `NETO PK`, `UNIDADES`, `PALL`, `PK`, `NETO SO`, `TN SO`, `CMG`, `PPP SO`, `ORIGEN`. `CMG` lleva el indicador visual `▼`.
- Filas informativas `SUBTOTAL SO ...` generadas por grupo de SO.
- Barra de estado con el formato `MODO: ... | RETENIDAS n | LIBERADAS n | SO n | LINEAS n | n filtros | Detalle de Descuentos Aplicados... | CSV`.

Los nombres y elementos anteriores son **hechos verificados por la captura y/o strings/símbolos del binario**. El layout exacto en píxeles, fórmulas de negocio y conteos originales son **inferencias conservadoras**.

## Mapeo de datos

`BuildLines` detecta la fila de encabezados mediante `headerScore` y utiliza los nombres reales en `Line.Values`. La grilla fija resuelve cada columna contra esos nombres con alias conservadores; si no existe el campo, muestra vacío. Los filtros se aplican por nombre real mediante `BuildFilteredSortedViewByHeaders`.

Los subtotales se agrupan por `SO` y suman únicamente campos numéricos reconocibles. Esto permite una representación visual útil, pero **no afirma reproducir la fórmula interna de V54**.

## Botones pendientes

Los controles cuyo comportamiento interno no es recuperable (`TOMAR EXCEL ABIERTO`, `RECARGAR`, `COLUMNAS...`, `FILTROS CABECERA...`, `EXPORTAR CSV`, `SIMULADOR`, `RESALTAR...`, `+/- COLOR...`, `DATOS CSV...`) están visibles y registran stubs en `%TEMP%\\GestionSO-V57-debug.log`. `TOMAR EXCEL ABIERTO` queda pendiente de validar porque solo se observa la referencia COM/Excel, no su protocolo completo.

## Descarga del ejecutable

El ejecutable se genera **exclusivamente como artifact de GitHub Actions**; no se commitean `.exe` ni `.zip`.

- Workflow: https://github.com/fernandezmarcosm-jpg/EXE/actions/workflows/build-exe.yml
- Run final de compilación: https://github.com/fernandezmarcosm-jpg/EXE/actions/runs/33695362962
- Artifact: **`GestionSO-V57`**, ID `9871584855`.
- SHA-256 del ZIP del artifact: `39aa70c2415ecb32d9042460b6da661df2e8c7ccfc4209a1684b047ae372731b`.
- El ejecutable extraído tiene 3.302.912 bytes y es PE32+ Windows x86-64 GUI.
- Para descargarlo: abrir el run y, al final de la página, entrar en **Artifacts → GestionSO-V57**.

## Compilación

```text
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o dist/GestionSO-V57.exe ./...
go vet ./...
```

El workflow de build usa un runner Windows y define `CGO_ENABLED`, `GOOS` y `GOARCH` a nivel de job porque el runner ejecuta los pasos con PowerShell. El workflow permanente de validación usa el mismo objetivo Windows x64.

## Límites funcionales

Esta es una reconstrucción. Para validar end-to-end el flujo `ABRIR XLSX` → motor se requieren `GestionSO-V54-engine.exe`, `GestionSO_Datos.csv` y XLSX de datos reales/de prueba compatibles. Esos materiales no están disponibles. Por lo tanto quedan pendientes las **fórmulas de negocio reales, subtotales exactos y compatibilidad funcional completa con V54**.

`feedEngineFile` no se modifica más allá de la parametrización existente mediante `GESTIONSO_V54_ENGINE` y logging.

## Trazabilidad

La separación entre evidencia verificada e inferencia está documentada en [`docs/EVIDENCIA_BINARIO.md`](docs/EVIDENCIA_BINARIO.md), incluyendo las correcciones de compilación y la discrepancia V54/V57.
