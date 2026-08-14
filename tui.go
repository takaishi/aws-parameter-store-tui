package awsparameterstoretui

import (
	"context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type viewState int

const (
	stateList viewState = iota
	stateDetail
)

var (
	titleStyle           = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	selectedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	normalStyle          = lipgloss.NewStyle()
	matchedStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Underline(true)
	selectedMatchedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).Underline(true)
	typeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	labelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
)

type parametersLoadedMsg []Parameter

type parameterValueMsg struct {
	name    string
	value   string
	forCopy bool
}

type errMsg struct{ err error }

type listItem struct {
	paramIndex   int
	matchedRunes []int
}

type model struct {
	ctx    context.Context
	client *SSMClient
	region string

	state   viewState
	input   textinput.Model
	spin    spinner.Model
	loading bool

	params   []Parameter
	names    []string
	filtered []listItem
	cursor   int
	offset   int

	detail      Parameter
	detailValue string
	revealed    bool
	viewport    viewport.Model

	width  int
	height int
	status string
	err    error
}

func newModel(ctx context.Context, client *SSMClient, region string) model {
	input := textinput.New()
	input.Placeholder = "type to filter parameters..."
	input.Prompt = "🔍 "
	input.Focus()

	spin := spinner.New()
	spin.Spinner = spinner.Dot

	return model{
		ctx:     ctx,
		client:  client,
		region:  region,
		state:   stateList,
		input:   input,
		spin:    spin,
		loading: true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadParameters(), m.spin.Tick, textinput.Blink)
}

func (m model) loadParameters() tea.Cmd {
	return func() tea.Msg {
		params, err := m.client.ListParameters(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		return parametersLoadedMsg(params)
	}
}

func (m model) fetchValue(name string, forCopy bool) tea.Cmd {
	return func() tea.Msg {
		value, err := m.client.GetParameterValue(m.ctx, name)
		if err != nil {
			return errMsg{err}
		}
		return parameterValueMsg{name: name, value: value, forCopy: forCopy}
	}
}

func (m *model) applyFilter() {
	query := strings.TrimSpace(m.input.Value())
	m.filtered = m.filtered[:0]
	if query == "" {
		for i := range m.params {
			m.filtered = append(m.filtered, listItem{paramIndex: i})
		}
	} else {
		for _, match := range fuzzy.Find(query, m.names) {
			m.filtered = append(m.filtered, listItem{
				paramIndex:   match.Index,
				matchedRunes: match.MatchedIndexes,
			})
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.clampScroll()
}

func (m *model) listHeight() int {
	// header + filter + status + help
	h := m.height - 5
	if h < 1 {
		h = 1
	}
	return h
}

func (m *model) clampScroll() {
	h := m.listHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *model) selectedParam() (Parameter, bool) {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return Parameter{}, false
	}
	return m.params[m.filtered[m.cursor].paramIndex], true
}

func (m *model) setDetailContent() {
	value := m.detailValue
	if m.detail.Type == "SecureString" && !m.revealed {
		value = maskValue(value)
	}
	m.viewport.SetContent(value)
}

func maskValue(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' {
			b.WriteRune(r)
		} else {
			b.WriteRune('•')
		}
	}
	return b.String()
}

func copyToClipboard(name, value string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(value); err != nil {
			return errMsg{fmt.Errorf("failed to copy to clipboard: %w", err)}
		}
		return statusMsg(fmt.Sprintf("Copied value of %s to clipboard", name))
	}
}

