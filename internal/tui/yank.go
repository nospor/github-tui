package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// clipboardWriteAll is indirected to allow stubbing in tests.
var clipboardWriteAll = clipboard.WriteAll

type yankOption struct {
	Key   string
	Label string
}

func (m Model) currentYankOptions() []yankOption {
	switch {
	case m.detailPR != nil:
		return []yankOption{
			{"i", "PR ID"},
			{"t", "Title"},
			{"d", "Description"},
			{"b", "Head Branch"},
			{"u", "URLs"},
		}
	case m.detailIssue != nil:
		return []yankOption{
			{"i", "Issue ID"},
			{"t", "Title"},
			{"d", "Description"},
			{"u", "URLs"},
		}
	case m.detailRun != nil:
		return []yankOption{
			{"i", "Run ID"},
			{"t", "Name"},
			{"u", "URL"},
		}
	default:
		return nil
	}
}

func (m Model) yankText(option string) string {
	switch {
	case m.detailPR != nil:
		switch option {
		case "i":
			return fmt.Sprintf("#%d", m.detailPR.Number)
		case "t":
			return m.detailPR.Title
		case "d":
			return m.detailPR.Body
		case "b":
			return m.detailPR.Head
		}
	case m.detailIssue != nil:
		switch option {
		case "i":
			return fmt.Sprintf("#%d", m.detailIssue.Number)
		case "t":
			return m.detailIssue.Title
		case "d":
			return m.detailIssue.Body
		}
	case m.detailRun != nil:
		switch option {
		case "i":
			return fmt.Sprintf("#%d", m.detailRun.ID)
		case "t":
			return m.detailRun.Name
		}
	}
	return ""
}

func (m Model) openYank() (tea.Model, tea.Cmd) {
	if len(m.currentYankOptions()) == 0 {
		return m, nil
	}
	m.yankOpen = true
	return m, nil
}

func (m *Model) closeYank() {
	m.yankOpen = false
	m.yankURLSelect = false
	m.yankItems = nil
	m.yankCursor = 0
}

func (m Model) handleYankPopupKey(key string) (tea.Model, tea.Cmd) {
	m.yankOpen = false
	for _, opt := range m.currentYankOptions() {
		if key == opt.Key {
			return m.performYank(opt.Key)
		}
	}
	return m, nil
}

func (m Model) handleYankURLSelectKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		if len(m.yankItems) > 0 {
			m.yankCursor++
			if m.yankCursor >= len(m.yankItems) {
				m.yankCursor = 0
			}
		}
	case "k", "up":
		if len(m.yankItems) > 0 {
			m.yankCursor--
			if m.yankCursor < 0 {
				m.yankCursor = len(m.yankItems) - 1
			}
		}
	case "enter":
		if m.yankCursor >= 0 && m.yankCursor < len(m.yankItems) {
			url := m.yankItems[m.yankCursor].URL
			m.closeYank()
			return m.copyYank(url, "URL")
		}
		m.closeYank()
	default:
		m.closeYank()
	}
	return m, nil
}

func (m Model) performYank(option string) (tea.Model, tea.Cmd) {
	if option == "u" {
		return m.yankURLs()
	}
	text := m.yankText(option)
	if text == "" {
		m.setStatus("Nothing to copy")
		return m, m.scheduleClear()
	}
	label := ""
	for _, opt := range m.currentYankOptions() {
		if opt.Key == option {
			label = strings.ToLower(opt.Label)
			break
		}
	}
	return m.copyYank(text, label)
}

func (m Model) yankURLs() (tea.Model, tea.Cmd) {
	items := m.collectLinks()
	switch len(items) {
	case 0:
		m.setStatus("No URLs found")
		return m, m.scheduleClear()
	case 1:
		return m.copyYank(items[0].URL, "URL")
	default:
		m.yankItems = items
		m.yankCursor = 0
		m.yankURLSelect = true
		return m, nil
	}
}

func (m Model) copyYank(text, label string) (tea.Model, tea.Cmd) {
	if err := clipboardWriteAll(text); err != nil {
		m.setStatus(fmt.Sprintf("Copy failed: %v", err))
		return m, m.scheduleClear()
	}
	if label == "" {
		m.setStatus("Copied to clipboard")
	} else {
		m.setStatus(fmt.Sprintf("Copied %s to clipboard", label))
	}
	return m, m.scheduleClear()
}

func yankBox(rows []string) string {
	return panelStyle.Padding(0, 1).Render(strings.Join(rows, "\n"))
}

func (m Model) viewYankPopup() string {
	rows := []string{subtitleStyle.Render("Yank")}
	for _, opt := range m.currentYankOptions() {
		key := lipgloss.NewStyle().Foreground(colorAccentAlt).Bold(true).Render("[" + opt.Key + "]")
		rows = append(rows, fmt.Sprintf("%s %s", key, dimStyle.Render(opt.Label)))
	}
	return yankBox(rows)
}

func (m Model) viewYankURLSelect() string {
	rows := []string{subtitleStyle.Render("Yank URL")}
	maxW := m.width - 12
	if maxW < 24 {
		maxW = 24
	}
	for idx, item := range m.yankItems {
		display := fit(item.URL, maxW)
		if idx == m.yankCursor {
			rows = append(rows, selectedStyle.Render("▶ "+display))
		} else {
			rows = append(rows, normalItemStyle.Render("  "+display))
		}
	}
	rows = append(rows, "", dimStyle.Render("↑↓ select · Enter copy · Esc cancel"))
	return yankBox(rows)
}

func (m Model) applyYankOverlays(view string) string {
	var popup string
	switch {
	case m.yankOpen:
		popup = m.viewYankPopup()
	case m.yankURLSelect:
		popup = m.viewYankURLSelect()
	default:
		return view
	}
	x := (m.width - 4) - lipgloss.Width(popup)
	if x < 0 {
		x = 0
	}
	return overlay(view, popup, m.width, lipgloss.Height(view), x, 0)
}
