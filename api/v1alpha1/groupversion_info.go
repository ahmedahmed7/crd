package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
)

// GroupVersion is group version used to register these objects
var (
	GroupVersion = schema.GroupVersion{Group: "apps.mycompany.com", Version: "v1alpha1"}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme = SchemeBuilder.AddToScheme
)

// Add the list of known types to Scheme
func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion,
		&WebApp{},
		&WebAppList{},
	)
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
