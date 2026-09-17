package main

import "strings"

var (
	appSettings = defaultDatasetSettings()
	viewDataset *MemoryDataset
)

func cleanCell(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(s)
}

func workbookSize(doc *xlsxDoc) (rows, cells int) {
	if doc == nil { return 0, 0 }
	for _, rs := range doc.Sheets {
		rows += len(rs)
		for _, r := range rs { cells += len(r) }
	}
	return rows, cells
}
