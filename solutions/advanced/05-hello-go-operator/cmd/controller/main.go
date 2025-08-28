package main

// 1. Needs to run forever.
// 2. Watch and detect changes in our Custom Resource Definition (ServiceDeployment).
// 3. A function `Reconciler` that does the operation (hence: Operator) for a change needs to be executed, everytime there is a change.
// 4. We do this with the help of the sigs.k8s.io/controller-runtime package.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	apiv1 "github.com/jayantasamaddar/quick-reference-kubernetes/solutions/hello-go-operator/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(apiv1.AddToScheme(scheme))

	// For Self-healing child resources, add them to the scheme
	utilruntime.Must(appsv1.AddToScheme(scheme)) // For Deployment
	utilruntime.Must(corev1.AddToScheme(scheme)) // For Service
}

// A reconciler struct that has a Reconcile function
type reconciler struct {
	client.Client
	scheme     *runtime.Scheme
	kubeClient *kubernetes.Clientset
}

// Implements a Kubernetes API for a specific Resource by Creating, Updating or Deleting Kubernetes objects,
// or by making changes to systems external to the cluster
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("servicedeployment", req.NamespacedName)
	log.Info("Reconciling servicedeployment...")

	// Create ServiceDeployment if not exists
	depClient := r.kubeClient.AppsV1().Deployments(req.Namespace)
	svcClient := r.kubeClient.CoreV1().Services(req.Namespace)

	// 1) Load the primary CR
	var sd apiv1.ServiceDeployment
	err := r.Get(ctx, req.NamespacedName, &sd)
	// If there is an error, it means ServiceDeployment got deleted, therefore delete underlying resources
	if err != nil {
		if k8serrors.IsNotFound(err) {
			err = depClient.Delete(ctx, req.Name, metav1.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't delete deployment: %s", err)
			}
			err = svcClient.Delete(ctx, fmt.Sprintf("%s-svc", req.Name), metav1.DeleteOptions{})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("couldn't delete service: %s", err)
			}
			return ctrl.Result{}, nil
		}
	}

	// 2) Ensure child Deployment
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      sd.Name,
		Namespace: sd.Namespace,
	}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
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
		return ctrl.Result{}, fmt.Errorf("apply deployment: %w", err)
	}

	// 3) Ensure child Service
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-svc", sd.Name),
		Namespace: sd.Namespace,
	}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
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
		return ctrl.Result{}, fmt.Errorf("apply service: %w", err)
	}

	log.Info("reconciled", "deployment", dep.Name, "service", svc.Name)
	return ctrl.Result{}, nil
}

func main() {
	var (
		config *rest.Config
		err    error
	)

	// Set kube config correctly
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfig); errors.Is(err, os.ErrNotExist) {
		// In a cluster use the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	} else {
		// Outside the cluster using kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	// Kubernetes client set
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	ctrl.SetLogger(zap.New()) // Set new logger

	// Manager can create controller(s) and `Start` running them until cancelled.
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create a new controller using builder pattern
	// ---------------------------------------------
	// - `For`: This struct is what the reconciler will reconcile the object into
	// - `Owns`: controller-runtime sets up a watch on this resource (Deployment and Service in our case).
	// 			 But it doesn’t blindly enqueue on all `Service` (for example) changes (that would be chaos).
	//			 Instead, it checks the `ownerReferences` on the `Service` (previously set by `controllerutil.SetControllerReference`).
	//			 Without `SetControllerReference`, .Owns(&corev1.Service{}) would never trigger a reconcile of the CR. So this is important.
	//				- If it finds an owner of kind ServiceDeployment with controller=true, it enqueues that owner’s {namespace, name}.
	// 			 Thus, that is how “child changed → reconcile parent” works.
	// - `Complete`: Takes the reconciler and builds the controller.
	//
	// Controllers can invoke the Reconcile function once the are running and receive events
	err = ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.ServiceDeployment{}).
		Owns(&appsv1.Deployment{}). // controller-runtime sets up a watch on Deployments
		Owns(&corev1.Service{}).    // controller-runtime sets up a watch on Services.
		Complete(&reconciler{
			Client:     mgr.GetClient(),
			scheme:     mgr.GetScheme(),
			kubeClient: clientset,
		})
	if err != nil {
		setupLog.Error(err, "Unable to create operator!")
		os.Exit(1)
	}

	// Start all controllers registered with the manager
	setupLog.Info("Starting manager...")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Error running manager!")
		os.Exit(1)
	}

}
