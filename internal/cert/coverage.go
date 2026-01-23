package cert

import "strings"

// CoverageStatus represents the coverage status of a certificate for a website
type CoverageStatus string

const (
	// CoverageStatusCovered means the certificate covers all website domains
	CoverageStatusCovered CoverageStatus = "covered"
	
	// CoverageStatusPartial means the certificate covers some but not all website domains
	CoverageStatusPartial CoverageStatus = "partial"
	
	// CoverageStatusNotCovered means the certificate does not cover any website domains
	CoverageStatusNotCovered CoverageStatus = "not_covered"
)

// CoverageResult represents the result of coverage calculation
type CoverageResult struct {
	Status         CoverageStatus `json:"status"`
	MissingDomains []string       `json:"missingDomains,omitempty"`
	CoveredDomains []string       `json:"coveredDomains,omitempty"`
}

// CalculateCoverage calculates the coverage status of certificate domains for website domains
// Returns:
// - covered: certificate_domains completely cover website_domains
// - partial: at least one covered, but not all
// - not_covered: none covered
func CalculateCoverage(certDomains, websiteDomains []string) CoverageResult {
	covered := []string{}
	missing := []string{}

	for _, wd := range websiteDomains {
		if IsCoveredBy(wd, certDomains) {
			covered = append(covered, wd)
		} else {
			missing = append(missing, wd)
		}
	}

	if len(missing) == 0 {
		return CoverageResult{
			Status:         CoverageStatusCovered,
			CoveredDomains: covered,
		}
	} else if len(covered) > 0 {
		return CoverageResult{
			Status:         CoverageStatusPartial,
			CoveredDomains: covered,
			MissingDomains: missing,
		}
	} else {
		return CoverageResult{
			Status:         CoverageStatusNotCovered,
			MissingDomains: missing,
		}
	}
}

// IsCoveredBy checks if a target domain is covered by any of the certificate domains
func IsCoveredBy(targetDomain string, certDomains []string) bool {
	for _, certDomain := range certDomains {
		if MatchDomain(certDomain, targetDomain) {
			return true
		}
	}
	return false
}

// MatchDomain checks if a certificate domain matches a target domain
// Supports exact match and wildcard match
func MatchDomain(certDomain, targetDomain string) bool {
	// Exact match
	if certDomain == targetDomain {
		return true
	}

	// Wildcard match
	if strings.HasPrefix(certDomain, "*.") {
		return MatchWildcard(certDomain, targetDomain)
	}

	return false
}

// MatchWildcard checks if a wildcard domain matches a target domain
// Rules:
// - *.example.com matches a.example.com, b.example.com
// - *.example.com does NOT match example.com (apex domain)
// - *.example.com does NOT match a.b.example.com (second-level subdomain)
func MatchWildcard(wildcardDomain, targetDomain string) bool {
	// Remove *. prefix
	baseDomain := strings.TrimPrefix(wildcardDomain, "*.")

	// Target domain must end with .baseDomain
	if !strings.HasSuffix(targetDomain, "."+baseDomain) {
		return false
	}

	// Extract prefix (before .baseDomain)
	prefix := strings.TrimSuffix(targetDomain, "."+baseDomain)

	// Prefix must not be empty (would match apex domain)
	if prefix == "" {
		return false
	}

	// Prefix must not contain dots (would match second-level subdomain)
	if strings.Contains(prefix, ".") {
		return false
	}

	return true
}
