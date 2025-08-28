package main

import (
	"reflect"

	apiv1 "github.com/jayantasamaddar/quick-reference-kubernetes/solutions/hello-crd-scaling/api/v1"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Accept spec or metadata changes; ignore status-only updates.
var serviceDeploymentPredicate = predicate.Funcs{
	CreateFunc:  func(e event.CreateEvent) bool { return true },
	DeleteFunc:  func(e event.DeleteEvent) bool { return true },
	GenericFunc: func(e event.GenericEvent) bool { return true },
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldObj, ok1 := e.ObjectOld.(*apiv1.ServiceDeployment)
		newObj, ok2 := e.ObjectNew.(*apiv1.ServiceDeployment)
		if !ok1 || !ok2 {
			// Not our type; be permissive.
			return true
		}

		// 1) Spec changed?
		if !reflect.DeepEqual(oldObj.Spec, newObj.Spec) {
			return true
		}

		// 2) Metadata changes we care about
		if !reflect.DeepEqual(oldObj.GetLabels(), newObj.GetLabels()) {
			return true
		}
		if !reflect.DeepEqual(oldObj.GetAnnotations(), newObj.GetAnnotations()) {
			return true
		}
		if !reflect.DeepEqual(oldObj.GetFinalizers(), newObj.GetFinalizers()) {
			return true
		}
		// 3) Deletion timestamp toggled?
		// If the object is being deleted (non-nil deletionTimestamp), or was deleted, reconcile so the controller can finalize or clean up.
		if (oldObj.GetDeletionTimestamp() == nil) != (newObj.GetDeletionTimestamp() == nil) {
			return true
		}

		// Otherwise: treat as status-only (or other noise) -> ignore
		return false
	},
}
