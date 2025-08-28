NAMESPACE=monitor-servicedeployments
CLUSTER_ROLE=monitor-servicedeployments
SERVICE_ACCOUNT="$(NAMESPACE)-acc"

# Initialize Scripts
.PHONY: init
init:
	@echo "Modifying permissions for script files...\n"
	chmod +x ./hooks/*

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
		--resource=servicedeployments.k8s.example.com,deployments.apps,services,pods \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl create clusterrolebinding $(CLUSTER_ROLE) --clusterrole=$(CLUSTER_ROLE) --serviceaccount=$(NAMESPACE):$(SERVICE_ACCOUNT)

#############################################################################
# Build, Deploy, Undeploy Shell Operator Image
#############################################################################
.PHONY: build deploy undeploy
build:
	make -s init
	docker build -t "k8s.example.com/shell-operator:v1.0.0" .

unbuild:
	docker rmi "k8s.example.com/shell-operator:v1.0.0"

deploy:
	kubectl -n $(NAMESPACE) apply -f ./k8s/shell-operator.yaml

undeploy:
	kubectl -n ${NAMESPACE} delete -f ./k8s/shell-operator.yaml

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
	kubectl delete clusterrolebinding $(CLUSTER_ROLE)
	kubectl delete clusterrole $(CLUSTER_ROLE)
	kubectl delete serviceaccount $(SERVICE_ACCOUNT) --namespace $(NAMESPACE)
	kubectl delete ns $(NAMESPACE)

clean-all:
	make -s clean-sd && make -s undeploy && make -s clean-rbac && make -s clean-crds
	make -s unbuild