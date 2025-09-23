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
├── mid                 # Mid-tier service (cache logic, sync with control panel, health checks)
└── origin-sample       # Sample origin server (static files for testing)
```

### Service Breakdown

#### **Control Panel**
- REST API to manage users, CDNs, and snapshots
- Persists data in MongoDB
- Subscribes to health updates from services via NATS

#### **Edge**
- Receives client requests
- Caches static content (images, CSS, JS, fonts, video, audio)
- Proxies dynamic/non-cacheable requests to origin servers
- Stores cache on disk

#### **Mid**
- Syncs CDN configurations from control panel (via snapshot API)
- Caches responses locally (on disk + in-memory metadata)
- Publishes service health status via NATS
- Subscribes to snapshot updates from control panel

#### **Origin Sample**
- Simple static file server (images, JSON, video) for testing CDN flows

---

## 🚀 How to Run

### 1. Clone the repo

```bash
git clone https://github.com/AmirAghaee/go-cdn-stack.git
cd go-cdn-stack
```

### 2. Run Control Panel

Make sure MongoDB & NATS are running.

```bash
cd control-panel
go run main.go
```

### 3. Run Mid Service

```bash
cd mid
go run main.go
```

### 4. Run Edge Service

```bash
cd edge
go run main.go
```

### 5. Run Origin Sample

```bash
cd origin-sample
go run main.go
```

---

## ⚡ Development Notes

- Cache rules: Only `image/*`, `font/*`, `text/css`, `text/javascript`, `application/javascript`, `video/*`, and `audio/*` responses are cached.
- Non-GET requests are proxied directly to origin.
- Each cached item has metadata stored alongside the cached file (headers + expiry time).
- Mid-tier syncs CDNs from Control Panel at startup and also via NATS events.
- Health check messages are published by services and consumed by Control Panel.

---

## 🛠️ Tech Stack

- **Language:** Go 1.25+
- **Frameworks:** Gin, NATS, MongoDB driver
- **Persistence:** MongoDB (control panel), Disk-based cache (edge/mid)

---

## 📌 TODO

- [ ] Add Docker Compose for local development (MongoDB + NATS + services)
- [ ] Add unit/integration tests
- [ ] Implement cache invalidation via NATS
- [ ] Add rate limiting and logging middleware

---
