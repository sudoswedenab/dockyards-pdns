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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pdnsv1 "github.com/powerdns-operator/powerdns-operator/api/v1alpha2"
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ZoneReconciler ensures PowerDNS zones are fully configured and mirrored to Dockyards resources.
type ZoneReconciler struct {
	client.Client
	*dyconfig.ConfigManager
}

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=workloads,verbs=create;patch;get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps;secrets;services,verbs=get;list;watch
// +kubebuilder:rbac:groups=dns.cav.enablers.ob,resources=rrsets,verbs=create;patch;get;list;watch

// Reconcile synchronizes RRsets and external DNS workloads once PowerDNS zones succeed.
func (r *ZoneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	var zone pdnsv1.Zone

	err := r.Get(ctx, req.NamespacedName, &zone)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if !zone.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if !zone.IsInExpectedStatus(1, "Succeeded") {
		logger.Info("Ignoring zone in non-Succeeded status", "zone", zone.Name, "syncStatus", zone.Status.SyncStatus)

		return ctrl.Result{}, nil
	}

	zoneLabels := zone.GetLabels()
	if zoneLabels[dockyardsv1.LabelClusterName] == "" {
		return ctrl.Result{}, nil
	}

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zoneLabels[dockyardsv1.LabelClusterName],
			Namespace: req.Namespace,
		},
	}

	err = r.Get(ctx, client.ObjectKeyFromObject(&cluster), &cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	ips, err := r.getPDNSIPs(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	if ips.DNSIP == "" {
		return ctrl.Result{}, errors.New("no available DNS addresses for PowerDNS")
	}
	if len(ips.APIIPs) == 0 {
		return ctrl.Result{}, errors.New("no available API addresses for PowerDNS")
	}

	_, err = r.reconcileRRsets(ctx, &zone, ips.DNSIP)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileExternalDNS(ctx, &zone, &cluster, ips.APIIPs)
}

// reconcileRRsets ensures SOA and NS records exist for the supplied zone and IP.
func (r *ZoneReconciler) reconcileRRsets(ctx context.Context, zone *pdnsv1.Zone, externalIP string) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	nsString := "ns1." + zone.Name + "."
	emailString := "hostmaster." + zone.Name + "."
	versionString := time.Now().Format("20060102") // DNS versioning format YYYYMMDDnn where nn is a counter

	record := []string{
		nsString,
		emailString,
		versionString + "01",
		strconv.Itoa(soaRefreshInterval),
		strconv.Itoa(soaRetryInterval),
		strconv.Itoa(soaExpireTime),
		strconv.Itoa(soaNegativeCache),
	}

	soaset := pdnsv1.RRset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "soa." + zone.Name,
			Namespace: zone.Namespace,
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &soaset, func() error {
		soaset.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: pdnsv1.GroupVersion.String(),
				Kind:       "Zone", // PDNS library does not offer ZoneKind
				Name:       zone.Name,
				UID:        zone.UID,
			},
		}
		soaset.Spec = pdnsv1.RRsetSpec{
			Type: "SOA",
			TTL:  uint32(3600),
			Name: zone.Name + ".",
			Records: []string{
				strings.Join(record, " "),
			},
			ZoneRef: pdnsv1.ZoneRef{
				Name: zone.Name,
				Kind: zone.Kind,
			},
		}

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Reconciled Zone SOA RRSet", "zone", zone.Name, "operationResult", operationResult)

	rrset := pdnsv1.RRset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ns1." + zone.Name,
			Namespace: zone.Namespace,
		},
	}

	operationResult, err = controllerutil.CreateOrPatch(ctx, r.Client, &rrset, func() error {
		rrset.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: pdnsv1.GroupVersion.String(),
				Kind:       "Zone", // PDNS library does not offer ZoneKind
				Name:       zone.Name,
				UID:        zone.UID,
			},
		}
		rrset.Spec = pdnsv1.RRsetSpec{
			Type: "A",
			TTL:  uint32(zoneTTL),
			Name: "ns1",
			Records: []string{
				externalIP,
			},
			ZoneRef: pdnsv1.ZoneRef{
				Name: zone.Name,
				Kind: zone.Kind,
			},
		}

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Reconciled Zone A RRSet", "zone", zone.Name, "operationResult", operationResult)

	return ctrl.Result{}, nil
}

