# Smile Clean CRM

A modern CRM platform being migrated from **AdminJS + Node.js** to a modular architecture built with:

- **Astro** — frontend application shell, routing, layouts, SSR
- **Vue 3** — primary CRM/business UI islands
- **Svelte** — lightweight interactive islands/widgets
- **Go** — backend API and business services
- **PostgreSQL** — primary database
- **Tailwind CSS** — UI styling
- **TypeScript** — frontend type safety

> **Important:** React is intentionally NOT used in this project.

---

# 1. Project Goal

Smile Clean Thailand is a Bangkok cleaning services company (house/condo/deep cleaning, move in/out, after renovation, junk removal, office cleaning, aircon service). This repo is an end-to-end CRM for it, built with:

```text
AdminJS
   ↓
Node.js
   ↓
Sequelize
   ↓
PostgreSQL
```

The goal is to gradually migrate the system to:

```text
                     Browser
                        │
                        ▼
                  ┌───────────┐
                  │   Astro   │
                  │ App Shell │
                  └─────┬─────┘
                        │
              ┌─────────┴─────────┐
              │                   │
              ▼                   ▼
        ┌───────────┐       ┌───────────┐
        │    Vue    │       │  Svelte   │
        │  Islands  │       │  Islands  │
        └─────┬─────┘       └─────┬─────┘
              │                   │
              └─────────┬─────────┘
                        │
                        ▼
                  ┌───────────┐
                  │  Go API   │
                  └─────┬─────┘
                        │
                        ▼
                  ┌───────────┐
                  │ PostgreSQL │
                  └───────────┘
```

The migration must be incremental.

Do **not** delete the existing AdminJS/Node.js system until the new implementation has replaced its required functionality and has been verified.

---

# 2. Core Architecture

## 2.1 Astro

Astro is the main frontend framework and application shell.

Astro is responsible for:

- Page routing
- Layouts
- Navigation
- Sidebar
- Header
- Page composition
- Server-side rendering where appropriate
- Static rendering where appropriate
- SEO
- Loading Vue islands
- Loading Svelte islands

Astro should NOT become a giant client-side application.

Use Astro for page structure and composition.

Example:

```astro
---
import DashboardLayout from "../../layouts/DashboardLayout.astro";
import LeadTable from "../../components/vue/LeadTable.vue";
---

<DashboardLayout title="Leads">
  <LeadTable client:load />
</DashboardLayout>
```

---

# 3. Vue Architecture

Vue is the **primary interactive UI framework** for the CRM.

Use Vue for complex business interfaces such as:

- Data tables
- CRUD interfaces
- Forms
- Filters
- Search
- Pagination
- Lead management
- Customer management
- Booking management
- Cleaner management
- Service record management
- Payment management
- Complex dialogs
- Business workflows

Recommended Vue components:

```text
src/components/vue/

├── leads/
│   ├── LeadTable.vue
│   ├── LeadForm.vue
│   ├── LeadDetails.vue
│   └── LeadFilters.vue
│
├── customers/
│   ├── CustomerTable.vue
│   └── CustomerForm.vue
│
├── bookings/
│   ├── BookingTable.vue
│   └── BookingForm.vue
│
├── cleaners/
│   ├── CleanerTable.vue
│   └── CleanerForm.vue
│
├── service-records/
│   ├── ServiceRecordTable.vue
│   └── ServiceRecordForm.vue
│
└── payments/
    ├── PaymentTable.vue
    └── PaymentForm.vue
```

Vue should be treated as the main CRM UI framework.

---

# 4. Svelte Architecture

Svelte is used for **small, isolated, lightweight interactive widgets**.

Use Svelte for:

- Notification bell
- Toast notifications
- Command palette
- Small dropdowns
- Lightweight UI interactions
- Small widgets
- Activity indicators
- Simple interactive components

Example:

```text
src/components/svelte/

├── NotificationBell.svelte
├── Toast.svelte
├── CommandPalette.svelte
└── StatusIndicator.svelte
```

Do not use Svelte for large CRM modules that are already implemented in Vue.

---

# 5. Important Component Rule

Vue and Svelte must remain independent islands.

Do NOT create:

```text
Vue
 └── Svelte
      └── Vue
```

or:

```text
Svelte
 └── Vue
```

Instead:

```text
Astro
│
├── Vue Island
│   └── LeadTable
│
├── Vue Island
│   └── LeadForm
│
├── Svelte Island
│   └── NotificationBell
│
└── Svelte Island
    └── Toast
```

