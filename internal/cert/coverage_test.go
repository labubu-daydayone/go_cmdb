package cert

import (
	"testing"
)

// TestMatchWildcard tests wildcard domain matching
func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		name           string
		wildcardDomain string
		targetDomain   string
		expected       bool
	}{
		// Positive cases
		{
			name:           "wildcard matches first-level subdomain",
			wildcardDomain: "*.example.com",
			targetDomain:   "a.example.com",
			expected:       true,
		},
		{
			name:           "wildcard matches another first-level subdomain",
			wildcardDomain: "*.example.com",
			targetDomain:   "www.example.com",
			expected:       true,
		},
		{
			name:           "wildcard matches api subdomain",
			wildcardDomain: "*.example.com",
			targetDomain:   "api.example.com",
			expected:       true,
		},
		// Negative cases
		{
			name:           "wildcard does NOT match apex domain",
			wildcardDomain: "*.example.com",
			targetDomain:   "example.com",
			expected:       false,
		},
		{
			name:           "wildcard does NOT match second-level subdomain",
			wildcardDomain: "*.example.com",
			targetDomain:   "a.b.example.com",
			expected:       false,
		},
		{
			name:           "wildcard does NOT match different base domain",
			wildcardDomain: "*.example.com",
			targetDomain:   "a.example.org",
			expected:       false,
		},
		{
			name:           "wildcard does NOT match unrelated domain",
			wildcardDomain: "*.example.com",
			targetDomain:   "test.com",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchWildcard(tt.wildcardDomain, tt.targetDomain)
			if result != tt.expected {
				t.Errorf("MatchWildcard(%q, %q) = %v, want %v",
					tt.wildcardDomain, tt.targetDomain, result, tt.expected)
			}
		})
	}
}

// TestMatchDomain tests domain matching (exact + wildcard)
func TestMatchDomain(t *testing.T) {
	tests := []struct {
		name         string
		certDomain   string
		targetDomain string
		expected     bool
	}{
		// Exact match
		{
			name:         "exact match",
			certDomain:   "example.com",
			targetDomain: "example.com",
			expected:     true,
		},
		{
			name:         "exact match with subdomain",
			certDomain:   "www.example.com",
			targetDomain: "www.example.com",
			expected:     true,
		},
		// Wildcard match
		{
			name:         "wildcard match",
			certDomain:   "*.example.com",
			targetDomain: "a.example.com",
			expected:     true,
		},
		// No match
		{
			name:         "no match - different domains",
			certDomain:   "example.com",
			targetDomain: "example.org",
			expected:     false,
		},
		{
			name:         "no match - wildcard vs apex",
			certDomain:   "*.example.com",
			targetDomain: "example.com",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchDomain(tt.certDomain, tt.targetDomain)
			if result != tt.expected {
				t.Errorf("MatchDomain(%q, %q) = %v, want %v",
					tt.certDomain, tt.targetDomain, result, tt.expected)
			}
		})
	}
}

// TestIsCoveredBy tests if a domain is covered by certificate domains
func TestIsCoveredBy(t *testing.T) {
	tests := []struct {
		name         string
		targetDomain string
		certDomains  []string
		expected     bool
	}{
		{
			name:         "covered by exact match",
			targetDomain: "example.com",
			certDomains:  []string{"example.com", "www.example.com"},
			expected:     true,
		},
		{
			name:         "covered by wildcard",
			targetDomain: "a.example.com",
			certDomains:  []string{"*.example.com"},
			expected:     true,
		},
		{
			name:         "covered by multiple domains",
			targetDomain: "www.example.com",
			certDomains:  []string{"example.com", "*.example.com"},
			expected:     true,
		},
		{
			name:         "not covered",
			targetDomain: "example.com",
			certDomains:  []string{"*.example.com"},
			expected:     false,
		},
		{
			name:         "not covered - second-level subdomain",
			targetDomain: "a.b.example.com",
			certDomains:  []string{"*.example.com"},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCoveredBy(tt.targetDomain, tt.certDomains)
			if result != tt.expected {
				t.Errorf("IsCoveredBy(%q, %v) = %v, want %v",
					tt.targetDomain, tt.certDomains, result, tt.expected)
			}
		})
	}
}

