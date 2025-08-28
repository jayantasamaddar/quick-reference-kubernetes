package v1

// The types here are for receiving json data from Kubernetes API

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceDeploymentStatus struct {
	DesiredReplicas   int32  `json:"desiredReplicas,omitempty"`
	ReadyReplicas     int32  `json:"readyReplicas,omitempty"`
	UpdatedReplicas   int32  `json:"updatedReplicas,omitempty"`
	AvailableReplicas int32  `json:"availableReplicas,omitempty"`
	Ready             string `json:"ready,omitempty"` // e.g. "3/3"

	ServiceType string `json:"serviceType,omitempty"`
	ClusterIP   string `json:"clusterIP,omitempty"`
	ExternalIPs string `json:"externalIPs,omitempty"`
	Ports       string `json:"ports,omitempty"` // e.g. "80/TCP,443/TCP"
}

type ServiceDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`

	Items []ServiceDeployment `json:"items"`
}

type ServiceDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec   ServiceDeploymentSpec   `json:"spec"`
	Status ServiceDeploymentStatus `json:"status,omitzero"`
}

type ServiceDeploymentSpec struct {
	Replicas   int32                        `json:"replicas"`
	Containers []corev1.Container           `json:"containers"`
	Service    ServiceDeploymentSpecService `json:"service"`
}

type ServiceDeploymentSpecService struct {
	Name  string               `json:"name,omitzero"`
	Type  corev1.ServiceType   `json:"type"`
	Ports []corev1.ServicePort `json:"ports,omitempty"`
}

// DeepCopyInto
func (in *ServiceDeployment) DeepCopyInto(out *ServiceDeployment) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta

	containersCopy := make([]corev1.Container, len(in.Spec.Containers))
	copy(containersCopy, in.Spec.Containers)

	portsCopy := make([]corev1.ServicePort, len(in.Spec.Service.Ports))
	copy(portsCopy, in.Spec.Service.Ports)

	out.Spec = ServiceDeploymentSpec{
		Replicas:   in.Spec.Replicas,
		Containers: containersCopy,
		Service: ServiceDeploymentSpecService{
			Name:  in.Spec.Service.Name,
			Type:  in.Spec.Service.Type,
			Ports: portsCopy,
		},
	}
	out.Status = in.Status // shallow copy is fine for value fields
}

// DeepCopy returns a pointer to a new ServiceDeploment by copying the receiver.
func (in *ServiceDeployment) DeepCopy() *ServiceDeployment {
	if in == nil {
		return nil
	}
	out := new(ServiceDeployment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of the receiver, creating a new runtime.Object.
func (in *ServiceDeployment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ServiceDeploymentList) DeepCopyObject() runtime.Object {
	out := new(ServiceDeploymentList)
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]ServiceDeployment, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
	return out
}
