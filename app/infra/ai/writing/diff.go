package writing

import (
	"strings"
)

const maxDiffMatrixCells = 100000

// BuildUnifiedDiff creates a deterministic line-oriented diff without
// modifying or persisting either input. For unusually large inputs it falls
// back to a bounded whole-content replacement so a preview request cannot
// allocate an unbounded LCS matrix.
func BuildUnifiedDiff(before, after string) string {
	beforeLines := diffLines(before)
	afterLines := diffLines(after)
	var builder strings.Builder
	builder.WriteString("--- original\n+++ preview\n@@\n")
	if before == after {
		for _, line := range beforeLines {
			builder.WriteString(" ")
			builder.WriteString(line)
			builder.WriteByte('\n')
		}
		return builder.String()
	}
	if len(beforeLines) > 0 && len(afterLines) > maxDiffMatrixCells/len(beforeLines) {
		return appendWholeReplacement(&builder, beforeLines, afterLines)
	}

	lcs := make([][]int, len(beforeLines)+1)
	for row := range lcs {
		lcs[row] = make([]int, len(afterLines)+1)
	}
	for row := len(beforeLines) - 1; row >= 0; row-- {
		for column := len(afterLines) - 1; column >= 0; column-- {
			if beforeLines[row] == afterLines[column] {
				lcs[row][column] = lcs[row+1][column+1] + 1
				continue
			}
			if lcs[row+1][column] >= lcs[row][column+1] {
				lcs[row][column] = lcs[row+1][column]
			} else {
				lcs[row][column] = lcs[row][column+1]
			}
		}
	}

	row, column := 0, 0
	for row < len(beforeLines) && column < len(afterLines) {
		switch {
		case beforeLines[row] == afterLines[column]:
			writeDiffLine(&builder, ' ', beforeLines[row])
			row++
			column++
		case lcs[row+1][column] >= lcs[row][column+1]:
			writeDiffLine(&builder, '-', beforeLines[row])
			row++
		default:
			writeDiffLine(&builder, '+', afterLines[column])
			column++
		}
	}
	for ; row < len(beforeLines); row++ {
		writeDiffLine(&builder, '-', beforeLines[row])
	}
	for ; column < len(afterLines); column++ {
		writeDiffLine(&builder, '+', afterLines[column])
	}
	return builder.String()
}

func diffLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

func writeDiffLine(builder *strings.Builder, prefix byte, line string) {
	builder.WriteByte(prefix)
	builder.WriteString(line)
	builder.WriteByte('\n')
}

func appendWholeReplacement(builder *strings.Builder, before, after []string) string {
	for _, line := range before {
		writeDiffLine(builder, '-', line)
	}
	for _, line := range after {
		writeDiffLine(builder, '+', line)
	}
	return builder.String()
}
