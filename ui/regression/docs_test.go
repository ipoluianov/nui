package regression

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Bug #12, fixed and kept: every ui.Name the documentation mentions exists
// in the package ui (the docs described ShowEnterStringDialog,
// ShowAboutDialog and UpdateMainForm, which never were)
func TestDocsMentionExistingAPI(t *testing.T) {
	exported := map[string]bool{}
	pkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Join("..", ""), func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for name, obj := range file.Scope.Objects {
				if ast.IsExported(name) && obj.Kind != ast.Bad {
					exported[name] = true
				}
			}
		}
	}
	if len(exported) < 100 {
		t.Fatalf("only %d exported names of ui: the check is broken", len(exported))
	}

	mention := regexp.MustCompile(`\bui\.([A-Z][A-Za-z0-9_]*)`)
	var docs []string
	filepath.Walk(filepath.Join("..", "..", "doc"), func(path string, info os.FileInfo, err error) error {
		// The plan names the APIs to come
		if err == nil && strings.HasSuffix(path, ".md") && filepath.Base(path) != "production-readiness.md" {
			docs = append(docs, path)
		}
		return nil
	})
	if len(docs) == 0 {
		t.Skip("no doc")
	}
	for _, doc := range docs {
		data, _ := os.ReadFile(doc)
		for _, m := range mention.FindAllStringSubmatch(string(data), -1) {
			if !exported[m[1]] {
				t.Errorf("%s mentions ui.%s, which doesn't exist", doc, m[1])
			}
		}
	}
}
