package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GroupName    string = "k8s.example.com"
	GroupVersion string = "v1"
	Kind         string = "ServiceDeployment"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)

	AddToScheme = SchemeBuilder.AddToScheme
)

// Function to ensure when we use a Kubernetes client in our controller,
// these types are embedded in our controller package so it is aware of our new schema
func addKnownTypes(scheme *runtime.Scheme) error {
	// Only can add objects that implement interface `runtime.Object`.
	scheme.AddKnownTypes(SchemeGroupVersion, &ServiceDeployment{}, &ServiceDeploymentList{})

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
