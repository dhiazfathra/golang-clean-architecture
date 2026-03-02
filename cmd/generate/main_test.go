package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// testDeps returns a Dependencies configured for testing.
// renderCalls accumulates every (tmpl, dest) pair passed to RenderTo.
func testDeps(
	stderr *bytes.Buffer,
	mkdirErr error,
	renderCalls *[][]string,
	now time.Time,
) Dependencies {
	if renderCalls == nil {
		calls := [][]string{}
		renderCalls = &calls
	}
	return Dependencies{
		Stderr: stderr,
		MkdirAll: func(path string, perm os.FileMode) error {
			return mkdirErr
		},
		RenderTo: func(spec ModuleSpec, tmpl, dest string) {
			*renderCalls = append(*renderCalls, []string{tmpl, dest})
		},
		Now: func() time.Time { return now },
		Printf: func(format string, a ...any) (int, error) {
			return 0, nil
		},
		Println: func(a ...any) (int, error) {
			return 0, nil
		},
	}
}

var fixedTime = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRun_MissingModuleFlag_ReturnsOne(t *testing.T) {
	var stderr bytes.Buffer
	calls := [][]string{}
	deps := testDeps(&stderr, nil, &calls, fixedTime)

	code := run([]string{}, deps)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("expected usage message in stderr, got: %q", stderr.String())
	}
	if len(calls) != 0 {
		t.Errorf("expected no render calls, got %d", len(calls))
	}
}

func TestRun_InvalidFlag_ReturnsOne(t *testing.T) {
	var stderr bytes.Buffer
	deps := testDeps(&stderr, nil, nil, fixedTime)

	code := run([]string{"-unknown-flag"}, deps)

	if code != 1 {
		t.Fatalf("expected exit code 1 for unknown flag, got %d", code)
	}
}

func TestRun_MkdirAllFails_OutDir_ReturnsOne(t *testing.T) {
	var stderr bytes.Buffer
	deps := testDeps(&stderr, errors.New("permission denied"), nil, fixedTime)

	code := run([]string{"-module=product"}, deps)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "permission denied") {
		t.Errorf("expected error message in stderr, got: %q", stderr.String())
	}
}

func TestRun_MkdirAllFails_ApiPathsDir_ReturnsOne(t *testing.T) {
	var stderr bytes.Buffer
	callCount := 0
	calls := [][]string{}
	deps := Dependencies{
		Stderr: &stderr,
		MkdirAll: func(path string, perm os.FileMode) error {
			callCount++
			// Fail on the second call (api/paths dir)
			if callCount == 2 {
				return errors.New("api paths mkdir failed")
			}
			return nil
		},
		RenderTo: func(spec ModuleSpec, tmpl, dest string) {
			calls = append(calls, []string{tmpl, dest})
		},
		Now: func() time.Time { return fixedTime },
		Printf: func(format string, a ...any) (int, error) {
			return 0, nil
		},
		Println: func(a ...any) (int, error) {
			return 0, nil
		},
	}

	code := run([]string{"-module=product"}, deps)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "api paths mkdir failed") {
		t.Errorf("expected error in stderr, got: %q", stderr.String())
	}
}

func TestRun_Success_AllFilesRendered(t *testing.T) {
	var stderr bytes.Buffer
	calls := [][]string{}
	deps := testDeps(&stderr, nil, &calls, fixedTime)

	code := run([]string{"-module=product", "-fields=name:string,price:float64"}, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}

	// 9 source + 5 test + 2 migration + 1 openapi = 17 render calls
	const expectedCalls = 17
	if len(calls) != expectedCalls {
		t.Errorf("expected %d render calls, got %d", expectedCalls, len(calls))
	}
}

func TestRun_Success_SpecFieldsCorrect(t *testing.T) {
	var stderr bytes.Buffer
	var capturedSpec ModuleSpec
	deps := Dependencies{
		Stderr:   &stderr,
		MkdirAll: func(path string, perm os.FileMode) error { return nil },
		RenderTo: func(spec ModuleSpec, tmpl, dest string) {
			capturedSpec = spec // capture last (all are same spec)
		},
		Now: func() time.Time { return fixedTime },
		Printf: func(format string, a ...any) (int, error) {
			return 0, nil
		},
		Println: func(a ...any) (int, error) {
			return 0, nil
		},
	}

	code := run([]string{"-module=order", "-fields=total:float64"}, deps)

	if code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	if capturedSpec.Name != "order" {
		t.Errorf("expected Name=order, got %q", capturedSpec.Name)
	}
	if capturedSpec.NameTitle != "Order" {
		t.Errorf("expected NameTitle=Order, got %q", capturedSpec.NameTitle)
	}
	if capturedSpec.NamePlural != "orders" {
		t.Errorf("expected NamePlural=orders, got %q", capturedSpec.NamePlural)
	}
	if capturedSpec.ModPath != "github.com/dhiazfathra/golang-clean-architecture" {
		t.Errorf("unexpected ModPath: %q", capturedSpec.ModPath)
	}
	if capturedSpec.Timestamp != "20240115120000" {
		t.Errorf("expected Timestamp=20240115120000, got %q", capturedSpec.Timestamp)
	}
	if len(capturedSpec.Fields) != 1 || capturedSpec.Fields[0].Name != "total" {
		t.Errorf("unexpected Fields: %+v", capturedSpec.Fields)
	}
}

