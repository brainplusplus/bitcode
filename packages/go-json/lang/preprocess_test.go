package lang

import (
	"testing"
)

func TestStripComments_NoComments(t *testing.T) {
	input := []byte(`{"name": "test", "value": 42}`)
	got := StripComments(input)
	if string(got) != string(input) {
		t.Errorf("expected unchanged, got %s", got)
	}
}

func TestStripComments_Empty(t *testing.T) {
	got := StripComments([]byte{})
	if len(got) != 0 {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestStripComments_LineComment(t *testing.T) {
	input := []byte("{\n// this is a comment\n\"name\": \"test\"\n}")
	expected := "{\n\n\"name\": \"test\"\n}"
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_BlockComment(t *testing.T) {
	input := []byte(`{"name": /* inline */ "test"}`)
	expected := `{"name":  "test"}`
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_MultiLineBlockComment(t *testing.T) {
	input := []byte("{\n/* multi\nline\ncomment */\n\"name\": \"test\"\n}")
	expected := "{\n\n\"name\": \"test\"\n}"
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_CommentInsideString_Preserved(t *testing.T) {
	input := []byte(`{"url": "https://example.com", "note": "use // carefully"}`)
	got := string(StripComments(input))
	if got != string(input) {
		t.Errorf("comment inside string should be preserved, got %s", got)
	}
}

func TestStripComments_BlockCommentInsideString_Preserved(t *testing.T) {
	input := []byte(`{"pattern": "/* not a comment */"}`)
	got := string(StripComments(input))
	if got != string(input) {
		t.Errorf("block comment inside string should be preserved, got %s", got)
	}
}

func TestStripComments_EscapedQuotes(t *testing.T) {
	input := []byte(`{"msg": "say \"hello\" // world"}`)
	got := string(StripComments(input))
	if got != string(input) {
		t.Errorf("escaped quotes should be handled, got %s", got)
	}
}

func TestStripComments_TrailingComma_Array(t *testing.T) {
	input := []byte(`[1, 2, 3,]`)
	expected := `[1, 2, 3]`
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_TrailingComma_Object(t *testing.T) {
	input := []byte(`{"a": 1, "b": 2,}`)
	expected := `{"a": 1, "b": 2}`
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_TrailingComma_WithWhitespace(t *testing.T) {
	input := []byte("[1, 2, 3,  \n  ]")
	expected := "[1, 2, 3  \n  ]"
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_MultipleTrailingCommas(t *testing.T) {
	input := []byte(`[1,,,]`)
	expected := `[1]`
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_UnterminatedBlockComment(t *testing.T) {
	input := []byte(`{"name": "test" /* unterminated`)
	expected := `{"name": "test" `
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_CommentAndTrailingComma(t *testing.T) {
	input := []byte("{\n  \"a\": 1, // comment\n  \"b\": 2,\n}")
	expected := "{\n  \"a\": 1, \n  \"b\": 2\n}"
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStripComments_CommaInsideString_Preserved(t *testing.T) {
	input := []byte(`{"msg": "hello, world,"}`)
	got := string(StripComments(input))
	if got != string(input) {
		t.Errorf("comma inside string should be preserved, got %s", got)
	}
}

func TestStripComments_NestedBlockCommentEndInsideString(t *testing.T) {
	input := []byte(`{"pattern": "end */ here"}`)
	got := string(StripComments(input))
	if got != string(input) {
		t.Errorf("*/ inside string should be preserved, got %s", got)
	}
}

func TestStripComments_URLInString(t *testing.T) {
	input := []byte(`{"url": "https://example.com/path"}`)
	got := string(StripComments(input))
	if got != string(input) {
		t.Errorf("URL in string should be preserved, got %s", got)
	}
}

func TestStripComments_ComplexJSONC(t *testing.T) {
	input := []byte(`{
  // Program name
  "name": "test",
  /* Version info */
  "go_json": "1",
  "steps": [
    {"let": "x", "value": 42}, // inline comment
    {"let": "url", "value": "https://example.com"},
    {"return": "x"},
  ]
}`)
	expected := `{
  
  "name": "test",
  
  "go_json": "1",
  "steps": [
    {"let": "x", "value": 42}, 
    {"let": "url", "value": "https://example.com"},
    {"return": "x"}
  ]
}`
	got := string(StripComments(input))
	if got != expected {
		t.Errorf("complex JSONC failed.\nexpected:\n%s\ngot:\n%s", expected, got)
	}
}
