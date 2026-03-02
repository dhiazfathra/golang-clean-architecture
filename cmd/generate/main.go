package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type ModuleSpec struct {
	Name       string // "product"
	NameTitle  string // "Product"
	NamePlural string // "products"
	ModPath    string // "github.com/dhiazfathra/golang-clean-architecture"
	Fields     []FieldSpec
	Timestamp  string // "20260228143022"
}

type FieldSpec struct {
	Name       string // "price"
	NameTitle  string // "Price"
	GoType     string // "float64"
	SQLType    string // "DOUBLE PRECISION"
	SQLDefault string // "DEFAULT 0"
	JSONTag    string // "price"
	DBTag      string // "price"
}

// Dependencies bundles I/O and FS operations so they can be swapped in tests.
type Dependencies struct {
	Stderr   io.Writer
	MkdirAll func(path string, perm os.FileMode) error
	RenderTo func(spec ModuleSpec, tmpl, dest string)
	Now      func() time.Time
	Exit     func(code int)
	Printf   func(format string, a ...any) (int, error)
	Println  func(a ...any) (int, error)
}

// defaultDeps returns production-ready dependencies.
func defaultDeps() Dependencies {
	return Dependencies{
		Stderr:   os.Stderr,
		MkdirAll: os.MkdirAll,
		RenderTo: renderTo,
		Now:      time.Now,
		Exit:     os.Exit,
		Printf:   fmt.Printf,  //nolint:forbidigo // allowed for CLI output
		Println:  fmt.Println, //nolint:forbidigo // allowed for CLI output
	}
}

// run contains all logic previously in main, making it fully testable.
// It returns an exit code (0 = success, non-zero = failure).
func run(args []string, deps Dependencies) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	modName := fs.String("module", "", "module name (e.g. product)")
	fieldsRaw := fs.String("fields", "", "comma-separated name:type pairs (e.g. name:string,price:float64)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *modName == "" {
		fmt.Fprintln(deps.Stderr, "usage: generate -module=<name> -fields=<name:type,...>")
		return 1
	}

	spec := ModuleSpec{
		Name:       *modName,
		NameTitle:  strings.Title(*modName), //nolint:staticcheck
		NamePlural: *modName + "s",
		ModPath:    "github.com/dhiazfathra/golang-clean-architecture",
		Timestamp:  deps.Now().UTC().Format("20060102150405"),
		Fields:     parseFields(*fieldsRaw),
	}

	outDir := filepath.Join("pkg", "module", spec.Name)
	// G301 — Directory permissions too broad if using 0o755
	if err := deps.MkdirAll(outDir, 0o750); err != nil {
		fmt.Fprintln(deps.Stderr, err)
		return 1
	}

	// Source files
	deps.RenderTo(spec, "templates/module/model.go.tmpl", filepath.Join(outDir, "model.go"))
	deps.RenderTo(spec, "templates/module/projections.go.tmpl", filepath.Join(outDir, "projections.go"))
	deps.RenderTo(spec, "templates/module/projector.go.tmpl", filepath.Join(outDir, "projector.go"))
	deps.RenderTo(spec, "templates/module/repository.go.tmpl", filepath.Join(outDir, "repository.go"))
	deps.RenderTo(spec, "templates/module/repository_pg.go.tmpl", filepath.Join(outDir, "repository_pg.go"))
	deps.RenderTo(spec, "templates/module/service.go.tmpl", filepath.Join(outDir, "service.go"))
	deps.RenderTo(spec, "templates/module/handler.go.tmpl", filepath.Join(outDir, "handler.go"))
	deps.RenderTo(spec, "templates/module/routes.go.tmpl", filepath.Join(outDir, "routes.go"))
	deps.RenderTo(spec, "templates/module/register.go.tmpl", filepath.Join(outDir, "register.go"))

	// Test files
	deps.RenderTo(spec, "templates/module/service_test.go.tmpl", filepath.Join(outDir, "service_test.go"))
	deps.RenderTo(spec, "templates/module/handler_test.go.tmpl", filepath.Join(outDir, "handler_test.go"))
	deps.RenderTo(spec, "templates/module/projector_test.go.tmpl", filepath.Join(outDir, "projector_test.go"))
	deps.RenderTo(spec, "templates/module/repository_pg_test.go.tmpl", filepath.Join(outDir, "repository_pg_test.go"))
	deps.RenderTo(spec, "templates/module/routes_test.go.tmpl", filepath.Join(outDir, "routes_test.go"))

	// Migrations
	migUp := filepath.Join("migrations", spec.Timestamp+"_"+spec.Name+"_read.up.sql")
	migDown := filepath.Join("migrations", spec.Timestamp+"_"+spec.Name+"_read.down.sql")
	deps.RenderTo(spec, "templates/module/migration.up.sql.tmpl", migUp)
	deps.RenderTo(spec, "templates/module/migration.down.sql.tmpl", migDown)

	// OpenAPI path fragment
	apiPathsDir := filepath.Join("api", "paths")
	// G301 — Directory permissions too broad if using 0o755
	if err := deps.MkdirAll(apiPathsDir, 0o750); err != nil {
		fmt.Fprintln(deps.Stderr, err)
		return 1
	}
	deps.RenderTo(spec, "templates/module/openapi_paths.yaml.tmpl",
		filepath.Join(apiPathsDir, spec.Name+".yaml"))

	deps.Printf("\n✓ Generated module: %s/ (9 files + 5 test files)\n", outDir)                            //nolint:forbidigo
	deps.Printf("✓ Generated migration: %s\n\n", migUp)                                                    //nolint:forbidigo
	deps.Printf("✓ Generated OpenAPI fragment: api/paths/%s.yaml\n", spec.Name)                            //nolint:forbidigo
	deps.Printf("  → Merge into api/openapi.yaml paths section\n\n")                                       //nolint:forbidigo
	deps.Printf("Add to cmd/server/main.go:\n\n")                                                          //nolint:forbidigo
	deps.Printf("    %[1]sProjector := %[1]s.NewProjector(db)\n", spec.Name)                               //nolint:forbidigo
	deps.Printf("    runner.Register(%[1]sProjector)\n", spec.Name)                                        //nolint:forbidigo
	deps.Printf("    %[1]sSvc := %[1]s.NewService(es, %[1]s.NewPgReadRepository(db))\n", spec.Name)        //nolint:forbidigo
	deps.Printf("    %[1]s.RegisterRoutes(protected, %[1]s.NewHandler(%[1]sSvc), rbacSvc)\n\n", spec.Name) //nolint:forbidigo
	deps.Println("Then run:")                                                                              //nolint:forbidigo
	deps.Println("    make migrate")                                                                       //nolint:forbidigo
	deps.Println("    make seed")                                                                          //nolint:forbidigo

	return 0
}

