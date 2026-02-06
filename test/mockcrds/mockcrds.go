package mockcrds

import (
	pdnsv1 "github.com/powerdns-operator/powerdns-operator/api/v1alpha2"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	DockyardsOrganization = mockCRD(dockyardsv1.OrganizationKind, "organization", dockyardsv1.GroupVersion.Group, dockyardsv1.GroupVersion.Version)
	DockyardsCluster      = mockCRD(dockyardsv1.ClusterKind, "clusters", dockyardsv1.GroupVersion.Group, dockyardsv1.GroupVersion.Version)
	DockyardsWorkload     = mockCRD(dockyardsv1.WorkloadKind, "workloads", dockyardsv1.GroupVersion.Group, dockyardsv1.GroupVersion.Version)
	PDNSZone              = mockCRD("Zone", "zones", pdnsv1.GroupVersion.Group, pdnsv1.GroupVersion.Version)
	PDNSRRset             = mockCRD("RRset", "rrsets", pdnsv1.GroupVersion.Group, pdnsv1.GroupVersion.Version)

	CRDs = []*apiextensionsv1.CustomResourceDefinition{
		DockyardsOrganization,
		DockyardsCluster,
		DockyardsWorkload,
		PDNSZone,
		PDNSRRset,
	}
)

func mockCRD(kind, plural, group, version string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiextensionsv1.SchemeGroupVersion.String(),
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: plural + "." + group,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: group,
			Scope: apiextensionsv1.NamespaceScoped,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural: plural,
				Kind:   kind,
			},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    version,
					Served:  true,
					Storage: true,
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"spec": {
									Type:                   "object",
									XPreserveUnknownFields: ptr.To(true),
								},
								"status": {
									Type:                   "object",
									XPreserveUnknownFields: ptr.To(true),
								},
							},
						},
					},
				},
			},
		},
	}
}
