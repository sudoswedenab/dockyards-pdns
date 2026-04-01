// Copyright 2025 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
