package template

import (
	"html/template"
	"strings"
	"testing"
)

func TestEngine_LoadStringAndRender(t *testing.T) {
	e := NewEngine()
	err := e.LoadString("hello", "Hello {{.Name}}!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := e.Render("hello", map[string]any{"Name": "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello World!" {
		t.Errorf("expected 'Hello World!', got '%s'", result)
	}
}

func TestEngine_RenderNotFound(t *testing.T) {
	e := NewEngine()
	_, err := e.Render("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestEngine_Has(t *testing.T) {
	e := NewEngine()
	if e.Has("test") {
		t.Error("should not have test template")
	}
	e.LoadString("test", "content")
	if !e.Has("test") {
		t.Error("should have test template")
	}
}

func TestEngine_Helpers(t *testing.T) {
	e := NewEngine()
	e.LoadString("test", `{{truncate .Text 5}}`)

	result, err := e.Render("test", map[string]any{"Text": "Hello World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello..." {
		t.Errorf("expected 'Hello...', got '%s'", result)
	}
}

func TestEngine_DictHelper(t *testing.T) {
	e := NewEngine()
	e.LoadString("test", `{{$d := dict "key" "value"}}{{$d.key}}`)

	result, err := e.Render("test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}
}

func TestEngine_PartialInLayout(t *testing.T) {
	e := NewEngine()

	e.LoadString("partials/header.html", `<header>{{.Title}}</header>`)
	err := e.LoadString("layout.html", `<html>{{template "partials/header.html" .}}<main>{{.Content}}</main></html>`)
	if err != nil {
		t.Fatalf("failed to load layout: %v", err)
	}

	result, err := e.Render("layout.html", map[string]any{
		"Title":   "Test",
		"Content": template.HTML("<p>Hello</p>"),
	})
	if err != nil {
		t.Fatalf("failed to render layout: %v", err)
	}

	if !strings.Contains(result, "<header>Test</header>") {
		t.Errorf("expected header partial rendered, got: %s", result)
	}
	if !strings.Contains(result, "<p>Hello</p>") {
		t.Errorf("expected content rendered, got: %s", result)
	}
}

func TestEngine_PartialLoadedAfterTemplate(t *testing.T) {
	e := NewEngine()

	err := e.LoadString("layout.html", `<html>{{template "partials/nav.html" .}}<main>{{.Content}}</main></html>`)
	if err != nil {
		t.Logf("layout load before partial exists: %v", err)
	}

	err = e.LoadString("partials/nav.html", `<nav>{{.Title}}</nav>`)
	if err != nil {
		t.Fatalf("failed to load partial: %v", err)
	}

	result, err := e.Render("layout.html", map[string]any{
		"Title":   "Nav Test",
		"Content": template.HTML("<p>Body</p>"),
	})
	if err != nil {
		t.Fatalf("failed to render after partial loaded: %v", err)
	}

	if !strings.Contains(result, "<nav>Nav Test</nav>") {
		t.Errorf("expected nav partial rendered, got: %s", result)
	}
}

func TestEngine_EqHelper(t *testing.T) {
	e := NewEngine()
	e.LoadString("test", `{{if eq .Status "active"}}YES{{else}}NO{{end}}`)

	result, _ := e.Render("test", map[string]any{"Status": "active"})
	if result != "YES" {
		t.Errorf("expected YES, got %s", result)
	}

	result2, _ := e.Render("test", map[string]any{"Status": "inactive"})
	if result2 != "NO" {
		t.Errorf("expected NO, got %s", result2)
	}
}
