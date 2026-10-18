# Event Booking System API

A production-grade RESTful API for event management and ticket booking, built with Go's standard library and clean architecture principles. Supports two user roles (Organizer and Customer), JWT authentication, async background job processing for email notifications, TTL caching, and rate limiting.

## Architecture & Design Decisions

### Layered Architecture (Separation of Concerns)

```
┌──────────────────────────────────────────────┐
│                  HTTP Layer                   │
│  router/ → middleware/ → handler/             │
├──────────────────────────────────────────────┤
│               Service Layer                   │
│  service/ (business logic, validation)        │
├──────────────────────────────────────────────┤
│             Infrastructure Layer              │
│  memory/ │ cache/ │ worker/ │ email/          │
├──────────────────────────────────────────────┤
│                Domain Layer                   │
│  domain/ (models, errors, no dependencies)    │
└──────────────────────────────────────────────┘
```

Each layer depends only on the layer below it. The domain layer has zero external dependencies.

### SOLID Principles Applied

| Principle | Where |
| --- | --- |
| **Single Responsibility** | Each package owns exactly one concern: `handler/` = HTTP serialization, `service/` = business rules, `memory/` = data access, `worker/` = async jobs, `email/` = mail delivery |
| **Open/Closed** | New repository backends (PostgreSQL, MySQL) implement `repository.EventRepository` without modifying services. New email providers implement `email.Mailer` without touching the worker |
| **Liskov Substitution** | `memory.EventRepo` is substitutable for any `repository.EventRepository` implementation. `email.NoopMailer` substitutes for `email.SMTPMailer` in tests |
| **Interface Segregation** | Repository interfaces are split per aggregate: `UserRepository`, `EventRepository`, `BookingRepository` — services depend only on what they need |
| **Dependency Inversion** | Services accept interfaces (`repository.EventRepository`, `email.Mailer`), never concrete types. `main.go` wires concrete implementations |

### Design Patterns Used

| Pattern | Implementation | Purpose |
| --- | --- | --- |
| **Repository** | `repository/interfaces.go` + `memory/repository.go` | Abstracts data access; swap in-memory for SQL without changing services |
| **Strategy** | `email.Mailer` interface with `SMTPMailer` and `NoopMailer` | Pluggable email providers; runtime selection based on config |
| **Producer-Consumer** | `worker.Pool` with bounded channel | Async background job processing with back-pressure |
| **Decorator** | `cache.EventCache` wrapping repository reads | Transparent TTL caching for event lookups |
| **Middleware Chain** | `middleware.Chain()` composing handlers | Layered cross-cutting concerns (auth, CORS, logging, rate-limit, recovery) |
| **Singleton** | `config.Get()` with `sync.Once` | Single configuration instance across the application |
| **Factory** | `main.go` composition root | Assembles all dependencies and wires the object graph |

### OOP Concepts

- **Encapsulation**: Repository internal state (`sync.RWMutex`, maps) is unexported; access only through methods
- **Polymorphism**: `email.Mailer` interface allows `SMTPMailer`/`NoopMailer` to be used interchangeably
- **Composition**: `EventService` composes `EventRepository`, `BookingRepository`, `EventCache`, and `Pool` via dependency injection
- **Abstraction**: Domain models define the ubiquitous language; infrastructure details are hidden behind interfaces

## API Routes

| Method | Path | Access | Description |
| --- | --- | --- | --- |
| GET | `/health` | Public | Health check with worker pool metrics |
| POST | `/api/auth/register` | Public | Register a new user (Organizer or Customer) |
| POST | `/api/auth/login` | Public | Login and receive JWT token |
| GET | `/api/events` | Public | List events with pagination (`?offset=0&limit=20`) |
| GET | `/api/events/{id}` | Public | Get event details (cached) |
| POST | `/api/events` | Organizer | Create a new event |
| PATCH | `/api/events/{id}` | Organizer (owner) | Update event; triggers async notifications to all booked customers |
| POST | `/api/events/{id}/bookings` | Customer | Book tickets with atomic capacity check; triggers async confirmation email |
| GET | `/api/bookings/me` | Customer | List authenticated customer's bookings |

