# OmniSales Pro — Real-Time Enterprise Sales & Inventory System

An enterprise-grade Go backend and reactive Vanilla JavaScript frontend built with Hexagonal (Ports & Adapters) and Modular Clean Architecture, asynchronous background workers, atomic batch WebSocket notifications, and fine-grained DOM delta patching.

---

## 🏛️ Architecture Overview

```
go-real-time/
├── cmd/
│   └── server/
│       └── main.go                         # Dependency Injection wireup & graceful shutdown
├── config/
│   └── config.go                           # Environment-based configuration
├── internal/
│   ├── core/                               # Hexagonal Domain Core (Independent of DB/HTTP)
│   │   ├── domain/                         # Pure Business Entities & Events
│   │   │   ├── user.go                     # User & Roles (Admin, Manager, Cashier)
│   │   │   ├── product.go                  # Product & Stock business invariants
│   │   │   ├── order.go                    # Sales Order & Order Items domain logic
│   │   │   ├── audit.go                    # Audit log entity
│   │   │   └── events.go                   # Single & Batch change events definitions
│   │   ├── ports/                          # Inbound & Outbound Interfaces
│   │   │   ├── repositories.go             # Persistence interfaces (UserRepository, etc.)
│   │   │   ├── services.go                 # Use-case service interfaces
│   │   │   ├── eventbus.go                 # Pub/Sub broadcaster interface
│   │   │   └── token.go                    # JWT & Hasher interfaces
│   │   └── services/                       # Application Use Cases
│   │       ├── auth_service.go             # Login, Register, JWT verification
│   │       ├── product_service.go          # Catalog management with delta events
│   │       ├── order_service.go            # Sales transactions + atomic batch notifications
│   │       └── audit_service.go            # Non-blocking async audit queue
│   ├── adapters/                           # Driving & Driven Adapters
│   │   ├── primary/                        # Inbound (Driving)
│   │   │   ├── http/                       # REST Handlers, Middlewares, SPA Subpath server
│   │   │   └── websocket/                  # Concurrent Hub, Client read/write pumps, Auth
│   │   └── secondary/                      # Outbound (Driven)
│   │       ├── postgres/                   # PostgreSQL repositories & schema
│   │       ├── eventbus/                   # High-throughput in-memory pub/sub
│   │       └── security/                   # JWT & Bcrypt implementations
│   └── infra/                              # Cross-Cutting Infrastructure
│       ├── logger/                         # Structured JSON logger
│       ├── tracker/                        # Telemetry & performance metrics
│       └── workerpool/                     # Bounded goroutine worker pool
├── web/                                    # Vanilla JS Modular GUI Project
│   ├── index.html                          # SPA Entry
│   ├── css/                                # Centralized Design System
│   │   ├── variables.css                   # Tokens, colors, dark theme palette
│   │   ├── base.css                        # Reset, animations, typography
│   │   ├── components.css                  # Reusable components & danger states
│   │   └── pages.css                       # View layouts & POS grids
│   └── js/                                 # Modular ES6 JavaScript
│       ├── app.js                          # SPA Router & lifecycle
│       ├── config.js                       # Endpoint configurations
│       ├── state/
│       │   ├── store.js                    # Centralized reactive state store
│       │   └── actions.js                  # REST API actions
│       ├── ws/
│       │   ├── socket.js                   # Reconnecting WebSocket client
│       │   └── dispatcher.js               # Event router & batch unboxer
│       ├── engine/
│       │   └── patcher.js                  # Fine-grained DOM delta patcher
│       ├── components/                     # Reusable UI widgets (Modal, Toast, Navbar, Table)
│       └── views/                          # Login, Register, Products, Sales, Audit views
├── go.mod
└── README.md
```

---

## ⚡ Key Features

### 1. Hexagonal & Clean Architecture
- **Zero Core Dependency Leakage**: The domain layer (`internal/core/domain`) has zero dependencies on databases, HTTP libraries, or third-party frameworks.
- **Inbound/Outbound Ports**: Everything interacts through interfaces in `internal/core/ports`.

### 2. Real-Time WebSocket Change Events & Batching
- **Backend-Originated State Notifications**: All database modifications made by this backend emit structured change events (`ENTITY_CREATED`, `ENTITY_UPDATED`, `ENTITY_DELETED`, `STOCK_CHANGED`).
- **Organized Batch Transactions**: When multiple records change simultaneously (such as in a Sales Order where multiple product inventory stocks are decremented and an order with line items is created inside an atomic transaction), the backend packages all changes into a single organized `BATCH_TRANSACTION` packet rather than flooding clients with disconnected events.

### 3. Asynchronous Background Logging & Audit Engine
- Non-blocking execution: Audit logs are pushed to a bounded background `WorkerPool`, keeping HTTP API response latencies minimal while persisting full change histories.
- Real-time telemetry: Live metrics track active connections, requests, emitted events, and background worker completions.

### 4. Vanilla JavaScript Reactive GUI & Delta Patcher
- **Active Resource Registry**: The client tracks which resources are currently requested/rendered in the active view.
- **Targeted Delta DOM Updates**: If a product's stock changes, the patcher locates only the specific table row / card in the DOM and patches its stock badge with visual highlight pulses, without refetching the dataset.
- **Smart Deleted State**: If a product is deleted by another user, its row transforms into an **unclickable danger state** (`.row-deleted-danger`) with disabled actions and danger badges.
- **New Item Announcement**: When a new product is added by another user, the GUI displays an unobtrusive banner *"New item added in catalog. [Click to Refresh]"*, preserving the user's current pagination and scroll position.

### 5. Subpath Static File Hosting
- The Go backend hosts the vanilla frontend under the configurable subpath `/app/` with SPA routing fallbacks and proper MIME caching headers.
