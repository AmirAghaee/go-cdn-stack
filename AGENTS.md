# AGENTS.md

This file defines the engineering rules for every directory in this repository.
Follow it when reviewing, changing, or adding code. More specific `AGENTS.md`
files may add local rules, but they must not weaken the dependency, security, or
correctness rules defined here.

## Product architecture

This repository is a CDN stack with two production services and one development
fixture:

- `control-panel`: management API and source of truth for CDN configuration.
- `edge`: data-plane service that serves cached content and proxies to origins.
- `origin-sample`: development-only origin server.
- `pkg`: narrowly scoped shared packages that are truly service-independent.

There is no mid-tier service. Do not introduce an extra network hop between edge
and origin unless an origin-shield requirement has been demonstrated with traffic
or capacity evidence.

The control plane and data plane must remain operationally separated:

- An edge must never call the control panel in the client-request path.
- An edge serves from an immutable, in-memory configuration snapshot.
- Edge configuration is refreshed at startup, periodically, and after a NATS
  notification.
- NATS notifications are hints to refresh; the control panel remains the source
  of truth.
- An edge must retain a last-known-good snapshot when the control panel or NATS
  is unavailable.
- Cache misses go directly from edge to the configured origin.
- Edge health is published directly to NATS.

## Clean Architecture dependency rule

Dependencies point inward. Inner layers never import outer layers.

```text
delivery/adapters -> application/use cases -> domain
infrastructure adapters implement application ports
composition root wires all dependencies
```

### Domain layer

The domain contains entities, value objects, invariants, and domain errors.

- Use plain Go. Do not import Gin, MongoDB, NATS, Viper, Prometheus, or filesystem
  packages into domain code.
- Do not put JSON, BSON, HTTP status, database, or transport concerns into core
  entities. Keep transport and persistence DTOs at their adapter boundaries.
- Represent important concepts explicitly, such as normalized CDN domains,
  validated origin URLs, cache TTLs, and snapshot versions.
- Keep entities valid after construction. Validation belongs in constructors or
  application commands, not scattered across handlers and repositories.
- Domain errors describe business outcomes and must be usable with `errors.Is`
  or `errors.As`.

### Application layer

The application layer implements use cases and owns the ports it consumes.

- Use cases accept `context.Context` plus command/query DTOs.
- Do not pass `*gin.Context`, MongoDB types, NATS messages, or HTTP requests into
  use cases.
- Define repository, publisher, clock, ID-generator, and external-client ports
  beside the use case that needs them.
- Keep orchestration and business decisions here; keep protocol translation out.
- Make transaction and event-publication ordering explicit.
- Return typed business errors. Do not return HTTP status codes from use cases.
- Constructors must receive dependencies explicitly. Avoid package-level mutable
  state and service locators.

### Adapters

Adapters translate between external systems and application ports.

- Inbound adapters include HTTP handlers and NATS subscribers.
- Outbound adapters include MongoDB repositories, NATS publishers, control-panel
  clients, filesystem snapshot stores, and origin HTTP clients.
- HTTP handlers bind and validate transport DTOs, call one use case, and map
  results to a stable response contract.
- Repositories perform persistence only. They must not contain HTTP behavior or
  business policy.
- Wrap external errors with operation context using `%w`; do not expose raw
  infrastructure errors to clients.
- Check MongoDB cursor errors, affected counts, duplicate-key errors, and context
  cancellation.

### HTTP handler and application service boundaries

A thin handler is defined by the decisions it owns, not by its line count. DTO
mapping and explicit error-to-response mapping may make a handler longer without
moving business logic into it.

- HTTP handlers own route registration, path/query/header extraction, request
  decoding, transport-level validation, application calls, HTTP status mapping,
  response DTO mapping, and response encoding.
- Transport-level validation checks whether the request can be understood, such
  as required JSON fields, valid JSON syntax, and syntactically valid email
  addresses.
- Application services or use cases own business validation and decisions, such
  as password policy, registration eligibility, uniqueness outcomes, credential
  verification, authorization policy, and orchestration across stores,
  publishers, token issuers, or other ports.
- Keep HTTP DTOs in the HTTP adapter. Do not add JSON tags to application or
  domain types merely to shorten response mapping code.
- A handler should call one application operation for a request. It must not
  reproduce business rules or directly coordinate multiple infrastructure
  adapters.
- Application operations must be transport-independent. They return results and
  typed errors, never HTTP status codes or framework response types.
- Define errors precisely enough for handlers to distinguish expected business
  outcomes, such as invalid credentials, not found, or conflict, from dependency
  failures and cancellation.
