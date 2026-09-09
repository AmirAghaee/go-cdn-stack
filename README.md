# GO CDN STACK

This project implements a **Content Delivery Network (CDN)** system with the following stack:

- **Gin** (HTTP framework)
- **React + Tailwind CSS** (administration dashboard)
- **MongoDB** (Control Panel persistence)
- **NATS** (messaging bus for events & health checks)
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

Docker Compose uses a development-only JWT secret by default. For a shared or
production deployment, set a strong `JWT_SECRET` in the environment or copy the
root `.env.example` to `.env` and replace its value before starting the stack.

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
