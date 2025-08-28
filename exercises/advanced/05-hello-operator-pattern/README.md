# About

Operators in Kubernetes, particularly those written in Go using frameworks like the Operator SDK or Kubebuilder, are essentially specialized controllers designed to manage Custom Resource Definitions (CRDs) that implement Operator design pattern. An operator can contain one or more controllers. Each handles a particular CRD. But multiple controllers can be bundled into a single operator. Every Operator is at least one or more controllers but not every controller is an Operator.

Using the `sigs.k8s.io/controller-runtime` library in Golang, one can create Controllers and thus Operators. This library is used in popular abstractions like:

- Operator SDK
- Kubebuilder

---

# Exercise

In the previous exercise, we tried to create a basic controller using `sigs.k8s.io/controller-runtime` for `ServiceDeployment`. However this controller does not truly implement Operator pattern and is NOT a full fledged Kubernetes Operator yet.

Why?

Notice how, if we delete 1 pod out of the 3 replicas, another springs right back in place. This self-healing mechanism is possible because the Deployment has `ownerReferences` to the underlying pods. This is done by in-built Kubernetes controllers. Actually what happens is the Deployment Controller manages ReplicaSets and ReplicaSet Controller manages underlying Pods.

We would want a similar situation here. What if we delete the related `Deployment` or `Service` (underlying resources) of `ServiceDeployment`. As things stand, nothing happens. We want to enable a self-healing mechanism here, such that, if either of `Deployment` or `Service` is deleted, they are created back. This leverages the true power of Kubernetes.

> Note: We don't have to worry about Pods, because ensuring the underlying Deployment is up, will in turn ensure the relevant Controllers for Deployment and thus ReplicaSets run, ensuring Pod deletion should self-heal automatically.

**Requirements**:

Copy your previous solution for `04-hello-controller-runtime`. Make any code modifications as necessary such that the following work as expected.

1. Creating or Modifying a `ServiceDeployment` creates or modifies relevant `Deployment` and `Service` (unchanged behaviour as earlier).
2. Additionally, deletion or manual modification of `Service` or `Deployment` (drift) is self-healed using the `ServiceDeployment` manifest as the single-source of truth.

   - Manual modification should revert to the configuration present in the `ServiceDeployment` manifest.
   - Deletion of Deployment or Service should re-create them.

---
