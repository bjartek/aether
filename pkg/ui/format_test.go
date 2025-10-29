package ui

import (
	"strings"
	"testing"

	"github.com/hexops/autogold"
)

func TestFormatFieldValueWithRegistry_SimpleTypes(t *testing.T) {
	tests := []struct {
		name     string
		val      interface{}
		indent   string
		maxWidth int
		want     string
	}{
		{
			name:     "short string no wrap",
			val:      "hello",
			indent:   "",
			maxWidth: 0,
			want:     "hello",
		},
		{
			name:     "integer",
			val:      42.0,
			indent:   "",
			maxWidth: 0,
			want:     "42",
		},
		{
			name:     "float",
			val:      3.14159,
			indent:   "",
			maxWidth: 0,
			want:     "3.14159",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValueWithRegistry(tt.val, tt.indent, nil, false, tt.maxWidth)
			if got != tt.want {
				t.Errorf("FormatFieldValueWithRegistry() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFieldValueWithRegistry_TextWrapping(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		indent   string
		maxWidth int
		wantFunc func(string) bool // Function to validate output
	}{
		{
			name:     "long string wraps",
			val:      "This is a very long string that should wrap when maxWidth is specified and the string exceeds the available width",
			indent:   "  ",
			maxWidth: 50,
			wantFunc: func(got string) bool {
				// Check that we got multiple lines
				lines := strings.Split(got, "\n")
				if len(lines) < 2 {
					return false
				}
				// Check that continuation lines are indented
				for i := 1; i < len(lines); i++ {
					if !strings.HasPrefix(lines[i], "  ") {
						return false
					}
				}
				// Check that no line exceeds maxWidth
				for _, line := range lines {
					if len(line) > 50 {
						return false
					}
				}
				return true
			},
		},
		{
			name:     "string within width no wrap",
			val:      "short",
			indent:   "  ",
			maxWidth: 50,
			wantFunc: func(got string) bool {
				// Should not wrap, single line
				return !strings.Contains(got, "\n") && got == "short"
			},
		},
		{
			name:     "maxWidth 0 no wrap",
			val:      "This is a very long string that should NOT wrap when maxWidth is 0",
			indent:   "  ",
			maxWidth: 0,
			wantFunc: func(got string) bool {
				// Should not wrap with maxWidth=0
				return !strings.Contains(got, "\n")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValueWithRegistry(tt.val, tt.indent, nil, false, tt.maxWidth)
			if !tt.wantFunc(got) {
				t.Errorf("FormatFieldValueWithRegistry() validation failed for %q\nGot:\n%s", tt.name, got)
			}
		})
	}
}

func TestFormatFieldValueWithRegistry_Maps(t *testing.T) {
	tests := []struct {
		name     string
		val      map[string]interface{}
		indent   string
		maxWidth int
		wantFunc func(string) bool
	}{
		{
			name: "simple map",
			val: map[string]interface{}{
				"name": "Alice",
				"age":  30.0,
			},
			indent:   "  ",
			maxWidth: 0,
			wantFunc: func(got string) bool {
				// Should start with newline for container types
				if !strings.HasPrefix(got, "\n") {
					return false
				}
				// Should contain both keys
				return strings.Contains(got, "name:") && strings.Contains(got, "age:")
			},
		},
		{
			name: "map with long string value wraps",
			val: map[string]interface{}{
				"description": "This is a very long description that should wrap when maxWidth is specified",
			},
			indent:   "  ",
			maxWidth: 50,
			wantFunc: func(got string) bool {
				// Should start with newline
				if !strings.HasPrefix(got, "\n") {
					return false
				}
				// Should contain the key
				if !strings.Contains(got, "description:") {
					return false
				}
				// Should have multiple lines due to wrapping
				lines := strings.Split(strings.TrimPrefix(got, "\n"), "\n")
				return len(lines) > 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValueWithRegistry(tt.val, tt.indent, nil, false, tt.maxWidth)
			if !tt.wantFunc(got) {
				t.Errorf("FormatFieldValueWithRegistry() validation failed for %q\nGot:\n%s", tt.name, got)
			}
		})
	}
}

func TestFormatFieldValueWithRegistry_Arrays(t *testing.T) {
	tests := []struct {
		name     string
		val      []interface{}
		indent   string
		maxWidth int
		wantFunc func(string) bool
	}{
		{
			name:     "simple array",
			val:      []interface{}{"one", "two", "three"},
			indent:   "  ",
			maxWidth: 0,
			wantFunc: func(got string) bool {
				// Should start with newline for container types
				if !strings.HasPrefix(got, "\n") {
					return false
				}
				// Should contain list markers
				return strings.Contains(got, "- one") && strings.Contains(got, "- two")
			},
		},
		{
			name:     "array with long strings wraps",
			val:      []interface{}{"This is a very long string in an array that should wrap when maxWidth is specified"},
			indent:   "  ",
			maxWidth: 50,
			wantFunc: func(got string) bool {
				// Should start with newline
				if !strings.HasPrefix(got, "\n") {
					return false
				}
				// Should have multiple lines due to wrapping
				lines := strings.Split(strings.TrimPrefix(got, "\n"), "\n")
				return len(lines) > 1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValueWithRegistry(tt.val, tt.indent, nil, false, tt.maxWidth)
			if !tt.wantFunc(got) {
				t.Errorf("FormatFieldValueWithRegistry() validation failed for %q\nGot:\n%s", tt.name, got)
			}
		})
	}
}

func TestFormatFieldValueWithRegistry_NestedStructures(t *testing.T) {
	tests := []struct {
		name     string
		val      interface{}
		indent   string
		maxWidth int
		wantFunc func(string) bool
	}{
		{
			name: "nested map",
			val: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  30.0,
				},
			},
			indent:   "  ",
			maxWidth: 0,
			wantFunc: func(got string) bool {
				// Should contain nested structure
				return strings.Contains(got, "user:") && strings.Contains(got, "name:") && strings.Contains(got, "age:")
			},
		},
		{
			name: "map with long nested value wraps",
			val: map[string]interface{}{
				"user": map[string]interface{}{
					"bio": "This is a very long biography that should wrap when maxWidth is specified and exceeds the available width",
				},
			},
			indent:   "  ",
			maxWidth: 60,
			wantFunc: func(got string) bool {
				// Should contain nested structure
				if !strings.Contains(got, "user:") || !strings.Contains(got, "bio:") {
					return false
				}
				// Should have multiple lines due to wrapping
				lines := strings.Split(strings.TrimPrefix(got, "\n"), "\n")
				return len(lines) > 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValueWithRegistry(tt.val, tt.indent, nil, false, tt.maxWidth)
			if !tt.wantFunc(got) {
				t.Errorf("FormatFieldValueWithRegistry() validation failed for %q\nGot:\n%s", tt.name, got)
			}
		})
	}
}

func TestFormatFieldValueWithRegistry_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		val      interface{}
		indent   string
		maxWidth int
		wantFunc func(string) bool
	}{
		{
			name:     "empty map",
			val:      map[string]interface{}{},
			indent:   "  ",
			maxWidth: 0,
			wantFunc: func(got string) bool {
				return strings.TrimSpace(got) == "{}"
			},
		},
		{
			name:     "empty array",
			val:      []interface{}{},
			indent:   "  ",
			maxWidth: 0,
			wantFunc: func(got string) bool {
				return strings.TrimSpace(got) == "[]"
			},
		},
		{
			name:     "very small maxWidth still works",
			val:      "hello",
			indent:   "      ",
			maxWidth: 10,
			wantFunc: func(got string) bool {
				// With indent of 6 and maxWidth of 10, available width is 4
				// But we have a minimum of 20, so it shouldn't wrap
				return got == "hello"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValueWithRegistry(tt.val, tt.indent, nil, false, tt.maxWidth)
			if !tt.wantFunc(got) {
				t.Errorf("FormatFieldValueWithRegistry() validation failed for %q\nGot:\n%s", tt.name, got)
			}
		})
	}
}

