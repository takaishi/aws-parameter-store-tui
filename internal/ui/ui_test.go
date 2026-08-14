package ui

import (
	"context"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestColumnsNavigation(t *testing.T) {
	childItems := []Item{{Name: "leaf-1"}, {Name: "leaf-2"}}
	child := &Screen{
		Title: "child",
		Noun:  "leaves",
		List: func(ctx context.Context) ([]Item, error) {
			return childItems, nil
		},
	}
	rootItems := []Item{
		{Name: "a", Child: func() *Screen { return child }},
		{Name: "b", Child: func() *Screen { return child }},
	}
	root := &Screen{
		Title: "root",
		Noun:  "items",
		List: func(ctx context.Context) ([]Item, error) {
			return rootItems, nil
		},
	}

	m := newModel(context.Background(), root)
	m.columns = true
	m.width = 80
	m.height = 24

	step := func(msg tea.Msg) {
		t.Helper()
		mm, _ := m.Update(msg)
		m = mm.(model)
	}

	step(itemsLoadedMsg{seq: m.seq, items: rootItems})
	if len(m.stack) != 1 || m.focus != 0 {
		t.Fatalf("after root load: stack=%d focus=%d, want 1/0", len(m.stack), m.focus)
	}

	// Descend into the selected item's child pane.
	step(tea.KeyMsg{Type: tea.KeyRight})
	if len(m.stack) != 2 || m.focus != 1 {
		t.Fatalf("after right: stack=%d focus=%d, want 2/1", len(m.stack), m.focus)
	}
	step(itemsLoadedMsg{seq: m.seq, items: childItems})
	if got := len(m.stack[1].items); got != 2 {
		t.Fatalf("child pane items = %d, want 2", got)
	}

	// Focus back to the parent pane; the child pane stays loaded.
	step(tea.KeyMsg{Type: tea.KeyLeft})
	if len(m.stack) != 2 || m.focus != 0 {
		t.Fatalf("after left: stack=%d focus=%d, want 2/0", len(m.stack), m.focus)
	}

	// Moving the cursor in the parent invalidates the child pane.
	step(tea.KeyMsg{Type: tea.KeyDown})
	if len(m.stack) != 1 || m.focus != 0 {
		t.Fatalf("after cursor move: stack=%d focus=%d, want 1/0", len(m.stack), m.focus)
	}
	if m.stack[0].cursor != 1 {
		t.Fatalf("root cursor = %d, want 1", m.stack[0].cursor)
	}

	// The columns view renders all visible panes without panicking.
	step(tea.KeyMsg{Type: tea.KeyRight})
	step(itemsLoadedMsg{seq: m.seq, items: childItems})
	view := m.View()
	for _, want := range []string{"root", "child", "leaf-1"} {
		if !strings.Contains(view, want) {
			t.Errorf("columns view missing %q", want)
		}
	}

	// Opening a leaf's detail keeps the column layout: ancestor panes on
	// the left, detail in the rightmost pane.
	step(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateDetail {
		t.Fatalf("after enter on leaf: state=%v, want stateDetail", m.state)
	}
	view = m.View()
	for _, want := range []string{"root", "child", "leaf-1"} {
		if !strings.Contains(view, want) {
			t.Errorf("detail columns view missing %q", want)
		}
	}
	step(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateList {
		t.Fatalf("after esc from detail: state=%v, want stateList", m.state)
	}
	if len(m.stack) != 2 || m.focus != 1 {
		t.Fatalf("after esc from detail: stack=%d focus=%d, want 2/1", len(m.stack), m.focus)
	}

	// esc walks focus back to the root pane, then quits.
	step(tea.KeyMsg{Type: tea.KeyEsc})
	if m.focus != 0 {
		t.Fatalf("after esc: focus=%d, want 0", m.focus)
	}
	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(model)
	if cmd == nil {
		t.Fatal("esc at root should quit")
	}
}

func TestColumnsPreview(t *testing.T) {
	childItems := []Item{{Name: "leaf-1"}}
	child := &Screen{
		Title: "child",
		Noun:  "leaves",
		List: func(ctx context.Context) ([]Item, error) {
			return childItems, nil
		},
	}
	rootItems := []Item{
		{Name: "a", Child: func() *Screen { return child }},
		{Name: "b", Child: func() *Screen { return child }},
	}
	root := &Screen{
		Title: "root",
		Noun:  "items",
		List: func(ctx context.Context) ([]Item, error) {
			return rootItems, nil
		},
	}

	m := newModel(context.Background(), root)
	m.columns = true
	m.width = 80
	m.height = 24

	step := func(msg tea.Msg) {
		t.Helper()
		mm, _ := m.Update(msg)
		m = mm.(model)
	}

	// Loading the root pane arms a debounced preview of its selection.
	step(itemsLoadedMsg{seq: m.seq, items: rootItems})
	if m.previewSeq == 0 {
		t.Fatal("root load should arm a preview")
	}

	// The preview tick pushes the child pane without moving focus.
	step(previewMsg{seq: m.previewSeq})
	if len(m.stack) != 2 || m.focus != 0 {
		t.Fatalf("after preview: stack=%d focus=%d, want 2/0", len(m.stack), m.focus)
	}
	step(itemsLoadedMsg{seq: m.seq, items: childItems})
	if got := len(m.stack[1].items); got != 1 {
		t.Fatalf("preview pane items = %d, want 1", got)
	}

	// Moving the cursor drops the stale child pane and re-arms the preview
	// for the new selection.
	prevSeq := m.previewSeq
	step(tea.KeyMsg{Type: tea.KeyDown})
	if len(m.stack) != 1 {
		t.Fatalf("after cursor move: stack=%d, want 1", len(m.stack))
	}
	if m.previewSeq <= prevSeq {
		t.Fatal("cursor move should re-arm the preview")
	}

	// A stale tick from before the cursor move is ignored.
	step(previewMsg{seq: prevSeq})
	if len(m.stack) != 1 {
		t.Fatalf("stale preview applied: stack=%d, want 1", len(m.stack))
	}
	step(previewMsg{seq: m.previewSeq})
	if len(m.stack) != 2 || m.focus != 0 {
		t.Fatalf("after re-preview: stack=%d focus=%d, want 2/0", len(m.stack), m.focus)
	}
}

func TestParseJSONObject(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []kvPair
		ok    bool
	}{
		{
			name:  "flat object preserves key order",
			input: `{"username":"admin","password":"p@ss","port":5432,"ssl":true}`,
			want: []kvPair{
				{key: "username", value: "admin"},
				{key: "password", value: "p@ss"},
				{key: "port", value: "5432"},
				{key: "ssl", value: "true"},
			},
			ok: true,
		},
		{
			name:  "nested values shown as compact JSON",
			input: `{"db": { "host" : "localhost", "port" : 5432 },"tags":[ "a", "b" ],"none":null}`,
			want: []kvPair{
				{key: "db", value: `{"host":"localhost","port":5432}`},
				{key: "tags", value: `["a","b"]`},
				{key: "none", value: "null"},
			},
			ok: true,
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  nil,
			ok:    true,
		},
		{name: "plain string", input: "hello", ok: false},
		{name: "quoted string", input: `"hello"`, ok: false},
		{name: "number", input: "42", ok: false},
		{name: "array", input: `[{"a":1}]`, ok: false},
		{name: "truncated object", input: `{"a":1`, ok: false},
		{name: "trailing garbage", input: `{"a":1} extra`, ok: false},
		{name: "empty string", input: "", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseJSONObject(tt.input)
			if ok != tt.ok {
				t.Fatalf("parseJSONObject(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJSONObject(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}