type statusMsg string

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 8
		if m.viewport.Height < 1 {
			m.viewport.Height = 1
		}
		m.clampScroll()
		return m, nil

	case parametersLoadedMsg:
		m.loading = false
		m.params = msg
		m.names = make([]string, len(m.params))
		for i, p := range m.params {
			m.names[i] = p.Name
		}
		m.err = nil
		m.applyFilter()
		m.status = fmt.Sprintf("Loaded %d parameters", len(m.params))
		return m, nil

	case parameterValueMsg:
		m.loading = false
		if msg.forCopy {
			return m, copyToClipboard(msg.name, msg.value)
		}
		m.detailValue = msg.value
		m.revealed = false
		m.state = stateDetail
		m.setDetailContent()
		m.viewport.GotoTop()
		return m, nil

	case statusMsg:
		m.status = string(msg)
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.state == stateDetail {
			return m.updateDetail(msg)
		}
		return m.updateList(msg)
	}

	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
			m.clampScroll()
		}
		return m, nil
	case "down", "ctrl+n":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			m.clampScroll()
		}
		return m, nil
	case "enter":
		if p, ok := m.selectedParam(); ok {
			m.detail = p
			m.loading = true
			m.status = ""
			m.err = nil
			return m, tea.Batch(m.fetchValue(p.Name, false), m.spin.Tick)
		}
		return m, nil
	case "ctrl+y":
		if p, ok := m.selectedParam(); ok {
			m.loading = true
			m.status = ""
			m.err = nil
			return m, tea.Batch(m.fetchValue(p.Name, true), m.spin.Tick)
		}
		return m, nil
	case "ctrl+r":
		m.loading = true
		m.status = ""
		m.err = nil
		return m, tea.Batch(m.loadParameters(), m.spin.Tick)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "q":
		m.state = stateList
		m.status = ""
		m.err = nil
		return m, nil
	case "y", "c":
		return m, copyToClipboard(m.detail.Name, m.detailValue)
	case "s":
		m.revealed = !m.revealed
		m.setDetailContent()
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	if m.state == stateDetail {
		return m.viewDetail()
	}
	return m.viewList()
}

func (m model) viewList() string {
	var b strings.Builder

	title := fmt.Sprintf("AWS Parameter Store (%s)", m.region)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")

	h := m.listHeight()
	if m.loading {
		b.WriteString(m.spin.View() + " loading...\n")
		h--
	}
	end := m.offset + h
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := m.offset; i < end; i++ {
		item := m.filtered[i]
		p := m.params[item.paramIndex]
		var line string
		if i == m.cursor {
			name := lipgloss.StyleRunes(p.Name, item.matchedRunes, selectedMatchedStyle, selectedStyle)
			line = selectedStyle.Render("> ") + name + " " + typeStyle.Render("("+p.Type+")")
		} else {
			name := lipgloss.StyleRunes(p.Name, item.matchedRunes, matchedStyle, normalStyle)
			line = "  " + name + " " + typeStyle.Render("("+p.Type+")")
		}
		b.WriteString(truncate(line, m.width))
		b.WriteString("\n")
	}
	for i := end - m.offset; i < h; i++ {
		b.WriteString("\n")
	}

	b.WriteString(m.statusLine())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: move  enter: view  ctrl+y: copy  ctrl+r: reload  esc: quit"))
	return b.String()
}

func (m model) viewDetail() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("AWS Parameter Store — Parameter Detail"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Name:"), m.detail.Name))
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Type:"), m.detail.Type))
	b.WriteString(fmt.Sprintf("%s %d\n", labelStyle.Render("Version:"), m.detail.Version))
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Last Modified:"), m.detail.LastModified))
	b.WriteString(labelStyle.Render("Value:"))
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	b.WriteString(m.statusLine())
	b.WriteString("\n")
	help := "y/c: copy  esc: back  ctrl+c: quit"
	if m.detail.Type == "SecureString" {
		help = "y/c: copy  s: reveal/mask  esc: back  ctrl+c: quit"
	}
	b.WriteString(helpStyle.Render(help))
	return b.String()
}

func (m model) statusLine() string {
	if m.err != nil {
		return truncate(errorStyle.Render("Error: "+m.err.Error()), m.width)
	}
	if m.status != "" {
		return truncate(statusStyle.Render(m.status), m.width)
	}
	if m.state == stateList {
		return typeStyle.Render(fmt.Sprintf("%d/%d parameters", len(m.filtered), len(m.params)))
	}
	return ""
}

func truncate(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
