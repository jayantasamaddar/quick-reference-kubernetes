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

	apiv1 "github.com/jayantasamaddar/quick-reference-kubernetes/solutions/hello-controller-runtime/api/v1"

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
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(apiv1.AddToScheme(scheme))
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

	// 2) So ServiceDeployment object was found, let's now check if Deployment exists.
	// (i) If doesn't exist, we have to create it.
	dep, err := depClient.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			_, err = depClient.Create(ctx, newDeploymentFromServiceDeployment(req.Name, sd.Spec.Containers, sd.Spec.Replicas), metav1.CreateOptions{})
			if err != nil && !k8serrors.IsAlreadyExists(err) {
				return ctrl.Result{}, fmt.Errorf("couldn't create deployment: %s", err)
			}
		}

		// (ii) Create the service
		_, err := svcClient.Create(ctx, newServiceFromServiceDeployment(req.Name, sd.Spec.Service.Type, sd.Spec.Service.Ports), metav1.CreateOptions{})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			return ctrl.Result{}, fmt.Errorf("couldn't create service: %s", err)
		}
	}

	// (3) Deployment is found, let's see if we need to update it
	if *dep.Spec.Replicas != sd.Spec.Replicas {
		dep.Spec.Replicas = &(sd.Spec.Replicas)
		_, err := depClient.Update(ctx, dep, metav1.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't update deployment: %s", err)
		}
		log.Info("servicedeployment with name " + sd.Name + " updated")
		return ctrl.Result{}, nil
	}

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

	// Create a new controller using the builder pattern
	// - `For` is what the reconciler will reconcile the object into
	// - `Complete` takes the reconciler and builds the controller
	//
	// Controllers can invoke the Reconcile function once the are running and receive events
	err = ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.ServiceDeployment{}).
		Complete(&reconciler{
			Client:     mgr.GetClient(),
			scheme:     mgr.GetScheme(),
			kubeClient: clientset,
		})
	if err != nil {
		setupLog.Error(err, "Unable to create controller!")
		os.Exit(1)
	}

	// Start all controllers registered with the manager
	setupLog.Info("Starting manager...")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Error running manager!")
		os.Exit(1)
	}

}

func newDeploymentFromServiceDeployment(name string, containers []corev1.Container, replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}
}

func newServiceFromServiceDeployment(name string, typ corev1.ServiceType, ports []corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-svc", name),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": name},
			Type:     typ,
			Ports:    ports,
		},
	}
}
