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
| DISPATCH | Dispatcher | Create/assign bookings, view sites, check themselves in/out | Money, contracts, quotes |
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
*Repeat* on “One-time” unless the customer asked for a schedule. Enter the
agreed *Price before VAT* (leave it blank to bill the catalog price).

### Step 4 — Dispatch (dispatch)
**Dispatch** board → assign crew. A cleaner taps **Accept** on the booking —
first tap wins, double-booking the same cleaner at the same time is blocked.

### Step 5 — The job (cleaner)
**My bookings → Accept → check in (Attendance) → do the work → set
`completed`.** Optionally attach a checklist (see §5).

### Step 6 — Invoice, payment & receipt (accountant)
Only `completed` bookings can be billed. Pick the flow that fits the customer:

- **Pays now (one-time customer)** — booking row → **Collect payment**. Enter
  the method (and slip number if any). The invoice is created and paid in one
  step and the customer gets the **receipt** (ใบเสร็จรับเงิน) by email/LINE.
- **Pays later (company / monthly)** — booking row → **Invoice**. The customer
  gets the invoice (ใบแจ้งหนี้) with a PromptPay QR. When money arrives, open
  the invoice → **Record payment**. Each payment issues its own receipt.
- **Part payments** are fine: enter the amount received. The invoice shows
  *Partially paid* and the balance due; record the rest later.

Every receipt is listed under **Receipts** (PDF / email). A mistaken payment
is undone with **Payments → Refund**: the receipt is cancelled and the amount
goes back onto the invoice. An invoice can only be voided while nothing has
been paid on it. Follow up open invoices in **Reports → Financial (AR aging)**.

---

## 4. Commercial customers (restaurants, condos, malls, offices)

Everything in §3, plus only what you need:

- **Sites** — one customer, many branches. Add them under **Sites**, then
  pick the site on each booking. The booking form auto-selects the default
  site.
- **Quotes** — **Quotes → New quote**: pick the customer (and site), add one
  line per service with quantity and price, set *Valid until*, tick *VAT 7%*
  if needed. **Save & download PDF** gives the Thai/English quotation
  (ใบเสนอราคา) — send it to the customer on LINE or email yourself, then tap
  **Mark as sent**. When they answer, tap **Accepted** or **Declined**. On an
  accepted quote, **Create booking** opens a new booking already filled in
  (customer, site, price before VAT); pick the date and cleaner and save.
  An accepted quote can't be re-priced — make a new quote instead.
  Every quotation PDF states the booking terms: a 50% deposit (amount
  calculated from the total) is required to confirm the booking.
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

## 5a. More tools

- **Thai tax documents & VAT** — set *Settings → Company* (legal name, 13-digit
  tax ID, branch) and *Settings → Payments* (PromptPay ID, bank account, VAT
  rate). VAT is **optional**: tick *Charge VAT* in Settings → Company only if
  the company is VAT registered. With it off, invoices carry no VAT and the
  receipt is a plain ใบเสร็จรับเงิน. With it on, VAT is added and each receipt
  is a ใบเสร็จรับเงิน/ใบกำกับภาษี (for services the tax invoice is issued when
  payment is received). Invoices are always ใบแจ้งหนี้. Give company
  customers their tax ID and *Withholding tax 3%* on the customer form;
  invoices show the deduction and net payable.
- **Deposits and invoice status** — an invoice can be made as soon as the
  booking exists (not only after the job): Bookings → **Invoice**. The
  status follows the money: **Unpaid** → **Deposit paid · 50%** (after a
  part payment) → **Paid**. On the invoice, **Record payment** has a
  *Deposit 50%* button and a *Full balance* button; each payment issues a
  receipt (ใบเสร็จ/ใบกำกับภาษี for that amount). Status can't be set by
  hand. If a payment was recorded by mistake, tap **Cancel payment** next
  to its receipt: the receipt is cancelled and the invoice goes back to
  Unpaid / Deposit paid. **Collect payment** (completed jobs) is for
  customers who pay the full amount on the spot.
  When an unpaid customer wants to pay the deposit first, send
  **Deposit 50% PDF** (invoice page): the same invoice, with the deposit
  amount and a PromptPay QR for the deposit. When the money arrives,
  **Record payment → Deposit 50%**.
  An invoice marked paid without any receipt (from an older version) shows
  **Mark as unpaid** instead. Fixing a wrong payment never needs a new
  booking or quote; only a wrong price needs the invoice voided (after
  cancelling its payments) and a new one made from the same booking.
