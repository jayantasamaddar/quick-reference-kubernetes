# Exercise

Create a `CustomResourceDefinition` that creates a CustomResource named `ServiceDeployment` that is namespace scoped.
`ServiceDeployment` allows the creation, updation and deletion of a `Deployment` and it's associated `Service` using a single `ServiceDeployment` manifest. Use the Shell Operator from [flant](https://github.com/flant/shell-operator), to orchestrate the CRUD operations of the CRDs.

```yaml
apiVersion: k8s.example.com/v1
kind: ServiceDeployment
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  replicas: 2
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

# References

- [Flant/Shell Operator](https://flant.github.io/shell-operator/)
