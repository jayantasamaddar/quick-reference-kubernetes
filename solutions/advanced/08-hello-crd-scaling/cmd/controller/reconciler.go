package main

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/jayantasamaddar/quick-reference-kubernetes/solutions/hello-crd-scaling/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// A reconciler struct that has a Reconcile function
type reconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
	recorder   record.EventRecorder
}

// Implements a Kubernetes API for a specific Resource by Creating, Updating or Deleting Kubernetes objects,
// or by making changes to systems external to the cluster
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("servicedeployment", req.NamespacedName)
	log.Info("Reconciling servicedeployment...")

	// Create ServiceDeployment if not exists
	depClient := r.kubeClient.AppsV1().Deployments(req.Namespace)
	svcClient := r.kubeClient.CoreV1().Services(req.Namespace)
	svcName := fmt.Sprintf("%s-svc", req.Name)

	// 1) Load the primary CR
	var sd apiv1.ServiceDeployment
	err := r.Get(ctx, req.NamespacedName, &sd)
	// If there is an error, it means ServiceDeployment got deleted, therefore delete underlying resources
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Manual failsafe deletion of resources.
			// We already provide OwnerReferences below via controllerutil.SetControllerReference for Service and Deployment so,
			// Kubernetes Garbage Collector is supposed to delete these resources automatically.
			// This is a failsafe, for any edge cases that may arise due to any unforseen circumstances.
			err = svcClient.Delete(ctx, svcName, metav1.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't delete service: %s", err)
			}
			err = depClient.Delete(ctx, req.Name, metav1.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't delete deployment: %s", err)
			}
			return ctrl.Result{}, nil
		}
	}

	// 2) Ensure child Deployment
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      sd.Name,
		Namespace: sd.Namespace,
	}}
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		// Set Labels
		if dep.Labels == nil {
			dep.Labels = make(map[string]string)
		}
		dep.Labels["app"] = sd.Name

		// Set Replicas
		replicas := sd.Spec.Replicas
		dep.Spec.Replicas = &replicas

		// Set Selector
		dep.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": sd.Name},
		}

		// Set Template
		if dep.Spec.Template.ObjectMeta.Name == "" {
			dep.Spec.Template.ObjectMeta.Name = sd.Name
		}
		dep.Spec.Template.ObjectMeta.Labels = map[string]string{"app": sd.Name}
		dep.Spec.Template.Spec.Containers = sd.Spec.Containers

		// [Very Important]: Set controller `ownerReferences` for GC + Owns()
		// It sets the OwnerReference on the Deployment object, pointing to ServiceDeployment (CR).
		return controllerutil.SetControllerReference(&sd, dep, r.scheme)
	})
	if err != nil {
		// Optimistic-concurrency conflict:
		// Sometimes the Deployment’s metadata.resourceVersion changed between your read and your write
		// (e.g., defaults/managedFields/status updates or two reconciles racing), so the API rejected your update once;
		// the next attempt used the latest object and succeeded—hence the subsequent “Updated” events.
		//
		// This is a benign race condition and we can treat it as transient and requeu without spamming as a Warning Event.
		if k8serrors.IsConflict(err) {
			log.Info("deployment update conflicted, will retry")
			return ctrl.Result{Requeue: true}, nil
		}
		// Record a Warning tied to the CR
		r.recorder.Eventf(&sd, corev1.EventTypeWarning, "ApplyDeploymentFailed", "Failed to apply Deployment %q: %v", dep.Name, err)
		return ctrl.Result{}, fmt.Errorf("apply deployment: %w", err)
	}
	// Record Normal event for created/updated/noop
	switch result {
	case controllerutil.OperationResultCreated:
		r.recorder.Eventf(&sd, corev1.EventTypeNormal, "DeploymentCreated", "Created Deployment %q", dep.Name)
	case controllerutil.OperationResultUpdated:
		r.recorder.Eventf(&sd, corev1.EventTypeNormal, "DeploymentUpdated", "Updated Deployment %q", dep.Name)
	default:
		// No change; keep noise low—skip or log a verbose message
	}

	// 3) Ensure child Service
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name:      svcName,
		Namespace: sd.Namespace,
	}}
	result, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		// Keep ClusterIP if it already exists (immutable for ClusterIP services)
		// CreateOrUpdate will load existing svc into object for us.
		svc.Labels = map[string]string{"app": sd.Name}

		svc.Spec.Selector = map[string]string{"app": sd.Name}
		svc.Spec.Type = sd.Spec.Service.Type

		// Preserve clusterIP on updates when Type == ClusterIP
		if svc.Spec.ClusterIP != "" {
			// leave as is
		}
		svc.Spec.Ports = sd.Spec.Service.Ports

		// [Very Important]: Set controller `ownerReferences` for GC + Owns()
		// It sets the OwnerReference on the Service object, pointing to ServiceDeployment (CR).
		return controllerutil.SetControllerReference(&sd, svc, r.scheme)
	})
	if err != nil {
		// Record a Warning tied to the CR
		r.recorder.Eventf(&sd, corev1.EventTypeWarning, "ApplyServiceFailed", "Failed to apply Service %q: %v", svc.Name, err)
		return ctrl.Result{}, fmt.Errorf("apply service: %w", err)
	}
	// Record Normal event for created/updated/noop
	switch result {
	case controllerutil.OperationResultCreated:
		r.recorder.Eventf(&sd, corev1.EventTypeNormal, "ServiceCreated", "Created Service %q", svc.Name)
	case controllerutil.OperationResultUpdated:
		r.recorder.Eventf(&sd, corev1.EventTypeNormal, "ServiceUpdated", "Updated Service %q", svc.Name)
	default:
		// No change; keep noise low—skip or log a verbose message
	}

	// 4) Sync Status
	err = r.SyncStatus(ctx, &sd, dep, svc)
	if err != nil {
		log.Error(err, "failed to sync status with deployment or service")
	}

	log.Info("reconciled", "deployment", dep.Name, "service", svc.Name)
	return ctrl.Result{}, nil
}

