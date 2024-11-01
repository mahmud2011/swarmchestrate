# The original upstream file is located at: https://github.com/sedflix/multi-cluster-istio-kind

#!/bin/bash
set -e

KIND_IMAGE="kindest/node:v1.33.0"

# Default cluster counts (can be overridden via environment variables)
NUM_CLOUD_CLUSTERS=${NUM_CLOUD_CLUSTERS:-3}
NUM_FOG_CLUSTERS=${NUM_FOG_CLUSTERS:-3}
NUM_EDGE_CLUSTERS=${NUM_EDGE_CLUSTERS:-3}
NUM_RAFT_CLUSTERS=${NUM_RAFT_CLUSTERS:-3}

NUM_CLUSTERS=$((NUM_CLOUD_CLUSTERS + NUM_FOG_CLUSTERS + NUM_EDGE_CLUSTERS))
OS=$(uname)

# Display usage if no arguments are provided
usage() {
  echo "Usage: $0 {kb|swagger|kind_create|kind_delete|metallb|remove_metallb|ingress|ns|raft_lb|all}"
  exit 1
}

# Runs the knowledge-base docker setup
kb() {
    echo "Starting knowledge-base setup..."
    cd knowledge-base
    docker compose up -d
    until docker exec postgres pg_isready -U foo -d knowledge_base; do
        echo "Waiting for database to be ready..."
        sleep 3
    done
    migrate -database "postgres://foo:pass@localhost:5432/knowledge_base?sslmode=disable" -path ./migrations up
    cd ..
    echo "+++ kb Complete"
}

kind_create() {
    echo "Creating ${NUM_CLUSTERS} clusters (Cloud=${NUM_CLOUD_CLUSTERS}, Fog=${NUM_FOG_CLUSTERS}, Edge=${NUM_EDGE_CLUSTERS})"

    # profile labels
    PROFILES=("energy" "cost" "performance")

    CPU_LIMIT="0.4"
    MEM_LIMIT="1g"

    for i in $(seq 1 ${NUM_CLUSTERS}); do
        # Determine domain and name prefix
        if [ $i -le $NUM_CLOUD_CLUSTERS ]; then
            DOMAIN="cloud"
        elif [ $i -le $((NUM_CLOUD_CLUSTERS + NUM_FOG_CLUSTERS)) ]; then
            DOMAIN="fog"
        else
            DOMAIN="edge"
        fi

        # Determine cluster profile based on cycle: 1 → energy, 2 → cost, 3 → performance
        PROFILE_INDEX=$(( (i - 1) % 3 ))
        PROFILE="${PROFILES[$PROFILE_INDEX]}"
        NAME="${DOMAIN}-${PROFILE}"

        echo "Creating cluster: ${NAME}"

        # Choose config file
        if [ "$PROFILE" = "performance" ]; then
            CONFIG_FILE="config/kind-config.yaml"
        else
            CONFIG_FILE="config/kind-config-reduced.yaml"
        fi

        kind create cluster --config "${CONFIG_FILE}" --name "${NAME}" --image="${KIND_IMAGE}"

        # Apply Docker limits to both control-plane and worker node
        for node in "${NAME}-control-plane" "${NAME}-worker"; do
            echo "Applying limits to $node (CPU=${CPU_LIMIT}, MEM=${MEM_LIMIT})"
            docker update --cpus="${CPU_LIMIT}" --memory="${MEM_LIMIT}" --memory-swap="${MEM_LIMIT}" "$node"
        done

        if [ "${OS}" != "Darwin" ]; then
            docker_ip=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${NAME}-control-plane")
            kubectl config set-cluster "kind-${NAME}" --server="https://${docker_ip}:6443"
        fi

        kubectl config rename-context "kind-${NAME}" "${NAME}"

        echo "Labeling worker nodes for ${NAME}"

        # Assign labels based on profile
        case "$PROFILE" in
            energy)
                ENERGY="0.0024042"
                PRICE="16.3884"
                BANDWIDTH="52500000000"
                ;;
            cost)
                ENERGY="0.0025689"
                PRICE="0.0042"
                BANDWIDTH="5000000000"
                ;;
            performance)
                ENERGY="0.0027335"
                PRICE="32.7726"
                BANDWIDTH="100000000000"
                ;;
        esac

        nodes=("${NAME}-worker")

        for node in "${nodes[@]}"; do
            kubectl label node "${node}" \
              swarmchestrate.tu-berlin.de/energy="${ENERGY}" \
              swarmchestrate.tu-berlin.de/price="${PRICE}" \
              swarmchestrate.tu-berlin.de/bandwidth="${BANDWIDTH}" \
              swarmchestrate.tu-berlin.de/location="Europe_Berlin" --overwrite
        done

        echo
    done

    kubectl config use-context cloud-energy
    cidr=$(docker network inspect -f '{{(index .IPAM.Config 0).Subnet}}' kind)
    echo "Kind CIDR is ${cidr}"
    echo "+++ kind_create Complete"
}

