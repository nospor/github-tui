package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorBg        = lipgloss.Color("#0d1117")
	colorBgPanel   = lipgloss.Color("#161b22")
	colorBgHover   = lipgloss.Color("#1f2937")
	colorBorder    = lipgloss.Color("#30363d")
	colorAccent    = lipgloss.Color("#7c3aed")
	colorAccentAlt = lipgloss.Color("#a78bfa")
	colorSuccess   = lipgloss.Color("#22c55e")
	colorWarning   = lipgloss.Color("#f59e0b")
	colorError     = lipgloss.Color("#ef4444")
	colorInfo      = lipgloss.Color("#3b82f6")
	colorMuted     = lipgloss.Color("#6b7280")
	colorText      = lipgloss.Color("#e2e8f0")
	colorTextDim   = lipgloss.Color("#94a3b8")
	colorGold      = lipgloss.Color("#f59e0b")
	colorTeal      = lipgloss.Color("#14b8a6")
	colorTitleFg   = lipgloss.Color("#ffffff")
)

var (
	baseStyle       lipgloss.Style
	titleBarStyle   lipgloss.Style
	subtitleStyle   lipgloss.Style
	mutedStyle      lipgloss.Style
	dimStyle        lipgloss.Style
	boldStyle       lipgloss.Style
	successStyle    lipgloss.Style
	warningStyle    lipgloss.Style
	errorStyle      lipgloss.Style
	infoStyle       lipgloss.Style
	accentStyle     lipgloss.Style
	selectedStyle   lipgloss.Style
	normalItemStyle lipgloss.Style
	dialogStyle     lipgloss.Style
)

func init() {
	InitTheme("catppuccin")
}

// InitTheme applies the catppuccin or teams palette.
func InitTheme(themeName string) {
	switch strings.ToLower(strings.TrimSpace(themeName)) {
	case "teams":
		colorBg = lipgloss.Color("#202020")
		colorBgPanel = lipgloss.Color("#303030")
		colorBgHover = lipgloss.Color("#404040")
		colorBorder = lipgloss.Color("#00d75f")
		colorAccent = lipgloss.Color("#00d75f")
		colorAccentAlt = lipgloss.Color("#00d7d7")
		colorSuccess = lipgloss.Color("#00d75f")
		colorWarning = lipgloss.Color("#ffd700")
		colorError = lipgloss.Color("#ff4444")
		colorInfo = lipgloss.Color("#00d7d7")
		colorMuted = lipgloss.Color("#888888")
		colorText = lipgloss.Color("#ffffff")
		colorTextDim = lipgloss.Color("#888888")
		colorGold = lipgloss.Color("#ffd700")
		colorTeal = lipgloss.Color("#00d7d7")
		colorTitleFg = lipgloss.Color("#202020")
	default:
		colorBg = lipgloss.Color("#0d1117")
		colorBgPanel = lipgloss.Color("#161b22")
		colorBgHover = lipgloss.Color("#1f2937")
		colorBorder = lipgloss.Color("#30363d")
		colorAccent = lipgloss.Color("#7c3aed")
		colorAccentAlt = lipgloss.Color("#a78bfa")
		colorSuccess = lipgloss.Color("#22c55e")
		colorWarning = lipgloss.Color("#f59e0b")
		colorError = lipgloss.Color("#ef4444")
		colorInfo = lipgloss.Color("#3b82f6")
		colorMuted = lipgloss.Color("#6b7280")
		colorText = lipgloss.Color("#e2e8f0")
		colorTextDim = lipgloss.Color("#94a3b8")
		colorGold = lipgloss.Color("#f59e0b")
		colorTeal = lipgloss.Color("#14b8a6")
		colorTitleFg = lipgloss.Color("#ffffff")
	}

	baseStyle = lipgloss.NewStyle().Background(colorBg).Foreground(colorText)
	titleBarStyle = lipgloss.NewStyle().Background(colorAccent).Foreground(colorTitleFg).Bold(true).Padding(0, 2)
	subtitleStyle = lipgloss.NewStyle().Foreground(colorAccentAlt).Bold(true)
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)
	dimStyle = lipgloss.NewStyle().Foreground(colorTextDim)
	boldStyle = lipgloss.NewStyle().Bold(true).Foreground(colorText)
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	warningStyle = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	infoStyle = lipgloss.NewStyle().Foreground(colorInfo).Bold(true)
	accentStyle = lipgloss.NewStyle().Foreground(colorAccentAlt).Bold(true)
	selectedStyle = lipgloss.NewStyle().Background(colorBgHover).Foreground(colorAccentAlt).Bold(true)
	normalItemStyle = lipgloss.NewStyle().Foreground(colorText)
	dialogStyle = lipgloss.NewStyle().
		Background(colorBgPanel).
		Foreground(colorText).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2)
}

func statusBadge(status string) string {
	s := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	switch status {
	case "opened", "open":
		return s.Background(colorSuccess).Foreground(lipgloss.Color("#000")).Render(" ● " + status + " ")
	case "merged":
		return s.Background(colorAccent).Foreground(lipgloss.Color("#fff")).Render(" ⎇ " + status + " ")
	case "closed":
		return s.Background(colorError).Foreground(lipgloss.Color("#fff")).Render(" ✕ " + status + " ")
	case "running":
		return s.Background(colorInfo).Foreground(lipgloss.Color("#fff")).Render(" ▶ " + status + " ")
	case "success":
		return s.Background(colorSuccess).Foreground(lipgloss.Color("#000")).Render(" ✓ " + status + " ")
	case "failed":
		return s.Background(colorError).Foreground(lipgloss.Color("#fff")).Render(" ✗ " + status + " ")
	case "pending":
		return s.Background(colorGold).Foreground(lipgloss.Color("#000")).Render(" … " + status + " ")
	case "canceled", "cancelled":
		return s.Background(colorMuted).Foreground(lipgloss.Color("#fff")).Render(" ⊘ " + status + " ")
	case "skipped":
		return s.Background(colorMuted).Foreground(lipgloss.Color("#fff")).Render(" » " + status + " ")
	case "draft":
		return s.Background(colorWarning).Foreground(lipgloss.Color("#000")).Render(" ✎ " + status + " ")
	default:
		return s.Background(colorMuted).Foreground(lipgloss.Color("#fff")).Render(" " + status + " ")
	}
}

func padStatusBadge(badge string, width int) string {
	w := lipgloss.Width(badge)
	if w >= width {
		return badge
	}
	return badge + strings.Repeat(" ", width-w)
}

func tabStyle(label string, active bool) string {
	if active {
		return lipgloss.NewStyle().Foreground(colorAccentAlt).Bold(true).Underline(true).Padding(0, 2).Render(label)
	}
	return lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 2).Render(label)
}

func keyHint(key, desc string) string {
	k := lipgloss.NewStyle().Background(colorBgHover).Foreground(colorAccentAlt).Padding(0, 1).Render(key)
	d := lipgloss.NewStyle().Foreground(colorMuted).Render(" " + desc)
	return k + d
}
