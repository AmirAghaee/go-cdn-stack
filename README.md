# GO CDN STACK

This project implements a **Content Delivery Network (CDN)** system using **Go (Golang)** with the following stack:

- **Gin** (HTTP framework)
- **MongoDB** (Control Panel persistence)
- **NATS** (messaging bus for events & health checks)
- **Monorepo** structure with multiple services

## 📂 Repository Structure

```
.
├── control-panel       # Management API + MongoDB persistence + NATS subscriber
├── edge                # Edge CDN service (serves cached content, proxies requests)
└── origin-sample       # Sample origin server (static files for testing)
```

### Service Breakdown

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

- **Language:** Go 1.25+
- **Frameworks:** Gin, NATS, MongoDB driver
- **Persistence:** MongoDB (control panel), disk-based cache and configuration snapshot (edge)

---

## 📌 TODO

- [ ] Expand unit and integration test coverage
- [ ] Implement cache invalidation via NATS
- [ ] Add rate limiting and logging middleware

---
