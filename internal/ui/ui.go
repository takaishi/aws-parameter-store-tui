package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

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

// Item is a single browsable entry in a Screen. Enter descends into Child
// when it is set; otherwise it opens the detail view, whose body is loaded
// via Value.
type Item struct {
	Name string
	// Meta is shown dimmed in parentheses after the name in the list view.
	Meta   string
	Fields []Field
	// Sensitive values are masked in the detail view until revealed.
	Sensitive bool
	// CopyValue, when set, is what ctrl+y copies in the list view (e.g. an
	// ARN). Otherwise ctrl+y copies the fetched Value, or the Name as a
	// last resort.
	CopyValue string
	// ValueLabel overrides the "Value" heading in the detail view.
	ValueLabel string
	Child      func() *Screen
	Value      func(ctx context.Context) (string, error)
}

// Screen is one level of the navigation stack.
type Screen struct {
	// Title is this screen's breadcrumb segment. The root screen's title
	// should identify the service and region, e.g. "AWS Parameter Store
	// (ap-northeast-1)".
	Title string
	// Noun is the plural noun used in status lines, e.g. "parameters".
	Noun string
	List func(ctx context.Context) ([]Item, error)
}

// Option configures the TUI.
type Option func(*model)

// WithColumns renders the navigation stack as side-by-side panes (up to
// three, Miller-column style) instead of one screen at a time. ←/→ (or
// tab/shift+tab) move focus between panes.
func WithColumns() Option {
	return func(m *model) { m.columns = true }
}

