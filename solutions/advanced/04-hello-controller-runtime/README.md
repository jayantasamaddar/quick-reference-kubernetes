# About

A Kubernetes Controller is a control loop that watches the shared state of the Kubernetes cluster through the API server and makes changes to move the current state towards the desired state. Essentially, controllers are responsible for ensuring that the actual state of your cluster matches the desired state you define in your resource manifests (e.g., YAML files).

Operators in Kubernetes, particularly those written in Go using frameworks like the Operator SDK or Kubebuilder, are essentially specialized controllers designed to manage Custom Resource Definitions (CRDs). An operator can contain one or more controllers. Each handles a particular CRD. But multiple controllers can be bundled into a single operator.

Using the `sigs.k8s.io/controller-runtime` library in Golang, one can create Controllers and thus Operators. This library is used in popular abstractions like:

- Operator SDK
- Kubebuilder

In this exercise, we will focus on using the base library: `sigs.k8s.io/controller-runtime` to complete controller/operator related tasks.

Usually the workflow is:

1. Write a `CRD`
2. Write the custom controller code that uses the `sigs.k8s.io/controller-runtime` as a Golang program.
3. Containerize it (Build image).
4. Deploy as a `Deployment` in the Kubernetes cluster.
5. Test changes to intended resources being handled by the controller correctly.

---

# Exercise

We will take the same problem we did in [hello-shell-operator](../01-hello-shell-operator/README.md) but solve it using the `sigs.k8s.io/controller-runtime` library.

Create a `CustomResourceDefinition` that creates a CustomResource named `ServiceDeployment` that is namespace scoped.
`ServiceDeployment` allows the creation, updation and deletion of a `Deployment` and it's associated `Service` using a single `ServiceDeployment` manifest. Use the `sigs.k8s.io/controller-runtime` Go library, to orchestrate the CRUD operations of the CRDs.

```yaml
apiVersion: k8s.example.com/v1
kind: ServiceDeployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  replicas: 3
  containers:
    - name: nginx
      image: nginx

  service:
    name: nginx-svc # Optional. If not provided, defaults to "{ServiceDeployment.metadata.name}-svc"
    type: ClusterIP # ClusterIP, NodePort or LoadBalancer
    ports:
      - port: 80
        targetPort: 80
        # nodePort: 30080 # If NodePort or LoadBalancer
```

**Requirements**:

1. The above manifest file should create when using `kubectl create` or `kubectl apply`:

   - A ServiceDeployment resource
   - A Deployment with 2 replicas
   - A Service exposing the `nginx` Deployment.

2. We should be able to view the `ServiceDeployment` resource(s) using both `kubectl get servicedeployment/service-deployment-name` (single resource) and `kubectl get servicedeployments` (all service deployments).
3. The above manifest when modified would modify the `ServiceDeployment` Kubernetes CustomResource and underlying resources accordingly when applied using `kubectl apply`.
4. The above manifest would delete `ServiceDeployment` CustomResource and the underlying resources and when `kubectl delete`.

---

# Solution

**The controller essentially must do the following**:

1. Needs to run forever.
2. Watch and detect changes in our Custom Resource Definition (`ServiceDeployment`).

**Steps to implement the above**:

1. A `Scheme` to correctly map Go Types -> GVK and vice versa.
2. A `Manager` (process supervisor) for any controllers we create tied to scheme.
3. Register controller(s) with the Manager using `NewControllerManagedBy` (from `sigs.k8s.io/controller-runtime`) and passing the `For` and `Complete`.
4. Every controller has a function named `Reconciler` that implements `TypedReconciler` interface from `sigs.k8s.io/controller-runtime/pkg/reconcile` package, that watches for changes on a Kubernetes resource and does the operation (operator pattern, hence: Operator). This is passed in the `Complete` function as part of the builder pattern for the `NewControllerManagedBy`.