- Map expected typed errors to the stable API contract. Log unexpected errors
  internally with structured context and return a generic client-safe error; do
  not expose raw database, messaging, token, or hashing errors.
- Do not split every endpoint into its own service or use-case type by default. A
  cohesive feature service is appropriate while its operations share a clear
  purpose and dependencies. Split it when workflows, dependencies, transactions,
  authorization rules, ownership, or test setup diverge materially.
- Do not introduce an interface only to rename a concrete service. Add a small
  consumer-owned interface when it creates a real substitution or test boundary.

### Composition root

`main` or a dedicated bootstrap package is the only place that knows concrete
implementations for all layers.

- Load and validate configuration before constructing services.
- Wire interfaces to implementations explicitly.
- Keep business logic out of `main`.
- Start background workers with cancellable contexts.
- Shut down HTTP servers, NATS, MongoDB, tickers, and workers gracefully.

## Target service layout

Use this layout for new code and move existing code toward it when touching a
feature. Do not perform unrelated bulk moves.

```text
<service>/
  cmd/<service>/main.go
  internal/
    domain/
    application/
      command/
      query/
      port/
    adapter/
      in/http/
      in/nats/
      out/mongodb/
      out/nats/
      out/http/
      out/filesystem/
    platform/
      config/
      observability/
```

Small features do not need empty ceremonial packages. Create a boundary when it
enforces the dependency rule or makes behavior independently testable.

## Go conventions

- Use standard Go naming: `CDN`, `URL`, `HTTP`, `JWT`, `ID`, and `NATS`.
- Keep interfaces small and behavior-focused. Define them on the consumer side.
- Accept interfaces and normally return concrete types from constructors.
- Prefer value types for immutable domain data and pointers where identity,
  mutation, or optionality is meaningful.
- Always propagate request contexts. Do not replace them with
  `context.Background()` inside a request flow.
- Reuse configured `http.Client` and `http.Transport` instances. Never allocate a
  new client per request.
- Every external call needs a timeout or deadline.
- Close response bodies and check body-read, JSON encode/decode, file, cursor,
  and server-start errors.
- Avoid `panic` and `log.Fatal` outside unrecoverable startup validation.
- Do not start unbounded goroutines. Every worker needs ownership, cancellation,
  and bounded concurrency.
- Use structured logs with stable fields such as service, instance, operation,
  domain, request ID, cache status, duration, and error.
- Never log passwords, tokens, cookies, authorization headers, or secret values.
- Keep comments focused on why behavior exists, not on restating the code.
- Run `gofmt` on changed Go files and `go mod tidy` only in modules whose imports
  changed.

## Control-panel rules

- MongoDB is the source of truth for CDN configuration.
- Normalize domains before uniqueness checks and persistence. Enforce uniqueness
  with a MongoDB unique index; an application pre-check alone is insufficient.
- Validate origin scheme and host. Production origins must not resolve to
  forbidden/private destinations unless explicitly allowed by policy.
- Create, update, and delete operations must preserve every submitted field,
  including `cache_ttl` and `is_active`.
- Successful configuration mutations publish a refresh notification. A publish
  failure must not falsely report that an already-committed database mutation
  failed; periodic edge reconciliation is the recovery mechanism.
- List/snapshot endpoints must return an error when MongoDB fails. Never convert
  a repository failure into an apparently valid empty snapshot.
- Separate human administration authorization from edge service authentication.
  Do not give an edge an unrestricted administrator identity.
- Public registration must not create control-plane administrators in production.

## Edge and HTTP proxy rules

Correct HTTP semantics take priority over cache hit rate.

- Normalize request hosts, including case and optional ports, before CDN lookup.
- Reject unknown and inactive CDN configurations.
- Build upstream URLs with `net/url`; preserve path escaping and query strings.
- A cache key must include the normalized host and complete request URI plus all
  supported representation dimensions required by `Vary`.
- Do not cache authenticated or cookie-dependent responses unless an explicit,
  tested cache policy allows it.
- Respect `Cache-Control`, `Expires`, `Vary`, `ETag`, `Last-Modified`, and
  `Set-Cookie` semantics.
- Persist and replay the original status code and permitted end-to-end headers.
- Strip hop-by-hop headers in both directions.
- Append to trusted forwarding headers rather than trusting arbitrary client
  values. Set the upstream host according to documented origin behavior.
- Support streaming and range requests for large content. Do not load unbounded
  response bodies into memory.
- Bound cacheable object size, total disk use, concurrency, and upstream response
  time.
- Use request collapsing/singleflight so concurrent misses for one key do not
  stampede an origin.