func TestFormatFieldValueWithRegistry_LongHexString(t *testing.T) {
	// Real-world test case - long hex string that must wrap correctly within viewport width
	hexString := "0x61636365737328616c6c2920636f6e74726163742046434c207b0a202061636365737328616c6c29206c65742073746f72616765506174683a2053746f72616765506174680a0a202061636365737328616c6c29207374727563742046434c4b6579207b0a20202020616363"

	tests := []struct {
		name     string
		indent   string
		maxWidth int
		want     autogold.Value
	}{
		{
			name:     "hex string wraps at 80 chars",
			indent:   "    ",
			maxWidth: 80,
			want: autogold.Want("hex string wraps at 80 chars", `0x61636365737328616c6c2920636f6e74726163742046434c207b0a20206163636573732861
    6c6c29206c65742073746f72616765506174683a2053746f72616765506174680a0a20206163
    6365737328616c6c29207374727563742046434c4b6579207b0a20202020616363`),
		},
		{
			name:     "hex string wraps at 50 chars with small indent",
			indent:   "  ",
			maxWidth: 50,
			want: autogold.Want("hex string wraps at 50 chars with small indent", `0x61636365737328616c6c2920636f6e7472616374204643
  4c207b0a202061636365737328616c6c29206c6574207374
  6f72616765506174683a2053746f72616765506174680a0a
  202061636365737328616c6c29207374727563742046434c
  4b6579207b0a20202020616363`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFieldValueWithRegistry(hexString, tt.indent, nil, false, tt.maxWidth)

			// Use autogold to compare the entire string
			tt.want.Equal(t, got)

			// Verify no line exceeds maxWidth
			lines := strings.Split(got, "\n")
			for i, line := range lines {
				if len(line) > tt.maxWidth {
					t.Errorf("Line %d exceeds maxWidth %d: length=%d", i, tt.maxWidth, len(line))
				}
			}
		})
	}
}
