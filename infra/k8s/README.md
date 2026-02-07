## Setting up the K8s Cluster

> 📌 **Note**: The following docs were meant for personal use. If you find any issue, feel free to contribute to the project.

Please read the [Contribution Guide](./CONTRIBUTING.md) before sending PR requests.

---

### Step 0: Prerequisites

Your system should have the following installed:

```
* This project cloned
* Docker
* kubectl
* kind (For Local Testing only)
* helm (Optional — already included via pre-generated ingress-controller.yaml)
```

---

### Step 1: Create a new Cluster

You can create a new cluster on Cloud (DigitalOcean, AWS, Azure) or test locally using `kind`.

To create a local Kind cluster:

```bash
kind create cluster --name devex-cluster
```

To verify the cluster is running:

```bash
kubectl get nodes
```

Sample output:

```
NAME                          STATUS   ROLES           AGE   VERSION
devex-cluster-control-plane   Ready    control-plane   45m   v1.33.1
```

---

### Step 2: Install Traefik Ingress Controller

We use Traefik as our Ingress Controller with `hostNetwork: true` to bind directly to the node's ports 80/443.

#### 1. (K3s only) Disable default Traefik
If you are using K3s, you must disable the built-in Traefik to avoid port conflicts:

```bash
# Temporarily scale down
kubectl delete daemonset traefik -n kube-system

# To permanently disable, add --disable traefik to your k3s startup args
```

#### 2. Install Traefik using Helm

```bash
helm repo add traefik https://traefik.github.io/charts
helm repo update
helm install traefik traefik/traefik -n traefik --create-namespace -f ./traefik_values.yaml
```

---

### Step 3: (Optional) Port Forwarding for Local Testing

If you aren't using a public IP, you can forward traffic:

```bash
kubectl -n traefik port-forward deployment/traefik 8000:8000 8443:8443
```

---

### Step 4: Load Local Docker Image into the Cluster (If Needed)

If you're not using a remote registry for your `runner-service` image:

```bash
kind load docker-image runner-service:latest
```

Skip if the image is pulled from GHCR.

---

### Step 5: Add Secrets for S3/R2 Access

For the `s3-downloader` and `runner` to fetch/upload workspace files:

```bash
kubectl create secret generic aws-creds \
  --from-literal=access_key=<YOUR_ACCESS_KEY> \
  --from-literal=secret_key=<YOUR_SECRET_KEY>
```

---

### Step 6: Install cert-manager and Traefik Issuer

We use **cert-manager** to automatically provision TLS certificates using **Let’s Encrypt**.

#### 1. Install cert-manager

```bash
kubectl apply -f ./cert-manager.yaml
```

#### 2. Create Let's Encrypt ClusterIssuer for Traefik

```bash
kubectl apply -f ./cert-issuer-traefik.yaml
```

This file defines a `ClusterIssuer` named `letsencrypt-cluster-issuer` using the `traefik` ingress class for HTTP-01 challenges.

---

### 🚀 Path Rewriting with Traefik Middleware

Unlike Nginx, Traefik uses a separate `Middleware` resource for path stripping. The `core-service` automatically generates a `Middleware` of type `StripPrefix` for each REPL to ensure the runner receives requests at `/` instead of `/<replId>`.

### 🔐 Important TLS Notes

* Your domain (`repl.parthkapoor.me`) must point to your cluster node's IP.
* Port `80` and `443` must be open and not occupied by other services.
* The ACME challenge requires access via HTTP on port 80.

---

### ✅ Congrats! Your K8s Cluster is Ready for TLS-enabled REPLs

You can now:

* Create dynamic workloads (`/repl-id`)
* Access them via `https://repl.parthkapoor.me/repl-id`
* Get certificates managed and auto-renewed via cert-manager

---
