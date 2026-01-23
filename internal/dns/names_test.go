package dns

import "testing"

func TestToFQDN(t *testing.T) {
	tests := []struct {
		name     string
		zone     string
		input    string
		expected string
	}{
		{
			name:     "@ converts to zone",
			zone:     "baidu.com",
			input:    "@",
			expected: "baidu.com",
		},
		{
			name:     "www converts to www.zone",
			zone:     "baidu.com",
			input:    "www",
			expected: "www.baidu.com",
		},
		{
			name:     "a.b converts to a.b.zone",
			zone:     "baidu.com",
			input:    "a.b",
			expected: "a.b.baidu.com",
		},
		{
			name:     "empty name defaults to @",
			zone:     "example.com",
			input:    "",
			expected: "example.com",
		},
		{
			name:     "already FQDN returns as-is",
			zone:     "example.com",
			input:    "test.example.com",
			expected: "test.example.com",
		},
		{
			name:     "zone itself returns as-is",
			zone:     "example.com",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "whitespace is trimmed",
			zone:     " baidu.com ",
			input:    " www ",
			expected: "www.baidu.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToFQDN(tt.zone, tt.input)
			if result != tt.expected {
				t.Errorf("ToFQDN(%q, %q) = %q; want %q", tt.zone, tt.input, result, tt.expected)
			}
		})
	}
}