Astro is the composition layer.

---

# 6. Backend Architecture

The backend is written in Go.

Recommended architecture:

```text
apps/api/

├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── auth/
│   ├── users/
│   ├── leads/
│   ├── customers/
│   ├── cleaners/
│   ├── bookings/
│   ├── service_records/
│   ├── payments/
│   ├── activities/
│   ├── notifications/
│   └── reports/
│
├── migrations/
│
├── pkg/
│   ├── database/
│   ├── logger/
│   └── response/
│
├── config/
│
├── go.mod
└── go.sum
```

---

# 7. Go Domain Architecture

Each business domain should be isolated.

Example:

```text
internal/leads/

├── handler.go
├── service.go
├── repository.go
├── model.go
├── dto.go
└── routes.go
```

Responsibilities:

### Handler

Handles:

- HTTP request
- Request validation
- Authentication context
- Calling service
- HTTP response

### Service

Handles:

- Business rules
- Business workflows
- Validation that depends on business logic
- Transactions when necessary

### Repository

Handles:

- Database queries
- PostgreSQL interaction
- Persistence

### Model

Represents domain/database entities.

### DTO

Represents API request/response structures.

---

# 8. Backend Request Flow

All business requests should follow:

```text
HTTP Request
      ↓
Middleware
      ↓
Handler
      ↓
Service
      ↓
Repository
      ↓
PostgreSQL
```

Do NOT put business logic directly inside HTTP handlers.

Bad:

```go
func createLead(w http.ResponseWriter, r *http.Request) {
    // validation
    // business rules
    // SQL
    // response
}
```

Preferred:

```go
func (h *Handler) CreateLead(w http.ResponseWriter, r *http.Request) {
    input := CreateLeadRequest{}

    // decode / validate

    lead, err := h.service.CreateLead(ctx, input)

    // response
}
```

---

# 9. API Design

Use versioned REST APIs.

Base URL:

```text
/api/v1
```

Examples:

## Authentication

```text
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh
GET    /api/v1/auth/me
```

## Leads

```text
GET    /api/v1/leads
GET    /api/v1/leads/:id
POST   /api/v1/leads
PATCH  /api/v1/leads/:id
DELETE /api/v1/leads/:id
```

## Customers

```text
GET    /api/v1/customers
GET    /api/v1/customers/:id
POST   /api/v1/customers
PATCH  /api/v1/customers/:id
DELETE /api/v1/customers/:id
```

## Cleaners

```text
GET    /api/v1/cleaners
GET    /api/v1/cleaners/:id
POST   /api/v1/cleaners
PATCH  /api/v1/cleaners/:id
DELETE /api/v1/cleaners/:id
```

## Bookings

```text
GET    /api/v1/bookings
GET    /api/v1/bookings/:id
POST   /api/v1/bookings
PATCH  /api/v1/bookings/:id
```

## Service Records

```text
GET    /api/v1/service-records
GET    /api/v1/service-records/:id
POST   /api/v1/service-records
PATCH  /api/v1/service-records/:id
```

## Payments

```text
GET    /api/v1/payments
GET    /api/v1/payments/:id
POST   /api/v1/payments
PATCH  /api/v1/payments/:id
```

---

# 10. Database

PostgreSQL remains the primary database.

The existing PostgreSQL database should initially be preserved during migration.

Do NOT migrate the database just because the backend changes from Node.js to Go.

Initial strategy:

```text
Existing PostgreSQL
       │
       ├───────────────┐
       │               │
       ▼               ▼
   Node/AdminJS       Go API
                       │
                       ▼
                     Astro
```

Once the Go implementation is stable, the old Node/AdminJS system can be retired.

---

# 11. Recommended Go Database Stack

Preferred:

```text
PostgreSQL
   +
pgx
   +
sqlc
   +
golang-migrate
```

Responsibilities:

- `pgx` → PostgreSQL driver
- `sqlc` → type-safe generated database code
- `golang-migrate` → database migrations

Avoid introducing an ORM unless there is a strong project-specific reason.

---

# 12. Frontend API Layer

Do not call the backend directly from every Vue/Svelte component.

Avoid:

```ts
fetch("http://localhost:8080/api/v1/leads");
```

inside every component.

Instead create:

```text
src/lib/
├── api.ts
├── auth.ts
├── leads.ts
├── customers.ts
├── cleaners.ts
├── bookings.ts
└── service-records.ts
```

