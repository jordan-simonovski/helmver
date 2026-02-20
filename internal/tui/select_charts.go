package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jordan-simonovski/helmver/internal/chart"
)

// selectChartsModel lets the user multi-select from all discovered charts.
// Stale charts are highlighted in blue; unchanged charts are shown in grey
// but remain fully selectable.
type selectChartsModel struct {
	charts   []*chart.Chart
	cursor   int
	selected map[int]bool
	done     bool
}

func newSelectChartsModel(charts []*chart.Chart) selectChartsModel {
	return selectChartsModel{
		charts:   charts,
		selected: make(map[int]bool),
	}
}

func (m selectChartsModel) Init() tea.Cmd {
	return nil
}

func (m selectChartsModel) Update(msg tea.Msg) (selectChartsModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.charts)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
			if !m.selected[m.cursor] {
				delete(m.selected, m.cursor)
			}
		case "a":
			// Toggle all
			if len(m.selected) == len(m.charts) {
				m.selected = make(map[int]bool)
			} else {
				for i := range m.charts {
					m.selected[i] = true
				}
			}
		case "enter":
			if len(m.selected) > 0 {
				m.done = true
			}
		}
	}
	return m, nil
}

func (m selectChartsModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	hintStyle := lipgloss.NewStyle().Faint(true)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	checkedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	// Stale = blue (changed), Clean = grey/dimmed (no changes detected)
	staleNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // blue
	cleanNameStyle := lipgloss.NewStyle().Faint(true)                      // grey
	staleVerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	cleanVerStyle := lipgloss.NewStyle().Faint(true)
	staleTagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	cleanTagStyle := lipgloss.NewStyle().Faint(true).Italic(true)
	pathStyle := lipgloss.NewStyle().Faint(true)

	b.WriteString(titleStyle.Render("Select charts to version bump"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[space] toggle  [a] all  [enter] confirm  [q] quit"))
	b.WriteString("\n\n")

	// Count stale for the legend
	staleCount := 0
	for _, c := range m.charts {
		if c.Stale {
			staleCount++
		}
	}
	cleanCount := len(m.charts) - staleCount

	if staleCount > 0 {
		fmt.Fprintf(&b, "  %s %d changed    %s %d unchanged\n\n",
			staleNameStyle.Render("*"),
			staleCount,
			cleanNameStyle.Render("*"),
			cleanCount,
		)
	} else {
		b.WriteString(cleanTagStyle.Render("  no changes detected (not a git repo or no diffs)"))
		b.WriteString("\n\n")
	}

	for i, c := range m.charts {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		checked := "[ ]"
		if m.selected[i] {
			checked = checkedStyle.Render("[x]")
		}

		var name, ver, tag string
		if c.Stale {
			name = staleNameStyle.Render(c.Name)
			ver = staleVerStyle.Render(fmt.Sprintf("(%s)", c.Version))
			tag = staleTagStyle.Render("changed")
		} else {
			name = cleanNameStyle.Render(c.Name)
			ver = cleanVerStyle.Render(fmt.Sprintf("(%s)", c.Version))
			tag = cleanTagStyle.Render("unchanged")
		}

		path := pathStyle.Render(c.Dir)

		b.WriteString(fmt.Sprintf("%s%s %s %s  %s  %s\n", cursor, checked, name, ver, tag, path))
	}

	return b.String()
}

// SelectedCharts returns the charts the user selected.
func (m selectChartsModel) SelectedCharts() []*chart.Chart {
	var out []*chart.Chart
	for i, c := range m.charts {
		if m.selected[i] {
			out = append(out, c)
		}
	}
	return out
}
