#!/usr/bin/env bash

set -euo pipefail # any failure in a pipeline (like jq errors) aborts the hook immediately.

if [[ ${1:-} == "--config" ]] ; then
  cat <<EOF
configVersion: v1
kubernetes:
- name: Monitor ServiceDeployments
  apiVersion: k8s.example.com/v1
  kind: ServiceDeployment
  executeHookOnEvent: ["Added", "Modified", "Deleted"]
EOF
  exit 0
fi

BCTX="$BINDING_CONTEXT_PATH"

NAME=$(jq -r '.[0].object.metadata.name' "$BCTX")
NAMESPACE=$(jq -r '.[0].object.metadata.namespace' "$BCTX")
EVENT_TYPE=$(jq -r '.[0].watchEvent // ""' "$BCTX" | tr '[:lower:]' '[:upper:]')

REPLICAS=$(jq -r '.[0].object.spec.replicas' "$BCTX")
SERVICE_NAME=$(jq -r '.[0].object.spec.service.name // empty' "$BCTX")
SERVICE_TYPE=$(jq -r '.[0].object.spec.service.type // "ClusterIP"' "$BCTX")

# Compact JSON array
CONTAINERS_JSON=$(jq -c '.[0].object.spec.containers' "$BCTX")
PORTS_JSON=$(jq -c '.[0].object.spec.service.ports' "$BCTX")

[[ -z "$SERVICE_NAME" || "$SERVICE_NAME" == "null" ]] && SERVICE_NAME="${NAME}-svc"

echo "🔔 ServiceDeployment '$NAME' in ns '$NAMESPACE' • event=$EVENT_TYPE • svc=$SERVICE_NAME ($SERVICE_TYPE) • replicas=$REPLICAS"

validate_spec() {
  # replicas integer
  if ! [[ "$REPLICAS" =~ ^[0-9]+$ ]]; then
    echo "❌ replicas must be an integer" >&2; exit 1
  fi

  # containers non-empty + name/image present
  if [[ "$CONTAINERS_JSON" == "[]" || "$CONTAINERS_JSON" == "null" ]]; then
    echo "❌ containers must not be empty" >&2; exit 1
  fi
  echo "$CONTAINERS_JSON" | jq -e 'all(.[]; has("name") and has("image"))' >/dev/null || {
    echo "❌ each container must have name and image" >&2; exit 1
  }

  # Must be array of objects
  echo "$PORTS_JSON" | jq -e '
    type=="array" and length>0 and all(.[]; type=="object")
  ' >/dev/null || {
    echo "❌ spec.service.ports must be a non-empty array of objects; got: $(echo "$PORTS_JSON" | jq -c 'type, (type=="array") as $a | if $a then map(type) else . end')" >&2
    exit 1
  }

  ## Ports by service type
  case "$SERVICE_TYPE" in
    "ClusterIP")
      echo "$PORTS_JSON" | jq -e '
        (type=="array") and (length>0) and
        all(.[];
          if ( (type=="object") and has("port") and has("targetPort") ) then
            ((.port|tonumber) >= 1 and (.port|tonumber) <= 65535) and
            (
              ((.targetPort|type)=="number" and (.targetPort >= 1 and .targetPort <= 65535)) or
              ((.targetPort|type)=="string" and (.targetPort|length) > 0)
            ) and
            (has("nodePort") | not)
          else
            false
          end
        )
      ' >/dev/null || { echo "❌ ClusterIP requires port & targetPort (int 1–65535 or non-empty string); nodePort must be absent"; exit 1; }
      ;;
    "NodePort"|"LoadBalancer")
      echo "$PORTS_JSON" | jq -e '
        (type=="array") and (length>0) and
        all(.[];
          if ( (type=="object") and has("port") and has("targetPort") and has("nodePort") ) then
            ((.port|tonumber) >= 1 and (.port|tonumber) <= 65535) and
            (
              ((.targetPort|type)=="number" and (.targetPort >= 1 and .targetPort <= 65535)) or
              ((.targetPort|type)=="string" and (.targetPort|length) > 0)
            ) and
            ((.nodePort|tonumber) >= 30000 and (.nodePort|tonumber) <= 32767)
          else
            false
          end
        )
      ' >/dev/null || { echo "❌ $SERVICE_TYPE requires port, targetPort (int 1–65535 or non-empty string), and nodePort 30000–32767"; exit 1; }
      ;;
    *)
      echo "❌ Unknown service type: $SERVICE_TYPE"; exit 1
      ;;
  esac
}

# Render YAML list items (proper indentation happens later via sed)
containers_yaml() {
  echo "$CONTAINERS_JSON" | jq -r '
    .[] | "- name: \(.name)
  image: \(.image)"
  '
}

ports_yaml() {
  echo "$PORTS_JSON" | jq -r '
    .[] | "- " +
      (if has("name") then "name: \(.name)
  " else "" end) +
      "port: \(.port)
  targetPort: \(.targetPort)" +
      (if has("protocol") then "
  protocol: \(.protocol)" else "" end) +
      (if has("nodePort") then "
  nodePort: \(.nodePort)" else "" end)
  '
}

echo "Ports element types: $(echo "$PORTS_JSON" | jq -c 'if type=="array" then map(type) else type end')"

case "$EVENT_TYPE" in
  "DELETED")
    kubectl -n "$NAMESPACE" delete deployment "$NAME" --ignore-not-found
    kubectl -n "$NAMESPACE" delete service "$SERVICE_NAME" --ignore-not-found
    echo "🗑️ Deleted deployment/service for '$NAME'"
    ;;

  "ADDED"|"MODIFIED")
    validate_spec

    CONTAINERS_YAML="$(containers_yaml)"
    PORTS_YAML="$(ports_yaml)"

    # Debug (optional):
    # echo "Containers YAML:"; echo "$CONTAINERS_YAML"
    # echo "Ports YAML:"; echo "$PORTS_YAML"

    # Deployment
    kubectl -n "$NAMESPACE" apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $NAME
spec:
  replicas: $REPLICAS
  selector:
    matchLabels:
      app: $NAME
  template:
    metadata:
      labels:
        app: $NAME
    spec:
      containers:
$(echo "$CONTAINERS_YAML" | sed 's/^/        /')
EOF

    # Service
    kubectl -n "$NAMESPACE" apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: $SERVICE_NAME
spec:
  selector:
    app: $NAME
  type: $SERVICE_TYPE
  ports:
$(echo "$PORTS_YAML" | sed 's/^/  /')
EOF
    ;;

  ""|"SYNCHRONIZATION")
    # Initial sync or empty; ignore.
    echo "ℹ️ Sync/empty event ignored."
    ;;

  *)
    echo "⚠️ Unknown event type: $EVENT_TYPE"
    ;;
esac