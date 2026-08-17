package contracts

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"testing"
)

func TestEnvelopeHelpers(t *testing.T) {
	ok := OK("payload")
	if ok.Data != "payload" || ok.ErrorCode != "" || ok.ErrorMessage != "" {
		t.Fatalf("unexpected OK envelope: %#v", ok)
	}
	fail := Fail("AUTH01001", "bad credentials")
	if fail.Data != nil || fail.ErrorCode != "AUTH01001" || fail.ErrorMessage != "bad credentials" {
		t.Fatalf("unexpected Fail envelope: %#v", fail)
	}
}

// TestErrorCodeFormat enforces the <FEATURE><SUBSET><ERROR> contract
// (4 uppercase feature letters, 2 subset digits, 3 error digits) over every
// constant in errors.go.
func TestErrorCodeFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Z]{4}[0-9]{2}[0-9]{3}$`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "errors.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, value := range valueSpec.Values {
				literal, ok := value.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				code, err := strconv.Unquote(literal.Value)
				if err != nil {
					continue
				}
				checked++
				if !pattern.MatchString(code) {
					t.Errorf("error code %q must match <FEATURE><SUBSET><ERROR>", code)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no error-code constants found in errors.go")
	}
}
