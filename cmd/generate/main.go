package main

import (
	"flag"
	"fmt"
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

func main() {
	modName := flag.String("module", "", "module name (e.g. product)")
	fieldsRaw := flag.String("fields", "", "comma-separated name:type pairs (e.g. name:string,price:float64)")
	flag.Parse()

	if *modName == "" {
		fmt.Fprintln(os.Stderr, "usage: generate -module=<name> -fields=<name:type,...>")
		os.Exit(1)
	}

	spec := ModuleSpec{
		Name:       *modName,
		NameTitle:  strings.Title(*modName), //nolint:staticcheck
		NamePlural: *modName + "s",
		ModPath:    "github.com/dhiazfathra/golang-clean-architecture",
		Timestamp:  time.Now().UTC().Format("20060102150405"),
		Fields:     parseFields(*fieldsRaw),
	}

	outDir := filepath.Join("pkg", "module", spec.Name)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}

	// Source files
	renderTo(spec, "templates/module/model.go.tmpl", filepath.Join(outDir, "model.go"))
	renderTo(spec, "templates/module/projections.go.tmpl", filepath.Join(outDir, "projections.go"))
	renderTo(spec, "templates/module/projector.go.tmpl", filepath.Join(outDir, "projector.go"))
	renderTo(spec, "templates/module/repository.go.tmpl", filepath.Join(outDir, "repository.go"))
	renderTo(spec, "templates/module/repository_pg.go.tmpl", filepath.Join(outDir, "repository_pg.go"))
	renderTo(spec, "templates/module/service.go.tmpl", filepath.Join(outDir, "service.go"))
	renderTo(spec, "templates/module/handler.go.tmpl", filepath.Join(outDir, "handler.go"))
	renderTo(spec, "templates/module/routes.go.tmpl", filepath.Join(outDir, "routes.go"))
	renderTo(spec, "templates/module/register.go.tmpl", filepath.Join(outDir, "register.go"))

	// Test files
	renderTo(spec, "templates/module/service_test.go.tmpl", filepath.Join(outDir, "service_test.go"))
	renderTo(spec, "templates/module/handler_test.go.tmpl", filepath.Join(outDir, "handler_test.go"))
	renderTo(spec, "templates/module/projector_test.go.tmpl", filepath.Join(outDir, "projector_test.go"))
	renderTo(spec, "templates/module/repository_pg_test.go.tmpl", filepath.Join(outDir, "repository_pg_test.go"))
	renderTo(spec, "templates/module/routes_test.go.tmpl", filepath.Join(outDir, "routes_test.go"))

	// Migrations
	migUp := filepath.Join("migrations", spec.Timestamp+"_"+spec.Name+"_read.up.sql")
	migDown := filepath.Join("migrations", spec.Timestamp+"_"+spec.Name+"_read.down.sql")
	renderTo(spec, "templates/module/migration.up.sql.tmpl", migUp)
	renderTo(spec, "templates/module/migration.down.sql.tmpl", migDown)

	// OpenAPI path fragment (merge into api/openapi.yaml manually)
	apiPathsDir := filepath.Join("api", "paths")
	if err := os.MkdirAll(apiPathsDir, 0o755); err != nil {
		panic(err)
	}
	renderTo(spec, "templates/module/openapi_paths.yaml.tmpl",
		filepath.Join(apiPathsDir, spec.Name+".yaml"))

	fmt.Printf("\n✓ Generated module: %s/ (9 files + 5 test files)\n", outDir)                            //nolint:forbidigo
	fmt.Printf("✓ Generated migration: %s\n\n", migUp)                                                    //nolint:forbidigo
	fmt.Printf("✓ Generated OpenAPI fragment: api/paths/%s.yaml\n", spec.Name)                            //nolint:forbidigo
	fmt.Printf("  → Merge into api/openapi.yaml paths section\n\n")                                       //nolint:forbidigo
	fmt.Printf("Add to cmd/server/main.go:\n\n")                                                          //nolint:forbidigo
	fmt.Printf("    %[1]sProjector := %[1]s.NewProjector(db)\n", spec.Name)                               //nolint:forbidigo
	fmt.Printf("    runner.Register(%[1]sProjector)\n", spec.Name)                                        //nolint:forbidigo
	fmt.Printf("    %[1]sSvc := %[1]s.NewService(es, %[1]s.NewPgReadRepository(db))\n", spec.Name)        //nolint:forbidigo
	fmt.Printf("    %[1]s.RegisterRoutes(protected, %[1]s.NewHandler(%[1]sSvc), rbacSvc)\n\n", spec.Name) //nolint:forbidigo
	fmt.Println("Then run:")                                                                              //nolint:forbidigo
	fmt.Println("    make migrate")                                                                       //nolint:forbidigo
	fmt.Println("    make seed")                                                                          //nolint:forbidigo
}

func renderTo(spec ModuleSpec, tmplPath, outPath string) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	t := template.Must(template.New(filepath.Base(tmplPath)).
		Funcs(funcMap).
		ParseFiles(tmplPath))
	f, err := os.Create(outPath)
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
