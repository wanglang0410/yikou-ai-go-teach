$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
$Overlay = if ($args.Count -gt 0) { $args[0] } else { "local" }

Write-Host "==> Building backend image..."
docker build -t yikou-ai-go-backend:latest $RootDir

Write-Host "==> Building frontend production image..."
docker build -f "$RootDir\yikou-ai-feiwu-front\Dockerfile.prod" `
  -t yikou-ai-go-frontend:latest `
  "$RootDir\yikou-ai-feiwu-front"

if (Get-Command minikube -ErrorAction SilentlyContinue) {
  try {
    minikube status | Out-Null
    Write-Host "==> Loading images into minikube..."
    minikube image load yikou-ai-go-backend:latest
    minikube image load yikou-ai-go-frontend:latest
  } catch {}
}

Write-Host "==> Applying Kubernetes manifests (overlay: $Overlay)..."
kubectl apply -k "$RootDir\k8s\overlays\$Overlay"

Write-Host "==> Waiting for deployments..."
kubectl rollout status deployment/mysql -n yikou-ai --timeout=180s
kubectl rollout status deployment/redis -n yikou-ai --timeout=120s
kubectl rollout status deployment/backend -n yikou-ai --timeout=180s
kubectl rollout status deployment/frontend -n yikou-ai --timeout=180s

Write-Host ""
Write-Host "Deployment complete."
Write-Host "  Frontend NodePort : http://localhost:30080"
Write-Host "  Backend  NodePort : http://localhost:30123/api/ping"
Write-Host "  Namespace         : yikou-ai"
Write-Host ""
Write-Host "Update AI API key:"
Write-Host "  kubectl edit secret backend-secret -n yikou-ai"