## Authentication

JWT-based authentication with bcrypt password hashing:
1. Register via `POST /api/auth/register` with `role: "ORGANIZER"` or `"CUSTOMER"`
2. Login via `POST /api/auth/login` to get a JWT token
3. Include `Authorization: Bearer <token>` header on protected routes

## Background Tasks

### Task 1: Booking Confirmation Email
- **Trigger**: Customer books tickets (`POST /api/events/{id}/bookings`)
- **Processing**: Job dispatched to worker pool channel → worker picks up → sends HTML email via SMTP
- **Email Content**: Booking ID, event title, venue, date, quantity

### Task 2: Event Update Notification
- **Trigger**: Organizer updates an event (`PATCH /api/events/{id}`)
- **Processing**: All bookings for the event are fetched → individual notification jobs dispatched to worker pool → each worker sends email
- **Email Content**: Event title, list of changes made

Both tasks use a **bounded channel-based worker pool** (Producer-Consumer pattern) with configurable pool size and queue depth. The pool provides back-pressure — when the queue is full, jobs are dropped with a log warning rather than blocking the HTTP handler.

## Middleware Stack

Applied in this order (outermost first):

1. **CORS** — Permissive cross-origin headers
2. **Recovery** — Catches panics, returns 500 JSON, logs stack trace
3. **Logging** — Logs method, path, status, duration per request
4. **Rate Limiting** — Per-IP token bucket algorithm (configurable RPS and burst)
5. **JWT Auth** (protected routes) — Validates Bearer token, injects claims into context
6. **RBAC** (role-specific routes) — Checks user role matches required role

## Run Locally

```bash
cp .env.example .env
# Edit .env with your SMTP credentials (Gmail App Password, Resend, SendGrid, etc.)
go run .
```

The server listens on `PORT` (default 8080).

### Example Workflow

```bash
# Register an organizer
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"org@example.com","password":"secret123","name":"Alice","role":"ORGANIZER"}'

# Create an event (use token from registration response)
curl -X POST http://localhost:8080/api/events \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <organizer-token>' \
  -d '{"title":"Go Conf","description":"Annual meetup","venue":"Hall A","starts_at":"2026-12-01T18:00:00Z","capacity":500}'

# Register a customer
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"cust@example.com","password":"secret123","name":"Bob","role":"CUSTOMER"}'

# Book tickets (triggers async confirmation email)
curl -X POST http://localhost:8080/api/events/evt_1/bookings \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <customer-token>' \
  -d '{"quantity":3}'

# Update event (triggers async notification to all booked customers)
curl -X PATCH http://localhost:8080/api/events/evt_1 \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <organizer-token>' \
  -d '{"venue":"New Hall B"}'
```

## Stress Test

Run the built-in stress test tool:

```bash
# Start the server first, then in another terminal:
BASE_URL=http://localhost:8080 go run scripts/stresstest.go
```

### Results (local, Apple Silicon, in-memory store)

| Scenario | Concurrency | Success | Avg Latency | Max Latency |
| --- | --- | --- | --- | --- |
| Booking | 10 | 100% | <1ms | 1ms |
| Booking | 50 | 100% | 2ms | 4ms |
| Booking | 100 | 100% | 6ms | 8ms |
| Booking | 200 | 100% | 9ms | 13ms |
| Booking | 500 | 100% | 13ms | 24ms |
| Booking | 1000 | 100% | 27ms | 40ms |
| Read | 2000 | 100% | 58ms | 82ms |
| Mixed (70R/30W) | 500 | 100% | 13ms | 21ms |

The system handles 1000 concurrent bookings with zero drops and sub-50ms latency. Performance degrades gradually due to mutex contention in the in-memory store.

### Optimization Strategies

