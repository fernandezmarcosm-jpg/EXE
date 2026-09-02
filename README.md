# GestionSO V57 — reconstrucción

Este repositorio contiene una **reconstrucción/reimplementación** de GestionSO V57 a partir de evidencia observable en el binario `GestionSO-V57.exe` disponible durante el análisis. **No es el código fuente original** y no debe presentarse como tal.

## Estado

- **Entrega 1:** reconstrucción Win32, botón `ABRIR XLSX`, selector múltiple y trazabilidad del hook.
- **Entrega 2:** núcleo XLSX, configuración/persistencia CSV, vistas/proceso y stubs explícitos.
- **Fidelidad UI:** la ventana ahora replica visualmente la pantalla de referencia real de **V54 / SO RETENIDAS**: barra superior, filtros de cabecera, grilla `SysListView32`, columnas fijas, subtotales por SO y barra de estado.
- **Build/vet:** automatizados con Go 1.23.2, `CGO_ENABLED=0`, `GOOS=windows`, `GOARCH=amd64`.

El binario analizado es Go 1.23.2, Windows x86-64. El símbolo `main.feedEngineFile` existe realmente; su contrato interno no puede recuperarse sin el motor V54.

## UI reconstruida — referencia V54

La pantalla de referencia usada es el programa real en modo `SO RETENIDAS`. Se reproduce:

- Título `Gestion SO V54 - SO RETENIDAS / CSV maestro`, derivado de `configData.Mode`. El ejecutable reconstruido sigue siendo V57; la mención V54 es deliberada porque corresponde a la captura de referencia.
- Barra superior: `ABRIR XLSX`, `TOMAR EXCEL ABIERTO`, `RECARGAR`, `COLUMNAS...`, `FILTROS CABECERA...`, `EXPORTAR CSV`, `SIMULADOR`, `RESALTAR...`, `+/- COLOR...`, `DATOS CSV...` y combo `10`.
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
- Artifact: **`GestionSO-V57`**.
- El ZIP contiene `GestionSO-V57.exe`, `README.md`, `EVIDENCIA_BINARIO.md` y `LEEME.txt`.

## Compilación

```text
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o dist/GestionSO-V57.exe ./...
```

También se ejecuta `go vet ./...`. Los workflows permanentes son `.github/workflows/validar-go.yml` y `.github/workflows/build-exe.yml`.

## Límites funcionales

Esta es una reconstrucción. Para validar end-to-end el flujo `ABRIR XLSX` → motor se requieren `GestionSO-V54-engine.exe`, `GestionSO_Datos.csv` y XLSX reales/de prueba compatibles. Esos materiales no están disponibles. Por lo tanto quedan pendientes las **fórmulas de negocio reales, subtotales exactos y compatibilidad funcional completa con V54**.

`feedEngineFile` no se modifica más allá de la parametrización existente mediante `GESTIONSO_V54_ENGINE` y logging.

## Trazabilidad

La separación entre evidencia verificada e inferencia está documentada en [`docs/EVIDENCIA_BINARIO.md`](docs/EVIDENCIA_BINARIO.md).
