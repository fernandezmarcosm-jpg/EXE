package main

import (
	_ "embed"
	"os"
	"path/filepath"
)

// El CSV del repositorio es el modelo de datos de GestionSO. Se embebe en el
// ejecutable para que el ZIP no dependa de que el usuario copie archivos extra.
// En ejecucion se materializa solo como cache temporal para reutilizar el
// cargador MasterData existente.
//
//go:embed acceso chatgpt/GestionSO_Datos.csv
var gestionSOModelCSV []byte

func init() {
	path := filepath.Join(os.TempDir(), "GestionSO_Datos.csv")
	// Se actualiza en cada arranque para que el modelo incluido en el EXE sea
	// exactamente el que se uso durante la compilacion.
	if err := os.WriteFile(path, gestionSOModelCSV, 0644); err != nil {
		logf("ERROR preparando modelo GestionSO_Datos.csv: %v", err)
	}
}
