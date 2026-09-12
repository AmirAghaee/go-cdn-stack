# GO CDN STACK

This project implements a **Content Delivery Network (CDN)** system with the following stack:

- **Gin** (HTTP framework)
- **React + Tailwind CSS** (administration dashboard)
- **MongoDB** (Control Panel persistence)
- **NATS** (messaging bus for events & health checks)
- **Prometheus** (edge metrics collection and storage)
- **Monorepo** structure with multiple services

## 📂 Repository Structure

```
.
├── dashboard           # React administration dashboard
├── control-panel       # Management API + MongoDB persistence + NATS subscriber
├── edge                # Edge CDN service (serves cached content, proxies requests)
└── origin-sample       # Sample origin server (static files for testing)
```

### Service Breakdown

#### **Dashboard**
- Modern administration interface for user lifecycle and credentials, CDN configurations, and edge-node health
- Uses a same-origin reverse proxy to access the control-panel API
- Available at `http://localhost:3000` when running with Docker Compose

#### **Control Panel**
- REST API to manage users, CDNs, and snapshots
- Persists data in MongoDB
- Publishes configuration-change notifications and subscribes to edge health via NATS

#### **Edge**
- Receives client requests
- Caches static content (images, CSS, JS, fonts, video, audio)
- Proxies cache misses and non-cacheable requests directly to origin servers
- Stores cache on disk
- Loads a last-known-good configuration snapshot from disk at startup
- Synchronizes configuration at startup, periodically, and after NATS events
- Publishes health status directly through NATS

#### **Origin Sample**
- Simple static file server (images, JSON, video) for testing CDN flows

---

## 🚀 How to Run

### Docker Compose

The complete local stack runs without a `.env` file:

```bash
docker compose up --build
```

Open the dashboard at [http://localhost:3000](http://localhost:3000). The
dashboard proxies API requests to the control-panel container, so no separate
browser-side API configuration is required.

Prometheus is available at [http://localhost:9090](http://localhost:9090). It
scrapes the edge metrics endpoint every 15 seconds and retains data for 30 days
in the persistent `prometheus_data` Docker volume.

Prometheus discovers edge nodes from `prometheus/targets/edge.json`. Add a target
with its display name to that file and Prometheus will load it within 30 seconds
without a restart:

```json
{
  "targets": ["edge-02:8090"],
  "labels": { "edge_node": "edge-02" }
}
```

Keep each target reachable from the Prometheus container. The Grafana dashboard
builds its **Edge Node** filter automatically from these labels.

Grafana is available at [http://localhost:3001](http://localhost:3001). Sign in
as `admin` using `GRAFANA_ADMIN_PASSWORD` (`admin` by default for local
development). The provisioned **CDN Edge Overview** dashboard visualizes traffic,
latency, cache efficiency, throughput, origin performance, storage, and errors.

Docker Compose uses development-only credentials by default. For a shared or
production deployment, set strong, different `JWT_SECRET` and `EDGE_SERVICE_TOKEN`
values in the environment, or copy the root `.env.example` to `.env` and replace
both values before starting the stack.

### 1. Clone the repo

```bash
git clone https://github.com/AmirAghaee/go-cdn-stack.git
cd go-cdn-stack
```

### 2. Run Control Panel

Make sure MongoDB and NATS are running. A `.env` file is optional; the service
uses the values documented in `control-panel/.env.example` as its defaults.

```bash
cd control-panel
go run ./cmd/control-panel
```

Create the first user from an interactive terminal before using the protected
registration API:

```bash
go run ./cmd/control-panel create-user --email admin@example.com
```

For automation, provide the password over standard input instead of a command
argument:

```bash
printf '%s\n' "$CONTROL_PANEL_USER_PASSWORD" | \
  go run ./cmd/control-panel create-user --email admin@example.com --password-stdin
```

When the Compose stack is running, the same command is available in the
control-panel container:

```bash
docker compose exec -it control-panel ./app create-user --email admin@example.com
```

After the initial user is created, `POST /api/register` requires a bearer token from
`POST /login`.

### Dashboard development

Run the control panel and its dependencies, then start the Vite development
server in a separate terminal:

```bash
cd dashboard
npm ci
npm run dev
```

The development server listens on `http://localhost:3000` and proxies
`/control-api` to the control panel at `http://127.0.0.1:9001`.

### 3. Run Edge Service

A `.env` file is optional; the service uses the values documented in
`edge/.env.example` as its defaults.

```bash
cd edge
go run main.go
```

The internal listener (port `8090` by default) exposes `/livez` for process
liveness, `/readyz` for configuration-snapshot readiness, and `/metrics`.
Readiness becomes successful after a complete local or control-panel snapshot is
loaded, including a valid empty snapshot, and remains successful while serving a
last-known-good snapshot during transient synchronization failures.

### 4. Run Origin Sample

```bash
cd origin-sample
go run main.go
```

---

## ⚡ Development Notes

- Cache rules: Only `image/*`, `font/*`, `text/css`, `text/javascript`, `application/javascript`, `video/*`, and `audio/*` responses are cached.
- Non-GET requests are proxied directly to origin.
- Each cached item has metadata stored alongside the cached file (headers + expiry time).
- Edges sync CDNs from Control Panel at startup, periodically, and after NATS events.
- Edge health messages are published directly to NATS and consumed by Control Panel.

---

## 🛠️ Tech Stack

- **Languages:** Go 1.25+, TypeScript
- **Frameworks:** Gin, React, Tailwind CSS, NATS, MongoDB driver
- **Persistence:** MongoDB (control panel), disk-based cache and configuration snapshot (edge)

---

## 📌 TODO

- [ ] Expand unit and integration test coverage
- [ ] Implement cache invalidation via NATS
- [ ] Add rate limiting and logging middleware

---
