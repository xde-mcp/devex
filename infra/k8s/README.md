# Devex Kubernetes Manifests

This folder contains only the manifests required for Devex REPL routing and TLS on Kubernetes.

## File Map

- `traefik-values.yaml`
  - Base Traefik Helm values (`hostNetwork: true`, ports `80/443`, no Traefik native ACME).
- `traefik-values-ingress-node.yaml`
  - Optional overlay to pin Traefik to the node labeled `devex.ingress=true`.
- `cert-issuer-staging.yaml`
  - Let’s Encrypt staging `ClusterIssuer` for safe validation.
- `cert-issuer-production.yaml`
  - Let’s Encrypt production `ClusterIssuer` named `letsencrypt-cluster-issuer`.
- `certificate-staging.yaml`
  - Staging certificate for `repl.parthkapoor.me` to `tls-secret`.
- `certificate-production.yaml`
  - Production certificate for `repl.parthkapoor.me` to `tls-secret`.
- `smoke-test-whoami.yaml`
  - End-to-end ingress + middleware + path-rewrite smoke test.
- `SETUP_GUIDE.md`
  - Complete cluster setup guide (cloud-provider agnostic) with verification at every step.

## Why This Set Is Minimal

- Core service dynamically creates per-REPL `Deployment`, `Service`, `Ingress`, and Traefik `Middleware`.
- These manifests only provide shared cluster prerequisites:
  - Ingress controller (Traefik) configuration
  - cert-manager issuers/certificates
  - smoke-test resources

For full setup instructions, use `infra/k8s/SETUP_GUIDE.md`.
