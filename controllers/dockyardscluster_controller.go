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

	pdns "github.com/powerdns-operator/powerdns-operator/api/v1alpha2"
	"github.com/sudoswedenab/dockyards-backend/api/apiutil"
	dyconfig "github.com/sudoswedenab/dockyards-backend/api/config"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=clusters/status,verbs=patch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=dns.cav.enablers.ob,resources=zones,verbs=create;get;list;watch;patch

// TODO: consider making this configurable through DY Config
const zoneTTL = 300

// DockyardsClusterReconciler orchestrates PowerDNS zones for Dockyards clusters.
type DockyardsClusterReconciler struct {
	client.Client
	dyconfig.DockyardsConfigReader
}

// Reconcile ensures a DNS zone exists for an owned cluster that is not being deleted.
func (r *DockyardsClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	var cluster dockyardsv1.Cluster
	err := r.Get(ctx, req.NamespacedName, &cluster)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if !cluster.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	ownerOrganization, err := apiutil.GetOwnerOrganization(ctx, r.Client, &cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	if ownerOrganization == nil {
		logger.Info("ignoring cluster without owner organization")

		return ctrl.Result{}, nil
	}

	return r.reconcileDNSZone(ctx, &cluster)
}

// reconcileDNSZone creates or patches the PowerDNS zone tied to the provided cluster.
func (r *DockyardsClusterReconciler) reconcileDNSZone(ctx context.Context, cluster *dockyardsv1.Cluster) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	zoneName := string(cluster.GetUID()) + "." + r.GetConfigKey("managementDomain", "")

	zone := pdns.Zone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zoneName,
			Namespace: cluster.Namespace,
		},
	}

	operationResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &zone, func() error {
		zone.Labels = map[string]string{
			dockyardsv1.LabelClusterName: cluster.Name,
		}
		zone.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: dockyardsv1.GroupVersion.String(),
				Kind:       dockyardsv1.ClusterKind,
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		}

		zone.Spec = pdns.ZoneSpec{
			Kind: "Native",
			Nameservers: []string{
				"ns1." + zoneName,
			},
		}

		return nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Reconciled DNS Zone", "cluster", cluster.Name, "operationResult", operationResult)

	return ctrl.Result{}, nil
}

// SetupWithManager configures the controller runtime to manage cluster and zone resources.
func (r *DockyardsClusterReconciler) SetupWithManager(manager ctrl.Manager) error {
	scheme := manager.GetScheme()

	_ = dockyardsv1.AddToScheme(scheme)
	_ = pdns.AddToScheme(scheme)

	err := ctrl.NewControllerManagedBy(manager).
		For(&dockyardsv1.Cluster{}).
		Owns(&pdns.Zone{}).
		Complete(r)
	if err != nil {
		return err
	}

	return nil
}
