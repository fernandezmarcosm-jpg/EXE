# GestionSO V57 — reconstrucción

Este repositorio contiene una **reconstrucción/reimplementación** de GestionSO V57 a partir de evidencia observable en el binario `GestionSO-V57.exe` disponible durante el análisis. **No es el código fuente original** y no debe presentarse como tal.

## Estado

- **Entrega 1:** reconstrucción de `main_windows.go`, centrada en Win32, botón `ABRIR XLSX`, selector múltiple y trazabilidad del hook.
- **Entrega 2:** `core.go` agregado por bloques: tipos/logging defensivo, lectura y merge XLSX, configuración/persistencia CSV, vistas/proceso y stubs explícitos para opciones/simulador.
- **Build integrado verificado:** el workflow `Validar Go` ejecutó `go build ./...` y `go vet ./...` con Go 1.23.2, `CGO_ENABLED=0`, `GOOS=windows`, `GOARCH=amd64`, y terminó en verde en el run 1.

El binario analizado es Go 1.23.2, Windows x86-64, `CGO_ENABLED=0`, buildmode `exe`. El símbolo `main.feedEngineFile` existe realmente en el binario; su contrato interno no puede recuperarse literalmente sin el motor V54.

## Descarga del ejecutable

El ejecutable se genera **exclusivamente como artifact de GitHub Actions**; no se commitean `.exe` ni `.zip` al repositorio.

- Workflow de compilación y descarga: https://github.com/fernandezmarcosm-jpg/EXE/actions/workflows/build-exe.yml
- El artifact se llama **`GestionSO-V57`** y contiene `GestionSO-V57.zip`.
- Dentro del ZIP están `GestionSO-V57.exe`, `README.md`, `EVIDENCIA_BINARIO.md` y `LEEME.txt`.

Para descargarlo: abrir el workflow en **Actions**, entrar al run terminado en verde y, en la sección **Artifacts**, seleccionar `GestionSO-V57`.

## Compilación

Build utilizado para la entrega Windows GUI:

```text
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o dist/GestionSO-V57.exe ./...
```

También se ejecuta:

```text
go vet ./...
```

La validación integrada de compilación y `go vet` quedó automatizada en `.github/workflows/validar-go.yml` y la generación del ejecutable/artifact en `.github/workflows/build-exe.yml`.

## Ejecución y límites

El programa es una reconstrucción y no una copia del fuente original. Para el flujo que depende del motor externo se requiere `GestionSO-V54-engine.exe`, configurable mediante `GESTIONSO_V54_ENGINE`. Ese motor no está incluido.

Tampoco se incluyen `GestionSO_Datos.csv` ni archivos XLSX de prueba. Por lo tanto, **la validación end-to-end del flujo `ABRIR XLSX` + motor V54 no está realizada**. El build verde confirma compilación integrada y `go vet`, no compatibilidad funcional completa.

## XLSX

La reconstrucción implementa lectura mediante `archive/zip` + `encoding/xml`, incluyendo shared strings, filas y selección de la hoja con mejor puntuación de encabezado. La operación de merge no modifica los XLSX originales.

El detalle exacto del formato de salida y del contrato con el motor V54 sigue siendo una inferencia mientras no esté disponible el ejecutable V54 original.

## Motor V54

El binario contiene la cadena `GestionSO-V54-engine.exe`, pero ese ejecutable no estaba incluido en el ZIP analizado. Por ello el flujo end-to-end con V54 no está validado. `feedEngineFile` queda parametrizado mediante `GESTIONSO_V54_ENGINE` y registra la situación en `%TEMP%\\GestionSO-V57-debug.log`; no se inventa un contrato de argumentos que no pueda verificarse.

## Datos

`GestionSO_Datos.csv` no forma parte del ZIP original según `LEEME-GestionSO-V57.txt`. No se incluye en este repositorio ni en el artifact.

## Trazabilidad

La evidencia de símbolos, strings, APIs Win32 y limitaciones está en [`docs/EVIDENCIA_BINARIO.md`](docs/EVIDENCIA_BINARIO.md).

No se incluyen binarios en el repositorio; los binarios de entrega viven únicamente como artifacts de Actions.
