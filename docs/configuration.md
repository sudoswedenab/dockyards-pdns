# Configuration

This operator relies on the Dockyards config store (`DockyardsConfigReader`) to source environment-specific values. The relevant keys are:

| Key | Description | Default |
| --- | ----------- | ------- |
| `managementDomain` | The DNS domain used for generated zones (e.g., `example.com`). | `` |
| `pdnsName` | Base name of the PowerDNS services (DNS/API). | `powerdns` |
| `pdnsNamespace` | Namespace where the PowerDNS services live. | `pdns` |
| `dockyardsPublicNamespace` | Namespace that exports the `external-dns` template. | `dockyards-public` |

`DockyardsClusterReconciler` appends the cluster UID to `managementDomain` for zone naming, and `ZoneReconciler` uses the other keys to find secrets, services, and workloads.
