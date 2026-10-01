package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	gh "github-tui/internal/github"
)

func (m *Model) clearPRDiff() {
	m.prDiffFiles = nil
	m.prDiffFileIdx = 0
	m.prDiffLineCursor = 0
	m.prDiffScrollOffset = 0
	m.prDiffPanelOpen = false
}

func (m *Model) prDiffLineCursorDown() {
	if len(m.prDiffFiles) == 0 {
		return
	}
	f := m.prDiffFiles[m.prDiffFileIdx]
	if m.prDiffLineCursor < len(f.Lines)-1 {
		m.prDiffLineCursor++
	} else if m.prDiffFileIdx < len(m.prDiffFiles)-1 {
		m.prDiffFileIdx++
		m.prDiffLineCursor = 0
		m.prDiffScrollOffset = 0
	}
}

func (m *Model) prDiffLineCursorUp() {
	if len(m.prDiffFiles) == 0 {
		return
	}
	if m.prDiffLineCursor > 0 {
		m.prDiffLineCursor--
	} else if m.prDiffFileIdx > 0 {
		m.prDiffFileIdx--
		m.prDiffScrollOffset = 0
		if n := len(m.prDiffFiles[m.prDiffFileIdx].Lines); n > 0 {
			m.prDiffLineCursor = n - 1
		}
	}
}

func (m *Model) prDiffNextHunk() {
	if len(m.prDiffFiles) == 0 {
		return
	}
	f := m.prDiffFiles[m.prDiffFileIdx]
	for i := m.prDiffLineCursor + 1; i < len(f.Lines); i++ {
		if f.Lines[i].Type == "hunk" {
			m.prDiffLineCursor = i
			return
		}
	}
}

func (m *Model) prDiffPrevHunk() {
	if len(m.prDiffFiles) == 0 {
		return
	}
	f := m.prDiffFiles[m.prDiffFileIdx]
	for i := m.prDiffLineCursor - 1; i >= 0; i-- {
		if f.Lines[i].Type == "hunk" {
			m.prDiffLineCursor = i
			return
		}
	}
}

func (m *Model) updatePRDiffScroll() {
	if len(m.prDiffFiles) == 0 {
		return
	}
	h := m.bodyHeight()
	startIdx := m.prDiffFileIdx - 1
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + 3
	if endIdx > len(m.prDiffFiles) {
		endIdx = len(m.prDiffFiles)
		startIdx = endIdx - 3
		if startIdx < 0 {
			startIdx = 0
		}
	}
	diffHeight := h - (4 + (endIdx - startIdx))
	if diffHeight < 1 {
		diffHeight = 1
	}
	if m.prDiffLineCursor < m.prDiffScrollOffset {
		m.prDiffScrollOffset = m.prDiffLineCursor
	} else if m.prDiffLineCursor >= m.prDiffScrollOffset+diffHeight {
		m.prDiffScrollOffset = m.prDiffLineCursor - diffHeight + 1
	}
	if m.prDiffScrollOffset < 0 {
		m.prDiffScrollOffset = 0
	}
}

func (m Model) splitPaneWidths() (leftW, rightW int) {
	leftW = m.width * 2 / 5
	if leftW < 20 {
		leftW = 20
	}
	rightW = m.width - leftW - 1
	if rightW < 20 {
		rightW = 20
		leftW = m.width - rightW - 1
		if leftW < 20 {
			leftW = 20
		}
	}
	return leftW, rightW
}

