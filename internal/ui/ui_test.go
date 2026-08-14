package ui

import (
	"reflect"
	"testing"
)

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
