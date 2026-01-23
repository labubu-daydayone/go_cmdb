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
