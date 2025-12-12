# Zone Reconciler

`controllers/ZoneReconciler` (see `controllers/zone_controller.go`) acts once PowerDNS reports a zone in the `Succeeded` state:

- Fetches the owning Dockyards cluster referenced through labels.
- Resolves the PowerDNS DNS and API service IPs using configuration keys (`pdnsName`, `pdnsNamespace`).
- Ensures the SOA RRset is present with a consistent serial, and that the `ns1` A record points at the DNS service external IP.
- Creates or patches a Dockyards `Workload` (named `<cluster>-external-dns`) that deploys ExternalDNS with the PowerDNS API credentials, domain filter, and target server.

By reconciling both RRsets and workloads, this controller keeps PowerDNS and Dockyards in sync.
