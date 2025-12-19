package controllers

// PDNSIPs holds DNS and API addresses used to configure Dockyards workloads.
type PDNSIPs struct {
	// DNSIP is a string containing an ExternalIP associated with PowerDNS's LoadBalancer service
	// intented for DNS-specific traffic
	DNSIP string
	// APIIPs is a slice of strings containing all ClusterIPs associated with PowerDNS's ClusterIP service
	// intented for API-specific traffic
	APIIPs []string
}
