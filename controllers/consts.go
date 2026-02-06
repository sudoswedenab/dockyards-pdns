package controllers

import (
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
)

const KeyManagementDomain dyconfig.Key = "managementDomain"

const (
	workloadTargetNamespace = "external-dns"
	secretPDNSAPIKey        = "PDNS_API_KEY"
)

const (
	KeyPDNSName      dyconfig.Key = "pdnsName"
	KeyPDNSNamespace dyconfig.Key = "pdnsNamespace"
)

const (
	zoneTTL            = 300
	soaRefreshInterval = 10800
	soaRetryInterval   = 3600
	soaExpireTime      = 604800
	soaNegativeCache   = 3600
)
