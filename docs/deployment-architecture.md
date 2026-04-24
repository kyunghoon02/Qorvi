# Qorvi Deployment Architecture

## Goal

Deploy the Qorvi backend cheaply enough for an early-stage prototype while keeping the existing web frontend where it already fits best.

- wallet detail API
- Redis caching
- Neo4j graph reads
- Postgres primary storage
- optional background backfill worker
- a Vercel-hosted frontend on `qorvi.app`

## Recommended Shape

- frontend stays on `qorvi.app` on Vercel
- backend runs on `api.qorvi.app` on GCP
- one GCP Compute Engine VM in `asia-southeast1`
- Docker Compose on that VM
- services on the VM:
  - `api`
  - `postgres`
  - `redis`
  - `neo4j`
  - optional `worker-backfill`

## Why This Is The Best Cost Shape Right Now

### Keep the frontend on Vercel

The frontend is already a Next.js app and is currently served by Vercel.

Moving it to GCP right now would mean taking on extra work for:

- frontend static hosting
- SSL and CDN routing
- origin rewrites
- cache behavior
- deployment workflow changes

That adds operator burden without improving the part of the stack that is actually expensive or stateful.

So the cheaper and simpler split is:

- `qorvi.app` on Vercel
- `api.qorvi.app` on GCP

### Keep managed services out for now

Using managed Postgres, managed Redis, and a separate graph/database tier would be cleaner, but it would immediately create fixed monthly cost across multiple products.

At the current stage, Qorvi does not need that separation yet. One VM is the cheapest practical shape that still matches the backend dependency graph.

### Use one backend VM, not many small services

The backend is not just a stateless Go API:

- Neo4j needs dedicated memory
- Postgres needs local storage
- Redis should stay close to the API
- the backfill worker benefits from local network access to all three

So the cheapest practical option is not "many tiny instances". It is one moderate VM.

### Recommended VM size

Start with:

- `e2-standard-2`
- `50 GB pd-standard`

This is the lowest-risk starting point for:

- Go API
- Postgres
- Redis
- Neo4j with reduced heap/page cache

If usage is very light, you can later try downsizing. If graph usage or worker activity increases, move to `e2-standard-4` before splitting the stack into managed services.

## Tradeoffs

### Pros

- lowest practical monthly cost
- simple Terraform
- simple operator workflow
- easy snapshot-and-replace deployment
- no need to rebuild the frontend delivery stack

### Cons

- single-node failure domain
- stateful services share CPU and memory
- not ideal for heavy indexing or concurrent graph workloads
- backend TLS must be handled on the VM unless you add a GCP load balancer later

## DNS And Traffic Flow

Recommended DNS layout:

- `qorvi.app` -> Vercel
- `www.qorvi.app` -> Vercel
- `api.qorvi.app` -> GCP static IP

Recommended request flow:

1. Browser loads `https://qorvi.app` from Vercel
2. Frontend calls `/v1/*`
3. Vercel rewrites or frontend origin config forwards that traffic to `https://api.qorvi.app`
4. `api.qorvi.app` terminates TLS on the VM
5. `nginx` proxies to the backend container on port `4000`

## TLS Recommendation

For the current stage, the cheapest setup is:

- use the Terraform-managed static IP
- point `api.qorvi.app` at that IP
- issue a Let's Encrypt certificate on the VM with `certbot --nginx`

Do not add a GCP HTTPS load balancer yet unless you actually need:

- multi-backend routing
- global edge termination
- managed certificates without VM termination
- higher availability than a single VM

## Deployment Sequence

1. `terraform apply`
2. copy the new static IP
3. create/update the `api.qorvi.app` DNS `A` record
4. wait for propagation
5. SSH into the VM and run `certbot`
6. set Vercel web env:
   - `NEXT_PUBLIC_API_BASE_URL=https://api.qorvi.app`
   - `API_PROXY_TARGET=https://api.qorvi.app`
7. redeploy the frontend on Vercel
8. verify frontend and backend together

## Scale Path

When Qorvi grows, scale in this order:

1. Increase VM size
2. Move Postgres to Cloud SQL
3. Move Redis to Memorystore
4. Keep Neo4j on its own VM or managed alternative
5. Split workers onto a second VM

That order keeps cost controlled while moving the most sensitive stateful parts out only when they actually need it.
