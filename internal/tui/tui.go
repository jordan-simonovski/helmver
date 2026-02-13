package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jordan-simonovski/helmver/internal/chart"
)

// phase tracks the TUI state machine.
type phase int

const (
	phaseSelectCharts phase = iota
	phaseSelectBump
	phaseInputMessage
	phaseConfirm
	phaseDone
)

// Model is the top-level bubbletea model that orchestrates all phases.
type Model struct {
	phase phase

	// Sub-models
	selectCharts selectChartsModel
	selectBump   selectBumpModel
	inputMessage inputMessageModel
	confirm      confirmModel

	// Accumulated state
	allCharts      []*chart.Chart
	selectedCharts []*chart.Chart
	changesets     []Changeset
	currentIdx     int // index into selectedCharts for bump/message flow

	// Result
	Applied bool
	Aborted bool
	Err     error
}

// New creates the top-level TUI model with all discovered charts.
// Charts with Stale=true are highlighted; others are dimmed but still selectable.
func New(charts []*chart.Chart) Model {
	return Model{
		phase:        phaseSelectCharts,
		allCharts:    charts,
		selectCharts: newSelectChartsModel(charts),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global quit
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c":
			m.Aborted = true
			return m, tea.Quit
		case "q":
			// Only quit on q if we're not in the text input phase
			if m.phase != phaseInputMessage {
				m.Aborted = true
				return m, tea.Quit
			}
		}
	}

	switch m.phase {
	case phaseSelectCharts:
		return m.updateSelectCharts(msg)
	case phaseSelectBump:
		return m.updateSelectBump(msg)
	case phaseInputMessage:
		return m.updateInputMessage(msg)
	case phaseConfirm:
		return m.updateConfirm(msg)
	}

	return m, nil
}

func (m Model) updateSelectCharts(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.selectCharts, cmd = m.selectCharts.Update(msg)

	if m.selectCharts.done {
		m.selectedCharts = m.selectCharts.SelectedCharts()
		m.changesets = nil
		m.currentIdx = 0
		m.phase = phaseSelectBump
		m.selectBump = newSelectBumpModel(m.selectedCharts[0])
	}

	return m, cmd
}

func (m Model) updateSelectBump(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.selectBump, cmd = m.selectBump.Update(msg)

	if m.selectBump.done {
		c := m.selectedCharts[m.currentIdx]
		bump := m.selectBump.selected
		m.phase = phaseInputMessage
		m.inputMessage = newInputMessageModel(c, bump)
		return m, m.inputMessage.Init()
	}

	return m, cmd
}

func (m Model) updateInputMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.inputMessage, cmd = m.inputMessage.Update(msg)

	if m.inputMessage.done {
		c := m.selectedCharts[m.currentIdx]
		bump := m.inputMessage.bump
		newVer, err := chart.BumpVersion(c.Version, bump)
		if err != nil {
			m.Err = err
			return m, tea.Quit
		}

		m.changesets = append(m.changesets, Changeset{
			Chart:   c,
			Bump:    bump,
			NewVer:  newVer,
			Message: m.inputMessage.message,
		})

		m.currentIdx++
		if m.currentIdx < len(m.selectedCharts) {
			// More charts to process
			m.phase = phaseSelectBump
			m.selectBump = newSelectBumpModel(m.selectedCharts[m.currentIdx])
		} else {
			// All done, show confirmation
			m.phase = phaseConfirm
			m.confirm = newConfirmModel(m.changesets)
		}
	}

	return m, cmd
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)

	if m.confirm.confirmed {
		m.Applied = true
		return m, tea.Quit
	}
	if m.confirm.aborted {
		m.Aborted = true
		return m, tea.Quit
	}

	return m, cmd
}

func (m Model) View() string {
	switch m.phase {
	case phaseSelectCharts:
		return m.selectCharts.View()
	case phaseSelectBump:
		return m.selectBump.View()
	case phaseInputMessage:
		return m.inputMessage.View()
	case phaseConfirm:
		return m.confirm.View()
	case phaseDone:
		return ""
	}
	return ""
}

// Changesets returns the accumulated changesets after the TUI exits.
func (m Model) Changesets() []Changeset {
	return m.changesets
}

// Run starts the TUI and returns the result.
// Pass all discovered charts; staleness is indicated by each chart's Stale field.
func Run(charts []*chart.Chart) ([]Changeset, error) {
	m := New(charts)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	final := result.(Model)
	if final.Err != nil {
		return nil, final.Err
	}
	if final.Aborted {
		return nil, nil
	}
	if final.Applied {
		return final.Changesets(), nil
	}
	return nil, nil
}