// Run starts the TUI at the given root screen.
func Run(ctx context.Context, root *Screen, opts ...Option) error {
	m := newModel(ctx, root)
	for _, opt := range opts {
		opt(&m)
	}
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

type itemsLoadedMsg struct {
	seq   int
	items []Item
}

type valueMsg struct {
	seq     int
	name    string
	value   string
	forCopy bool
}

type errMsg struct{ err error }

type statusMsg string

// previewDelay debounces child-pane preview loads while the user is still
// moving the cursor, so each cursor step doesn't fire an API call.
const previewDelay = 200 * time.Millisecond

type previewMsg struct{ seq int }

type listItem struct {
	itemIndex    int
	matchedRunes []int
}

// frame is one level of the navigation stack: a screen plus its loaded
// items and filter state, preserved when descending into a child screen.
type frame struct {
	screen   *Screen
	input    textinput.Model
	items    []Item
	names    []string
	filtered []listItem
	cursor   int
	offset   int
}

func (f *frame) applyFilter(height int) {
	query := strings.TrimSpace(f.input.Value())
	f.filtered = f.filtered[:0]
	if query == "" {
		for i := range f.items {
			f.filtered = append(f.filtered, listItem{itemIndex: i})
		}
	} else {
		for _, match := range fuzzy.Find(query, f.names) {
			f.filtered = append(f.filtered, listItem{
				itemIndex:    match.Index,
				matchedRunes: match.MatchedIndexes,
			})
		}
	}
	if f.cursor >= len(f.filtered) {
		f.cursor = len(f.filtered) - 1
	}
	if f.cursor < 0 {
		f.cursor = 0
	}
	f.clampScroll(height)
}

func (f *frame) clampScroll(height int) {
	if f.cursor < f.offset {
		f.offset = f.cursor
	}
	if f.cursor >= f.offset+height {
		f.offset = f.cursor - height + 1
	}
	if f.offset < 0 {
		f.offset = 0
	}
}

type model struct {
	ctx context.Context

	stack   []frame
	columns bool
	// focus is the index of the frame receiving input. In single-screen
	// mode it always tracks the deepest frame; in columns mode ←/→ move
	// it along the stack.
	focus   int
	state   viewState
	spin    spinner.Model
	loading bool
	// seq invalidates in-flight loads when the user navigates away before
	// they complete; stale itemsLoadedMsg/valueMsg are dropped.
	seq int
	// previewSeq invalidates pending debounced preview ticks.
	previewSeq int

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

func newFilterInput(noun string) textinput.Model {
	input := textinput.New()
	input.Placeholder = fmt.Sprintf("type to filter %s...", noun)
	input.Prompt = "🔍 "
	input.Focus()
	return input
}

func newModel(ctx context.Context, root *Screen) model {
	spin := spinner.New()
	spin.Spinner = spinner.Dot

	return model{
		ctx:     ctx,
		stack:   []frame{{screen: root, input: newFilterInput(root.Noun)}},
		state:   stateList,
		spin:    spin,
		loading: true,
		seq:     1,
	}
}

func (m *model) focused() *frame {
	return &m.stack[m.focus]
}

func (m *model) deepest() *frame {
	return &m.stack[len(m.stack)-1]
}

func listCmd(ctx context.Context, s *Screen, seq int) tea.Cmd {
	return func() tea.Msg {
		items, err := s.List(ctx)
		if err != nil {
			return errMsg{err}
		}
		return itemsLoadedMsg{seq: seq, items: items}
	}
}

func valueCmd(ctx context.Context, item Item, seq int, forCopy bool) tea.Cmd {
	return func() tea.Msg {
		value, err := item.Value(ctx)
		if err != nil {
			return errMsg{err}
		}
		return valueMsg{seq: seq, name: item.Name, value: value, forCopy: forCopy}
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(listCmd(m.ctx, m.stack[0].screen, m.seq), m.spin.Tick, textinput.Blink)
}

func (m *model) loadCurrent() tea.Cmd {
	m.loading = true
	m.status = ""
	m.err = nil
	m.seq++
	return tea.Batch(listCmd(m.ctx, m.deepest().screen, m.seq), m.spin.Tick)
}

func (m *model) pushScreen(s *Screen) tea.Cmd {
	m.stack = append(m.stack, frame{screen: s, input: newFilterInput(s.Noun)})
	m.focus = len(m.stack) - 1
	return tea.Batch(m.loadCurrent(), textinput.Blink)
}

func (m *model) popScreen() {
	m.stack = m.stack[:len(m.stack)-1]
	m.focus = len(m.stack) - 1
	m.seq++ // drop results of any in-flight load for the abandoned screen
	m.loading = false
	m.status = ""
	m.err = nil
	m.focused().clampScroll(m.listHeight())
}

// truncateToFocus discards frames deeper than the focused one. In columns
// mode child panes become stale whenever the focused pane's selection
// changes, so they are dropped until the user descends again.
func (m *model) truncateToFocus() {
	if len(m.stack) > m.focus+1 {
		m.stack = m.stack[:m.focus+1]
		m.seq++
		m.loading = false
	}
}

func (m *model) fetchValue(item Item, forCopy bool) tea.Cmd {
	m.loading = true
	m.status = ""
	m.err = nil
	m.seq++
	return tea.Batch(valueCmd(m.ctx, item, m.seq, forCopy), m.spin.Tick)
}

func (m *model) listHeight() int {
	// Single screen: breadcrumb + filter + status + help.
	// Columns: per-pane title + filter, shared status + help.
	chrome := 5
	if m.columns {
		chrome = 4
	}
	h := m.height - chrome
	if h < 1 {
		h = 1
	}
	return h
}

func frameSelected(f *frame) (Item, bool) {
	if len(f.filtered) == 0 || f.cursor >= len(f.filtered) {
		return Item{}, false
	}
	return f.items[f.filtered[f.cursor].itemIndex], true
}

func (m *model) selectedItem() (Item, bool) {
	return frameSelected(m.focused())
}

// pushPreview appends and loads a child pane without moving focus, so the
// user keeps navigating the parent pane while its selection's children
// render on the right.
func (m *model) pushPreview(s *Screen) tea.Cmd {
	m.stack = append(m.stack, frame{screen: s, input: newFilterInput(s.Noun)})
	return m.loadCurrent()
}

// schedulePreview arms a debounced preview of the focused pane's selected
// item, when that selection can descend and has no child pane yet.
func (m *model) schedulePreview() tea.Cmd {
	if !m.columns || m.focus != len(m.stack)-1 {
		return nil
	}
	item, ok := frameSelected(m.focused())
	if !ok || item.Child == nil {
		return nil
	}
	m.previewSeq++
	seq := m.previewSeq
	return tea.Tick(previewDelay, func(time.Time) tea.Msg {
		return previewMsg{seq: seq}
	})
}

func (m *model) resizeViewport() {
	if m.columns {
		// The detail is rendered as the rightmost pane, next to up to two
		// ancestor list panes.
		k := len(m.stack)
		if k > 2 {
			k = 2
		}
		widths := paneWidths(m.width, k+1)
		m.viewport.Width = widths[k]
		// pane height minus title + value label + fields
		h := (m.height - 2) - (2 + len(m.detail.Fields))
		if h < 1 {
			h = 1
		}
		m.viewport.Height = h
		return
	}
	// title + blank + fields + value label + status + help
	h := m.height - (5 + len(m.detail.Fields))
	if h < 1 {
		h = 1
	}
	m.viewport.Width = m.width
	m.viewport.Height = h
}

func (m *model) openDetail(value string) {
	m.detailValue = value
	m.revealed = false
	m.kvPairs, m.isKV = parseJSONObject(value)
	m.rawView = false
	m.state = stateDetail
	m.resizeViewport()
	m.setDetailContent()
	m.viewport.GotoTop()
}

func (m *model) setDetailContent() {
	masked := m.detail.Sensitive && !m.revealed
	var content string
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
		content = b.String()
	} else {
		content = m.detailValue
		if masked {
			content = maskValue(content)
		}
	}
	if m.columns && m.viewport.Width > 0 {
		// Soft-wrap to the detail pane width so long lines stay readable.
		content = lipgloss.NewStyle().Width(m.viewport.Width).Render(content)
	}
	m.viewport.SetContent(content)
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
		if m.state == stateDetail {
			m.setDetailContent() // re-wrap to the new pane width
		}
		for i := range m.stack {
			m.stack[i].clampScroll(m.listHeight())
		}
		return m, nil

	case itemsLoadedMsg:
		if msg.seq != m.seq {
			return m, nil
		}
		m.loading = false
		f := m.deepest()
		f.items = msg.items
		f.names = make([]string, len(f.items))
		for i, item := range f.items {
			f.names[i] = item.Name
		}
		m.err = nil
		f.applyFilter(m.listHeight())
		m.status = fmt.Sprintf("Loaded %d %s", len(f.items), f.screen.Noun)
		// Preview one level below the focused pane (initial load / reload).
		return m, m.schedulePreview()

	case valueMsg:
		if msg.seq != m.seq {
			return m, nil
		}
		m.loading = false
		if msg.forCopy {
			return m, copyToClipboard(msg.name, msg.value)
		}
		m.openDetail(msg.value)
		return m, nil

	case previewMsg:
		if msg.seq != m.previewSeq || m.state != stateList {
			return m, nil
		}
		if item, ok := frameSelected(m.deepest()); ok && item.Child != nil && m.focus == len(m.stack)-1 {
			return m, m.pushPreview(item.Child())
		}
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
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.columns {
			if m.focus > 0 {
				m.focus--
				m.status = ""
				m.err = nil
				return m, nil
			}
			return m, tea.Quit
		}
		if len(m.stack) > 1 {
			m.popScreen()
			return m, nil
		}
		return m, tea.Quit
	case "left", "shift+tab":
		if m.columns {
			if m.focus > 0 {
				m.focus--
			}
			return m, nil
		}
		// In single-screen mode these keys belong to the filter input.
	case "right", "tab":
		if m.columns {
			if m.focus < len(m.stack)-1 {
				m.focus++
				// Arm a preview for the newly focused pane's selection.
				return m, m.schedulePreview()
			}
			if item, ok := m.selectedItem(); ok && item.Child != nil {
				return m, m.pushScreen(item.Child())
			}
			return m, nil
		}
	case "up", "ctrl+p":
		f := m.focused()
		if f.cursor > 0 {
			f.cursor--
			f.clampScroll(m.listHeight())
			m.truncateToFocus()
			return m, m.schedulePreview()
		}
		return m, nil
	case "down", "ctrl+n":
		f := m.focused()
		if f.cursor < len(f.filtered)-1 {
			f.cursor++
			f.clampScroll(m.listHeight())
			m.truncateToFocus()
			return m, m.schedulePreview()
		}
		return m, nil
	case "enter":
		if item, ok := m.selectedItem(); ok {
			if item.Child != nil {
				if m.columns && m.focus < len(m.stack)-1 {
					// The child pane for this selection is still loaded.
					m.focus++
					return m, m.schedulePreview()
				}
				return m, m.pushScreen(item.Child())
			}
			m.detail = item
			if item.Value == nil {
				m.openDetail("")
				return m, nil
			}
			return m, m.fetchValue(item, false)
		}
		return m, nil
	case "ctrl+y":
		if item, ok := m.selectedItem(); ok {
			if item.CopyValue != "" {
				return m, copyToClipboard(item.Name, item.CopyValue)
			}
			if item.Value == nil {
				return m, copyToClipboard(item.Name, item.Name)
			}
			return m, m.fetchValue(item, true)
		}
		return m, nil
	case "ctrl+r":
		m.truncateToFocus()
		return m, m.loadCurrent()
	}

	f := m.focused()
	var cmd tea.Cmd
	before := f.input.Value()
	f.input, cmd = f.input.Update(msg)
	f.applyFilter(m.listHeight())
	if f.input.Value() != before {
		m.truncateToFocus()
		return m, tea.Batch(cmd, m.schedulePreview())
	}
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

func (m model) breadcrumb() string {
	titles := make([]string, len(m.stack))
	for i, f := range m.stack {
		titles[i] = f.screen.Title
	}
	return strings.Join(titles, " > ")
}

func (m model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	if m.state == stateDetail {
		if m.columns {
			return m.viewDetailColumns()
		}
		return m.viewDetail()
	}
	if m.columns {
		return m.viewColumns()
	}
	return m.viewList()
}

func (m model) viewList() string {
	var b strings.Builder
	f := m.stack[len(m.stack)-1]

	b.WriteString(truncate(titleStyle.Render(m.breadcrumb()), m.width))
	b.WriteString("\n")
	b.WriteString(f.input.View())
	b.WriteString("\n")

	h := m.listHeight()
	if m.loading {
		b.WriteString(m.spin.View() + " loading...\n")
		h--
	}
	end := f.offset + h
	if end > len(f.filtered) {
		end = len(f.filtered)
	}
	for i := f.offset; i < end; i++ {
		b.WriteString(truncate(m.renderLine(&f, i, true), m.width))
		b.WriteString("\n")
	}
	for i := end - f.offset; i < h; i++ {
		b.WriteString("\n")
	}

	b.WriteString(m.statusLine())
	b.WriteString("\n")
	escLabel := "esc: quit"
	if len(m.stack) > 1 {
		escLabel = "esc: back"
	}
	b.WriteString(helpStyle.Render("↑/↓: move  enter: open  ctrl+y: copy  ctrl+r: reload  " + escLabel))
	return b.String()
}

// renderLine renders one list row of the frame. Selection highlighting is
// dimmed when the frame is not focused.
func (m model) renderLine(f *frame, i int, focused bool) string {
	li := f.filtered[i]
	item := f.items[li.itemIndex]
	meta := ""
	if item.Meta != "" {
		meta = " " + metaStyle.Render("("+item.Meta+")")
	}
	if i == f.cursor {
		if focused {
			name := lipgloss.StyleRunes(item.Name, li.matchedRunes, selectedMatchedStyle, selectedStyle)
			return selectedStyle.Render("> ") + name + meta
		}
		return metaStyle.Render("> "+item.Name) + meta
	}
	name := lipgloss.StyleRunes(item.Name, li.matchedRunes, matchedStyle, normalStyle)
	return "  " + name + meta
}

func (m model) viewColumns() string {
	// Show a window of up to three consecutive frames that includes both
	// the focused and, when possible, the deepest frame.
	ws := len(m.stack) - 3
	if ws < 0 {
		ws = 0
	}
	if m.focus < ws {
		ws = m.focus
	}
	end := ws + 3
	if end > len(m.stack) {
		end = len(m.stack)
	}
	widths := paneWidths(m.width, end-ws)
	paneH := m.height - 2 // status + help
	if paneH < 3 {
		paneH = 3
	}

	panes := make([]string, 0, (end-ws)*2)
	for gi := ws; gi < end; gi++ {
		if gi > ws {
			panes = append(panes, sepPane(paneH))
		}
		loading := m.loading && gi == len(m.stack)-1
		panes = append(panes, m.renderPane(&m.stack[gi], widths[gi-ws], paneH, gi == m.focus, loading))
	}

	var b strings.Builder
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panes...))
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: move  ←/→: pane  enter: open  ctrl+y: copy  ctrl+r: reload  esc: back"))
	return b.String()
}

