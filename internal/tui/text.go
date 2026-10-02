package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	urlRE      = regexp.MustCompile(`https?://[^\s<>\])}"']+`)
	keyRE      = regexp.MustCompile(`\b[A-Z][A-Z0-9]+-\d+\b`)
	ansiRE     = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	mdInlineRE = regexp.MustCompile(
		"`([^`]+)`" +
			`|\[([^\]]+)\]\(([^)]+)\)` +
			`|!\[([^\]]*)\]\([^)]+\)` +
			`|~~(.+?)~~` +
			`|\*\*(.+?)\*\*` +
			`|__(.+?)__` +
			`|\*([^*]+)\*` +
			`|https?://[^\s<>\])}"']+`,
	)
	mdFenceRE   = regexp.MustCompile("^\\s*```")
	mdHeadingRE = regexp.MustCompile(`^\s{0,3}(#{1,6})\s+(.*)$`)
	mdListRE    = regexp.MustCompile(`^(\s*)([-*+]|\d+\.)\s+(.*)$`)
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

func markdownToStyled(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out []string
	inFence := false
	for _, line := range strings.Split(text, "\n") {
		if mdFenceRE.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			out = append(out, dimStyle.Render(line))
			continue
		}
		if m := mdHeadingRE.FindStringSubmatch(line); m != nil {
			out = append(out, boldStyle.Render(styleInline(strings.TrimSpace(m[2]))))
			continue
		}
		if m := mdListRE.FindStringSubmatch(line); m != nil {
			out = append(out, m[1]+"• "+styleInline(m[3]))
			continue
		}
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, styleInline(line))
	}
	return strings.Join(out, "\n")
}

func styleInline(s string) string {
	return applyMDInline(s, func(plain string) string { return plain })
}

func styleSystemNote(s string) string {
	return applyMDInline(s, func(plain string) string {
		return dimStyle.Italic(true).Render(plain)
	})
}

func applyMDInline(s string, plain func(string) string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range mdInlineRE.FindAllStringSubmatchIndex(s, -1) {
		if m[0] > last {
			b.WriteString(plain(s[last:m[0]]))
		}
		switch {
		case m[2] != -1: // `code`
			b.WriteString(accentStyle.Render(s[m[2]:m[3]]))
		case m[4] != -1: // [text](url)
			b.WriteString(infoStyle.Render(s[m[4]:m[5]]))
			b.WriteString(dimStyle.Render(" (" + s[m[6]:m[7]] + ")"))
		case m[8] != -1: // ![alt](url)
			alt := strings.TrimSpace(s[m[8]:m[9]])
			if alt == "" {
				alt = "image"
			}
			b.WriteString(dimStyle.Render(alt))
		case m[10] != -1: // ~~strike~~
			b.WriteString(lipgloss.NewStyle().Strikethrough(true).Foreground(colorTextDim).Render(s[m[10]:m[11]]))
		case m[12] != -1: // **bold**
			b.WriteString(boldStyle.Render(s[m[12]:m[13]]))
		case m[14] != -1: // __bold__
			b.WriteString(boldStyle.Render(s[m[14]:m[15]]))
		case m[16] != -1: // *italic*
			b.WriteString(lipgloss.NewStyle().Italic(true).Foreground(colorText).Render(s[m[16]:m[17]]))
		default: // bare URL
			b.WriteString(infoStyle.Render(s[m[0]:m[1]]))
		}
		last = m[1]
	}
	if last < len(s) {
		b.WriteString(plain(s[last:]))
	}
	return b.String()
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
