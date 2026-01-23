package dnstypes

// DNSRecord represents a DNS record for provider operations
type DNSRecord struct {
	Type    string // A, AAAA, CNAME, TXT
	Name    string // FQDN (e.g., www.example.com)
	Value   string // IP address or target
	TTL     int    // Time to live
	Proxied bool   // Cloudflare proxy (orange cloud)
}
