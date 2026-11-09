package diff

import (
	"strings"
	"testing"
)

func TestCompareSameContent(t *testing.T) {
	if result := Compare("hello", "hello"); result != "No changes" {
		t.Fatalf("Compare() = %q, want no changes", result)
	}
}

func TestCompareDifferentContent(t *testing.T) {
	result := Compare("hello", "world")
	for _, expected := range []string{"--- snapshot", "+++ snapshot", "-hello", "+world"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("diff %q does not contain %q", result, expected)
		}
	}
}

func TestCompareKeepsUnchangedLinesAsContext(t *testing.T) {
	result := Compare("package main\n\nfunc main() {\n\told()\n}\n", "package main\n\nfunc main() {\n\tnew()\n}\n")
	for _, expected := range []string{" package main", " func main() {", "-\told()", "+\tnew()", " }"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("diff %q does not contain %q", result, expected)
		}
	}
}

func TestCompareHandlesAddedAndRemovedLines(t *testing.T) {
	added := Compare("one\ntwo\n", "one\ntwo\nthree\n")
	if !strings.Contains(added, "+three") || strings.Contains(added, "-three") {
		t.Fatalf("unexpected addition diff: %q", added)
	}

	removed := Compare("one\ntwo\nthree\n", "one\nthree\n")
	if !strings.Contains(removed, "-two") {
		t.Fatalf("unexpected removal diff: %q", removed)
	}
}

func TestCompareHandlesEmptyFiles(t *testing.T) {
	added := Compare("", "first line\n")
	if !strings.Contains(added, "+first line") {
		t.Fatalf("empty-file addition diff = %q", added)
	}
	removed := Compare("first line\n", "")
	if !strings.Contains(removed, "-first line") {
		t.Fatalf("empty-file removal diff = %q", removed)
	}
}

func TestCompareFallsBackForVeryLargeInputs(t *testing.T) {
	oldContent := strings.Repeat("old line\n", 1500)
	newContent := strings.Repeat("new line\n", 1500)
	result := Compare(oldContent, newContent)
	if !strings.Contains(result, "-old line") || !strings.Contains(result, "+new line") {
		t.Fatalf("large diff fallback missing old/new lines")
	}
}
