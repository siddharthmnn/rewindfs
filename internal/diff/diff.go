package diff

import (
	"fmt"
	"strings"
)

func Compare(oldContent, newContent string) string {

	if oldContent == newContent {
		return "No changes"
	}

	var result strings.Builder

	result.WriteString(fmt.Sprintf("- %s\n", oldContent))
	result.WriteString(fmt.Sprintf("+ %s\n", newContent))

	return result.String()
}