// reconcileExternalDNS configures a Dockyards Workload that runs ExternalDNS against PowerDNS.
func (r *ZoneReconciler) reconcileExternalDNS(ctx context.Context, zone *pdnsv1.Zone, cluster *dockyardsv1.Cluster, internalIPs []string) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	pdnsName := r.GetValueOrDefault(KeyPDNSName, "")
	if pdnsName == "" {
		return ctrl.Result{}, fmt.Errorf("no value for config key `%s`", KeyPDNSName)
	}

	pdnsNamespace := r.GetValueOrDefault(KeyPDNSNamespace, "")
	if pdnsNamespace == "" {
		return ctrl.Result{}, fmt.Errorf("no value for config key `%s`", KeyPDNSNamespace)
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pdnsName,
			Namespace: pdnsNamespace,
		},
	}
	err := r.Get(ctx, client.ObjectKeyFromObject(&secret), &secret)
	if err != nil {
		return ctrl.Result{}, err
	}

	publicNamespace := r.GetValueOrDefault(dyconfig.KeyPublicNamespace, "")
	if publicNamespace == "" {
		return ctrl.Result{}, fmt.Errorf("no value for config key `%s`", dyconfig.KeyPublicNamespace)
	}

	workload := dockyardsv1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-external-dns",
			Namespace: cluster.Namespace,
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &workload, func() error {
		if metav1.HasAnnotation(workload.ObjectMeta, dockyardsv1.AnnotationSkipRemediation) {
			return nil
		}

		workload.Labels = map[string]string{
			dockyardsv1.LabelClusterName: cluster.Name,
		}

		workload.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.ClusterKind,
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		}

		workload.Spec.Provenience = dockyardsv1.ProvenienceDockyards
		workload.Spec.ClusterComponent = true
		workload.Spec.TargetNamespace = workloadTargetNamespace

		workload.Spec.WorkloadTemplateRef = &corev1.TypedObjectReference{
			Kind:      dockyardsv1.WorkloadTemplateKind,
			Name:      workloadTargetNamespace,
			Namespace: &publicNamespace,
		}

		apiKey, ok := secret.Data[secretPDNSAPIKey]
		if !ok || len(apiKey) == 0 {
			return fmt.Errorf("%s missing from secret", secretPDNSAPIKey)
		}

		raw, err := json.Marshal(map[string]any{
			"provider": "pdns",
			"sources": []string{
				"ingress",
			},
			"credentials": map[string]string{
				"pdnsApiKey": string(apiKey),
			},
			"env": map[string]string{
				"EXTERNAL_DNS_PDNS_SERVER":   "http://" + internalIPs[0] + ":8081",
				"EXTERNAL_DNS_DOMAIN_FILTER": zone.Name,
			},
		})
		if err != nil {
			return err
		}

		workload.Spec.Input = &apiextensionsv1.JSON{
			Raw: raw,
		}

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Reconciled Workload", "cluster", cluster.Name, "workload", workload.Name, "operationResult", operationResult)

	return ctrl.Result{}, nil
}

// getPDNSIPs fetches the DNS and API service addresses exported by PowerDNS components.
func (r *ZoneReconciler) getPDNSIPs(ctx context.Context) (*PDNSIPs, error) {
	pdnsName := r.GetValueOrDefault(KeyPDNSName, "")
	if pdnsName == "" {
		return nil, fmt.Errorf("no value for config key `%s`", KeyPDNSName)
	}

	pdnsNamespace := r.GetValueOrDefault(KeyPDNSNamespace, "")
	if pdnsNamespace == "" {
		return nil, fmt.Errorf("no value for config key `%s`", KeyPDNSNamespace)
	}

	pdnsDNSService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pdnsName + "-dns",
			Namespace: pdnsNamespace,
		},
	}

	err := r.Get(ctx, client.ObjectKeyFromObject(&pdnsDNSService), &pdnsDNSService)
	if err != nil {
		return &PDNSIPs{}, err
	}

	pdnsAPIService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pdnsName + "-api",
			Namespace: pdnsNamespace,
		},
	}

	err = r.Get(ctx, client.ObjectKeyFromObject(&pdnsAPIService), &pdnsAPIService)
	if err != nil {
		return &PDNSIPs{}, err
	}

	return &PDNSIPs{DNSIP: pdnsDNSService.Status.LoadBalancer.Ingress[0].IP, APIIPs: pdnsAPIService.Spec.ClusterIPs}, nil
}

// SetupWithManager registers the Zone controller with the provided manager.
func (r *ZoneReconciler) SetupWithManager(manager ctrl.Manager) error {
	scheme := manager.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)
	_ = pdnsv1.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(manager).
		For(&pdnsv1.Zone{}).
		Complete(r)
	if err != nil {
		return err
	}

	return nil
}
