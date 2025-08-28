package main

// 1. Needs to run forever.
// 2. Watch and detect changes in our Custom Resource Definition (ServiceDeployment).
// 3. A function `Reconciler` that does the operation (hence: Operator) for a change needs to be executed, everytime there is a change.
// 4. We do this with the help of the sigs.k8s.io/controller-runtime package.

import (
	"errors"
	"os"
	"path/filepath"

	apiv1 "github.com/jayantasamaddar/quick-reference-kubernetes/solutions/hello-printer-columns/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
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
	// - `For`: This struct is what the reconciler will reconcile the object into. Can use custom predicates (optionally).
	// - `Owns`: controller-runtime sets up a watch on this resource (Deployment and Service in our case).
	// 			 But it doesn’t blindly enqueue on all `Service` (for example) changes (that would be chaos).
	//			 Instead, it checks the `ownerReferences` on the `Service` (previously set by `controllerutil.SetControllerReference`).
	//			 Without `SetControllerReference`, .Owns(&corev1.Service{}) would never trigger a reconcile of the CR. So this is important.
	//				- If it finds an owner of kind ServiceDeployment with controller=true, it enqueues that owner’s {namespace, name}.
	// 			 Thus, that is how “child changed → reconcile parent” works.
	//			 Can use custom predicates (optionally).
	// - `Complete`: Takes the reconciler and builds the controller.
	//
	// Controllers can invoke the Reconcile function once the are running and receive events
	err = ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.ServiceDeployment{}, builder.WithPredicates(serviceDeploymentPredicate)).
		Owns(&appsv1.Deployment{}). // controller-runtime sets up a watch on Deployments
		Owns(&corev1.Service{}).    // controller-runtime sets up a watch on Services.
		Complete(&reconciler{
			Client:     mgr.GetClient(),
			scheme:     mgr.GetScheme(),
			kubeClient: clientset,
			recorder:   mgr.GetEventRecorderFor("servicedeployments-operator"),
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
