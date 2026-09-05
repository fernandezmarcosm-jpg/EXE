# GestionSO V57 — configuración persistente

Las preferencias de la grilla se guardan mediante `saveDatasetSettings` en `os.UserConfigDir()`.

En Windows, la ruta efectiva es:

`%LOCALAPPDATA%\GestionSO V57\dataset.txt`

El archivo es JSON aunque conserva la extensión histórica `.txt`.

Se persisten, entre otros: títulos visibles, orden de columnas, visibilidad, decimales por columna, tipo por columna, porcentaje, resaltado de negativos, fórmula, columnas calculadas, subtotales y tamaño de fuente.

## Fórmulas

- `[COLUMNA]` referencia una columna XLSX por título.
- `[CSV:COLUMNA]` referencia una columna proveniente de `GestionSO_Datos.csv`.
- Operadores admitidos: `+`, `-`, `*`, `/` y paréntesis.
- Los nombres se insertan desde el selector de configuración para evitar errores de tipeo.

La reconstrucción no afirma que esta sintaxis sea la del motor V54 original: es la sintaxis implementada en esta reconstrucción a partir de la estructura existente del parser.
