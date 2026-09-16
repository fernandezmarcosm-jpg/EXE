//go:build windows

package main

// Compatibility alias for the existing column-view references.
// Keeps the change surgical without altering the working column-view logic.
func datasetColummKey(c DatasetColumn) string {
	return datasetColumnKey(c)
}
