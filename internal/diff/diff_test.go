package diff

import "testing"

func TestCompareSameContent(t *testing.T) {

        result := Compare("hello", "hello")

        if result != "No changes" {
                t.Errorf("expected 'No changes', got %s", result)
        }
}
func TestCompareDifferentContent(t *testing.T) {

        result := Compare("hello", "world")

        if result == "No changes" {
                t.Errorf("expected diff output")
        }
}
