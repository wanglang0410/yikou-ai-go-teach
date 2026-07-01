#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OVERLAY="${1:-local}"

echo "==> Building backend image..."
docker build -t yikou-ai-go-backend:latest "$ROOT_DIR"

echo "==> Building frontend production image..."
docker build -f "$ROOT_DIR/yikou-ai-feiwu-front/Dockerfile.prod" \
  -t yikou-ai-go-frontend:latest \
  "$ROOT_DIR/yikou-ai-feiwu-front"

if command -v minikube >/dev/null 2>&1 && minikube status >/dev/null 2>&1; then
  echo "==> Loading images into minikube..."
  minikube image load yikou-ai-go-backend:latest
  minikube image load yikou-ai-go-frontend:latest
elif command -v kind >/dev/null 2>&1; then
  echo "==> Loading images into kind (cluster: kind)..."
  kind load docker-image yikou-ai-go-backend:latest --name kind || true
  kind load docker-image yikou-ai-go-frontend:latest --name kind || true
fi

echo "==> Applying Kubernetes manifests (overlay: $OVERLAY)..."
kubectl apply -k "$ROOT_DIR/k8s/overlays/$OVERLAY"

echo "==> Waiting for deployments..."
kubectl rollout status deployment/mysql -n yikou-ai --timeout=180s
kubectl rollout status deployment/redis -n yikou-ai --timeout=120s
kubectl rollout status deployment/backend -n yikou-ai --timeout=180s
kubectl rollout status deployment/frontend -n yikou-ai --timeout=180s

echo ""
echo "Deployment complete."
echo "  Frontend NodePort : http://localhost:30080"
echo "  Backend  NodePort : http://localhost:30123/api/ping"
echo "  MySQL    NodePort : localhost:30306 (root / yikou123456, db: yikou_ai)"
echo "  Redis    NodePort : localhost:30379"
echo "  Namespace         : yikou-ai"
echo ""
echo "On minikube docker driver (Mac/Windows), use port-forward or minikube service:"
echo "  kubectl port-forward -n yikou-ai svc/mysql 30306:3306"
echo "  kubectl port-forward -n yikou-ai svc/redis 30379:6379"
echo ""
echo "Update AI API key:"
echo "  kubectl edit secret backend-secret -n yikou-ai"
