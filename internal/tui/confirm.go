package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jordan-simonovski/helmver/internal/chart"
)

// Changeset captures the user's choices for a single chart.
type Changeset struct {
	Chart   *chart.Chart
	Bump    string
	NewVer  string
	Message string
}

// confirmModel shows a summary and asks for y/n confirmation.
type confirmModel struct {
	changesets []Changeset
	confirmed  bool
	aborted    bool
}

func newConfirmModel(cs []Changeset) confirmModel {
	return confirmModel{changesets: cs}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (confirmModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			m.confirmed = true
		case "n", "N", "q":
			m.aborted = true
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	nameStyle := lipgloss.NewStyle().Bold(true)
	hintStyle := lipgloss.NewStyle().Faint(true)
	msgStyle := lipgloss.NewStyle().Faint(true).PaddingLeft(4)

	b.WriteString(titleStyle.Render("Summary"))
	b.WriteString("\n\n")

	for _, cs := range m.changesets {
		arrow := okStyle.Render("->")
		b.WriteString(fmt.Sprintf("  %s  %s %s %s\n",
			nameStyle.Render(cs.Chart.Name),
			cs.Chart.Version,
			arrow,
			okStyle.Render(cs.NewVer),
		))
		// Show first line of message as preview
		lines := strings.SplitN(cs.Message, "\n", 2)
		preview := lines[0]
		if len(preview) > 60 {
			preview = preview[:57] + "..."
		}
		b.WriteString(msgStyle.Render(preview))
		b.WriteString("\n\n")
	}

	b.WriteString(hintStyle.Render("[y] apply  [n] abort"))
	b.WriteString("\n")

	return b.String()
}
