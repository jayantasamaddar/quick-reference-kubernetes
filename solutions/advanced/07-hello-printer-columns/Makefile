NAMESPACE=servicedeployments-operator
CLUSTER_ROLE=servicedeployments-operator
SERVICE_ACCOUNT=servicedeployments-operator
IMAGE=k8s.example.com/servicedeployments-operator:v1.2.0

#############################################################################
# CRDS
#############################################################################
.PHONY: crds
crds:
	kubectl apply -f ./k8s/crds.yaml

#############################################################################
# RBAC
#############################################################################
.PHONY: rbac
rbac:
	@echo "Creating RBAC...\n"
	kubectl create ns $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f ./k8s/rbac.yaml

#############################################################################
# Build, Deploy, Undeploy Shell Operator Image
#############################################################################
.PHONY: build unbuild deploy undeploy
build:
	docker build -t $(IMAGE) .

unbuild:
	docker rmi $(IMAGE)

deploy:
	kubectl -n $(NAMESPACE) apply -f ./k8s/operator.yaml

undeploy:
	kubectl -n ${NAMESPACE} delete -f ./k8s/operator.yaml

#############################################################################
# SERVICE DEPLOYMENT
#############################################################################
.PHONY: sd
sd:
	kubectl apply -f ./k8s/servicedeployment.yaml

#############################################################################
# DEBUG
#############################################################################
.PHONY: debug debug-image-os
debug:
	docker run --rm -it --entrypoint /bin/sh $(IMAGE)

debug-image-os:
	docker image inspect $(IMAGE) --format '{{.Os}}/{{.Architecture}}'

#############################################################################
# Clean Up
#############################################################################
.PHONY: clean-rbac clean-crds clean-sd clean-all

clean-sd:
	kubectl delete -f ./k8s/servicedeployment.yaml

clean-crds:
	@echo "Cleaning up CRDs...\n"
	kubectl delete -f ./k8s/crds.yaml

clean-rbac:
	@echo "Cleaning up RBAC...\n"
	kubectl delete -f ./k8s/rbac.yaml
	kubectl delete ns $(NAMESPACE)

clean-all:
	make -s clean-sd && make -s undeploy && make -s clean-rbac && make -s clean-crds
	make -s unbuild