func TestRun_Success_OutputPaths(t *testing.T) {
	var stderr bytes.Buffer
	calls := [][]string{}
	deps := testDeps(&stderr, nil, &calls, fixedTime)

	code := run([]string{"-module=widget"}, deps)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Verify some key output destinations
	outDir := filepath.Join("pkg", "module", "widget")
	wantDests := []string{
		filepath.Join(outDir, "model.go"),
		filepath.Join(outDir, "service.go"),
		filepath.Join(outDir, "handler.go"),
		filepath.Join(outDir, "service_test.go"),
		filepath.Join("api", "paths", "widget.yaml"),
	}

	destSet := map[string]bool{}
	for _, c := range calls {
		destSet[c[1]] = true
	}
	for _, want := range wantDests {
		if !destSet[want] {
			t.Errorf("expected render dest %q not found in calls", want)
		}
	}

	// Verify migration files contain the timestamp
	ts := fixedTime.UTC().Format("20060102150405")
	migUpDest := filepath.Join("migrations", ts+"_widget_read.up.sql")
	migDownDest := filepath.Join("migrations", ts+"_widget_read.down.sql")
	if !destSet[migUpDest] {
		t.Errorf("expected migration up dest %q not found", migUpDest)
	}
	if !destSet[migDownDest] {
		t.Errorf("expected migration down dest %q not found", migDownDest)
	}
}

func TestRun_NoFields_SucceedsWithEmptyFields(t *testing.T) {
	var stderr bytes.Buffer
	var capturedSpec ModuleSpec
	deps := Dependencies{
		Stderr:   &stderr,
		MkdirAll: func(path string, perm os.FileMode) error { return nil },
		RenderTo: func(spec ModuleSpec, tmpl, dest string) {
			capturedSpec = spec
		},
		Now: func() time.Time { return fixedTime },
		Printf: func(format string, a ...any) (int, error) {
			return 0, nil
		},
		Println: func(a ...any) (int, error) {
			return 0, nil
		},
	}

	code := run([]string{"-module=thing"}, deps)

	if code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	if len(capturedSpec.Fields) != 0 {
		t.Errorf("expected empty fields, got %+v", capturedSpec.Fields)
	}
}

func TestDefaultDeps_NotNil(t *testing.T) {
	deps := defaultDeps()
	if deps.Stderr == nil {
		t.Error("Stderr should not be nil")
	}
	if deps.MkdirAll == nil {
		t.Error("MkdirAll should not be nil")
	}
	if deps.RenderTo == nil {
		t.Error("RenderTo should not be nil")
	}
	if deps.Now == nil {
		t.Error("Now should not be nil")
	}
	if deps.Exit == nil {
		t.Error("Exit should not be nil")
	}
	if deps.Printf == nil {
		t.Error("Printf should not be nil")
	}
	if deps.Println == nil {
		t.Error("Println should not be nil")
	}
}

// ---------------------------------------------------------------------------
// goType
// ---------------------------------------------------------------------------