- **Signatures on documents** — PDFs follow Thai practice: each has two
  signature blocks ("ในนาม …", the line, the name in brackets, the role and
  the date). Quotation: ผู้อนุมัติสั่งซื้อ (customer) / ผู้เสนอราคา (us).
  Invoice: ผู้วางบิล / ผู้รับวางบิล. Receipt / tax invoice: ผู้มีอำนาจลงนาม
  (us) / ผู้อนุมัติ/ผู้รับเงิน. Print, sign and stamp when the customer
  needs a paper copy. The date under our signature is filled in with the
  document date; the customer's date line stays blank. Invoices also show a
  *Due date*: issue date + the overdue-reminder days in
  *Settings → Notifications*.
- **Invoice / receipt layout (company pattern)** — invoice and receipt PDFs
  come as two copies in one file: page 1 ต้นฉบับ / Original (for the
  customer), page 2 สำเนา / Copy (for our files). Dates are Thai Buddhist
  era (10/07/2569). The header shows the Thai legal name and the English
  name (*Settings → Company*: Legal name = บริษัท สไมล์ คลีน จำกัด, Name =
  Smile Clean CO.,LTD.). The customer's tax ID has ☒ สำนักงานใหญ่ /
  ☐ สาขาที่ boxes. The line reads "1 งาน" with the booking's site
  (สถานที่ปฏิบัติงาน); the receipt also names the invoice it pays. The
  receipt's *Paid by* boxes (cash / transfer to our account / cheque /
  other) are ticked from the payment method, with the bank account from
  *Settings → Payments*. The PDFs are ruled like the paper forms: a bordered
  item table with empty rows, boxed totals and boxed signature blocks. Document numbers stay INV-2026-0042 / RC-2026-0001.
- **Roles & permissions** — *Settings → Roles & permissions*. A Super admin
  can tick/untick what each role may do (Admin, Manager, Dispatch,
  Accountant, Cleaner) and **Save changes**; **reset** puts a role back to
  the built-in defaults. Super admin always keeps every permission. The API
  applies changes at once; menus update on each person's next page load.
  Outlined boxes differ from the default; ⚠ marks sensitive permissions
  (team, settings, audit log, editing payments/invoices).
- **Two-factor sign-in (2FA)** — office roles (Super admin, Admin, Manager,
  Accountant, Dispatch) must use an authenticator app (Google Authenticator
  or Microsoft Authenticator). At the first sign-in the app shows a QR code:
  scan it, type the 6-digit code, and **save the 10 backup codes**. After
  that, sign-in asks for the code; tick *Remember this device for 30 days* on
  your own computer/phone. Cleaners can turn it on under *Settings →
  Security*. Lost phone: sign in with a backup code, or ask a Super admin to
  press **Reset 2FA** on your row in *Users*, then set it up again.
- **Team chat** — sidebar / bottom bar → **Chat** → **New message**. Send
  text and voice messages (microphone button, up to 1 minute). Messages,
  "typing…" and "Seen" appear instantly (live connection; if it drops, the
  app reconnects by itself and catches up). Voice
  messages are deleted automatically 30 days after they were sent (the text
  of the chat stays) so the server's disk never fills up — photos are not
  sent in chat for the same reason. Who can message whom: admin, manager and
  dispatch can message anyone and anyone can message them; other staff
  (cleaners, accountant) can message each other only when they share a chat
  group. ADMIN sets up groups under **Chat → Chat groups** (e.g. "Bang Na
  team"). Every group also has its own **group chat** (👥 at the top of
  the chat list) where all its members talk together; people added later
  see the earlier messages, people removed lose access, and deleting the
  group deletes its group chat. Tap **Turn on notifications** once per phone to get a push alert
  for new messages — on iPhone, first add the app to the Home Screen (Share →
  Add to Home Screen), then open it from there.
- **LINE messages** — customers who came through LINE get booking
  confirmation, a reminder the day before, a job-done message and their
  invoice in that chat automatically. Look for the **LINE ✓** badge.
