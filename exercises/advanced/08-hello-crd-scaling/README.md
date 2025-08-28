# About

To scale a default Deployment resource, we can do either of the following:

- Manually modify the manifest YAML and change `spec.replicas` and do a `kubectl apply -f <modified-manifest.yaml>`.
- Do a `kubectl scale deployment/nginx --replicas={number}` command.
- Use a [Horizontal Pod Autoscaler (HPA)](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) to scale up and down a deployment based on memory and cpu usage.
- Other third party tools.

Write now to scale the underlying Deployments in `ServiceDeployments`, we can only change the manifest file's `spec.replicas` and do a `kubectl apply`.

Neither of the following work by default:

- `kubectl scale servicedeployment/nginx --replicas=5` command.
- Horizontal Pod Autoscaling.

---

## Subresources

If you want `kubectl scale` and HPA to work against your CR (instead of directly against the child Deployment), the **`scale` subresource** on your CRD must be implemented. Kubernetes natively supports this for CRDs.

**How the `scale` subresource works**

You tell Kubernetes:

- where to read/write the desired replicas on your CR (`specReplicasPath`), and
- where to read the current replicas (`statusReplicasPath`),
- optionally where to read the label selector (`labelSelectorPath`) so HPA/UI can show matching pods.

Kubernetes then exposes `/scale` for your CR.

- `kubectl scale servicedeployment/nginx --replicas=5` writes to your CR at `specReplicasPath`.
- HPA can target your CR (kind: ServiceDeployment) if `/scale` exists.

> **IMPORTANT**: Apart from adding subresource `scale`, controller changes (if any) must be done to propagate changes from the `spec.replicas` on `ServiceDeployment` to `spec.replicas` on the underlying `Deployment`. Additionally, the Deployment’s live state must be continuously mirrored into the `ServiceDeployment` (CR) status.

Find more in the [Kubernetes Documentation for Scale Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource)

---

# Exercise

Implement the following:

1. Following commands should work:

   - `kubectl scale servicedeployment/nginx --replicas=5`
   - `kubectl scale servicedeployment/nginx --replicas=3`

2. Keep `additionalPrinterColumns` and make sure the status changes from `Deployment` are propagated to `ServiceDeployment` when using `kubectl scale` on `ServiceDeployment`.

3. Write a `HorizontalPodAutoscaler` ([starter](./k8s/hpa.yaml)) and test if it is working by driving load up and down.

---

# References

- [Kubernetes Documentation for Scale Subresource](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource)
