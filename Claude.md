You are working on the Smile Clean CRM codebase.

Before modifying anything, perform a full read-only analysis of the existing architecture, database schema/models, Go API, frontend, authentication/RBAC, bookings, invoices, customers, services, attendance, and existing recurring-booking logic.

IMPORTANT BUSINESS REQUIREMENT:

Smile Clean serves BOTH:

1. One-time / daily customers

   * A customer may book cleaning for only one day.
   * They may never sign a contract.
   * They may make multiple independent bookings over time.
   * Example: a restaurant requests 5 cleaners for one day.

2. Recurring / commercial customers

   * Restaurants, condos, factories, malls, offices, etc.
   * They may have weekly/monthly recurring services.
   * Some may have formal monthly/annual contracts.
   * Some recurring customers may NOT have a formal contract.

Therefore, DO NOT make Contract mandatory for every customer or booking.

The architecture must support:

Customer
→ Sites
→ Bookings
→ Optional Contract
→ Optional Recurring Schedule
→ Invoices
→ Checklists / Proof of Work

The core booking workflow must continue to work without a Contract.

---

## PHASE 1 — AUDIT FIRST

Inspect the current codebase and identify:

* Customer model/entity
* Customer address/location model
* Booking model/entity
* Recurring booking implementation
* Invoice model/entity
* Quote/quotation functionality
* Services
* Staff/attendance
* Existing customer portal
* RBAC/permissions
* Existing migrations
* Existing API routes
* Existing frontend pages/components
* Existing AdminJS resources if applicable

Do NOT immediately modify code.

Produce a short architecture assessment first.

Specifically answer:

1. What currently represents a customer's address/site?
2. Can one customer currently have multiple locations?
3. How are one-time bookings represented?
4. How are recurring bookings represented?
5. How are invoices linked to bookings/customers?
6. Is there already any concept similar to Contract?
7. Where should Site logically live in the existing architecture?
8. What existing functionality could break if Customer → Site is introduced?

---

## PHASE 2 — DESIGN THE DOMAIN MODEL

Design the system around the following principles.

### CUSTOMER

Customer represents the commercial relationship/customer.

A customer can have:

* One site
* Multiple sites
* No formal contract
* One contract
* Multiple contracts over time

Do not force every customer to have a contract.

### SITE

Introduce a first-class Site entity if the current architecture does not already have an equivalent.

Conceptually:

Customer
└── Sites
├── Site A
├── Site B
└── Site C

A Site should support fields appropriate to commercial cleaning, such as:

* id
* customer_id
* name
* address
* contact person
* phone
* email
* notes
* status
* latitude/longitude if the current system supports location
* created_at
* updated_at

Do not duplicate customer data unnecessarily.

A customer's billing/contact information should remain at Customer level unless there is a clear existing business reason for site-level overrides.

A Site represents the physical location where cleaning work happens.

Examples:

ABC Restaurant
├── Sukhumvit Branch
├── Silom Branch
└── Asoke Branch

ABC Condo
├── Lobby
├── Gym
├── Car Park
└── Building A

If the current system already has an address/location abstraction that can safely become Site, reuse it rather than creating unnecessary duplicate concepts.

---

## PHASE 3 — BOOKING DESIGN

Bookings must remain usable without contracts.

A booking should conceptually support:

* customer_id
* site_id
* service
* scheduled date/time
* assigned staff
* status
* price
* invoice relationship where appropriate
* recurring information where appropriate
* optional contract_id

IMPORTANT:

contract_id MUST be nullable/optional.

Examples:

ONE-TIME:

Customer
→ Site
→ Booking
→ Invoice

No Contract required.

RECURRING WITHOUT CONTRACT:

Customer
→ Site
→ Recurring Schedule
→ Bookings

No Contract required.

CONTRACT-BASED:

Customer
→ Site
→ Contract
→ Recurring Schedule
→ Bookings
→ Invoices

All three workflows must coexist.

Do not force one workflow into another.

---

## PHASE 4 — CONTRACTS

Introduce a Contract entity only if the codebase does not already have one.

Contract should be optional.

Conceptually:

Customer
└── Contract
├── Site(s)
├── Start date
├── End date
├── Renewal date
├── Contract value
├── Billing frequency
├── SLA terms
├── Agreed rates
└── Status

Possible statuses:

* draft
* active
* expiring
* expired
* cancelled
* renewed

Support:

* monthly
* quarterly
* annual
* custom billing cycles if appropriate

Do not assume every Contract applies to only one Site.

A commercial customer may have:

Customer
└── Contract
├── Site A
├── Site B
└── Site C

Choose the cleanest relational design based on the existing database architecture.

---

## PHASE 5 — RECURRING BILLING

Do not replace the existing recurring-booking system blindly.

First understand how recurring bookings currently work.

The system should eventually support:

A. Recurring booking without contract

Example:

Restaurant requests cleaning every Monday.

B. Recurring booking attached to a contract

Example:

Mall has a 12-month contract with monthly invoicing.

For contract-based billing, support:

Contract
→ Billing schedule
→ Invoice generation

The invoice generation process must be idempotent.

Never generate duplicate invoices if the scheduler runs more than once.

Use stable identifiers / unique constraints / idempotency keys where appropriate.

Also fix the previously identified recurring-booking duplication bug:

If creating the next recurring occurrence succeeds but disabling/updating the source recurring record fails, the scheduler must NOT generate the same occurrence repeatedly.

Design this transactionally/idempotently.

---

## PHASE 6 — QUOTATIONS

Audit whether quotation functionality already exists.

If it does not, design it so that commercial sales can follow:

Customer
→ Site Survey
→ Quote
→ Accepted Quote
→ Optional Contract
→ Booking
→ Invoice

Quotes must not require a Contract.

Support:

* quote number
* customer
* site
* services
* pricing
* validity date
* status
* version
* notes
* accepted/rejected timestamp
* conversion relationship to booking/contract where appropriate

Avoid duplicating pricing logic already present in the existing price calculator.

The price calculator should remain a calculation tool.

The Quote should be the persistent business record.

---

## PHASE 7 — CHECKLISTS & PROOF OF WORK

Design a reusable checklist system.

A booking may optionally have a checklist.

Do not make checklists mandatory for simple one-time residential bookings unless the current business rules require it.

Commercial bookings should be able to use:

Booking
→ Checklist
→ Checklist Items
→ Before Photos
→ After Photos
→ Staff completion
→ Client confirmation/signature

Checklist templates should be reusable by service type.

Example:

Factory Deep Cleaning Checklist

* Production area cleaned
* Floor cleaned
* Toilets cleaned
* Waste removed
* Equipment area cleaned

Store:

* completed/not completed
* completed_by
* completed_at
* notes

Photos must be associated with the appropriate booking/service record and should not be stored as publicly accessible files if the existing architecture has private-storage requirements.

---

## PHASE 8 — BACKWARD COMPATIBILITY

This is critical.

Do NOT break existing customers or existing bookings.

Before changing the Customer/Booking schema:

* inspect all existing queries
* inspect all API handlers
* inspect all frontend forms
* inspect AdminJS resources
* inspect customer portal
* inspect reports
* inspect invoices
* inspect recurring bookings

Existing records must continue to work.

If existing customers currently have an address field, determine a safe migration strategy.

For example:

Existing Customer
address = current address

can be migrated to:

Customer
Sites
└── Default Site

But ONLY if this matches the current schema and business behavior.

Do not blindly duplicate data.

Use proper database migrations.

Do not modify production data destructively.

---

## PHASE 9 — RBAC

Review existing RBAC/permissions.

Add permissions only where necessary.

Potential permissions may include:

* sites.read
* sites.create
* sites.update
* sites.delete
* contracts.read
* contracts.create
* contracts.update
* contracts.delete
* quotes.read
* quotes.create
* quotes.update
* quotes.approve
* checklists.read
* checklists.manage

However, do NOT invent a parallel permission architecture.

Use the existing PERMISSION_MATRIX and role system.

Ensure SUPER_ADMIN has appropriate access.

Ensure existing roles do not unexpectedly gain access to sensitive commercial features.

---

## PHASE 10 — API

Follow the existing Go API architecture and conventions.

Potential endpoints:

GET    /api/v1/sites
POST   /api/v1/sites
GET    /api/v1/sites/:id
PATCH  /api/v1/sites/:id
DELETE /api/v1/sites/:id

GET    /api/v1/contracts
POST   /api/v1/contracts
GET    /api/v1/contracts/:id
PATCH  /api/v1/contracts/:id

GET    /api/v1/quotes
POST   /api/v1/quotes
GET    /api/v1/quotes/:id
PATCH  /api/v1/quotes/:id

Do NOT blindly create these exact routes if the project has a different routing convention.

Follow existing API patterns.

Validate:

* customer ownership
* site ownership
* contract/customer consistency
* booking/site consistency
* permissions
* input validation

A booking belonging to Customer A must never be assignable to Customer B's Site.

---

## PHASE 11 — FRONTEND

Follow the existing frontend architecture and UI conventions.

Add functionality only after understanding the current frontend structure.

Commercial customer workflow should eventually be:

Customer
→ Sites
→ Site Details
→ Bookings
→ Contracts
→ Quotes
→ Invoices
→ Checklists / Completion Reports

But one-time customers should have a simpler workflow:

Customer
→ Booking
→ Invoice

Do not force commercial UI onto every customer.

The UI should make the distinction clear without creating unnecessary complexity.

---

## PHASE 12 — REPORTING

Prepare the data model so future reporting can support:

* Revenue by customer
* Revenue by site
* Cost by site
* Profitability by site
* Booking frequency
* Contract value
* Contract renewal pipeline
* Quote win rate
* Service completion rate
* Checklist completion
* Complaint rate
* Staff hours by site

Do not build every dashboard immediately unless the current project architecture makes it cheap.

First make the underlying data reliable.

---

## PHASE 13 — PERFORMANCE & DATA INTEGRITY

While implementing, specifically review:

1. N+1 queries in booking/site/assignment loading.
2. Proper indexes for:

   * customer_id
   * site_id
   * contract_id
   * booking dates
   * invoice dates
3. Unique constraints where needed.
4. Transaction boundaries.
5. Recurring-job idempotency.
6. Invoice idempotency.
7. Foreign-key integrity.

The existing review identified an N+1 query in bookings. Fix it as part of this work if it is within the affected code path.

---

## PHASE 14 — SECURITY

Also fix the previously identified authentication vulnerability.

The frontend SSR JWT verification must validate the same security claims required by the Go API, including:

* signature
* expiry
* issuer
* audience

A customer portal token with audience:

odysight-portal

must NEVER be accepted as a staff web session token whose expected audience is:

odysight-web

Use the existing auth architecture rather than creating a second authentication mechanism.

---

## IMPLEMENTATION RULES

1. Do not rewrite the application.
2. Do not introduce unnecessary frameworks.
3. Reuse existing patterns.
4. Prefer incremental migrations.
5. Preserve backward compatibility.
6. Do not make Contract mandatory.
7. Do not make Site mandatory for historical records until migration is safely handled.
8. Do not break one-time bookings.
9. Do not assume every customer is commercial.
10. Do not assume every recurring customer has a contract.
11. Do not duplicate existing pricing, invoice, or booking logic unnecessarily.
12. Do not modify unrelated features.
13. Do not change files until the architecture audit is complete.

---

## DELIVERABLE

First provide:

1. Current architecture findings.
2. Proposed data model.
3. Migration strategy.
4. API changes.
5. Frontend changes.
6. RBAC changes.
7. Risks/backward-compatibility concerns.
8. Implementation order.

Then wait for approval before making code changes.

After approval, implement incrementally.

After implementation run at minimum:

Backend:

* go build
* go vet
* go test ./...

Frontend:

* existing type/check commands
* existing unit tests
* astro check if applicable

Also add/update tests for:

* one-time booking without contract
* recurring booking without contract
* contract-based recurring booking
* multiple sites per customer
* site/customer ownership validation
* quote acceptance
* checklist completion
* recurring booking idempotency
* recurring invoice idempotency
* staff-vs-portal JWT audience validation

Final response must clearly report:

* files changed
* migrations added
* API endpoints added/changed
* frontend pages/components added/changed
* tests added
* test results
* remaining known issues

Do not claim browser/E2E testing was performed unless it actually was.
