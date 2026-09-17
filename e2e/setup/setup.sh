#!/usr/bin/env bash
set -aeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "Running setup.sh"

echo "Deploying FusionAuth + PostgreSQL..."
${KUBECTL} apply -f "${PROJECT_ROOT}/e2e/setup/fusionauth.yaml"

echo "Waiting for PostgreSQL to be ready..."
${KUBECTL} -n fusionauth wait deployment/postgresql \
  --for=condition=Available --timeout=5m

echo "Waiting for FusionAuth to be ready..."
${KUBECTL} -n fusionauth wait deployment/fusionauth \
  --for=condition=Available --timeout=10m

echo "Creating provider credentials secret pointing to local FusionAuth..."
${KUBECTL} -n crossplane-system create secret generic provider-secret \
  --from-literal=credentials='{"host":"http://fusionauth.fusionauth.svc.cluster.local:9011","api_key":"e2e-test-api-key-00000000000000000000"}' \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -

echo "Waiting until provider is healthy..."
${KUBECTL} wait provider.pkg --all --for condition=Healthy --timeout 5m

echo "Waiting for all pods to come online..."
${KUBECTL} -n crossplane-system wait --for=condition=Available deployment --all --timeout=5m

echo "Creating default (cluster)provider configs..."
${KUBECTL} apply -f ${SCRIPT_DIR}/providerconfigs.yaml

${KUBECTL} wait provider.pkg --all --for condition=Healthy --timeout 5m
${KUBECTL} -n crossplane-system wait --for=condition=Available deployment --all --timeout=5m