func (m model) renderPane(f *frame, w, h int, focused, loading bool) string {
	pad := lipgloss.NewStyle().Width(w)
	tp := func(s string) string { return pad.Render(truncate(s, w)) }

	titleSt := metaStyle
	if focused {
		titleSt = titleStyle
	}
	rows := make([]string, 0, h)
	rows = append(rows, tp(titleSt.Render(f.screen.Title)))
	rows = append(rows, tp(f.input.View()))

	itemH := h - 2
	if loading {
		rows = append(rows, tp(m.spin.View()+" loading..."))
		itemH--
	}
	end := f.offset + itemH
	if end > len(f.filtered) {
		end = len(f.filtered)
	}
	for i := f.offset; i < end; i++ {
		rows = append(rows, tp(m.renderLine(f, i, focused)))
	}
	for len(rows) < h {
		rows = append(rows, tp(""))
	}
	return strings.Join(rows, "\n")
}

func sepPane(h int) string {
	line := metaStyle.Render("│")
	rows := make([]string, h)
	for i := range rows {
		rows[i] = line
	}
	return strings.Join(rows, "\n")
}

// paneWidths splits the total width among k panes (plus k-1 one-column
// separators): narrow parent panes, a wide rightmost pane.
func paneWidths(total, k int) []int {
	avail := total - (k - 1)
	if avail < k {
		avail = k
	}
	switch k {
	case 1:
		return []int{avail}
	case 2:
		w := avail / 3
		return []int{w, avail - w}
	default:
		w := avail / 4
		return []int{w, w, avail - 2*w}
	}
}