# Deletes all kind clusters
kind_delete() {
    echo "Deleting ${NUM_CLUSTERS} clusters"

    # Keep a profile label cycling through energy → cost → performance
    PROFILES=("energy" "cost" "performance")

    for i in $(seq 1 ${NUM_CLUSTERS}); do
        # Determine domain
        if [ $i -le $NUM_CLOUD_CLUSTERS ]; then
            DOMAIN="cloud"
        elif [ $i -le $((NUM_CLOUD_CLUSTERS + NUM_FOG_CLUSTERS)) ]; then
            DOMAIN="fog"
        else
            DOMAIN="edge"
        fi

        # Determine cluster profile
        PROFILE_INDEX=$(( (i - 1) % 3 ))
        PROFILE="${PROFILES[$PROFILE_INDEX]}"
        NAME="${DOMAIN}-${PROFILE}"

        echo "Deleting cluster: ${NAME}"

        # Rename context back to kind format before deletion (optional if needed)
        kubectl config rename-context "${NAME}" "kind-${NAME}" || true

        # Delete the cluster
        kind delete cluster --name "${NAME}"
    done

    echo "+++ kind_delete Complete"
}

# Deploys MetalLB in all clusters (creates metallb-system namespace if needed)
metallb() {
    echo "Deploying MetalLB in ${NUM_CLUSTERS} clusters"

    BASE_SUBNET=$(docker network inspect kind -f '{{(index .IPAM.Config 0).Subnet}}')
    echo "Base subnet: ${BASE_SUBNET}"

    BASE_IP=$(echo "${BASE_SUBNET}" | cut -d'.' -f1-3)
    START_OCTET=$(echo "${BASE_SUBNET}" | cut -d'.' -f4 | cut -d'/' -f1)

    IPS_PER_CLUSTER=5
    PROFILES=("energy" "cost" "performance")

    for i in $(seq 1 ${NUM_CLUSTERS}); do
        # Determine domain
        if [ $i -le $NUM_CLOUD_CLUSTERS ]; then
            DOMAIN="cloud"
        elif [ $i -le $((NUM_CLOUD_CLUSTERS + NUM_FOG_CLUSTERS)) ]; then
            DOMAIN="fog"
        else
            DOMAIN="edge"
        fi

        # Determine profile and full cluster name
        PROFILE_INDEX=$(( (i - 1) % 3 ))
        PROFILE="${PROFILES[$PROFILE_INDEX]}"
        NAME="${DOMAIN}-${PROFILE}"

        echo "Starting MetalLB deployment in ${NAME}"

        kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.9/config/manifests/metallb-native.yaml --context "${NAME}"

        kubectl wait --namespace metallb-system \
            --for=condition=ready pod \
            --selector=app=metallb \
            --timeout=300s \
            --context "${NAME}"

        START_IP=$((START_OCTET + (i - 1) * IPS_PER_CLUSTER + 1))
        END_IP=$((START_IP + IPS_PER_CLUSTER - 1))
        ADDR_RANGE="${BASE_IP}.${START_IP}-${BASE_IP}.${END_IP}"
        echo "${NAME} will use address range: ${ADDR_RANGE}"

        yq eval 'select(di == 0) | .spec.addresses = ["'"${ADDR_RANGE}"'"]' config/metallb-cr.yaml > /tmp/ipaddresspool.yaml
        yq eval 'select(di == 1)' config/metallb-cr.yaml > /tmp/l2advertisement.yaml

        kubectl apply -f /tmp/ipaddresspool.yaml --context "${NAME}"
        kubectl apply -f /tmp/l2advertisement.yaml --context "${NAME}"

        echo "----"
    done

    echo "+++ metallb Complete"
}

remove_metallb() {
    echo "Removing MetalLB from ${NUM_CLUSTERS} clusters"

    PROFILES=("energy" "cost" "performance")

    for i in $(seq 1 ${NUM_CLUSTERS}); do
        if [ $i -le $NUM_CLOUD_CLUSTERS ]; then
            DOMAIN="cloud"
        elif [ $i -le $((NUM_CLOUD_CLUSTERS + NUM_FOG_CLUSTERS)) ]; then
            DOMAIN="fog"
        else
            DOMAIN="edge"
        fi

        PROFILE_INDEX=$(( (i - 1) % 3 ))
        PROFILE="${PROFILES[$PROFILE_INDEX]}"
        NAME="${DOMAIN}-${PROFILE}"

        echo "Removing MetalLB from cluster: ${NAME}"

        # Delete CRs if they exist
        kubectl delete ipaddresspool --all --namespace metallb-system --context "${NAME}" 2>/dev/null
        kubectl delete l2advertisement --all --namespace metallb-system --context "${NAME}" 2>/dev/null

        # Delete MetalLB core components
        kubectl delete -f https://raw.githubusercontent.com/metallb/metallb/v0.14.9/config/manifests/metallb-native.yaml --context "${NAME}" 2>/dev/null

        # Delete the namespace
        kubectl delete namespace metallb-system --context "${NAME}" --ignore-not-found

        echo "---"
    done

    echo "+++ remove_metallb Complete"
}