func fillFromDeploymentStatus(dst *apiv1.ServiceDeploymentStatus, sd *apiv1.ServiceDeployment, dep *appsv1.Deployment) {
	dst.DesiredReplicas = sd.Spec.Replicas
	dst.ReadyReplicas = dep.Status.ReadyReplicas
	dst.UpdatedReplicas = dep.Status.UpdatedReplicas
	dst.AvailableReplicas = dep.Status.AvailableReplicas
	dst.Ready = fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, sd.Spec.Replicas)

	// Build the selector string for the scale subresource
	if dep.Spec.Selector != nil {
		if sel, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector); err == nil {
			dst.Selector = sel.String() // e.g. "app=nginx"
		}
	}
}

func fillFromServiceStatus(dst *apiv1.ServiceDeploymentStatus, svc *corev1.Service) {
	dst.ServiceType = string(svc.Spec.Type)
	dst.ClusterIP = svc.Spec.ClusterIP
	dst.ExternalIPs = strings.Join(svc.Spec.ExternalIPs, ",")
	var parts []string
	for _, p := range svc.Spec.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = corev1.ProtocolTCP
		}
		parts = append(parts, fmt.Sprintf("%d/%s", p.Port, proto))
	}
	dst.Ports = strings.Join(parts, ",")
}

func (r *reconciler) SyncStatus(ctx context.Context, sd *apiv1.ServiceDeployment, dep *appsv1.Deployment, svc *corev1.Service) error {
	desired := sd.Status
	fillFromDeploymentStatus(&desired, sd, dep)
	fillFromServiceStatus(&desired, svc)

	// If there are no changes, do nothing
	if equality.Semantic.DeepEqual(sd.Status, desired) {
		return nil
	}
	orig := sd.DeepCopy()
	sd.Status = desired
	return r.Status().Patch(ctx, sd, client.MergeFrom(orig))
}
