# Cluster Reconciler

`controllers/DockyardsClusterReconciler` (see `controllers/dockyardscluster_controller.go`) monitors `dockyards.io/v1alpha3` clusters. Its responsibilities:

- Skip clusters with no owning organization or those that are marked for deletion.
- For each active cluster, construct a PowerDNS `Zone` whose name combines the cluster UID with the configured `managementDomain`.
- Label and owner-reference the `Zone` so changes propagate back to the owning cluster.
- Provide the `ns1.<zone>` nameserver that PowerDNS relies on.

This controller uses controller-runtime's `CreateOrPatch` to make its operations idempotent and registers both Dockyards and PowerDNS schemes with the manager (`SetupWithManager`).
