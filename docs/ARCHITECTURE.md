# Architecture & Refactoring Plan (SOLID & DRY)

Our goal is to evolve the `open-whats` daemon from a monolithic `main.go` file into a scalable, testable, and maintainable application by applying SOLID principles, DRY philosophy, and proper design patterns (primarily Hexagonal/Clean Architecture principles).

## 1. Directory Structure (Separation of Concerns)

We will reorganize the codebase using the standard Go project layout:

```text
open-whats/
├── cmd/
│   └── open-whats/
│       └── main.go           # Application entry point, dependency injection, and wiring.
├── internal/
│   ├── api/                  # HTTP Server, Handlers, routing.
│   ├── domain/               # Core business logic and interfaces (Ports).
│   ├── store/                # SQLite initialization and device management (Adapters).
│   └── whatsapp/             # whatsmeow client integration, event listeners (Adapters).
├── public/                   # Frontend assets (HTML, CSS, JS).
└── ARCHITECTURE.md           # This document.
```

## 2. SOLID Principles in Action

### S - Single Responsibility Principle (SRP)
Currently, `main.go` does database initialization, connects to WhatsApp, handles WhatsApp events, and spins up an HTTP server.
*   **Plan:** 
    *   `api` package will *only* handle HTTP traffic and payload decoding.
    *   `whatsapp` package will *only* handle the `whatsmeow` client lifecycle and event routing.
    *   `store` package will *only* handle SQLite connections.

### O - Open/Closed Principle (OCP)
The system should be open for extension (adding new message types like images, documents) but closed for modification.
*   **Plan:** Implement a Strategy Pattern for message sending and receiving. The `whatsapp` package will expose generic methods to register event handlers, allowing us to add new handlers for different WhatsApp events without modifying the core client code.

### L - Liskov Substitution Principle (LSP) & I - Interface Segregation Principle (ISP)
Clients should not be forced to depend on methods they do not use.
*   **Plan:** Define small, focused interfaces in the `domain` package. 

```go
package domain

import "context"

// MessageSender is a minimal interface for sending messages.
type MessageSender interface {
    SendMessage(ctx context.Context, jid string, text string) error
}

// EventDispatcher allows registering callbacks for events.
type EventDispatcher interface {
    OnMessageReceived(callback func(msg Message))
}
```

### D - Dependency Inversion Principle (DIP)
High-level modules (like `api`) should not depend on low-level modules (like `whatsmeow`). Both should depend on abstractions.
*   **Plan:** The `api.Server` will take a `domain.MessageSender` interface in its constructor. The `whatsapp.Client` will implement this interface. This breaks the hard coupling to `whatsmeow`, making the `api` easily testable with mock senders.

## 3. Design Patterns to Implement

1.  **Dependency Injection (DI):** Through constructor functions (e.g., `api.NewServer(sender domain.MessageSender)`). `main.go` acts as the DI container.
2.  **Adapter Pattern:** Wrap the `whatsmeow.Client` inside a custom `whatsapp.Client` struct so our application only talks to our interface, isolating us from third-party library changes.
3.  **Observer Pattern:** For WhatsApp events. `whatsmeow` already uses this internally, but we will abstract it so that different parts of our app can subscribe to specific events (e.g., text messages vs. connection status changes) via a custom `EventDispatcher`.

## 4. DRY Philosophy (Don't Repeat Yourself)

*   **Error Handling:** Consolidate API JSON error responses into a single utility function in the `api` package (`respondWithError(w, code, msg)`).
*   **JID Parsing:** Wrap JID parsing into a domain utility so we aren't repeating `types.ParseJID(req.JID)` across multiple API routes in the future.
*   **Logging:** Instantiate a shared, structured logger (e.g., `slog` or `zerolog`) and pass it down instead of scattering `fmt.Printf` and `panic()` everywhere.

## 5. Execution Steps for Refactoring

- [x] **Step 1: Create `internal/domain`** and define the core structs (`Message`) and interfaces (`MessageSender`).
- [x] **Step 2: Create `internal/store`** and migrate the `sqlstore.New` logic into a `store.InitDatabase(path string)` function.
- [x] **Step 3: Create `internal/whatsapp`** and build an adapter struct that embeds or holds `whatsmeow.Client`. Implement the `MessageSender` interface.
- [x] **Step 4: Create `internal/api`**, define the `Server` struct, and inject the `MessageSender`. Migrate the `/send` handler logic.
- [x] **Step 5: Rewrite `cmd/open-whats/main.go`** to simply wire the `store`, `whatsapp`, and `api` together, handle OS signals, and start the app gracefully.

By following this plan, adding WebSockets, handling images, or swapping the database will touch isolated components rather than forcing us to untangle spaghetti code.
