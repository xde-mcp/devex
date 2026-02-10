# Devex Kubernetes Cluster Setup Guide (Any Cloud Provider)

This guide sets up Kubernetes so Devex routes REPL traffic as:

- `https://repl.parthkapoor.me/<repl-id>/<route>` -> REPL pod `/<route>`

Design choice used here:

- No external cloud load balancer
- Traefik runs with `hostNetwork: true` and binds directly on node ports `80` and `443`

Use this with any provider (Civo, DigitalOcean, AWS, GCP, Azure, Hetzner, etc.) as long as:

- You can get at least one public node IP
- You can open inbound ports `80/443`
- You can point DNS to that node

---

## 1) Prerequisites

### Required tools

- `kubectl`
- `helm`
- `curl`
- `openssl`

### Required DNS

- `A` record: `repl.parthkapoor.me` -> `<public-ingress-node-ip>`

### Required network rules

Allow inbound from the internet to the ingress node:

- TCP `80`
- TCP `443`

### Verify

```bash
kubectl version --client
helm version
nslookup repl.parthkapoor.me
```

Expected:

- Tools available
- DNS resolves to your intended public ingress node IP

---

## 2) Choose Ingress Node

Pick exactly one node to receive external traffic.

```bash
kubectl get nodes -o wide
kubectl label node <INGRESS_NODE_NAME> devex.ingress=true --overwrite
kubectl get nodes --show-labels | grep devex.ingress
```

Expected:

- One node has label `devex.ingress=true`
- That node should be the one with the public IP used in DNS

---

## 3) Ensure No Conflicting Ingress Controller

If using K3s, default Traefik may already exist. Disable/remove it before installing your managed Traefik release.

Check:

```bash
kubectl -n kube-system get deploy,ds,svc | grep -i traefik || true
```

Expected:

- No active built-in Traefik that would conflict with your Helm-managed deployment

---

## 4) Install Traefik (Host Network Mode)

Install with base values:

```bash
helm repo add traefik https://traefik.github.io/charts
helm repo update
helm upgrade --install traefik traefik/traefik \
  --namespace traefik \
  --create-namespace \
  --reset-values \
  -f infra/k8s/traefik-values.yaml
```

Pin Traefik to your ingress node:

```bash
helm upgrade --install traefik traefik/traefik \
  --namespace traefik \
  --create-namespace \
  --reset-values \
  -f infra/k8s/traefik-values.yaml \
  -f infra/k8s/traefik-values-ingress-node.yaml
```

### Verify

```bash
kubectl -n traefik rollout status deploy/traefik --timeout=180s
kubectl -n traefik get pods -o wide
kubectl -n traefik get deploy traefik -o yaml | grep certificatesresolvers || true
curl -v http://repl.parthkapoor.me/
curl -vk https://repl.parthkapoor.me/
```

Expected:

- Traefik pod is `Running`
- Pod scheduled on ingress-labeled node
- No `certificatesresolvers` args (Traefik native ACME disabled)
- HTTP/HTTPS respond from Traefik (404 is acceptable before app routes exist)

---

## 5) Install cert-manager (Helm)

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.18.1 \
  --set crds.enabled=true
```

Wait until all cert-manager deployments are ready:

```bash
kubectl wait --for=condition=Available deployment/cert-manager -n cert-manager --timeout=180s
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=180s
kubectl wait --for=condition=Available deployment/cert-manager-cainjector -n cert-manager --timeout=180s
kubectl get pods -n cert-manager
```

Expected:

- `cert-manager`, `cert-manager-webhook`, and `cert-manager-cainjector` are all `Available`

---

## 6) Issue Staging TLS Certificate First

Apply staging issuer and certificate:

```bash
kubectl apply -f infra/k8s/cert-issuer-staging.yaml
kubectl apply -f infra/k8s/certificate-staging.yaml
```

### Verify

```bash
kubectl get clusterissuer letsencrypt-staging
kubectl get certificate,certificaterequest,order,challenge -n default
```

Expected:

- `clusterissuer/letsencrypt-staging` is `READY=True`
- `certificate/repl-root-certificate` becomes `READY=True`
- temporary ACME `Challenge` resources disappear after success

---

## 7) Deploy and Verify Whoami Smoke Test

```bash
kubectl apply -f infra/k8s/smoke-test-whoami.yaml
kubectl get deploy,svc,ingress,middleware -n default | grep whoami
```

Test rewrite behavior:

```bash
curl -vk https://repl.parthkapoor.me/test-repl
curl -vk https://repl.parthkapoor.me/test-repl/anything
```

Expected:

- `/test-repl` reaches backend `/`
- `/test-repl/anything` reaches backend `/anything`
- Response body includes whoami details

---

## 8) Switch to Production TLS

Apply production issuer and production certificate:

```bash
kubectl apply -f infra/k8s/cert-issuer-production.yaml
kubectl apply -f infra/k8s/certificate-production.yaml
```

### Verify

```bash
kubectl get clusterissuer letsencrypt-cluster-issuer
kubectl get certificate,certificaterequest,order,challenge -n default
openssl s_client -connect repl.parthkapoor.me:443 -servername repl.parthkapoor.me </dev/null 2>/dev/null | openssl x509 -noout -issuer -subject -dates
```

Expected:

- Issuer is ready
- Certificate is ready
- Issuer line shows production Let’s Encrypt CA (not staging)

---

## 9) Prepare Secrets Needed by Core REPL Lifecycle

Core REPL init/deactivation flow expects secret `aws-creds` in `default` namespace.

```bash
kubectl -n default create secret generic aws-creds \
  --from-literal=access_key='<ACCESS_KEY>' \
  --from-literal=secret_key='<SECRET_KEY>'
