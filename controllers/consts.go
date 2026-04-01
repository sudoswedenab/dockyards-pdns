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

import (
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
)

const KeyManagementDomain dyconfig.Key = "dockyards-pdns.managementDomain"

const (
	workloadTargetNamespace = "external-dns"
	secretPDNSAPIKey        = "PDNS_API_KEY"
)

const (
	KeyPDNSName      dyconfig.Key = "dockyards-pdns.pdnsName"
	KeyPDNSNamespace dyconfig.Key = "dockyards-pdns.pdnsNamespace"
)

const (
	zoneTTL            = 300
	soaRefreshInterval = 10800
	soaRetryInterval   = 3600
	soaExpireTime      = 604800
	soaNegativeCache   = 3600
)