Example:

```ts
const API_URL = import.meta.env.PUBLIC_API_URL ?? "http://localhost:8080";

export async function getLeads() {
  const response = await fetch(`${API_URL}/api/v1/leads`);

  if (!response.ok) {
    throw new Error("Failed to fetch leads");
  }

  return response.json();
}
```

Vue components should consume API functions instead of knowing backend URLs.

---

# 13. Authentication

The CRM requires authentication and role-based authorization.

Initial roles:

```text
SUPER_ADMIN
ADMIN
MANAGER
DISPATCH
ACCOUNTANT
CLEANER
```

Permissions should be granular.

Examples:

```text
leads.read
leads.create
leads.update
leads.delete

customers.read
customers.create
customers.update
customers.delete

bookings.read
bookings.create
bookings.update

cleaners.read
cleaners.create
cleaners.update

service_records.read
service_records.create
service_records.update

payments.read
payments.create
payments.update
```

Do not rely only on frontend permission checks.

Authorization must also be enforced by the Go API.

---

# 14. Security Rules

Never trust the frontend.

The Go backend must validate:

- Authentication
- Authorization
- Input validation
- Ownership/access rules
- File upload restrictions
- Resource IDs
- Business rules

Never expose secrets in Astro client-side code.

Environment variables intended for the browser must use Astro's public environment variable convention.

Never put:

```text
DATABASE_URL
JWT_SECRET
PRIVATE_API_KEY
```

into public frontend variables.

---

# 15. Astro Frontend Structure

Recommended:

```text
apps/web/

├── src/
│
│   ├── components/
│   │   ├── vue/
│   │   │   ├── leads/
│   │   │   ├── customers/
│   │   │   ├── bookings/
│   │   │   ├── cleaners/
│   │   │   ├── service-records/
│   │   │   └── payments/
│   │   │
│   │   └── svelte/
│   │
│   ├── layouts/
│   │   ├── DashboardLayout.astro
│   │   └── AuthLayout.astro
│   │
│   ├── pages/
│   │   ├── index.astro
│   │   ├── login.astro
│   │   ├── dashboard.astro
│   │   ├── leads/
│   │   ├── customers/
│   │   ├── bookings/
│   │   ├── cleaners/
│   │   ├── service-records/
│   │   ├── payments/
│   │   └── settings/
│   │
│   ├── lib/
│   │   ├── api.ts
│   │   ├── auth.ts
│   │   └── utils.ts
│   │
│   └── styles/
│       └── global.css
│
├── astro.config.mjs
├── package.json
└── tsconfig.json
```

---

# 16. Dashboard Layout

Astro should own the global CRM layout.

Example:

```text
┌──────────────────────────────────────────────┐
│ Header                         Notifications │
├───────────────┬──────────────────────────────┤
│               │                              │
│ Dashboard     │                              │
│ Leads         │        Page Content          │
│ Customers     │                              │
│ Bookings      │                              │
│ Cleaners      │                              │
│ Service Recs  │                              │
│ Payments      │                              │
│ Reports       │                              │
│ Settings      │                              │
│               │                              │
└───────────────┴──────────────────────────────┘
```

Sidebar and layout should remain Astro components unless they require significant interactivity.

---

# 17. Astro Island Rules

Use the smallest possible island.

Examples:

```astro
<LeadTable client:load />
```

Use client-side hydration only where needed.

Prefer:

```text
Astro HTML
   +
small Vue/Svelte island
```

instead of:

```text
Entire page
   ↓
Vue application
```

The goal is to preserve Astro's performance benefits.

---

# 18. Vue Rules

Use Vue for:

- Complex forms
- Tables
- CRUD
- Filters
- Pagination
- Business workflows
- Complex client-side state

Keep Vue components focused.

Avoid creating one giant:

```text
CRM.vue
```

Instead use:

```text
LeadTable.vue
LeadFilters.vue
LeadForm.vue
LeadDetails.vue
```

---

# 19. Svelte Rules

Use Svelte for small isolated interactions.

Good:

```text
NotificationBell
Toast
CommandPalette
Small dropdown
Status indicator
```

Avoid:

```text
Entire CRM Dashboard in Svelte
```

when the main CRM application is already using Vue.

---

# 20. Styling

Use Tailwind CSS.

Primary UI should be consistent.

Recommended principles:

- Consistent spacing
- Consistent typography
- Accessible contrast
- Responsive layout
- Reusable UI patterns
- Avoid unnecessary custom CSS
- Avoid duplicated utility classes where reusable components are more appropriate