- **Cleaner app language** — cleaners tap English / ไทย / မြန်မာ at the top
  of My Jobs or Check-in.
- **Attendance** — everyone checks only **themselves** in and out (office
  staff on the *Team staff* tab, cleaners on *Check-in*). Only SUPER_ADMIN
  and ADMIN can record a check-in/out for someone else, e.g. a cleaner
  without a phone. Change who may do this in *Settings → Roles &
  permissions → Check in / out for others*.
- **GPS check-in** — set *Settings → Booking defaults → check-in radius*
  (e.g. 200 m) and pin each site (*Sites → GPS pin*: paste a Google Maps
  link or use your location on site). Cleaners then must be at the site to
  check in; history shows how far they were.
- **Dispatch drag & drop** — drag a job to another day/time, or use *By
  cleaner* and drag it onto another cleaner. Clashes are refused.
- **Payroll** (ADMIN, ACCOUNTANT) — set each cleaner's rate (per hour / day /
  job / monthly, overtime ×) and read pay for any period; *Export CSV* for
  the bank. Days without a check-out are flagged and not paid.
- **Complaints** — *Log complaint* on the customer (and job). High = fix in
  24h, medium 48h, low 72h. *Book re-clean* makes a free follow-up job;
  write what was done, then *Mark resolved*.
- **Supplies** — add items with opening stock and a reorder level; *Buy*,
  *Use* (on a job or site) and *Adjust* keep stock right. *Cost by site*
  compares supply cost with each site's revenue.

---

## 6. Daily rhythm per role

- **MANAGER** — Dashboard: new leads, today's bookings, expiring contracts,
  quote win rate (**Reports → Commercial**).
- **DISPATCH** — Dispatch board (drag to reschedule/reassign); complaints and re-cleans; record supply usage.
- **CLEANER** — Accept jobs, check in/out (at the site), complete + checklist + photos.
- **ACCOUNTANT** — Completed bookings → collect payment or invoice → record
  payments (receipts go out automatically); payroll each period;
  revenue by site/contract in **Reports → Commercial**.
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
| Cleaner can't check in: "you are … from …" | They are outside the site's check-in radius; move closer, or check the site's GPS pin |
| Cleaner check-in: "location is required" | Allow location access for the site in the phone's browser |
| No QR on the invoice | Set a PromptPay ID under Settings → Payments; paid/void invoices have no QR |
| "this booking has no price" | Set *Price before VAT* on the booking, or type the amount in the dialog |
| "payment exceeds the balance due" | The amount is more than what is still owed — check the invoice balance |
| Can't void an invoice | It has payments — refund them on **Payments** first |
| Authenticator code "invalid" | The phone's clock must be automatic (Settings → Date & time → Set automatically); use the newest code |
| Lost phone and no backup codes | A Super admin presses **Reset 2FA** in Users. If the only Super admin is locked out, on the server run: `docker compose exec postgres psql -U $POSTGRES_USER -d $POSTGRES_DB -c "UPDATE users SET totp_enabled=false, totp_secret='' WHERE email='owner@…'"` and set 2FA up again |
| Everyone's 2FA stopped working after a server change | The 2FA secrets are encrypted with `JWT_SECRET`; if it was changed, put the old value back (or reset everyone's 2FA) |
| Can't message a cleaner/colleague | You share no chat group — ask ADMIN to add you both to one (Chat → Chat groups) |
| No chat notifications on the phone | Tap "Turn on notifications" in Chat; on iPhone the app must be opened from the Home Screen icon; check the browser didn't block notifications |
| VAT appears / doesn't appear | Controlled by *Charge VAT* in Settings → Company; existing invoices keep what they were issued with |
| Supply usage refused "only N in stock" | Record the purchase (Buy) or a stock count (Adjust) first |

---

## 8. Golden rules

1. Never share logins — permissions keep money and customer data safe.
2. Phone numbers: always with country code (`+66 …`).
3. One-time jobs need only Lead → Customer → Booking → Collect payment.
4. Add Sites/Quotes/Contracts/Checklists only when the customer needs them.
5. When in doubt, ask your ADMIN — don't work around permissions.