func (m model) detailHelp() string {
	help := "y/c: copy"
	if m.detail.Sensitive {
		help += "  s: reveal/mask"
	}
	if m.isKV {
		help += "  t: kv/raw"
	}
	help += "  esc: back  ctrl+c: quit"
	return help
}

func (m model) detailValueLabel() string {
	if m.detail.ValueLabel != "" {
		return m.detail.ValueLabel
	}
	return "Value"
}

func (m model) viewDetail() string {
	var b strings.Builder

	b.WriteString(truncate(titleStyle.Render(m.breadcrumb()+" > "+m.detail.Name), m.width))
	b.WriteString("\n\n")
	for _, f := range m.detail.Fields {
		b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render(f.Label+":"), f.Value))
	}
	b.WriteString(labelStyle.Render(m.detailValueLabel() + ":"))
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	b.WriteString(m.statusLine())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.detailHelp()))
	return b.String()
}

// viewDetailColumns keeps the column layout while a detail is open: up to
// two ancestor list panes stay on the left and the detail fills the
// rightmost pane.
func (m model) viewDetailColumns() string {
	start := len(m.stack) - 2
	if start < 0 {
		start = 0
	}
	k := len(m.stack) - start + 1
	widths := paneWidths(m.width, k)
	paneH := m.height - 2 // status + help
	if paneH < 3 {
		paneH = 3
	}

	panes := make([]string, 0, k*2)
	for gi := start; gi < len(m.stack); gi++ {
		panes = append(panes, m.renderPane(&m.stack[gi], widths[gi-start], paneH, false, false))
		panes = append(panes, sepPane(paneH))
	}
	panes = append(panes, m.renderDetailPane(widths[k-1], paneH))

	var b strings.Builder
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, panes...))
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.detailHelp()))
	return b.String()
}

