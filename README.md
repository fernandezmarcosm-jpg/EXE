# GestionSO V57 — reconstrucción

Este repositorio contiene una **reconstrucción/reimplementación** de GestionSO V57 a partir de evidencia observable en el binario `GestionSO-V57.exe` disponible durante el análisis. **No es el código fuente original** y no debe presentarse como tal.

## Estado

- **Entrega 1:** reconstrucción de `main_windows.go`, centrada en Win32, botón `ABRIR XLSX`, selector múltiple y trazabilidad del hook.
- **Entrega 2:** `core.go` agregado por bloques: tipos/logging defensivo, lectura y merge XLSX, configuración/persistencia CSV, vistas/proceso y stubs explícitos para opciones/simulador.

El binario analizado es Go 1.23.2, Windows x86-64, `CGO_ENABLED=0`, buildmode `exe`. El símbolo `main.feedEngineFile` existe realmente en el binario; su contrato interno no puede recuperarse literalmente sin el motor V54.

## Compilación

Objetivo de compilación:

```text
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o GestionSO-V57-reconstructed.exe .
```

`core.go` fue verificado de forma aislada con ese objetivo de plataforma. La compilación completa del proyecto debe confirmarse en Windows/Go equivalente antes de una entrega funcional.

## XLSX

La reconstrucción implementa lectura mediante `archive/zip` + `encoding/xml`, incluyendo shared strings, filas y selección de la hoja con mejor puntuación de encabezado. La operación de merge no modifica los XLSX originales.

El detalle exacto del formato de salida y del contrato con el motor V54 sigue siendo una inferencia mientras no esté disponible el ejecutable V54 original.

## Motor V54

El binario contiene la cadena `GestionSO-V54-engine.exe`, pero ese ejecutable no estaba incluido en el ZIP analizado. Por ello el flujo end-to-end con V54 no está validado. `feedEngineFile` queda parametrizado mediante `GESTIONSO_V54_ENGINE` y registra la situación en `%TEMP%\\GestionSO-V57-debug.log`; no se inventa un contrato de argumentos que no pueda verificarse.

## Datos

`GestionSO_Datos.csv` no forma parte del ZIP original según `LEEME-GestionSO-V57.txt`. No se incluye en este repositorio.

## Trazabilidad

La evidencia de símbolos, strings, APIs Win32 y limitaciones está en [`docs/EVIDENCIA_BINARIO.md`](docs/EVIDENCIA_BINARIO.md).

No se incluyen binarios, ZIP originales, logs ni datos de runtime.
