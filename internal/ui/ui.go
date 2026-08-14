package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// Field is one metadata row shown in the detail view.
type Field struct {
	Label string
	Value string
}

// Item is a single browsable entry provided by a Backend.
type Item struct {
	Name string
	// Meta is shown dimmed in parentheses after the name in the list view.
	Meta   string
	Fields []Field
	// Sensitive values are masked in the detail view until revealed.
	Sensitive bool
}

// Backend connects the TUI to an AWS service.
type Backend interface {
	ServiceName() string
	// ItemNoun is the plural noun used in status lines, e.g. "parameters".
	ItemNoun() string
	List(ctx context.Context) ([]Item, error)
	GetValue(ctx context.Context, name string) (string, error)
}

// Run starts the TUI for the given backend.
func Run(ctx context.Context, backend Backend, region string) error {
	m := newModel(ctx, backend, region)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}

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
	metaStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	labelStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
)

type itemsLoadedMsg []Item

type valueMsg struct {
	name    string
	value   string
	forCopy bool
}

type errMsg struct{ err error }

type statusMsg string

type listItem struct {
	itemIndex    int
	matchedRunes []int
}

type model struct {
	ctx     context.Context
	backend Backend
	region  string

	state   viewState
	input   textinput.Model
	spin    spinner.Model
	loading bool

	items    []Item
	names    []string
	filtered []listItem
	cursor   int
	offset   int

	detail      Item
	detailValue string
	revealed    bool
	kvPairs     []kvPair
	isKV        bool
	rawView     bool
	viewport    viewport.Model

	width  int
	height int
	status string
	err    error
}

func newModel(ctx context.Context, backend Backend, region string) model {
	input := textinput.New()
	input.Placeholder = fmt.Sprintf("type to filter %s...", backend.ItemNoun())
	input.Prompt = "🔍 "
	input.Focus()

	spin := spinner.New()
	spin.Spinner = spinner.Dot

	return model{
		ctx:     ctx,
		backend: backend,
		region:  region,
		state:   stateList,
		input:   input,
		spin:    spin,
		loading: true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadItems(), m.spin.Tick, textinput.Blink)
}

