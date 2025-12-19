# Operations Guide

## Deployment

1. Make sure the Dockyards CRDs, PowerDNS, and PowerDNS operator are installed in your cluster.
2. Build the controller binary (`go build ./...`) and package it into an image.
3. Deploy the operator and grant it RBAC access to `dockyards.io` clusters, PowerDNS zones, and RRsets.
4. Populate the Dockyards config with the keys above (`managementDomain`, `pdnsName`, `pdnsNamespace`, and `publicNamespace`) so the controller knows where to find PowerDNS services and templates.
5. The operator watches clusters and zones automatically once running.

## Troubleshooting

- Logs mention missing zones or workloads? Verify the namespace defined by `publicNamespace` exports the `external-dns` template and that the `dockyards-backend` APIs are reachable.
- ExternalDNS workload fails to start? Confirm the `PDNS_API_KEY` secret exists in the PowerDNS namespace and the API service has healthy ClusterIPs.
- Zone stuck in non-`Succeeded` status? Inspect the PowerDNS operator or backend for syncing issues.

## Backstage TechDocs

Backstage TechDocs builds this site via `mkdocs.yml`. Point TechDocs at the repository and enable the MkDocs generator so the `docs/` directory becomes searchable documentation.
