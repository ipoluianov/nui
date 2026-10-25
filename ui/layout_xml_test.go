package ui

import (
	"os"
	"strings"
	"testing"
)

// Every element of the XML is in its documentation
func TestXMLElementsDocumented(t *testing.T) {
	doc, err := os.ReadFile("../doc/ui/content/xml_layout.md")
	if err != nil {
		t.Fatal(err)
	}
	for name, e := range xmlElements {
		if !strings.Contains(string(doc), "`<"+name+">`") {
			t.Errorf("<%s> is not in xml_layout.md", name)
		}
		for attr := range e.attrs {
			if !strings.Contains(string(doc), "`"+attr+"`") {
				t.Errorf("the attribute %s of <%s> is not in xml_layout.md", attr, name)
			}
		}
		for _, ev := range e.events {
			if !strings.Contains(string(doc), "`"+ev+"`") {
				t.Errorf("the event %s of <%s> is not in xml_layout.md", ev, name)
			}
		}
	}
	for attr := range commonXMLAttrs {
		if !strings.Contains(string(doc), "`"+attr+"`") {
			t.Errorf("the attribute %s is not in xml_layout.md", attr)
		}
	}
}