func (m Model) viewPRDetailSplit() string {
	bodyH := m.bodyHeight()
	leftW, rightW := m.splitPaneWidths()

	m.clampDetail()
	height := bodyH
	start := m.detailScroll
	var leftLines []string
	for i := 0; i < height; i++ {
		idx := start + i
		if idx >= len(m.detailLines) {
			leftLines = append(leftLines, "")
			continue
		}
		leftLines = append(leftLines, ansi.Truncate(m.detailLines[idx], leftW, ""))
	}
	left := lipgloss.NewStyle().Width(leftW).Height(bodyH).MaxHeight(bodyH).Render(strings.Join(leftLines, "\n"))

	sepContent := strings.Repeat("│\n", bodyH)
	if bodyH > 0 {
		sepContent = sepContent[:len(sepContent)-1]
	}
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(sepContent)

	right := lipgloss.NewStyle().Width(rightW).Height(bodyH).MaxHeight(bodyH).Render(
		viewDiffFilesPanel(m.prDiffFiles, m.prDiffFileIdx, m.prDiffLineCursor, m.prDiffScrollOffset, false, rightW, bodyH),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

func viewDiffFilesPanel(files []*gh.DiffFile, fileIdx, lineCursor, scrollOffset int, loading bool, w, h int) string {
	var lines []string
	if loading {
		return strings.Join([]string{
			subtitleStyle.Render("  Changes"),
			"",
			dimStyle.Render("  Loading diffs..."),
		}, "\n")
	}
	if len(files) == 0 {
		return strings.Join([]string{
			subtitleStyle.Render("  Changes"),
			"",
			dimStyle.Render("  No files changed or no diff found."),
		}, "\n")
	}
	if fileIdx < 0 {
		fileIdx = 0
	}
	if fileIdx >= len(files) {
		fileIdx = len(files) - 1
	}

	headerLine := subtitleStyle.Render("  Changes ") +
		dimStyle.Render(fmt.Sprintf("(%d file(s))  n/p=file, J/K=hunk", len(files)))
	lines = append(lines, headerLine)

	startIdx := fileIdx - 1
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + 3
	if endIdx > len(files) {
		endIdx = len(files)
		startIdx = endIdx - 3
		if startIdx < 0 {
			startIdx = 0
		}
	}

	var fileTabs []string
	for i := startIdx; i < endIdx; i++ {
		f := files[i]
		name := f.NewPath
		counts := diffCounts(f)
		limit := w - 7 - len(counts)
		if limit < 35 {
			limit = 35
		}
		name = truncatePath(name, limit)
		label := fmt.Sprintf("%s %s", counts, name)
		if i == fileIdx {
			fileTabs = append(fileTabs, accentStyle.Render(" ▶ "+label))
		} else {
			fileTabs = append(fileTabs, dimStyle.Render("   "+label))
		}
	}
	lines = append(lines, strings.Join(fileTabs, "\n"))
	lines = append(lines, lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", max(2, w-2))))

	tabsLen := endIdx - startIdx
	diffHeight := h - (4 + tabsLen)
	if diffHeight < 1 {
		diffHeight = 1
	}

	f := files[fileIdx]
	renderedCount := 0
	if len(f.Lines) == 0 {
		lines = append(lines, dimStyle.Render("  (diff unavailable — file is too large or collapsed)"))
		renderedCount++
	}
	for i := scrollOffset; i < len(f.Lines) && renderedCount < diffHeight; i++ {
		dl := f.Lines[i]
		selected := i == lineCursor
		content := dl.Content
		avail := w - 5
		if avail < 1 {
			avail = 1
		}
		if lipgloss.Width(content) > avail {
			content = ansi.Truncate(content, avail-1, "…")
		}
		var rendered string
		switch dl.Type {
		case "added":
			st := lipgloss.NewStyle().Foreground(colorSuccess)
			if selected {
				st = st.Background(colorBgHover).Bold(true)
			}
			rendered = st.Render("▶ " + content)
		case "removed":
			st := lipgloss.NewStyle().Foreground(colorError)
			if selected {
				st = st.Background(colorBgHover).Bold(true)
			}
			rendered = st.Render("▶ " + content)
		case "hunk":
			st := lipgloss.NewStyle().Foreground(colorInfo).Italic(true)
			if selected {
				st = st.Background(colorBgHover).Bold(true)
			}
			rendered = st.Render("  " + content)
		default:
			st := lipgloss.NewStyle().Foreground(colorTextDim)
			if selected {
				st = st.Background(colorBgHover)
			}
			rendered = st.Render("  " + content)
		}
		lines = append(lines, rendered)
		renderedCount++
	}
	return strings.Join(lines, "\n")
}