func (m model) renderDetailPane(w, h int) string {
	pad := lipgloss.NewStyle().Width(w)
	tp := func(s string) string { return pad.Render(truncate(s, w)) }

	rows := make([]string, 0, h)
	rows = append(rows, tp(titleStyle.Render(m.detail.Name)))
	for _, f := range m.detail.Fields {
		rows = append(rows, tp(labelStyle.Render(f.Label+":")+" "+f.Value))
	}
	rows = append(rows, tp(labelStyle.Render(m.detailValueLabel()+":")))
	for _, line := range strings.Split(m.viewport.View(), "\n") {
		rows = append(rows, tp(line))
	}
	if len(rows) > h {
		rows = rows[:h]
	}
	for len(rows) < h {
		rows = append(rows, tp(""))
	}
	return strings.Join(rows, "\n")
}

func (m model) statusLine() string {
	if m.err != nil {
		return truncate(errorStyle.Render("Error: "+m.err.Error()), m.width)
	}
	if m.status != "" {
		return truncate(statusStyle.Render(m.status), m.width)
	}
	if m.state == stateList {
		f := m.stack[m.focus]
		return metaStyle.Render(fmt.Sprintf("%d/%d %s", len(f.filtered), len(f.items), f.screen.Noun))
	}
	return ""
}

func truncate(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}