func TestGoType(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"string", "string"},
		{"int", "int"},
		{"int64", "int64"},
		{"float64", "float64"},
		{"bool", "bool"},
		{"time", "time.Time"},
		{"uuid", "string"},
		{"unknown", "string"},
	}
	for _, c := range cases {
		if got := goType(c.in); got != c.want {
			t.Errorf("goType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// sqlType
// ---------------------------------------------------------------------------

func TestSqlType(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"string", "TEXT"},
		{"int", "INTEGER"},
		{"int64", "BIGINT"},
		{"float64", "DOUBLE PRECISION"},
		{"bool", "BOOLEAN"},
		{"time", "TIMESTAMPTZ"},
		{"uuid", "UUID"},
		{"unknown", "TEXT"},
	}
	for _, c := range cases {
		if got := sqlType(c.in); got != c.want {
			t.Errorf("sqlType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// sqlDefault
// ---------------------------------------------------------------------------

func TestSqlDefault(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"string", "DEFAULT ''"},
		{"int", "DEFAULT 0"},
		{"int64", "DEFAULT 0"},
		{"float64", "DEFAULT 0"},
		{"bool", "DEFAULT false"},
		{"time", "DEFAULT now()"},
		{"uuid", ""},
		{"unknown", ""},
	}
	for _, c := range cases {
		if got := sqlDefault(c.in); got != c.want {
			t.Errorf("sqlDefault(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// parseFields
// ---------------------------------------------------------------------------

func TestParseFields_Empty(t *testing.T) {
	fields := parseFields("")
	if fields != nil {
		t.Errorf("expected nil, got %v", fields)
	}
}

func TestParseFields_Valid(t *testing.T) {
	fields := parseFields("name:string,price:float64,active:bool")
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	if fields[0].Name != "name" || fields[0].GoType != "string" || fields[0].SQLType != "TEXT" {
		t.Errorf("unexpected field[0]: %+v", fields[0])
	}
	if fields[1].Name != "price" || fields[1].GoType != "float64" || fields[1].SQLType != "DOUBLE PRECISION" {
		t.Errorf("unexpected field[1]: %+v", fields[1])
	}
	if fields[2].Name != "active" || fields[2].GoType != "bool" || fields[2].SQLType != "BOOLEAN" {
		t.Errorf("unexpected field[2]: %+v", fields[2])
	}
}

func TestParseFields_NameTitle(t *testing.T) {
	fields := parseFields("price:float64")
	if fields[0].NameTitle != "Price" {
		t.Errorf("NameTitle = %q, want %q", fields[0].NameTitle, "Price")
	}
}

func TestParseFields_JSONAndDBTags(t *testing.T) {
	fields := parseFields("email:string")
	if fields[0].JSONTag != "email" || fields[0].DBTag != "email" {
		t.Errorf("unexpected tags: json=%q db=%q", fields[0].JSONTag, fields[0].DBTag)
	}
}

func TestParseFields_SkipMalformed(t *testing.T) {
	// "badpair" has no colon — should be skipped
	fields := parseFields("badpair,name:string")
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Name != "name" {
		t.Errorf("unexpected field name: %q", fields[0].Name)
	}
}

func TestParseFields_AllTypes(t *testing.T) {
	raw := "a:string,b:int,c:int64,d:float64,e:bool,f:time,g:uuid,h:unknown"
	fields := parseFields(raw)
	if len(fields) != 8 {
		t.Fatalf("expected 8 fields, got %d", len(fields))
	}
}

func TestParseFields_Whitespace(t *testing.T) {
	fields := parseFields("  name : string  , price : float64  ")
	// After TrimSpace on the pair the colon split still works;
	// leading/trailing spaces on name/type are NOT trimmed (by design) — just verify no panic.
	if len(fields) == 0 {
		t.Fatal("expected at least one field")
	}
}

// ---------------------------------------------------------------------------
// renderTo — path traversal guard
// ---------------------------------------------------------------------------

func TestRenderTo_PathEscapesWorkingDir(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for path escaping working dir")
		}
		msg := ""
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		}
		if !strings.Contains(msg, "escapes working directory") {
			t.Errorf("unexpected panic message: %v", r)
		}
	}()

	spec := ModuleSpec{Name: "test"}
	// /tmp is outside the working directory
	renderTo(spec, "/tmp/evil.tmpl", "/tmp/evil.go")
}

// ---------------------------------------------------------------------------
// renderTo — happy path with real templates
// ---------------------------------------------------------------------------

func TestRenderTo_HappyPath(t *testing.T) {
	// Create a temporary directory inside the working directory to act as
	// both the template source and the output destination.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// --- template file ---
	tmplDir := filepath.Join(wd, "testdata_tmp_templates")
	if err := os.MkdirAll(tmplDir, 0o750); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmplDir)

	tmplPath := filepath.Join(tmplDir, "simple.tmpl")
	if err := os.WriteFile(tmplPath, []byte(`hello {{.Name}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// --- output file ---
	outDir := filepath.Join(wd, "testdata_tmp_out")
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)

	outPath := filepath.Join(outDir, "out.txt")

	spec := ModuleSpec{Name: "world"}
	renderTo(spec, tmplPath, outPath)

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("output = %q, want %q", string(data), "hello world")
	}
}

// TestRenderTo_WithAddFunc verifies the "add" FuncMap entry works in templates.
func TestRenderTo_WithAddFunc(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tmplDir := filepath.Join(wd, "testdata_tmp_add_tmpl")
	if err := os.MkdirAll(tmplDir, 0o750); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmplDir)

	tmplPath := filepath.Join(tmplDir, "add.tmpl")
	// template uses the "add" func
	if err := os.WriteFile(tmplPath, []byte(`{{add 3 4}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(wd, "testdata_tmp_add_out")
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)

	outPath := filepath.Join(outDir, "out.txt")
	renderTo(ModuleSpec{}, tmplPath, outPath)

	data, _ := os.ReadFile(outPath)
	if string(data) != "7" {
		t.Errorf("add func output = %q, want %q", string(data), "7")
	}
}
