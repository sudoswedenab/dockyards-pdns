package controllers

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	pdnsv1 "github.com/powerdns-operator/powerdns-operator/api/v1alpha2"
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"github.com/sudoswedenab/dockyards-pdns/test/mockcrds"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestClusterReconciler(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	env := envtest.Environment{
		CRDs: mockcrds.CRDs,
	}

	textHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	slogr := logr.FromSlogHandler(textHandler)

	ctrl.SetLogger(slogr)

	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := env.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cancel()
		err := env.Stop()
		if err != nil {
			panic(err)
		}
	})

	scheme := runtime.NewScheme()

	_ = corev1.AddToScheme(scheme)
	_ = dockyardsv1.AddToScheme(scheme)
	_ = pdnsv1.AddToScheme(scheme)

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}

	mgr, err := manager.New(cfg, manager.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	if !mgr.GetCache().WaitForCacheSync(ctx) {
		t.Fatalf("could not sync cache")
	}

	dockyardsConfigManager := dyconfig.NewFakeConfigManager(map[string]string{
		string(KeyManagementDomain):         "test.com",
		string(KeyPDNSName):                 "test-pdns",
		string(KeyPDNSNamespace):            "test-ns",
		string(dyconfig.KeyPublicNamespace): "public-ns",
	})

	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-ns-",
		},
	}

	err = c.Create(ctx, &namespace)
	if err != nil {
		t.Fatal(err)
	}

	organization := dockyardsv1.Organization{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-org-",
			Namespace:    namespace.Name,
		},
	}

	err = c.Create(ctx, &organization)
	if err != nil {
		t.Fatal(err)
	}

	cluster := dockyardsv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-cluster-",
			Namespace:    organization.Namespace,
		},
	}

	err = c.Create(ctx, &cluster)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test cluster reconciliation", func(t *testing.T) {
		r := DockyardsClusterReconciler{c, dockyardsConfigManager}
		_, err = r.reconcileDNSZone(ctx, &cluster)
		if err != nil {
			t.Fatal(err)
		}

		managementDomain, found := dockyardsConfigManager.GetValueForKey(KeyManagementDomain)
		if !found {
			t.Fatalf("unable to get key %s from dockyards config", KeyManagementDomain)
		}

		zoneName := string(cluster.GetUID()) + "." + managementDomain
		zone := pdnsv1.Zone{
			ObjectMeta: metav1.ObjectMeta{
				Name:      zoneName,
				Namespace: cluster.Namespace,
			},
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(&zone), &zone)
		if err != nil {
			t.Fatal(err)
		}

		if zone.Spec.Nameservers == nil {
			t.Error("Unable to find Zone")
		}

		expectedZoneOwner := []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.ClusterKind,
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		}
		if !cmp.Equal(expectedZoneOwner, zone.ObjectMeta.OwnerReferences) {
			t.Error(cmp.Diff(expectedZoneOwner, zone.ObjectMeta.OwnerReferences))
		}

		expectedZoneSpec := pdnsv1.ZoneSpec{
			Kind: "Native",
			Nameservers: []string{
				"ns1." + zoneName,
			},
		}

		if !cmp.Equal(expectedZoneSpec, zone.Spec) {
			t.Error(cmp.Diff(expectedZoneSpec, zone.Spec))
		}

		externalIP := "1.2.3.4"
		z := ZoneReconciler{c, dockyardsConfigManager}
		_, err = z.reconcileRRsets(ctx, &zone, externalIP)
		if err != nil {
			t.Fatal(err)
		}

		rrsetSOA := pdnsv1.RRset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "soa." + zone.Name,
				Namespace: zone.Namespace,
			},
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(&rrsetSOA), &rrsetSOA)
		if err != nil {
			t.Fatal(err)
		}

		expectedSOAOwner := []metav1.OwnerReference{
			{
				APIVersion: pdnsv1.GroupVersion.String(),
				Kind:       "Zone",
				Name:       zone.Name,
				UID:        zone.UID,
			},
		}

		if !cmp.Equal(expectedSOAOwner, rrsetSOA.ObjectMeta.OwnerReferences) {
			t.Error(cmp.Diff(expectedSOAOwner, rrsetSOA.ObjectMeta.OwnerReferences))
		}

		nsString := "ns1." + zone.Name + "."
		emailString := "hostmaster." + zone.Name + "."
		versionString := time.Now().Format("20060102")

		record := []string{
			nsString,
			emailString,
			versionString + "01",
			strconv.Itoa(soaRefreshInterval),
			strconv.Itoa(soaRetryInterval),
			strconv.Itoa(soaExpireTime),
			strconv.Itoa(soaNegativeCache),
		}

		expectedSOASpec := pdnsv1.RRsetSpec{
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

		if !cmp.Equal(expectedSOASpec, rrsetSOA.Spec) {
			t.Error(cmp.Diff(expectedSOASpec, rrsetSOA.Spec))
		}

		rrsetA := pdnsv1.RRset{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ns1." + zone.Name,
				Namespace: zone.Namespace,
			},
		}

		err = c.Get(ctx, client.ObjectKeyFromObject(&rrsetA), &rrsetA)
		if err != nil {
			t.Fatal(err)
		}

		expectedAOwner := []metav1.OwnerReference{
			{
				APIVersion: pdnsv1.GroupVersion.String(),
				Kind:       "Zone",
				Name:       zone.Name,
				UID:        zone.UID,
			},
		}

		if !cmp.Equal(expectedAOwner, rrsetA.ObjectMeta.OwnerReferences) {
			t.Error(cmp.Diff(expectedAOwner, rrsetA.ObjectMeta.OwnerReferences))
		}

		expectedASpec := pdnsv1.RRsetSpec{
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

		if !cmp.Equal(expectedASpec, rrsetA.Spec) {
			t.Error(cmp.Diff(expectedASpec, rrsetA.Spec))
		}

		pdnsNamespace, found := dockyardsConfigManager.GetValueForKey(KeyPDNSNamespace)
		if !found {
			t.Errorf("Key %s missing from dockyards config\n", KeyPDNSNamespace)
		}

		pdnsNS := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: pdnsNamespace,
			},
		}

		err = c.Create(ctx, &pdnsNS)
		if err != nil {
			t.Fatal(err)
		}

		secretName, found := dockyardsConfigManager.GetValueForKey(KeyPDNSName)
		if !found {
			t.Errorf("Key %s missing from dockyards config\n", KeyPDNSName)
		}

		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: pdnsNamespace,
			},
			StringData: map[string]string{
				secretPDNSAPIKey: "test-api-key",
			},
		}

		err = c.Create(ctx, &secret)
		if err != nil {
			t.Fatal(err)
		}

		internalIPs := []string{
			"5.6.7.8",
		}
		_, err = z.reconcileExternalDNS(ctx, &zone, &cluster, internalIPs)
		if err != nil {
			t.Fatal(err)
		}

		workload := dockyardsv1.Workload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-external-dns",
				Namespace: cluster.Namespace,
			},
		}
		err = c.Get(ctx, client.ObjectKeyFromObject(&workload), &workload)

		expectedWorkloadOwner := []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.ClusterKind,
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		}

		if !cmp.Equal(expectedWorkloadOwner, workload.ObjectMeta.OwnerReferences) {
			t.Error(cmp.Diff(expectedWorkloadOwner, workload.ObjectMeta.OwnerReferences))
		}

		publicNamespace, found := dockyardsConfigManager.GetValueForKey(dyconfig.KeyPublicNamespace)
		if !found {
			t.Fatalf("unable to get key %s from dockyards config", KeyManagementDomain)
		}

		expectedInput, err := json.Marshal(map[string]any{
			"provider": "pdns",
			"sources": []string{
				"ingress",
			},
			"credentials": map[string]string{
				"pdnsApiKey": string(secret.Data[secretPDNSAPIKey]),
			},
			"env": map[string]string{
				"EXTERNAL_DNS_PDNS_SERVER":   "http://" + internalIPs[0] + ":8081",
				"EXTERNAL_DNS_DOMAIN_FILTER": zone.Name,
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		expectedWorkloadSpec := dockyardsv1.WorkloadSpec{
			Provenience:      dockyardsv1.ProvenienceDockyards,
			ClusterComponent: true,
			TargetNamespace:  workloadTargetNamespace,
			WorkloadTemplateRef: &corev1.TypedObjectReference{
				Kind:      dockyardsv1.WorkloadTemplateKind,
				Name:      workloadTargetNamespace,
				Namespace: &publicNamespace,
			},
			Input: &apiextensionsv1.JSON{
				Raw: expectedInput,
			},
		}

		if !cmp.Equal(expectedWorkloadSpec, workload.Spec) {
			t.Error(cmp.Diff(expectedWorkloadSpec, workload.Spec))
		}
	})
}
