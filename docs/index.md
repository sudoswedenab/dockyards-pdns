# Dockyards PowerDNS Operator

The Dockyards PowerDNS Operator keeps Dockyards clusters, PowerDNS zones, and ExternalDNS workloads coordinated. It runs as a standard controller-runtime service and performs the following steps:

1. Watches `dockyards.io/v1alpha3` clusters and creates a PowerDNS `Zone` for each owned, active cluster.
2. Watches the resulting PowerDNS `Zone` resources to ensure SOA/NS `RRsets` are present and that Dockyards workloads such as ExternalDNS are configured against the PowerDNS API.
3. Relies on the Kubernetes API to discover things like the management domain, PowerDNS service names, and the public namespace that holds shared templates.

This repository ships a `main.go` entrypoint, two reconcilers under `./controllers`, and a `DockyardsConfigReader` implementation provided by `dockyards-backend/api/config`.

TechDocs in Backstage consume this MkDocs layout (`mkdocs.yml`) and the Markdown files under `docs/`. Deploy the generated site through Backstage TechDocs to expose the operator's description, configuration options, and operational guidance.
