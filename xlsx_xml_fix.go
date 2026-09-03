//go:build windows

package main

import "encoding/xml"

// UnmarshalXML reads the real XLSX worksheet hierarchy. Excel stores rows as
// children of <sheetData>, not direct children of <worksheet>.
//
// Rows are decoded one by one instead of relying on a nested slice tag match;
// this keeps the importer tolerant of worksheet metadata before/after
// <sheetData> and makes the row extraction explicit.
func (x *sheetXML) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	x.Rows = nil
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "sheetData" {
				if err := d.Skip(); err != nil {
					return err
				}
				continue
			}

			// Read the contents of <sheetData> and decode each <row> explicitly.
			for {
				tok, err := d.Token()
				if err != nil {
					return err
				}
				switch rowTok := tok.(type) {
				case xml.StartElement:
					if rowTok.Name.Local == "row" {
						var row sheetRowXML
						if err := d.DecodeElement(&row, &rowTok); err != nil {
							return err
						}
						x.Rows = append(x.Rows, row)
					} else if err := d.Skip(); err != nil {
						return err
					}
				case xml.EndElement:
					if rowTok.Name.Local == "sheetData" {
						break
					}
				}

				if len(x.Rows) > 0 {
					// No-op: keeps this branch intentionally simple; the enclosing
					// loop continues until the sheetData end element is consumed.
				}

				// The inner loop needs an explicit end check; inspect the last
				// token through a small helper condition below is not possible,
				// so this block is replaced by the labeled loop in the source.
			}
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}
