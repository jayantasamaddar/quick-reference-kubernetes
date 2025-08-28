# About

Concepts that will be used in this exercise:

- **[AdditionalColumns](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#additional-printer-columns)**
- **[Predicates](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/predicate)**

I'm sure you have noticed these print messages when doing a `LIST` or `GET` request to the Kubernetes API using `kubectl`.

```
$ kubectl get all
NAME                         READY   STATUS    RESTARTS   AGE
pod/nginx-7c766b6b59-2lfb7   1/1     Running   0          54m
pod/nginx-7c766b6b59-8fbdm   1/1     Running   0          54m
pod/nginx-7c766b6b59-zf2vt   1/1     Running   0          54m

NAME                 TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)   AGE
service/kubernetes   ClusterIP   10.43.0.1     <none>        443/TCP   405d
service/nginx-svc    ClusterIP   10.43.42.71   <none>        80/TCP    54m

NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/nginx   3/3     3            3           54m

NAME                               DESIRED   CURRENT   READY   AGE
replicaset.apps/nginx-7c766b6b59   3         3         3       54m
```

What happens when we do a get for our current `ServiceDeployment`

```
$ kubectl get servicedeployments
NAME    AGE
nginx   56m
```

We do not get any other printed columns except `NAME` and `AGE`. These are set by Kubernetes by default. But the others have to be added by us.

This is done by [`additionalPrinterColumns`](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#additional-printer-columns).

---

# Exercise

1. **Additional Columns in Table Output**

   Copy any of your solutions between `04` and `06` and use them as the base. Modify the CRD and controller, such that:

   Ensure we can get a nice printed output when we do a `kubectl get servicedeployments` that displays both `Deployments` and `Service` info:

   ```
   $ kubectl get servicedeployments
   NAME    READY   SERVICE-TYPE   CLUSTER-IP      PORT(S)   AGE
   nginx   3/3     ClusterIP      10.43.173.140   80/TCP    2m

   $ kubectl get servicedeployments -o wide
   NAME    READY   UP-TO-DATE   AVAILABLE   SERVICE-TYPE   CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
   nginx   3/3     3            3           ClusterIP      10.43.173.140                 80/TCP    2m
   ```

2. **Predicates**

   Any updates to `ServiceDeployments` would trigger a Reconcile by default. This includes any Status updates. We must ensure a change such that:

   - Status-only updates of `ServiceDeployment` is ignored (shouldn't trigger Reconcile) as we will update it constantly whenever Deployment or Service changes, so this will trigger extra Reconcile runs unless we do something about it.

   > **Concept**: Predicates allow controllers to selectively respond to events, preventing unnecessary reconciliations or processing of irrelevant changes. This reduces the load on the API server and improves controller efficiency. The `controller-runtime` predicates implements the `Predicate` interface (`sigs.k8s.io/controller-runtime/pkg/predicate`). One can also create custom predicates by implementing the `Predicate` interface.

---

# Solution

1. Since we’re managing the CRD via YAML (not Kubebuilder markers), we will get pretty kubectl get output by:

   - Adding a `Status` subresource and `additionalPrinterColumns` to the CRD.
   - Adding a `Status` struct to our Go type.
   - Syncing `Status` updates of child resources (`Deployment` and `Service`) with `ServiceDeployment` in our reconciler.

2. Write a custom predicate

---

# References

- [Additional Printer Columns](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#additional-printer-columns)
