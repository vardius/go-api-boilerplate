// Package archtest enforces the import rules that keep the layers apart.
// They are listed in AGENTS.md; this test is what makes them true.
package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/vardius/go-api-boilerplate"

// pkg is one directory's imports, from every Go file in it whatever its
// build tags, split into production and test code.
type pkg struct {
	path        string
	imports     map[string][]string // import → the files that import it
	testImports map[string][]string
}

func load(t *testing.T) map[string]*pkg {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	pkgs := map[string]*pkg{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "testdata", "website", "web", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		importPath := module + "/" + filepath.ToSlash(rel)
		p := pkgs[importPath]
		if p == nil {
			p = &pkg{path: importPath, imports: map[string][]string{}, testImports: map[string][]string{}}
			pkgs[importPath] = p
		}
		target := p.imports
		if strings.HasSuffix(path, "_test.go") {
			target = p.testImports
		}
		for _, spec := range f.Imports {
			imp, _ := strconv.Unquote(spec.Path.Value)
			target[imp] = append(target[imp], filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) == 0 {
		t.Fatal("found no Go packages")
	}
	return pkgs
}

func within(path, prefix string) bool {
	full := module + "/" + prefix
	return path == full || strings.HasPrefix(path, full+"/")
}

func stdlib(imp string) bool {
	first, _, _ := strings.Cut(imp, "/")
	return !strings.Contains(first, ".")
}

func check(t *testing.T, pkgs map[string]*pkg, applies func(p *pkg) bool, allowed func(imp string) bool, why string) {
	t.Helper()
	for _, p := range pkgs {
		if !applies(p) {
			continue
		}
		for _, imp := range slices.Sorted(maps.Keys(p.imports)) {
			if !allowed(imp) {
				t.Errorf("%s imports %s (%s): %s", strings.TrimPrefix(p.path, module+"/"), imp, strings.Join(p.imports[imp], ", "), why)
			}
		}
	}
}

func TestPlatformImportsNoService(t *testing.T) {
	check(t, load(t),
		func(p *pkg) bool { return within(p.path, "internal/platform") },
		func(imp string) bool {
			return !within(imp, "internal/auth") && !within(imp, "internal/user") && !within(imp, "gen") && !within(imp, "cmd")
		},
		"the platform kit is shared, so it can't depend on a service")
}

func TestServicesStayApart(t *testing.T) {
	pkgs := load(t)
	check(t, pkgs,
		func(p *pkg) bool { return within(p.path, "internal/auth") },
		func(imp string) bool { return !within(imp, "internal/user") },
		"services talk over Connect through gen/, never by importing each other")
	check(t, pkgs,
		func(p *pkg) bool { return within(p.path, "internal/user") },
		func(imp string) bool { return !within(imp, "internal/auth") },
		"services talk over Connect through gen/, never by importing each other")
}

func TestUserDomainIsPure(t *testing.T) {
	check(t, load(t),
		func(p *pkg) bool { return within(p.path, "internal/user/domain") },
		func(imp string) bool {
			return stdlib(imp) || within(imp, "internal/platform/es") || within(imp, "internal/platform/apperr")
		},
		"the domain holds the rules and nothing else: no transport, storage or framework")
}

func TestTestHelpersStayInTests(t *testing.T) {
	helpers := []string{"testing", "net/http/httptest", "github.com/testcontainers/testcontainers-go"}
	isHelper := func(imp string) bool {
		if slices.Contains(helpers, imp) || strings.HasPrefix(imp, "github.com/testcontainers/") {
			return true
		}
		last := imp[strings.LastIndex(imp, "/")+1:]
		return strings.HasPrefix(imp, module+"/") && strings.HasSuffix(last, "test") && last != "archtest"
	}
	check(t, load(t),
		func(p *pkg) bool {
			last := p.path[strings.LastIndex(p.path, "/")+1:]
			return !strings.HasSuffix(last, "test")
		},
		func(imp string) bool { return !isHelper(imp) },
		"test helpers and testcontainers must not be linked into production binaries")
}

func TestModuleRootIsThisRepo(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "module "+module+"\n") {
		t.Fatalf("go.mod doesn't declare %s; update the constant", module)
	}
}
