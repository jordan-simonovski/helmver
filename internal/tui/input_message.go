package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jsimonovski/helmver/internal/chart"
)

// inputMessageModel lets the user write a multiline changelog message.
type inputMessageModel struct {
	chart    *chart.Chart
	bump     string
	textarea textarea.Model
	message  string
	done     bool
}

func newInputMessageModel(c *chart.Chart, bump string) inputMessageModel {
	ta := textarea.New()
	ta.Placeholder = "Describe your changes..."
	ta.Focus()
	ta.SetWidth(72)
	ta.SetHeight(6)
	ta.CharLimit = 2000

	return inputMessageModel{
		chart:    c,
		bump:     bump,
		textarea: ta,
	}
}

func (m inputMessageModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m inputMessageModel) Update(msg tea.Msg) (inputMessageModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+d":
			val := strings.TrimSpace(m.textarea.Value())
			if val != "" {
				m.message = val
				m.done = true
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m inputMessageModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	hintStyle := lipgloss.NewStyle().Faint(true)

	newVer, _ := chart.BumpVersion(m.chart.Version, m.bump)

	b.WriteString(titleStyle.Render(fmt.Sprintf("Changelog for %s (%s -> %s)", m.chart.Name, m.chart.Version, newVer)))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[ctrl+d] submit  [q is just a letter here]"))
	b.WriteString("\n\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n")

	return b.String()
}
