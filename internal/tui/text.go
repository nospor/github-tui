package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	urlRE  = regexp.MustCompile(`https?://[^\s<>\])}"']+`)
	keyRE  = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
	ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
)

func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > width-1 {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func wrapBlock(text string, width int) []string {
	if width < 8 {
		width = 8
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		para = strings.TrimRight(para, " ")
		if strings.TrimSpace(para) == "" {
			lines = append(lines, "")
			continue
		}
		var cur string
		for _, word := range strings.Fields(para) {
			if cur == "" {
				cur = word
				continue
			}
			if lipgloss.Width(cur+" "+word) > width {
				lines = append(lines, cur)
				cur = word
			} else {
				cur += " " + word
			}
		}
		if cur != "" {
			lines = append(lines, cur)
		}
	}
	return lines
}

func findURLs(s string) []string {
	found := urlRE.FindAllString(s, -1)
	for i, raw := range found {
		found[i] = strings.TrimRight(raw, ".,);")
	}
	return found
}

func findIssueKeys(s string) []string {
	return keyRE.FindAllString(s, -1)
}

func plain(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}
