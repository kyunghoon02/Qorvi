# Qorvi Terraform

This Terraform layout targets the cheapest practical GCP deployment shape for the Qorvi backend while keeping the existing Next.js frontend on Vercel.

- `qorvi.app` frontend stays on `Vercel`
- `api.qorvi.app` points to `1x Compute Engine VM`
- one public static IP
- `nginx` on the host
- `api + postgres + redis + neo4j` on the same VM via Docker Compose
- optional `worker-backfill` profile on the same VM
- optional private GitHub repo bootstrap via `app_repo_token`

## Recommended Architecture

For the current product stage, the most cost-efficient shape is:

- frontend stays on `qorvi.app` on Vercel
- backend runs on `api.qorvi.app` on GCP
- one Singapore VM hosts everything stateful

Why this shape:

- `Cloud SQL + Memorystore + separate workers` would add fixed monthly cost immediately.
- Qorvi's backend is still in a pre-scale stage, so keeping Postgres, Redis, and Neo4j on one VM is cheaper and simpler.
- `e2-standard-2` is a safer minimum than `e2-medium` because Neo4j, Postgres, Redis, and the Go API share the same host.
- keeping the web frontend on Vercel avoids adding GCP load balancer/CDN/static hosting complexity for no real benefit right now

## Final Topology

```text
User
  -> https://qorvi.app              (Vercel / Next.js frontend)
       -> /v1/* rewrite or direct client fetch
            -> https://api.qorvi.app (GCP VM / nginx)
                 -> api
                 -> postgres
                 -> redis
                 -> neo4j
```

## DNS Shape

Recommended DNS records:

- `qorvi.app` -> keep on Vercel
- `www.qorvi.app` -> keep on Vercel if used
- `api.qorvi.app` -> `A` record to the Terraform-managed static IP

This keeps the product split clean:

- Vercel handles the public web app and Next.js deployment
- GCP handles the stateful backend

## Vercel Environment

Set these in the Qorvi web project on Vercel:

```text
NEXT_PUBLIC_APP_BASE_URL=https://qorvi.app
NEXT_PUBLIC_API_BASE_URL=https://api.qorvi.app
API_PROXY_TARGET=https://api.qorvi.app
```

Keep backend-only secrets out of Vercel.

## GCP Environment

In `terraform.tfvars`, keep:

```text
APP_BASE_URL=https://qorvi.app
NEXT_PUBLIC_APP_BASE_URL=https://qorvi.app
NEXT_PUBLIC_API_BASE_URL=https://api.qorvi.app
API_PROXY_TARGET=https://api.qorvi.app
app_domain=api.qorvi.app
```

The backend VM does not need to host the frontend.

## TLS

Terraform provisions `nginx` and now installs `certbot`, but certificate issuance should happen after DNS for `api.qorvi.app` is already pointing at the VM.

Cheapest recommended path:

1. apply Terraform
2. point `api.qorvi.app` to the static IP
3. wait for DNS propagation
4. SSH to the VM and run:

```bash
sudo certbot --nginx -d api.qorvi.app --redirect -m ops@example.com --agree-tos --no-eff-email
```

That keeps TLS on the VM and avoids paying for a GCP HTTPS load balancer too early.

## Rollout Order

1. Deploy backend VM with Terraform
2. Add/update `api.qorvi.app` DNS `A` record to the Terraform static IP
3. Issue TLS cert with `certbot`
4. Set Vercel web env to `https://api.qorvi.app`
5. Redeploy the Vercel frontend
6. Verify:
   - `https://qorvi.app`
   - `https://api.qorvi.app/healthz`
   - wallet detail page reads from the new API origin

## Structure

```text
terraform/
  environments/
    prod/
  modules/
    project_services/
    network/
    firewall/
    static_ip/
    compute_vm/
```

## First Use

1. Copy `terraform/environments/prod/terraform.tfvars.example` to `terraform.tfvars`.
2. Fill in your GCP project, SSH settings, repo URL, and `app_env_content`.
   - If the repo is private, also set `app_repo_token` to a GitHub token with read access.
3. Authenticate with GCP:

```bash
gcloud auth application-default login
```

4. Apply:

```bash
cd terraform/environments/prod
terraform init
terraform plan
terraform apply
```

## Notes

- The default region is `asia-southeast1` because you explicitly wanted Singapore.
- The startup script clones the repo, writes `.env.backend`, starts Docker Compose, applies Postgres and Neo4j migrations, and brings up the API.
- When `app_repo_token` is set, startup uses an HTTP auth header for clone/fetch so the token is not written into the git remote URL.
- Use an `https://...` clone URL for private repos so the auth header can scope cleanly to the remote host.
- The optional worker is disabled by default to keep cost and background provider usage down.
- If you later need live warm-up/backfill on the server, enable `QORVI_ENABLE_BACKFILL_WORKER=true`.
- The Terraform output `app_urls.domain` assumes TLS on `https://${var.app_domain}` after `certbot` has been run.
