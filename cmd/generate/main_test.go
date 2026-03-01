package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
