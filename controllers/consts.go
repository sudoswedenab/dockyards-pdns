package controllers

import (
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
)

const zoneTTL = 300

const KeyManagementDomain dyconfig.Key = "managementDomain"

const (
	KeyPDNSName      dyconfig.Key = "pdnsName"
	KeyPDNSNamespace dyconfig.Key = "pdnsNamespace"
)

const (
	soaRefreshInterval = 10800
	soaRetryInterval   = 3600
	soaExpireTime      = 604800
	soaNegativeCache   = 3600
)