# Deploys ingress controllers
ingress() {
    echo "Deploying ingress controller in ${NUM_CLUSTERS} clusters"

    PROFILES=("energy" "cost" "performance")

    for i in $(seq 1 ${NUM_CLUSTERS}); do
        # Determine domain
        if [ $i -le $NUM_CLOUD_CLUSTERS ]; then
            DOMAIN="cloud"
        elif [ $i -le $((NUM_CLOUD_CLUSTERS + NUM_FOG_CLUSTERS)) ]; then
            DOMAIN="fog"
        else
            DOMAIN="edge"
        fi

        # Determine profile and full cluster name
        PROFILE_INDEX=$(( (i - 1) % 3 ))
        PROFILE="${PROFILES[$PROFILE_INDEX]}"
        NAME="${DOMAIN}-${PROFILE}"

        echo "Starting ingress controller deployment in ${NAME}"
        kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml --context "${NAME}"
        kubectl wait --namespace ingress-nginx \
            --for=condition=ready pod \
            --selector=app.kubernetes.io/component=controller \
            --timeout=90s \
            --context "${NAME}"
        echo "----"
    done

    echo "+++ ingress Complete"
}

# Creates swarmchestrate namespace in all clusters
ns() {
    echo "Creating swarmchestrate namespace in ${NUM_CLUSTERS} clusters"

    PROFILES=("energy" "cost" "performance")

    for i in $(seq 1 ${NUM_CLUSTERS}); do
        # Determine domain
        if [ $i -le $NUM_CLOUD_CLUSTERS ]; then
            DOMAIN="cloud"
        elif [ $i -le $((NUM_CLOUD_CLUSTERS + NUM_FOG_CLUSTERS)) ]; then
            DOMAIN="fog"
        else
            DOMAIN="edge"
        fi

        PROFILE_INDEX=$(( (i - 1) % 3 ))
        PROFILE="${PROFILES[$PROFILE_INDEX]}"
        NAME="${DOMAIN}-${PROFILE}"

        until kubectl --context "${NAME}" get namespace default >/dev/null 2>&1; do
            echo "Waiting for ${NAME} API server to be ready..."
            sleep 2
        done

        kubectl apply --validate=false -f config/ns.yaml --context "${NAME}"
        echo "----"
    done

    echo "+++ ns Complete"
}

# Applies config/lb.yaml to the first NUM_RAFT_CLUSTERS
raft_lb() {
    echo "Applying config/lb.yaml to first ${NUM_RAFT_CLUSTERS} clusters (raft nodes)"

    if [ "${NUM_RAFT_CLUSTERS}" -gt "${NUM_CLOUD_CLUSTERS}" ]; then
        echo "Error: NUM_RAFT_CLUSTERS cannot exceed NUM_CLOUD_CLUSTERS (${NUM_CLOUD_CLUSTERS})"
        exit 1
    fi

    PROFILES=("energy" "cost" "performance")

    for i in $(seq 1 ${NUM_RAFT_CLUSTERS}); do
        PROFILE_INDEX=$(( (i - 1) % 3 ))
        PROFILE="${PROFILES[$PROFILE_INDEX]}"
        NAME="cloud-${PROFILE}"

        echo "Applying config/lb.yaml to ${NAME}"
        kubectl apply -f config/lb.yaml -n swarmchestrate --context "${NAME}"
    done

    echo "+++ raft_lb Complete"
}

swagger() {
    echo "Generating swagger spec"

    cd resource-lead-agent
    swag init --parseDependency --parseInternal --parseDepth 1 -md ./documentation -o ./swagger
    cd ..

    echo "+++ swagger spec generation Complete"
}

# Runs the full setup in order
all() {
    kind_create
    # metallb
    ingress
    ns
    raft_lb
}

# Main execution dispatcher
if [ "$#" -eq 0 ]; then
    usage
fi

COMMAND=$1
shift

case "$COMMAND" in
    kb)
        kb "$@"
        ;;
    swagger)
        swagger "$@"
        ;;
    kind_create)
        kind_create "$@"
        ;;
    kind_delete)
        kind_delete "$@"
        ;;
    metallb)
        metallb "$@"
        ;;
    remove_metallb)
        remove_metallb "$@"
        ;;
    ingress)
        ingress "$@"
        ;;
    ns)
        ns "$@"
        ;;
    raft_lb)
        raft_lb "$@"
        ;;
    all)
        all "$@"
        ;;
    *)
        echo "Unknown command: $COMMAND"
        usage
        ;;
esac
