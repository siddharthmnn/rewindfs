package diff

import (
	"fmt"
	"strings"
)

const maxDiffCells = 2_000_000

type edit struct {
	kind byte
	line string
}

// Compare returns a line-oriented diff. The bounded LCS table keeps the
// comparison predictable for large snapshots; very large inputs fall back to
// a compact whole-content replacement instead of allocating an unbounded table.
func Compare(oldContent, newContent string) string {
	if oldContent == newContent {
		return "No changes"
	}

	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	if len(oldLines) == 0 && len(newLines) == 0 {
		return "No changes"
	}

	if len(oldLines) > 0 && len(newLines) > maxDiffCells/len(oldLines) {
		return replacementDiff(oldContent, newContent)
	}

	edits := lineEdits(oldLines, newLines)
	var result strings.Builder
	result.WriteString("--- snapshot\n")
	result.WriteString("+++ snapshot\n")
	result.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(oldLines), len(newLines)))
	for _, item := range edits {
		result.WriteByte(item.kind)
		result.WriteString(item.line)
		result.WriteByte('\n')
	}
	return result.String()
}

func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	if strings.HasSuffix(content, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func lineEdits(oldLines, newLines []string) []edit {
	n, m := len(oldLines), len(newLines)
	lcs := make([][]uint32, n+1)
	for i := range lcs {
		lcs[i] = make([]uint32, m+1)
	}

	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	edits := make([]edit, 0, n+m)
	for i, j := 0, 0; i < n || j < m; {
		switch {
		case i < n && j < m && oldLines[i] == newLines[j]:
			edits = append(edits, edit{kind: ' ', line: oldLines[i]})
			i++
			j++
		case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
			edits = append(edits, edit{kind: '-', line: oldLines[i]})
			i++
		default:
			edits = append(edits, edit{kind: '+', line: newLines[j]})
			j++
		}
	}
	return edits
}

func replacementDiff(oldContent, newContent string) string {
	var result strings.Builder
	result.WriteString("--- snapshot\n+++ snapshot\n")
	for _, line := range splitLines(oldContent) {
		result.WriteString("-")
		result.WriteString(line)
		result.WriteByte('\n')
	}
	for _, line := range splitLines(newContent) {
		result.WriteString("+")
		result.WriteString(line)
		result.WriteByte('\n')
	}
	return result.String()
}