```

If it already exists, update it safely:

```bash
kubectl -n default delete secret aws-creds
kubectl -n default create secret generic aws-creds \
  --from-literal=access_key='<ACCESS_KEY>' \
  --from-literal=secret_key='<SECRET_KEY>'
```

### Verify

```bash
kubectl -n default get secret aws-creds
```

Expected:

- Secret exists and is readable by workloads in `default`

---

## 10) Core-Service Integration Checks

Core code uses:

- Ingress host from `RUNNER_CLUSTER_IP` (should be `repl.parthkapoor.me`)
- TLS secret `tls-secret`
- cert-manager annotation `cert-manager.io/cluster-issuer=letsencrypt-cluster-issuer`

Run core with correct environment and activate a REPL.

### Verify created resources

```bash
kubectl get deploy,svc,ingress,middleware -n default | grep <repl-id>
curl -vk https://repl.parthkapoor.me/<repl-id>/ping
```

### Verify cleanup after deactivation

```bash
kubectl get deploy,svc,ingress,middleware -n default | grep <repl-id> || true
```

Expected:

- On activation: all resources exist and endpoint responds
- On deactivation: resources are deleted and workspace snapshot is uploaded

---

## 11) Common Failure Modes and Exact Fixes

### A) Traefik pod stuck `Pending`

Symptom:

- `0/2 nodes are available: node(s) didn't match Pod's node affinity/selector`

Fix:

```bash
kubectl get nodes --show-labels | grep devex.ingress
kubectl label node <INGRESS_NODE_NAME> devex.ingress=true --overwrite
helm upgrade --install traefik traefik/traefik -n traefik --create-namespace --reset-values \
  -f infra/k8s/traefik-values.yaml \
  -f infra/k8s/traefik-values-ingress-node.yaml
```

### B) Issuer apply fails with webhook endpoint error

Symptom:

- `failed calling webhook ... no endpoints available for service "cert-manager-webhook"`

Fix:

```bash
kubectl wait --for=condition=Available deployment/cert-manager-webhook -n cert-manager --timeout=180s
kubectl apply -f infra/k8s/cert-issuer-staging.yaml
```

### C) ACME challenge pending with `connection refused`

Symptom:

- cert-manager challenge self-check cannot connect to `http://repl.parthkapoor.me/.well-known/...`

Root cause:

- DNS points to public node A, but Traefik scheduled on node B (no public IP) or port 80 closed

Fix:

- Pin Traefik to DNS-target node
- Open inbound TCP 80 in provider firewall/security group

### D) ACME challenge pending with `404`

Symptom:

- wrong status `404`, expected `200`

Root cause candidates:

- stale challenge resources
- conflicting Traefik native ACME configuration

Fix:

```bash
kubectl delete certificate repl-root-certificate -n default --ignore-not-found
kubectl delete secret tls-secret -n default --ignore-not-found
kubectl delete order,challenge,certificaterequest -n default --all
kubectl apply -f infra/k8s/cert-issuer-staging.yaml
kubectl apply -f infra/k8s/certificate-staging.yaml
```

Also ensure Traefik has no `certificatesresolvers.*` args.

---

## 12) Final Acceptance Checklist

- `curl -v http://repl.parthkapoor.me/` reaches Traefik
- `curl -vk https://repl.parthkapoor.me/` reaches Traefik
- Staging certificate issuance succeeds
- Production certificate issuance succeeds
- whoami rewrite test passes on both `/test-repl` and `/test-repl/anything`
- Core-created REPL ingress path works end-to-end
- Deactivation removes resources and persists workspace

