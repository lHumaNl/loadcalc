// Package tui implements the interactive terminal user interface.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"loadcalc/internal/engine"
	"loadcalc/internal/output"
	"loadcalc/internal/tui/views"
)

// ViewMode represents the current TUI view.
type ViewMode int

const (
	ViewSummary  ViewMode = iota
	ViewSteps    ViewMode = iota
	ViewScenario ViewMode = iota
	ViewTimeline ViewMode = iota
)

// ExportMsg signals that an XLSX export was triggered.
type ExportMsg struct {
	Err error
}

// Model is the bubbletea model for the TUI application.
type Model struct {
	OutputPath  string
	StatusMsg   string
	Results     engine.CalculationResults
	CurrentView ViewMode
	SelectedRow int
	ColumnGroup views.ColumnGroup
}

// NewModel creates a new TUI model with the given calculation results.
func NewModel(results engine.CalculationResults, outputPath string) Model {
	return Model{
		Results:     results,
		CurrentView: ViewSummary,
		OutputPath:  outputPath,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ExportMsg:
		if msg.Err != nil {
			m.StatusMsg = fmt.Sprintf("Export failed: %v", msg.Err)
		} else {
			m.StatusMsg = fmt.Sprintf("Exported to %s", m.OutputPath)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "esc":
		return m.handleEsc()
	case "up", "k":
		return m.handleNavigateUp()
	case "down", "j":
		return m.handleNavigateDown()
	case "left":
		return m.handleNavigateLeft()
	case "right":
		return m.handleNavigateRight()
	case "enter":
		return m.handleEnter()
	case "tab":
		return m.handleTab()
	case "s", "S":
		return m.handleSwitchToSteps()
	case "t", "T":
		return m.handleSwitchToTimeline()
	case "e", "E":
		return m, m.exportXLSX()
	}
	return m, nil
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	if m.CurrentView == ViewSummary {
		return m, tea.Quit
	}
	m.CurrentView = ViewSummary
	m.StatusMsg = ""
	return m, nil
}

func (m Model) handleNavigateUp() (tea.Model, tea.Cmd) {
	if (m.CurrentView == ViewSummary || m.CurrentView == ViewSteps) && m.SelectedRow > 0 {
		m.SelectedRow--
	}
	return m, nil
}

func (m Model) handleNavigateDown() (tea.Model, tea.Cmd) {
	maxRow := len(m.Results.ScenarioResults) - 1
	if (m.CurrentView == ViewSummary || m.CurrentView == ViewSteps) && m.SelectedRow < maxRow {
		m.SelectedRow++
	}
	return m, nil
}

func (m Model) handleNavigateLeft() (tea.Model, tea.Cmd) {
	if m.CurrentView == ViewSteps && m.SelectedRow > 0 {
		m.SelectedRow--
	}
	return m, nil
}

func (m Model) handleNavigateRight() (tea.Model, tea.Cmd) {
	if m.CurrentView == ViewSteps {
		maxRow := len(m.Results.ScenarioResults) - 1
		if m.SelectedRow < maxRow {
			m.SelectedRow++
		}
	}
	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.CurrentView == ViewSummary {
		m.CurrentView = ViewScenario
		return m, nil
	}
	return m, nil
}

func (m Model) handleTab() (tea.Model, tea.Cmd) {
	if m.CurrentView == ViewSummary {
		if m.ColumnGroup == views.ColumnGroupBase {
			m.ColumnGroup = views.ColumnGroupTool
		} else {
			m.ColumnGroup = views.ColumnGroupBase
		}
	}
	return m, nil
}

func (m Model) handleSwitchToSteps() (tea.Model, tea.Cmd) {
	if m.CurrentView != ViewSteps {
		m.CurrentView = ViewSteps
		m.StatusMsg = ""
	}
	return m, nil
}

func (m Model) handleSwitchToTimeline() (tea.Model, tea.Cmd) {
	if m.CurrentView != ViewTimeline {
		m.CurrentView = ViewTimeline
		m.StatusMsg = ""
	}
	return m, nil
}

func (m Model) exportXLSX() tea.Cmd {
	return func() tea.Msg {
		if m.OutputPath == "" {
			return ExportMsg{Err: fmt.Errorf("no output path specified")}
		}
		w := &output.XLSXWriter{}
		err := w.Write(m.Results, m.OutputPath)
		return ExportMsg{Err: err}
	}
}

// View implements tea.Model.
func (m Model) View() string {
	var s string
	switch m.CurrentView {
	case ViewSummary:
		s = views.RenderSummary(m.Results, m.SelectedRow, m.ColumnGroup)
	case ViewSteps:
		s = views.RenderSteps(m.Results, m.SelectedRow)
	case ViewScenario:
		s = views.RenderScenario(m.Results, m.SelectedRow)
	case ViewTimeline:
		s = views.RenderTimeline(m.Results)
	}

	if m.StatusMsg != "" {
		s += "\n" + m.StatusMsg
	}
	return s
}

// Run starts the TUI application.
func Run(results engine.CalculationResults, outputPath string) error {
	m := NewModel(results, outputPath)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
