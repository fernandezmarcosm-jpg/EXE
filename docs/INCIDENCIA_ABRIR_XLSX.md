# Incidencia ABRIR XLSX — diagnóstico y corrección

Fecha: 2026-09-03

## Síntoma observado

El botón visible `ABRIR XLSX` recibía el clic, pero el selector de archivos no aparecía en la aplicación reconstruida.

## Causa técnica identificada

La reconstrucción llamaba a `GetOpenFileNameW` mediante `OPENFILENAMEW`, pero la estructura Go no coincidía con el layout Win64 de la estructura nativa.

La definición anterior omitía el padding necesario para alinear punteros después de miembros `DWORD` y también omitía los campos finales `pvReserved`, `dwReserved` y `FlagsEx`.

Esto hacía que `GetOpenFileNameW` leyera algunos campos desde offsets incorrectos. La consecuencia observable era que el selector podía no abrirse aunque `WM_COMMAND` y el ID `ID_ABRIR_XLSX` fueran correctos.

La documentación de Microsoft define `OPENFILENAMEW` con esos miembros y exige que `lStructSize` sea el tamaño de la estructura. La estructura Win64 requiere respetar su alineación nativa. citeturn0search0turn0search4

## Corrección aplicada

`main_windows.go` fue corregido para que `OPENFILENAMEW` incluya:

- padding después de `lStructSize`;
- padding después de `nMaxFile`;
- padding después de `nMaxFileTitle`;
- `pvReserved`;
- `dwReserved`;
- `FlagsEx`.

`LStructSize` continúa calculándose con `unsafe.Sizeof(OPENFILENAMEW{})`, por lo que el valor entregado a Win32 corresponde a la estructura Go corregida.

También se corrigió `parseMultiSelectBuffer`: antes utilizaba `len(first)` para calcular una posición dentro de un buffer UTF-16. `len(string)` está expresado en bytes, mientras que el buffer se indexa en unidades UTF-16. Ahora se busca explícitamente el primer `NUL` dentro de `[]uint16`.

## Estado

- Fuente corregida en `main`.
- `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` validado localmente en entorno Go 1.23.2.
- `go vet ./...` validado localmente.
- El ejecutable producido localmente se identificó como PE32+ GUI x86-64.

## Qué queda por probar

La corrección del ABI elimina el defecto técnico identificado en la llamada a `GetOpenFileNameW`, pero la validación definitiva del comportamiento requiere ejecutar el `.exe` en Windows y comprobar el clic real sobre `ABRIR XLSX`, selección simple y selección múltiple de archivos `.xlsx`.

Esto no implica que el flujo de negocio completo esté validado: `GestionSO-V54-engine.exe` y los datos reales continúan siendo externos a esta reconstrucción.
