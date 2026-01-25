package dns

import "strings"

// ToFQDN converts a relative DNS name to a Fully Qualified Domain Name (FQDN)
//
// Rules:
// - zone = "baidu.com"
// - name = "@"    -> fqdn = "baidu.com"
// - name = "www"  -> fqdn = "www.baidu.com"
// - name = "a.b"  -> fqdn = "a.b.baidu.com"
//
// If name is already a FQDN (contains the zone), it will be returned as-is.
func ToFQDN(zone string, name string) string {
	// Normalize inputs
	zone = strings.TrimSpace(zone)
	name = strings.TrimSpace(name)

	// Handle empty name (default to @)
	if name == "" {
		name = "@"
	}

	// Handle @ (root domain)
	if name == "@" {
		return zone
	}

	// If name already contains the zone, return as-is (already FQDN)
	if strings.HasSuffix(name, "."+zone) || name == zone {
		return name
	}

	// Otherwise, append zone to name
	return name + "." + zone
}

// NormalizeRelativeName converts any name format to a relative name (non-FQDN)
//
// Rules:
// - zone = "example.com"
// - name = "example.com"           -> "@"
// - name = "www.example.com"       -> "www"
// - name = "a.b.example.com"       -> "a.b"
// - name = "www.example.com."      -> "www" (trailing dot removed)
// - name = "@"                     -> "@"
// - name = "abc"                   -> "abc"
// - name = "a.b"                   -> "a.b"
//
// This ensures domain_dns_records.name always stores relative names.
func NormalizeRelativeName(name, zone string) string {
	// Normalize inputs
	zone = strings.TrimSpace(zone)
	name = strings.TrimSpace(name)
	
	// Remove trailing dot
	name = strings.TrimSuffix(name, ".")
	zone = strings.TrimSuffix(zone, ".")
	
	// Handle empty name (default to @)
	if name == "" {
		return "@"
	}
	
	// If name equals zone, return @
	if name == zone {
		return "@"
	}
	
	// If name ends with ".zone", extract the relative part
	if strings.HasSuffix(name, "."+zone) {
		relName := strings.TrimSuffix(name, "."+zone)
		if relName == "" {
			return "@"
		}
		return relName
	}
	
	// If name is already @ or a relative name (abc, a.b), return as-is
	return name
}
