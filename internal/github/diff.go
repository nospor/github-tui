package github

import "fmt"

// DiffLine is one line in a unified diff.
type DiffLine struct {
	OldLine int
	NewLine int
	Type    string // "added", "removed", "context", "hunk"
	Content string
}

// DiffFile is a changed file with optional parsed diff lines.
type DiffFile struct {
	OldPath     string
	NewPath     string
	Added       int
	Deleted     int
	Lines       []DiffLine
	TooLarge    bool
	Collapsed   bool
	Overflow    bool
	NewFile     bool
	DeletedFile bool
	RenamedFile bool
	AMode       string
	BMode       string
}

func parseDiffLines(raw string) []DiffLine {
	var lines []DiffLine
	oldLine, newLine := 0, 0
	for _, l := range splitDiffLines(raw) {
		if len(l) == 0 {
			continue
		}
		switch l[0] {
		case '@':
			var oStart, oLen, nStart, nLen int
			if _, err := fmt.Sscanf(l, "@@ -%d,%d +%d,%d", &oStart, &oLen, &nStart, &nLen); err != nil {
				fmt.Sscanf(l, "@@ -%d +%d", &oStart, &nStart) //nolint
			}
			oldLine = oStart
			newLine = nStart
			lines = append(lines, DiffLine{Type: "hunk", Content: l})
		case '+':
			lines = append(lines, DiffLine{NewLine: newLine, Type: "added", Content: l})
			newLine++
		case '-':
			lines = append(lines, DiffLine{OldLine: oldLine, Type: "removed", Content: l})
			oldLine++
		default:
			lines = append(lines, DiffLine{OldLine: oldLine, NewLine: newLine, Type: "context", Content: l})
			oldLine++
			newLine++
		}
	}
	return lines
}

func splitDiffLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		out = append(out, line)
	}
	return out
}

func mapPatchFile(filename, patch string) *DiffFile {
	f := &DiffFile{NewPath: filename, OldPath: filename}
	if filename == "" {
		f.NewPath = "(unknown)"
		f.OldPath = "(unknown)"
	}
	lines := parseDiffLines(patch)
	for _, l := range lines {
		switch l.Type {
		case "added":
			f.Added++
		case "removed":
			f.Deleted++
		}
	}
	f.Lines = lines
	return f
}
