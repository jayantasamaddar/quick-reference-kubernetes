package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"sigs.k8s.io/yaml"
)

const (
	NAMESPACE  string = "example-nginx"
	APP_NAME   string = "nginx"
	IMAGE_NAME        = "nginx"
)

func main() {
	ctx := context.Background()

	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	// Optional: QPS/Burst tuning for heavy operations
	config.QPS = 20
	config.Burst = 40

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	/************************************************************************************************************/
	// Task 1: Create Namespace - `example-nginx` and print out the name
	/************************************************************************************************************/
	_, err = clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: NAMESPACE},
	}, metav1.CreateOptions{
		DryRun: []string{"All"},
	})

	var ns *corev1.Namespace
	if err == nil {
		ns, err = clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: NAMESPACE},
		}, metav1.CreateOptions{})
		if err != nil {
			log.Fatalln("error creating namespace:", err)
		}
		fmt.Println("\nNamespace created:", ns.Name)
	} else {
		fmt.Println("\nNamespace already exists:", NAMESPACE)
	}

	/************************************************************************************************************/
	// Task 2: Create a nginx `Deployment` named `nginx` with 3 replicas in `example-nginx` namespace.
	/************************************************************************************************************/
	replicas := int32(3)
	dep, err := clientset.AppsV1().Deployments(NAMESPACE).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   APP_NAME,
			Labels: map[string]string{"app": APP_NAME},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": APP_NAME,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   APP_NAME,
					Labels: map[string]string{"app": APP_NAME},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            APP_NAME,
							Image:           IMAGE_NAME,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})

	if err != nil {
		log.Fatalln("error creating deployment:", err)
	}
	fmt.Println("\nDeployment created:", dep.Name)

	/************************************************************************************************************/
	// Task 3: Expose it via a `NodePort` `Service` named `nginx-svc` on port 30007.
	/************************************************************************************************************/
	svc, err := clientset.CoreV1().Services(NAMESPACE).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-svc", APP_NAME),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": APP_NAME},
			Type:     corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt32(80),
					NodePort:   int32(30007),
				},
			},
		},
	}, metav1.CreateOptions{})

	if err != nil {
		log.Fatalln("error creating service:", err)
	}
	fmt.Println("\nService created:", svc.Name)

	/************************************************************************************************************/
	// Task 4: List Pods in Namespace
	/************************************************************************************************************/
	pods, err := clientset.CoreV1().Pods(NAMESPACE).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalln("error listing pods:", err)
	}
	time.Sleep(500 * time.Millisecond) // Artificial Delay
	fmt.Printf("\nListing pods in namespace: %s...\n", NAMESPACE)
	for _, pod := range pods.Items {
		fmt.Println("- ", pod.Name)
	}

	/************************************************************************************************************/
	// Task 5: Update the Deployment's replicas to 2
	/************************************************************************************************************/
	originalReplicas := *dep.Spec.Replicas
	replicas = int32(2)
	scale, err := clientset.AppsV1().Deployments(NAMESPACE).UpdateScale(ctx, dep.Name, &autoscalingv1.Scale{
		ObjectMeta: metav1.ObjectMeta{Name: dep.Name},
		Spec:       autoscalingv1.ScaleSpec{Replicas: replicas},
	}, metav1.UpdateOptions{})

	if err != nil {
		log.Fatalln("error scaling deployment:", err)
	}
	fmt.Printf("\nDeployment Scaled from %d replicas to %d!\n", originalReplicas, scale.Spec.Replicas)

	/************************************************************************************************************/
	// Task 6: Read and Extract the YAML file for `nginx`
	/************************************************************************************************************/
	dep, err = clientset.AppsV1().Deployments(NAMESPACE).Get(ctx, dep.Name, metav1.GetOptions{})
	if err != nil {
		log.Fatalln("error fetching deployment:", err)
	}

	deploymentYAML, err := yaml.Marshal(dep)
	if err != nil {
		log.Fatalln("error marshalling deployment to YAML:", err)
	}

	if err = os.WriteFile(fmt.Sprintf("%s.yaml", dep.Name), deploymentYAML, 0644); err != nil {
		log.Fatalf("error writing manifest YAML to %s.yaml\n", dep.Name)
	}
	fmt.Printf("\nDeployment manifest YAML created at: '%s.yaml'!\n", dep.Name)

	/************************************************************************************************************/
	// Task 7: Delete the Service, Deployment and lastly the Namespace.
	/************************************************************************************************************/
	// Delete Service
	if err = clientset.CoreV1().Services(NAMESPACE).Delete(ctx, svc.Name, metav1.DeleteOptions{}); err != nil {
		log.Fatalln("error deleting service:", err)
	}
	fmt.Printf("\nService '%s' deleted...\n", svc.Name)

	// Delete Deployment
	if err = clientset.AppsV1().Deployments(NAMESPACE).Delete(ctx, dep.Name, metav1.DeleteOptions{}); err != nil {
		log.Fatalln("error deleting deployment:", err)
	}
	fmt.Printf("\nDeployment '%s' deleted...\n", dep.Name)

	// Delete Namespace
	time.Sleep(1 * time.Second) // Artificial delay
	if err = clientset.CoreV1().Namespaces().Delete(ctx, NAMESPACE, metav1.DeleteOptions{}); err != nil {
		log.Fatalln("error deleting namespace:", err)
	}
	fmt.Printf("\nNamespace '%s' deleted...\n", NAMESPACE)

	// Remove File
	// if err = os.Remove(fmt.Sprintf("%s.yaml", dep.Name)); err != nil {
	// 	log.Fatalln("error deleting file:", err)
	// }
	// fmt.Printf("\nFile '%s.yaml' deleted...\n", dep.Name)
}