// TestCalculateCoverage tests coverage calculation
func TestCalculateCoverage(t *testing.T) {
	tests := []struct {
		name            string
		certDomains     []string
		websiteDomains  []string
		expectedStatus  CoverageStatus
		expectedMissing []string
		expectedCovered []string
	}{
		// Scenario 1: partial coverage (missing apex domain)
		{
			name:            "partial - missing apex domain",
			certDomains:     []string{"*.example.com"},
			websiteDomains:  []string{"example.com", "www.example.com"},
			expectedStatus:  CoverageStatusPartial,
			expectedMissing: []string{"example.com"},
			expectedCovered: []string{"www.example.com"},
		},
		// Scenario 2: covered
		{
			name:            "covered - wildcard matches all",
			certDomains:     []string{"*.example.com"},
			websiteDomains:  []string{"a.example.com"},
			expectedStatus:  CoverageStatusCovered,
			expectedMissing: nil,
			expectedCovered: []string{"a.example.com"},
		},
		// Scenario 3: not covered (second-level subdomain)
		{
			name:            "not covered - second-level subdomain",
			certDomains:     []string{"*.example.com"},
			websiteDomains:  []string{"a.b.example.com"},
			expectedStatus:  CoverageStatusNotCovered,
			expectedMissing: []string{"a.b.example.com"},
			expectedCovered: nil,
		},
		// Scenario 4: covered with exact match
		{
			name:            "covered - exact match",
			certDomains:     []string{"example.com", "www.example.com"},
			websiteDomains:  []string{"example.com", "www.example.com"},
			expectedStatus:  CoverageStatusCovered,
			expectedMissing: nil,
			expectedCovered: []string{"example.com", "www.example.com"},
		},
		// Scenario 5: covered with wildcard + exact
		{
			name:            "covered - wildcard + exact",
			certDomains:     []string{"example.com", "*.example.com"},
			websiteDomains:  []string{"example.com", "www.example.com"},
			expectedStatus:  CoverageStatusCovered,
			expectedMissing: nil,
			expectedCovered: []string{"example.com", "www.example.com"},
		},
		// Scenario 6: partial with multiple missing
		{
			name:            "partial - multiple missing",
			certDomains:     []string{"*.example.com"},
			websiteDomains:  []string{"example.com", "www.example.com", "api.example.com"},
			expectedStatus:  CoverageStatusPartial,
			expectedMissing: []string{"example.com"},
			expectedCovered: []string{"www.example.com", "api.example.com"},
		},
		// Scenario 7: not covered - completely different domain
		{
			name:            "not covered - different domain",
			certDomains:     []string{"*.example.com"},
			websiteDomains:  []string{"example.org"},
			expectedStatus:  CoverageStatusNotCovered,
			expectedMissing: []string{"example.org"},
			expectedCovered: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCoverage(tt.certDomains, tt.websiteDomains)

			// Check status
			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}

			// Check missing domains
			if !equalStringSlices(result.MissingDomains, tt.expectedMissing) {
				t.Errorf("MissingDomains = %v, want %v", result.MissingDomains, tt.expectedMissing)
			}

			// Check covered domains
			if !equalStringSlices(result.CoveredDomains, tt.expectedCovered) {
				t.Errorf("CoveredDomains = %v, want %v", result.CoveredDomains, tt.expectedCovered)
			}
		})
	}
}

// equalStringSlices checks if two string slices are equal (order-independent)
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Handle nil vs empty slice
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	// Convert to map for order-independent comparison
	aMap := make(map[string]bool)
	for _, s := range a {
		aMap[s] = true
	}

	for _, s := range b {
		if !aMap[s] {
			return false
		}
	}

	return true
}