func main() {
	deps := defaultDeps()
	code := run(os.Args[1:], deps)
	if code != 0 {
		deps.Exit(code)
	}
}

func renderTo(spec ModuleSpec, tmplPath, outPath string) {
	// Validate both paths stay within the working directory to avoid G304 — path traversal
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for _, p := range []string{tmplPath, outPath} {
		abs, err := filepath.Abs(p)
		if err != nil {
			panic(err)
		}
		if !strings.HasPrefix(abs, wd+string(os.PathSeparator)) {
			panic(fmt.Sprintf("renderTo: path escapes working directory: %s", p))
		}
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	t := template.Must(template.New(filepath.Base(tmplPath)).
		Funcs(funcMap).
		ParseFiles(tmplPath))

	// Use os.OpenFile with explicit flags instead of os.Create to avoid G304 — path traversal
	f, err := os.OpenFile(filepath.Clean(outPath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := t.Execute(f, spec); err != nil {
		panic(err)
	}
}

func parseFields(raw string) []FieldSpec {
	if raw == "" {
		return nil
	}
	var fields []FieldSpec
	for _, pair := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			continue
		}
		name, typ := parts[0], parts[1]
		fields = append(fields, FieldSpec{
			Name:       name,
			NameTitle:  strings.Title(name), //nolint:staticcheck
			GoType:     goType(typ),
			SQLType:    sqlType(typ),
			SQLDefault: sqlDefault(typ),
			JSONTag:    name,
			DBTag:      name,
		})
	}
	return fields
}

func goType(t string) string {
	switch t {
	case "string":
		return "string"
	case "int":
		return "int"
	case "int64":
		return "int64"
	case "float64":
		return "float64"
	case "bool":
		return "bool"
	case "time":
		return "time.Time"
	case "uuid":
		return "string"
	}
	return "string"
}

func sqlType(t string) string {
	switch t {
	case "string":
		return "TEXT"
	case "int":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "float64":
		return "DOUBLE PRECISION"
	case "bool":
		return "BOOLEAN"
	case "time":
		return "TIMESTAMPTZ"
	case "uuid":
		return "UUID"
	}
	return "TEXT"
}

func sqlDefault(t string) string {
	switch t {
	case "string":
		return "DEFAULT ''"
	case "int", "int64", "float64":
		return "DEFAULT 0"
	case "bool":
		return "DEFAULT false"
	case "time":
		return "DEFAULT now()"
	}
	return ""
}