1. **Sharded locks**: Replace single `sync.RWMutex` with per-event locks to eliminate cross-event contention
2. **Connection pooling**: When migrating to SQL, use bounded connection pools with prepared statements
3. **Read replicas**: Separate read path from write path; cache serves most reads
4. **Batch notification**: Aggregate update notifications into a single batch email instead of N individual sends
5. **Horizontal scaling**: Stateless HTTP layer + shared PostgreSQL + Redis-backed job queue enables multi-instance deployment

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | 8080 | HTTP listen port |
| `JWT_SECRET` | (required) | Secret key for JWT signing |
| `JWT_EXPIRY` | 24h | Token expiry duration |
| `SMTP_HOST` | — | SMTP server hostname |
| `SMTP_PORT` | 587 | SMTP server port |
| `SMTP_USERNAME` | — | SMTP auth username |
| `SMTP_PASSWORD` | — | SMTP auth password |
| `EMAIL_FROM` | — | Sender email address |
| `WORKER_POOL_SIZE` | 5 | Number of background worker goroutines |
| `JOB_QUEUE_SIZE` | 100 | Buffered channel capacity for job queue |
| `CACHE_TTL` | 30s | Event cache time-to-live |
| `RATE_LIMIT_RPS` | 100 | Rate limiter: requests per second per IP |
| `RATE_LIMIT_BURST` | 200 | Rate limiter: burst capacity per IP |

## Deploy

### Render (recommended)

1. Push to GitHub
2. Connect repository on [Render](https://render.com)
3. Render detects `render.yaml` and deploys automatically
4. Set SMTP environment variables in Render dashboard

### Docker

```bash
docker build -t event-booking-system .
docker run -p 8080:8080 -e JWT_SECRET=your-secret event-booking-system
```

## Project Structure

```
├── main.go                          # Composition root (Factory pattern)
├── Dockerfile                       # Multi-stage Docker build
├── render.yaml                      # Render deployment config
├── scripts/
│   └── stresstest.go                # Stress test tool
└── internal/
    ├── config/config.go             # Singleton configuration
    ├── domain/
    │   ├── models.go                # User, Event, Booking aggregates
    │   └── errors.go                # Domain error types
    ├── repository/
    │   └── interfaces.go            # ISP-compliant repository contracts
    ├── memory/
    │   └── repository.go            # In-memory repository (thread-safe)
    ├── service/
    │   ├── auth_service.go          # Registration, login, JWT
    │   ├── event_service.go         # Event CRUD + async notifications
    │   ├── booking_service.go       # Ticket booking + async confirmation
    │   └── helpers.go               # Time parsing utilities
    ├── handler/
    │   ├── response.go              # Standardized JSON response helpers
    │   ├── auth_handler.go          # Auth HTTP handlers
    │   ├── event_handler.go         # Event HTTP handlers
    │   ├── booking_handler.go       # Booking HTTP handlers
    │   └── health_handler.go        # Health check with metrics
    ├── router/
    │   └── router.go                # Route registration + middleware wiring
    ├── middleware/
    │   ├── auth.go                  # JWT validation + RBAC
    │   ├── cors.go                  # CORS headers
    │   ├── logging.go               # Request logging
    │   ├── ratelimit.go             # Token bucket rate limiter
    │   ├── recovery.go              # Panic recovery
    │   └── chain.go                 # Middleware composition utility
    ├── worker/
    │   ├── pool.go                  # Bounded worker pool (Producer-Consumer)
    │   └── email_handler.go         # Email job processor
    ├── cache/
    │   └── events.go                # TTL cache (Decorator pattern)
    └── email/
        └── mailer.go                # SMTP mailer + Noop strategy
```

## Tech Stack

- **Language**: Go 1.23 (standard library `net/http` router, no web framework)
- **Auth**: JWT (`github.com/golang-jwt/jwt/v5`) + bcrypt (`golang.org/x/crypto`)
- **Email**: Go `net/smtp` (works with Gmail, SendGrid, Resend, Mailgun, etc.)
- **Storage**: In-memory with `sync.RWMutex` (production: swap repository interface for PostgreSQL)
- **Job Queue**: Channel-based bounded worker pool (production: swap for Redis-backed queue)
- **Deployment**: Docker, Render, Heroku, Railway, Fly.io
