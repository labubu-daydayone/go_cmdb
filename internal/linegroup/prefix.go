package linegroup

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateCNAMEPrefix generates a unique CNAME prefix for line groups
// Format: lg-<16 hex characters>
// Example: lg-a0b719f2b1d6f6aa
func GenerateCNAMEPrefix() string {
	// Generate 8 random bytes (will become 16 hex characters)
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a default pattern if random generation fails
		// This should never happen in practice
		return "lg-0000000000000000"
	}
	
	// Convert to hex string
	hexStr := hex.EncodeToString(bytes)
	
	// Return with "lg-" prefix
	return "lg-" + hexStr
}
