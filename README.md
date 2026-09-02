# GestionSO V57 — reconstrucción

Este repositorio contiene una **reconstrucción/reimplementación** de GestionSO V57 a partir de evidencia observable en el binario `GestionSO-V57.exe` disponible durante el análisis. **No es el código fuente original** y no debe presentarse como tal.

## Estado

Entrega 1: reconstrucción de `main_windows.go`, centrada en Win32, botón `ABRIR XLSX`, selector múltiple y trazabilidad del hook.

El binario analizado es Go 1.23.2, Windows x86-64, `CGO_ENABLED=0`, buildmode `exe`. El símbolo `main.feedEngineFile` existe realmente en el binario; su implementación interna no puede recuperarse literalmente.

## Compilación

```text
go build -trimpath -o GestionSO-V57-reconstructed.exe .
```

Para reproducir el objetivo del binario analizado:

```text
set CGO_ENABLED=0
go build -trimpath -o GestionSO-V57-reconstructed.exe .
```

## Motor V54

El binario contiene la cadena `GestionSO-V54-engine.exe`, pero ese ejecutable no estaba incluido en el ZIP analizado. Por ello el flujo end-to-end con V54 no está validado. La reconstrucción deja la selección de archivos y `feedEngineFile` parametrizados mediante `GESTIONSO_V54_ENGINE` y registra la situación en `%TEMP%\\GestionSO-V57-debug.log`.

## Datos

`GestionSO_Datos.csv` no forma parte del ZIP original según `LEEME-GestionSO-V57.txt`. No se incluye en este repositorio.