func (m model) loadItems() tea.Cmd {
	return func() tea.Msg {
		items, err := m.backend.List(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		return itemsLoadedMsg(items)
	}
}

func (m model) fetchValue(name string, forCopy bool) tea.Cmd {
	return func() tea.Msg {
		value, err := m.backend.GetValue(m.ctx, name)
		if err != nil {
			return errMsg{err}
		}
		return valueMsg{name: name, value: value, forCopy: forCopy}
	}
}

func (m *model) applyFilter() {
	query := strings.TrimSpace(m.input.Value())
	m.filtered = m.filtered[:0]
	if query == "" {
		for i := range m.items {
			m.filtered = append(m.filtered, listItem{itemIndex: i})
		}
	} else {
		for _, match := range fuzzy.Find(query, m.names) {
			m.filtered = append(m.filtered, listItem{
				itemIndex:    match.Index,
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

func (m *model) selectedItem() (Item, bool) {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return Item{}, false
	}
	return m.items[m.filtered[m.cursor].itemIndex], true
}

func (m *model) resizeViewport() {
	// title + blank + fields + value label + status + help
	h := m.height - (5 + len(m.detail.Fields))
	if h < 1 {
		h = 1
	}
	m.viewport.Width = m.width
	m.viewport.Height = h
}

func (m *model) setDetailContent() {
	masked := m.detail.Sensitive && !m.revealed
	if m.isKV && !m.rawView {
		var b strings.Builder
		for i, p := range m.kvPairs {
			if i > 0 {
				b.WriteString("\n")
			}
			value := p.value
			if masked {
				value = maskValue(value)
			}
			b.WriteString(labelStyle.Render(p.key+":") + " " + value)
		}
		m.viewport.SetContent(b.String())
		return
	}
	value := m.detailValue
	if masked {
		value = maskValue(value)
	}
	m.viewport.SetContent(value)
}

type kvPair struct {
	key   string
	value string
}

// parseJSONObject parses s as a single JSON object, preserving key order.
// It returns false if s is anything other than a JSON object.
func parseJSONObject(s string) ([]kvPair, bool) {
	dec := json.NewDecoder(strings.NewReader(s))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false
	}
	var pairs []kvPair
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, false
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, false
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, false
		}
		pairs = append(pairs, kvPair{key: key, value: formatJSONValue(raw)})
	}
	if _, err := dec.Token(); err != nil {
		return nil, false
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, false
	}
	return pairs, true
}

// formatJSONValue renders a JSON value for the k/v view: strings are
// unquoted, everything else is shown as compact JSON.
func formatJSONValue(raw json.RawMessage) string {
	// Decoding the JSON literal null into a RawMessage leaves it nil.
	if len(raw) == 0 {
		return "null"
	}
	// Unmarshal leaves the target unchanged for null, so only unquote
	// actual JSON strings.
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return string(raw)
	}
	return buf.String()
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		m.clampScroll()
		return m, nil

	case itemsLoadedMsg:
		m.loading = false
		m.items = msg
		m.names = make([]string, len(m.items))
		for i, item := range m.items {
			m.names[i] = item.Name
		}
		m.err = nil
		m.applyFilter()
		m.status = fmt.Sprintf("Loaded %d %s", len(m.items), m.backend.ItemNoun())
		return m, nil

	case valueMsg:
		m.loading = false
		if msg.forCopy {
			return m, copyToClipboard(msg.name, msg.value)
		}
		m.detailValue = msg.value
		m.revealed = false
		m.kvPairs, m.isKV = parseJSONObject(msg.value)
		m.rawView = false
		m.state = stateDetail
		m.resizeViewport()
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
		if item, ok := m.selectedItem(); ok {
			m.detail = item
			m.loading = true
			m.status = ""
			m.err = nil
			return m, tea.Batch(m.fetchValue(item.Name, false), m.spin.Tick)
		}
		return m, nil
	case "ctrl+y":
		if item, ok := m.selectedItem(); ok {
			m.loading = true
			m.status = ""
			m.err = nil
			return m, tea.Batch(m.fetchValue(item.Name, true), m.spin.Tick)
		}
		return m, nil
	case "ctrl+r":
		m.loading = true
		m.status = ""
		m.err = nil
		return m, tea.Batch(m.loadItems(), m.spin.Tick)
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
	case "t":
		if m.isKV {
			m.rawView = !m.rawView
			m.setDetailContent()
			m.viewport.GotoTop()
		}
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

	title := fmt.Sprintf("%s (%s)", m.backend.ServiceName(), m.region)
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
		li := m.filtered[i]
		item := m.items[li.itemIndex]
		meta := ""
		if item.Meta != "" {
			meta = " " + metaStyle.Render("("+item.Meta+")")
		}
		var line string
		if i == m.cursor {
			name := lipgloss.StyleRunes(item.Name, li.matchedRunes, selectedMatchedStyle, selectedStyle)
			line = selectedStyle.Render("> ") + name + meta
		} else {
			name := lipgloss.StyleRunes(item.Name, li.matchedRunes, matchedStyle, normalStyle)
			line = "  " + name + meta
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

	b.WriteString(titleStyle.Render(m.backend.ServiceName() + " — Detail"))
	b.WriteString("\n\n")
	for _, f := range m.detail.Fields {
		b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render(f.Label+":"), f.Value))
	}
	b.WriteString(labelStyle.Render("Value:"))
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	b.WriteString(m.statusLine())
	b.WriteString("\n")
	help := "y/c: copy"
	if m.detail.Sensitive {
		help += "  s: reveal/mask"
	}
	if m.isKV {
		help += "  t: kv/raw"
	}
	help += "  esc: back  ctrl+c: quit"
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
		return metaStyle.Render(fmt.Sprintf("%d/%d %s", len(m.filtered), len(m.items), m.backend.ItemNoun()))
	}
	return ""
}

func truncate(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
