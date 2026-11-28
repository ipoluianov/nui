package ui

import (
	"encoding/xml"
	"io"
	"strings"
)

func xmlCheck(s string) error {
	d := xml.NewDecoder(strings.NewReader(s))
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if se, ok := tok.(xml.StartElement); ok {
			for _, a := range se.Attr {
				if a.Name.Local == "onload" {
					return &xml.SyntaxError{Msg: "injected attribute onload on <" + se.Name.Local + ">"}
				}
			}
		}
	}
}
