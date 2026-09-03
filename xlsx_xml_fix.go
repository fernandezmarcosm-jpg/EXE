//go:build windows

package main

import "encoding/xml"

// UnmarshalXML reads the real XLSX worksheet hierarchy. Excel stores rows as
// children of <sheetData>, not direct children of <worksheet>.
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

			// Decode each row explicitly. This avoids depending on a direct
			// worksheet->row XML mapping and handles normal Excel worksheet XML.
			for {
				rowTok, err := d.Token()
				if err != nil {
					return err
				}
				switch r := rowTok.(type) {
				case xml.StartElement:
					if r.Name.Local == "row" {
						var row sheetRowXML
						if err := d.DecodeElement(&row, &r); err != nil {
							return err
						}
						x.Rows = append(x.Rows, row)
					} else if err := d.Skip(); err != nil {
						return err
					}
				case xml.EndElement:
					if r.Name.Local == "sheetData" {
						goto sheetDataDone
					}
				}
			}

		sheetDataDone:
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}
