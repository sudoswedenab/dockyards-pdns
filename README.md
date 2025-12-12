# Dockyards PowerDNS Operator

This repository supplies a small Kubernetes controller that keeps Dockyards clusters in sync with PowerDNS zones. As clusters are created they get healthy DNS zones managed by the operator, and Dockyards workloads such as ExternalDNS are reconciled to publish records through PowerDNS.

## Key Components

- **`DockyardsClusterReconciler`** (controllers/dockyardscluster_controller.go) watches `dockyards.io` clusters. When a cluster is owned by an organization and not being deleted it ensures a PowerDNS `Zone` exists with the right labels, ownership, and nameserver records.
- **`ZoneReconciler`** (controllers/zone_controller.go) watches the PowerDNS `Zone` resource. Once a zone enters a succeeded state it create/patches the SOA/NS RRsets, discovers PowerDNS API and DNS endpoints, and configures a Dockyards `Workload` that runs ExternalDNS pointing at PowerDNS.
- **`Configuration`** is driven by the Dockyards config reader; keys such as `managementDomain`, `pdnsName`, `pdnsNamespace`, and `dockyardsPublicNamespace` tell the controller how to wire everything together.

## Requirements

- Go 1.25+
- Kubernetes-style controller-runtime tooling (already captured in `go.mod`)
- Access to the Dockyards backend custom resources and PowerDNS services referenced in the config

## Development

```bash
# build the controller binary
go build ./...

# run gofmt or tests if needed
go test ./...
```

`main.go` wires the reconcilers with a controller-runtime manager and relies on `generate.go` for helper annotations when the code is scaffolded.

## Deployment

1. Build/push the controller image as needed.
2. Deploy the controller into a cluster that has the Dockyards CRDs and PowerDNS operator installed.
3. Set the required config keys via the Dockyards config API so `managementDomain`, `pdnsName`, `pdnsNamespace`, and `dockyardsPublicNamespace` reflect your environment.

## Troubleshooting

- Check controller logs for failure hints when zone or workload reconciliation fails.
- Validate that the PowerDNS services expose LoadBalancer DNS and ClusterIP addresses (`<pdnsName>-dns` and `<pdnsName>-api`).
- Confirm the `Workload` custom resource template `external-dns` exists in the public namespace referenced by the config.

## License

Apache 2.0. See [LICENSE](./LICENSE.md) if provided in the broader project.
