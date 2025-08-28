# About

The [Go Client SDK for Kubernetes](k8s.io/client-go) and its supporting libraries, help us use Golang to do CRUD operations using the Kubernetes API - i.e. Create, Update, Read, List, Watch, Delete resources in a Kubernetes cluster.

---

# Exercise

Using the SDK, do the following:

1. Create a `Namespace`: `example-nginx` in such a way that it only creates the namespace if it doesn't exist.
2. Create a nginx `Deployment` named `nginx` with 3 replicas in `example-nginx` namespace.
3. Expose it via a `NodePort` `Service` named `nginx-svc`.
4. List the names of the `Pods` that got created.
5. Scale down the `Deployment` replicas to 2.
6. Read and Extract the YAML file for `nginx-deployment`.
7. Delete the `Service`, `Deployment` and lastly the `Namespace`.