- Hash cache keys with a stable digest for filenames. Do not use raw or hex-encoded
  URLs as filenames.
- Write cache bodies and metadata atomically. A cache write failure must normally
  degrade to an uncached successful response, not a client-facing 500.
- Treat missing or corrupt cache files as misses, remove bad metadata, and refetch.
- Ensure memory eviction and disk deletion have deliberate, consistent behavior.
- Configuration snapshot replacement must be atomic for readers. Never expose a
  partially updated domain map.
- Persist snapshots atomically and replace the in-memory last-known-good snapshot
  only after receiving a complete, valid control-panel response.

## Messaging and eventual consistency

- Give subjects stable, documented names and version message schemas when they
  evolve.
- Validate every inbound message before executing a use case.
- Assume messages can be duplicated, delayed, reordered, or missed.
- Subscribers must be idempotent.
- For events that must survive outages, use JetStream or another durable mechanism;
  plain NATS delivery alone is not durable.
- Event-triggered refresh and periodic reconciliation must be safe to run
  concurrently.
- Retry transient failures with bounded exponential backoff and jitter. Never stop
  a permanent worker after one transient failure.

## Configuration and security

- Environment variables configure deployments; `.env.example` documents every
  supported setting without containing real secrets.
- Production must fail startup when required secrets are missing and must not
  retain known development defaults.
- Control-panel and edge must agree on service-authentication configuration.
- Prefer scoped service credentials and key rotation over sharing the human JWT
  signing secret with edge nodes.
- Pin container and infrastructure image versions; do not deploy `latest`.
- Do not disable checksum verification for production builds.
- Expose only public service ports. MongoDB, NATS monitoring, metrics, and internal
  endpoints require network restrictions and authentication where appropriate.
- Validate and constrain configured origins to prevent SSRF and access to cloud
  metadata or internal control-plane services.

## Observability and operations

- Provide separate liveness and readiness endpoints.
- Readiness must reflect whether an edge has a usable configuration snapshot, not
  whether NATS happens to be connected at that instant.
- Metrics labels must have bounded cardinality. Do not use raw paths, query values,
  user IDs, or arbitrary hosts as labels without an allowlist.
- Measure request count, status, duration, bytes, cache hit/miss/bypass, origin
  latency, origin errors, cache size, evictions, snapshot age, and sync failures.
- Use UTC for machine timestamps.
- Handle SIGTERM with enough time for in-flight requests and workers to stop.

## Testing policy

Tests belong at the layer that owns the contract.

- Domain tests cover invariants and value-object normalization.
- Use-case tests use fakes for ports and cover success, business failure, external
  failure, cancellation, and idempotency.
- HTTP tests cover validation, status codes, response bodies, and error mapping.
- Repository integration tests cover unique indexes, not-found behavior, update
  counts, cursor failures, and MongoDB serialization.
- Edge tests cover query-aware cache keys, inactive CDNs, headers, status codes,
  cache directives, range behavior, stale/corrupt files, concurrent misses, and
  last-known-good configuration.
- Subscriber tests cover malformed, duplicate, and failed messages.
- Avoid real network calls in unit tests; use `httptest.Server` for HTTP adapter
  integration tests.
- Run concurrency-sensitive tests with `go test -race` on a supported 64-bit
  platform.

Minimum verification for a change:

```bash
cd control-panel && go test ./... && go vet ./...
cd ../edge && go test ./... && go vet ./...
cd ../pkg && go test ./... && go vet ./...
git diff --check
```

Also validate `docker compose config` and run focused integration tests when
Compose or cross-service behavior changes.

## Change workflow

1. Start from the named route, use case, configuration, or failure and trace the
   actual call path before editing.
2. Check `git status` and preserve unrelated user changes.
3. State the invariant or contract being changed.
4. Make the smallest coherent change that improves the architecture.
5. Add or update tests at the owning layer.
6. Format, test, vet, and inspect the final diff.
7. Report verified behavior separately from environment-dependent assumptions.

Do not mix unrelated refactors with behavior changes. Do not create abstraction
layers solely to rename existing code. When legacy code violates this guide,
improve the touched path incrementally and record any important remaining debt in
the handoff.

## Definition of done

A change is complete only when:

- The dependency rule is preserved or improved.
- Business behavior is expressed outside frameworks and infrastructure.
- Errors and cancellation are handled deliberately.
- Security boundaries and secrets are not weakened.
- Tests cover the changed contract and pass.
- Static analysis and formatting pass.
- Configuration and documentation match runtime behavior.
- No unrelated files or generated artifacts are included.
