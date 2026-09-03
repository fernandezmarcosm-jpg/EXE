package main

import "testing"

func TestWorkbookSizeCountsRowsAndCells(t *testing.T) {
	doc := &xlsxDoc{Sheets: map[string][][]string{
		"Hoja1": {
			{"A", "B", "C"},
			{"1", "2", "3"},
		},
	}}

	rows, cells := workbookSize(doc)
	if rows != 2 || cells != 6 {
		t.Fatalf("workbookSize() = rows=%d cells=%d; want rows=2 cells=6", rows, cells)
	}
}
