//go:build windows

package main

import "encoding/xml"

// UnmarshalXML fixes the XLSX worksheet layout used by Excel. In a normal
// worksheet, <row> elements are children of <sheetData>, not direct children
// of <worksheet>. The original sheetXML shape predates this fix, so the
// custom unmarshaler keeps the existing types/API while reading real XLSX.
func (x *sheetXML) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
    x.Rows = nil
    for {
        tok, err := d.Token()
        if err != nil {
            if err == nil {
                return nil
            }
            return err
        }
        switch t := tok.(type) {
        case xml.StartElement:
            if t.Name.Local == "sheetData" {
                if err := d.DecodeElement(&x.Rows, &t); err != nil {
                    return err
                }
            } else if err := d.Skip(); err != nil {
                return err
            }
        case xml.EndElement:
            if t.Name == start.Name {
                return nil
            }
        }
    }
}
