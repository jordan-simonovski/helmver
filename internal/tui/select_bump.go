package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jordan-simonovski/helmver/internal/chart"
)

var bumpTypes = []string{"patch", "minor", "major"}

// selectBumpModel lets the user pick a semver bump type for a single chart.
type selectBumpModel struct {
	chart    *chart.Chart
	cursor   int
	selected string
	done     bool
}

func newSelectBumpModel(c *chart.Chart) selectBumpModel {
	return selectBumpModel{chart: c}
}

func (m selectBumpModel) Init() tea.Cmd {
	return nil
}

func (m selectBumpModel) Update(msg tea.Msg) (selectBumpModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(bumpTypes)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = bumpTypes[m.cursor]
			m.done = true
		}
	}
	return m, nil
}

func (m selectBumpModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	hintStyle := lipgloss.NewStyle().Faint(true)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	descStyle := lipgloss.NewStyle().Faint(true)

	b.WriteString(titleStyle.Render(fmt.Sprintf("Bump type for %s (%s)", m.chart.Name, m.chart.Version)))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[up/down] navigate  [enter] select"))
	b.WriteString("\n\n")

	descriptions := map[string]string{
		"patch": "bug fixes, no API changes",
		"minor": "new features, backwards compatible",
		"major": "breaking changes",
	}

	for i, bt := range bumpTypes {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		preview, _ := chart.BumpVersion(m.chart.Version, bt)
		desc := descStyle.Render(fmt.Sprintf("- %s (%s -> %s)", descriptions[bt], m.chart.Version, preview))

		label := bt
		if i == m.cursor {
			label = cursorStyle.Render(bt)
		}

		fmt.Fprintf(&b, "%s%s %s\n", cursor, label, desc)
	}

	return b.String()
}
