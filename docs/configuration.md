# Configuration

This operator relies on the Dockyards config store (`DockyardsConfigReader`) to source environment-specific values. The relevant keys are:

| Key | Description | Default |
| --- | ----------- | ------- |
| `managementDomain` | The DNS domain used for generated zones (e.g., `example.com`). | `` |
| `pdnsName` | Base name of the PowerDNS services (DNS/API) and the secret that provides `PDNS_API_KEY`. | `powerdns` |
| `pdnsNamespace` | Namespace where the PowerDNS services live. | `pdns` |
| `publicNamespace` | Namespace that exports the `external-dns` template used to render workloads. | `dockyards-public` |

`DockyardsClusterReconciler` appends the cluster UID to `managementDomain` for zone naming, and `ZoneReconciler` uses the other keys to find secrets, services, and workloads.
