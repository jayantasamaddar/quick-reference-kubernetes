NAMESPACE=servicedeployments-controller
CLUSTER_ROLE=servicedeployments-controller
SERVICE_ACCOUNT=servicedeployments-controller
IMAGE=k8s.example.com/servicedeployments-controller:v1.0.0

#############################################################################
# RBAC
#############################################################################
.PHONY: rbac
rbac:
	@echo "Creating Role-based access controls...\n"
	kubectl create ns $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	kubectl create serviceaccount $(SERVICE_ACCOUNT) --namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	kubectl create clusterrole $(CLUSTER_ROLE) \
		--verb=get,list,watch,create,update,patch,delete \
		--resource=servicedeployments.k8s.example.com \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl create clusterrolebinding $(CLUSTER_ROLE) --clusterrole=$(CLUSTER_ROLE) --serviceaccount=$(NAMESPACE):$(SERVICE_ACCOUNT) \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl create role $(CLUSTER_ROLE) \
		--verb=get,list,watch,create,update,patch,delete \
		--resource=deployments.apps,services,pods \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl create rolebinding $(CLUSTER_ROLE) --role=$(CLUSTER_ROLE) --serviceaccount=$(NAMESPACE):$(SERVICE_ACCOUNT) \
		--dry-run=client -o yaml | kubectl apply -f -

#############################################################################
# Build, Deploy, Undeploy Shell Operator Image
#############################################################################
.PHONY: build deploy undeploy
build:
	docker build -t $(IMAGE) .

unbuild:
	docker rmi $(IMAGE)

deploy:
	kubectl -n $(NAMESPACE) apply -f ./k8s/controller.yaml

undeploy:
	kubectl -n ${NAMESPACE} delete -f ./k8s/controller.yaml

#############################################################################
# CRDS
#############################################################################
crds:
	kubectl apply -f ./k8s/crds.yaml

#############################################################################
# SERVICE DEPLOYMENT
#############################################################################
sd:
	kubectl apply -f ./k8s/servicedeployment.yaml

#############################################################################
# DEBUG
#############################################################################
debug:
	docker run --rm -it --entrypoint /bin/sh $(IMAGE)

debug-image-os:
	docker image inspect $(IMAGE) --format '{{.Os}}/{{.Architecture}}'

#############################################################################
# Clean Up
#############################################################################
.PHONY: clean-rbac clean-crds clean-sd clean

clean-sd:
	kubectl delete -f ./k8s/servicedeployment.yaml

clean-crds:
	@echo "Cleaning up CRDs...\n"
	kubectl delete -f ./k8s/crds.yaml

clean-rbac:
	@echo "Cleaning up RBAC...\n"
	kubectl delete rolebinding $(CLUSTERROLE)
	kubectl delete role $(CLUSTERROLE)
	kubectl delete clusterrolebinding $(CLUSTER_ROLE)
	kubectl delete clusterrole $(CLUSTER_ROLE)
	kubectl delete serviceaccount $(SERVICE_ACCOUNT) --namespace $(NAMESPACE)
	kubectl delete ns $(NAMESPACE)

clean-all:
	make -s clean-sd && make -s undeploy && make -s clean-rbac && make -s clean-crds
	make -s unbuild