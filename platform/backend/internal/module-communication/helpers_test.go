package communication

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateStringKeepsLimitIncludingEllipsis(t *testing.T) {
	result := truncateString(strings.Repeat("x", 140), 128)
	if got := utf8.RuneCountInString(result); got != 128 {
		t.Fatalf("expected 128 runes, got %d", got)
	}
	if !strings.HasSuffix(result, "...") {
		t.Fatalf("expected ellipsis, got %q", result)
	}
}

func TestBuildHTMLDocumentDOCX(t *testing.T) {
	data := buildHTMLDocumentDOCX("Test title", "<p>First paragraph</p><p>Second paragraph</p>")
	if len(data) == 0 || !bytes.HasPrefix(data, []byte("PK")) {
		t.Fatal("expected a DOCX zip payload")
	}
}
