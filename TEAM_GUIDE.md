# Smile Clean CRM — Team Guidebook

For every staff member: what your role can do, and how one order flows from
first contact to paid invoice. Keep this open during your first week.

- Web app: ask your admin for the URL (local dev: `http://localhost:4321`)
- Sign in at `/login`. There is **no self-registration** — accounts are
  created by an admin under **Settings → Team**.

---

## 1. Roles at a glance

| Role | Typical person | Can | Cannot |
|---|---|---|---|
| SUPER_ADMIN | Owner | Everything, incl. team management and audit log | — |
| ADMIN | Co-owner / ops head | Everything except managing users and audit log | `users.manage`, `audit.read` |
| MANAGER | Branch / sales manager | Leads, customers, bookings, sites, quotes (create/edit), checklists | Create invoices/payments, approve quotes, manage contracts |
| DISPATCH | Dispatcher | Create/assign bookings, manage attendance, view sites | Money, contracts, quotes |
| ACCOUNTANT | Finance | Invoices, payments, expenses | Leads/bookings edits, contracts edits |
| CLEANER | Field staff | View own bookings, accept jobs, checklists, check in/out | Everything else |

If a button you expect is missing, you almost certainly lack the permission —
ask your ADMIN, don't share logins.

---

## 2. First day (SUPER_ADMIN / ADMIN)

1. Sign in with the seeded admin account from `.env`
   (`SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD`).
2. **Settings → Team**: create one account per staff member with the correct
   role from the table above.
3. **Settings → Workspace**: check service catalog, tax rate, and company
   details (these print on invoices).
4. **Settings → Team**: each cleaner needs a cleaner profile linked to their
   login so dispatch can assign them.

---

## 3. One order, start to finish (simple cleaning job)

### Step 1 — Lead (sales / manager / admin)
**Leads → New Lead.** Name + phone are required; pick the true source
(LINE, Facebook, walk-in…). Status flow: `new → contacted → quote_sent /
booked → won / lost`.

> If saving fails, read the error message — it tells you exactly what's
> wrong (usually the phone number format).

### Step 2 — Convert to customer (manager / admin)
Open the lead → **Convert**. Address and area are required. A *Default
Site* is created automatically — you don't need to touch Sites for simple
jobs.

### Step 3 — Booking (manager / dispatch)
**Bookings → New Booking.** Pick the customer (name, email and address
pre-fill), choose service, date/time and the primary cleaner. Leave
*Repeat* on “One-time” unless the customer asked for a schedule.

### Step 4 — Dispatch (dispatch)
**Dispatch** board → assign crew. A cleaner taps **Accept** on the booking —
first tap wins, double-booking the same cleaner at the same time is blocked.

### Step 5 — The job (cleaner)
**My bookings → Accept → check in (Attendance) → do the work → set
`completed`.** Optionally attach a checklist (see §5).

### Step 6 — Invoice & payment (accountant)
Booking row → **Invoice** (only `completed` bookings can be billed; one
active invoice per booking — retries never duplicate). Mark `paid` when
money arrives. Follow up open invoices in **Reports → Financial (AR aging)**.

---

## 4. Commercial customers (restaurants, condos, malls, offices)

Everything in §3, plus only what you need:

- **Sites** — one customer, many branches. Add them under **Sites**, then
  pick the site on each booking. The booking form auto-selects the default
  site.
- **Quotes** — **Quotes → Add** (customer + service lines; totals and tax
  compute automatically) → **Send** → **Approve** when the customer accepts.
  An accepted quote is terminal and ready to become a booking/contract.
- **Contracts** — ONLY for formal agreements. Most recurring customers never
  need one. Lifecycle: `draft → Activate → active → Renew` next period
  (the old row stays as history, a successor draft opens). Illegal jumps
  (e.g. draft → expired) are rejected.
- **Recurring** — set *Repeat* to weekly / biweekly / monthly on the
  booking. The scheduler rolls the series forward automatically and can
  never create the same occurrence twice.

Rule of thumb: **no contract, no problem.** One-time, recurring without
contract, and contract-based billing all coexist.

---

## 5. Checklists & proof of work (optional)

Nobody is forced to use checklists. When a commercial client wants proof:

1. Booking row → **Checklist** (or open **Checklists** and pick the booking).
2. **New from template** (e.g. *Factory Deep Cleaning*).
3. Cleaner ticks items, attaches **Before/After** photos (jpg/png/webp,
   max 8 MB — stored privately, staff-only download).
4. Client confirms with signature.

---

## 6. Daily rhythm per role

- **MANAGER** — Dashboard: new leads, today's bookings, expiring contracts,
  quote win rate (**Reports → Commercial**).
- **DISPATCH** — Dispatch board assignments; **Attendance** check-ins.
- **CLEANER** — Accept jobs, check in/out, complete + checklist + photos.
- **ACCOUNTANT** — Completed bookings → invoice → paid; revenue by
  site/contract in **Reports → Commercial**.
- **ADMIN / SUPER_ADMIN** — Team accounts, settings, audit log
  (SUPER_ADMIN only), contract approvals and renewals.

---

## 7. Troubleshooting

| Problem | Cause / fix |
|---|---|
| Can't save a lead/customer | Read the error toast: usually phone format (`+66 …`, min 6 chars) or a duplicate email |
| Missing button (Invoice, Approve, Delete…) | Your role lacks the permission — see §1, ask ADMIN |
| Booking won't accept a site/contract | It belongs to a different customer — pick the right customer first |
| Contract status won't change | Only allowed transitions work (see §4); renewal goes through **Renew** |
| Quote can't be reopened | Accepted quotes are terminal by design — create a new version |
| Invoice button missing | Booking must be `completed` first |
| Duplicate invoice warning | The booking already has an active invoice — open it instead |

---

## 8. Golden rules

1. Never share logins — permissions keep money and customer data safe.
2. Phone numbers: always with country code (`+66 …`).
3. One-time jobs need only Lead → Customer → Booking → Invoice.
4. Add Sites/Quotes/Contracts/Checklists only when the customer needs them.
5. When in doubt, ask your ADMIN — don't work around permissions.
