package linegroup

import (
	"strings"
	"testing"
)

func TestGenerateCNAMEPrefix(t *testing.T) {
	t.Run("format validation", func(t *testing.T) {
		prefix := GenerateCNAMEPrefix()
		
		// Must start with "lg-"
		if !strings.HasPrefix(prefix, "lg-") {
			t.Errorf("prefix must start with 'lg-', got: %s", prefix)
		}
		
		// Total length should be 19 (3 for "lg-" + 16 for hex)
		if len(prefix) != 19 {
			t.Errorf("prefix length must be 19, got: %d (prefix: %s)", len(prefix), prefix)
		}
		
		// Hex part should be valid hex characters
		hexPart := prefix[3:]
		if len(hexPart) != 16 {
			t.Errorf("hex part length must be 16, got: %d", len(hexPart))
		}
		
		// Check if all characters in hex part are valid hex digits
		for _, c := range hexPart {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("invalid hex character in prefix: %c (prefix: %s)", c, prefix)
			}
		}
	})
	
	t.Run("uniqueness check", func(t *testing.T) {
		// Generate 100 prefixes and check for duplicates
		generated := make(map[string]bool)
		count := 100
		
		for i := 0; i < count; i++ {
			prefix := GenerateCNAMEPrefix()
			
			if generated[prefix] {
				t.Errorf("duplicate prefix generated: %s", prefix)
			}
			
			generated[prefix] = true
		}
		
		if len(generated) != count {
			t.Errorf("expected %d unique prefixes, got: %d", count, len(generated))
		}
	})
}