---

# 21. Migration Strategy

Migration must be incremental.

## Phase 0 — Existing System

```text
AdminJS
   ↓
Node.js
   ↓
Sequelize
   ↓
PostgreSQL
```

Keep this system working.

---

## Phase 1 — Frontend Foundation

Build:

```text
Astro
Vue
Svelte
Tailwind
TypeScript
```

Create:

- Dashboard layout
- Sidebar
- Header
- Authentication page
- Basic routing

---

## Phase 2 — Go API

Build:

```text
Go
Chi
pgx
PostgreSQL
```

Start with:

```text
GET /api/v1/leads
```

Then implement:

```text
POST
PATCH
DELETE
```

---

## Phase 3 — Leads Migration

Replace AdminJS Leads UI with:

```text
Astro
 +
Vue LeadTable
 +
Vue LeadForm
 +
Go Leads API
```

Keep the existing Node/AdminJS implementation available as fallback.

---

## Phase 4 — Customers

Migrate:

```text
Customers
Customer details
Customer property
Customer status
```

---

## Phase 5 — Cleaners

Migrate:

```text
Cleaners
Cleaner availability
Cleaner assignments
Cleaner notes
```

---

## Phase 6 — Bookings

Implement:

```text
Booking schedule
Service type
Assignment
Booking status
Booking notes
```

---

## Phase 7 — Service Records

Implement:

```text
Service record
Rating
Completion status
Completed at
```

---

## Phase 7 — Payments

Implement:

```text
Payment records
Payment status
Invoices
Transactions
Payment history
```

---

## Phase 8 — Authentication & RBAC

Implement:

```text
Login
Logout
Session
Roles
Permissions
Access control
```

---

## Phase 9 — Notifications

Use Svelte for:

```text
NotificationBell
Toast
Real-time notification UI
```

Go can later provide:

```text
WebSocket
SSE
```

if real-time functionality is required.

---

## Phase 10 — Remove AdminJS

Only after all required modules are stable:

```text
AdminJS
   ↓
REMOVE

Node.js
   ↓
REMOVE
```

Final:

```text
Astro
 ↓
Vue / Svelte
 ↓
Go
 ↓
PostgreSQL
```

---

# 22. Development Rules for OpenCode

When modifying this project, follow these rules.

## Rule 1 — No React

Do not install or introduce:

```text
react
react-dom
@astrojs/react
```

unless explicitly requested.

---

## Rule 2 — Vue is the primary CRM UI framework

Complex CRM UI should use Vue.

---

## Rule 3 — Svelte is for isolated widgets

Do not move large CRM modules into Svelte without explicit architectural approval.

---

## Rule 4 — Astro owns composition

Use Astro for:

- Routing
- Layout
- Page composition
- Static/SSR content

---

## Rule 5 — Go owns business logic

Business rules must live in Go.

Do not implement important business rules only in frontend code.

---

## Rule 6 — PostgreSQL is the source of truth

Do not introduce another database without explicit approval.

---

## Rule 7 — Do not rewrite everything

Prefer incremental migration.

Existing functionality must remain usable until its replacement is verified.

---

## Rule 8 — API versioning

All APIs must use:

```text
/api/v1/...
```

---

## Rule 9 — TypeScript

Avoid:

```ts
any;
```

unless absolutely necessary.

Prefer explicit interfaces/types.

---

## Rule 10 — Security

Never expose:

- Database credentials
- JWT secrets
- Private API keys
- Internal service credentials

to the browser.

---

# 23. Local Development

Frontend:

```bash
cd apps/web
npm install
npm run dev
```

Expected:

```text
http://localhost:4321
```

Backend:

```bash
cd apps/api
go run ./cmd/api
```

Expected:

```text
http://localhost:8080
```

API example:

```text
http://localhost:8080/api/v1/leads
```

---

# 24. Environment Variables

Frontend:

```env
PUBLIC_API_URL=http://localhost:8080
```

Backend:

```env
APP_ENV=development
PORT=8080

DATABASE_URL=postgres://user:password@localhost:5432/odysight_crm
```

Production secrets must be provided through the deployment environment.

Do not commit `.env` files containing secrets.

---

# 25. Docker Architecture

Production target:

```text
                    Internet
                       │
                       ▼
                 Reverse Proxy
                       │
             ┌─────────┴─────────┐
             │                   │
             ▼                   ▼
           Astro                Go API
             │                   │
             └─────────┬─────────┘
                       │
                       ▼
                  PostgreSQL
```

Potential services:

```text
docker-compose.yml

services:

  web:
    Astro

  api:
    Go

  postgres:
    PostgreSQL
```

---

# 26. Current Technology Decision

The current architectural decision is:

| Layer                  | Technology     |
| ---------------------- | -------------- |
| Frontend shell         | Astro          |
| Primary interactive UI | Vue 3          |
| Lightweight islands    | Svelte         |
| Language               | TypeScript     |
| Styling                | Tailwind CSS   |
| Backend                | Go             |
| HTTP Router            | Chi            |
| Database               | PostgreSQL     |
| PostgreSQL Driver      | pgx            |
| SQL Generation         | sqlc           |
| Migrations             | golang-migrate |
| API Style              | REST           |
| API Version            | v1             |
| Authentication         | Go backend     |
| Deployment             | Docker         |
| React                  | **Not used**   |

---

# 27. Current Migration Status

## Frontend

- [x] Astro project created
- [x] Vue dependency installed
- [x] Vue integration configured
- [x] Svelte integration installed
- [x] Tailwind configured
- [x] Dashboard layout
- [x] Sidebar
- [x] Header
- [x] Authentication UI
- [x] Leads UI
- [x] Customers UI
- [x] Bookings UI
- [x] Cleaners UI
- [x] Service Records UI
- [x] Payments UI

## Backend

- [x] Go project initialized
- [x] HTTP server
- [x] PostgreSQL connection
- [x] Database migrations
- [x] Leads API
- [x] Customers API
- [x] Cleaners API
- [x] Bookings API
- [x] Service Records API
- [x] Payments API
- [x] Authentication
- [x] RBAC

## Migration

- [ ] Analyze existing AdminJS resources
- [ ] Analyze existing Sequelize models
- [ ] Map existing database schema
- [ ] Migrate Leads
- [ ] Migrate Customers
- [ ] Migrate Cleaners
- [ ] Migrate Bookings
- [ ] Migrate Service Records
- [ ] Migrate Payments
- [ ] Verify data integrity
- [ ] Remove AdminJS
- [ ] Remove Node.js backend

---

# 28. Development Priority

OpenCode should work in this order:

```text
1. Frontend foundation
       ↓
2. Go API foundation
       ↓
3. PostgreSQL connection
       ↓
4. Leads
       ↓
5. Customers
       ↓
6. Cleaners
       ↓
7. Bookings
       ↓
8. Service Records
       ↓
9. Payments
       ↓
10. Authentication
       ↓
11. RBAC
       ↓
12. Reports
       ↓
13. AdminJS removal
```

Do not jump directly to advanced features before the foundation is stable.

---

# 29. Definition of Done

A migrated module is considered complete only when:

- Frontend UI works
- Vue/Svelte island is properly isolated
- Go API is implemented
- API validation exists
- Authorization exists where required
- PostgreSQL persistence works
- Error handling exists
- Loading states exist
- Empty states exist
- Frontend does not contain business logic that belongs in backend
- Existing AdminJS functionality has been verified against the new implementation
- No React dependency has been introduced

---

# 30. Final Architecture

The target architecture is:

```text
                         SMILE CLEAN CRM
                              │
             ┌────────────────┴────────────────┐
             │                                 │
             ▼                                 ▼
       ┌─────────────┐                   ┌─────────────┐
       │    Astro    │                   │    Go API   │
       │             │                   │             │
       │ Routing     │◄──── REST ──────►│ Auth        │
       │ Layout      │                   │ Business    │
       │ SSR         │                   │ Services    │
       │ Pages       │                   │ Validation  │
       └──────┬──────┘                   └──────┬──────┘
              │                                 │
       ┌──────┴──────┐                          │
       │             │                          │
       ▼             ▼                          ▼
    ┌──────┐     ┌───────┐                ┌───────────┐
    │ Vue  │     │Svelte │                │PostgreSQL │
    │      │     │       │                │           │
    │ CRM  │     │Widgets│                │ Source of │
    │ UI   │     │       │                │   Truth   │
    └──────┘     └───────┘                └───────────┘
```

## Architectural Principle

> **Astro composes. Vue manages complex CRM interactions. Svelte handles small interactive islands. Go owns business logic. PostgreSQL owns persistent data.**

This separation should remain the guiding principle throughout the migration.
