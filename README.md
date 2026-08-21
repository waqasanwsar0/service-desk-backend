# Service Desk & Field Engineer Management System (Go)

Custom-built backend for the Service Desk system (replacing the earlier
Odoo-based plan). Pure Go standard library — **zero external
dependencies** — so it runs anywhere with just the Go toolchain, and the
whole thing builds as a single static binary.

## All 6 core modules + SOW-depth additions — built and tested end-to-end

1. **Auth & Roles** — register, login, HMAC-signed session tokens
   (JWT-like, stdlib-only), role-based access control for five roles:
   `admin`, `service_desk`, `engineer`, `recruiter`, `accounts`.
2. **Ticket Intake** — full status workflow
   `New → Assigned → Onsite → Timesheet Pending → Completed → Invoice → Paid`,
   validated so you can't skip or reverse steps. Includes a stub endpoint
   for the future Outlook (Microsoft Graph) email-to-ticket integration.
3. **Engineer Database & Search** — the SOW's "Service Desk Search
   Feature": filter by location, skill, availability, and Named-Project
   eligibility, cheapest matching engineer shown first.
4. **FTE Attendance** — "I am ON-SITE" check-in, duplicate-check-in
   prevention, in-portal leave-request / approval flow.
5. **Timesheets & Billing** — upload proof of work, compute billed amount
   from the engineer's hourly/day rate, auto-advances the ticket
   (Onsite → Timesheet Pending → Completed).
6. **Accounting** — multi-entity (multi-company), multi-currency
   invoicing: pulls approved timesheets into an invoice, blocks
   double-invoicing and currency mismatches, tracks
   Draft → Sent → Partially Paid → Paid (or → Overdue / Cancelled), and
   pushes linked tickets through Invoice → Paid. Plus:
   - **Vendor bills** — the cost side of a ticket (what you pay the
     engineer), separate from what the client is billed.
   - **Profitability per ticket** — billed amount vs. vendor cost.
   - **Credit / debit notes** against an invoice.
   - **Partial payments** — `RecordPayment` applies an amount and moves
     the invoice through Partially Paid → Paid automatically.
7. **Recruitment ATS** — applicant pipeline
   (Applied → Screening → Interview → Offer → Hired, or Rejected from any
   stage), with a per-applicant note thread ("Chat with Recruiter"). Plus:
   - **Multi-project assignment ("re-hire")** — one engineer can hold
     several active project assignments at once, each with its own
     rate and travel/tools allowances.
   - **Project-wise recruitment dashboard** — pipeline counts grouped
     by job title/project.
   - **Recruiter performance** — candidates handled, hired, hire rate
     per recruiter.
8. **Contracts / SOW tracking** — client contracts with scope, billing
   terms, start/expiry dates, and an "expiring within N days" query for
   renewal reminders.
9. **Dashboard** (`GET /api/dashboard`) — the SOW's "VERY IMPORTANT"
   section 8: tickets by client, tickets by project, upcoming visits
   (sorted by SLA due time), daily/weekly/monthly ticket counts, the
   timesheet-missing list, engineer availability, and the leave calendar.

All stores are in-memory today, but every one sits behind an interface
so a PostgreSQL-backed implementation can be swapped in later without
touching any handler code.

### Still not built (needs external services/credentials, not just code)

- **Outlook auto-import** — `POST /api/tickets/import/email` exists as a
  landing point, but nothing calls it yet. Needs a Microsoft Graph API
  poller/webhook with real Azure AD app credentials.
- **Bank reconciliation** — needs actual bank statement files/API access
  to build and test against.
- **WhatsApp intake, GPS attendance, OCR timesheets, job-board imports**
  — explicitly "Future Scope" in the original SOW; correctly deferred.
- **VAT/GST compliance reports, multi-language invoice templates** — not
  started.
- **Data persistence** — everything above resets on server restart until
  it's wired to PostgreSQL (interfaces are ready for that swap).

## Endpoints

| Method | Path | Role required | Purpose |
|---|---|---|---|
| POST | `/api/auth/register` | *(open — see note below)* | Create a user |
| POST | `/api/auth/login` | — | Get a session token |
| GET | `/api/auth/me` | any authenticated user | Current user's profile |
| POST | `/api/tickets` | service_desk, admin | Create a ticket |
| GET | `/api/tickets` | any authenticated user | List/filter tickets |
| GET | `/api/tickets/{id}` | any authenticated user | Get one ticket |
| PATCH | `/api/tickets/{id}/assign` | service_desk, admin | Assign an engineer |
| PATCH | `/api/tickets/{id}/status` | service_desk, admin, engineer | Move to next status |
| POST | `/api/tickets/import/email` | service_desk, admin | Outlook import landing point (stub) |
| POST | `/api/engineers` | service_desk, admin, recruiter | Add an engineer |
| GET | `/api/engineers` | any authenticated user | Search engineers |
| PATCH | `/api/engineers/{id}/availability` | service_desk, admin, engineer | Toggle availability |
| POST | `/api/attendance/checkin` | engineer, admin | "I am ON-SITE" |
| POST | `/api/attendance/leave-request` | engineer, admin | Request leave |
| PATCH | `/api/attendance/leave-request/{id}/decision` | service_desk, admin | Approve/reject leave |
| GET | `/api/attendance` | any authenticated user | Attendance + leave history |
| POST | `/api/timesheets` | service_desk, admin, engineer | Upload timesheet |
| GET | `/api/timesheets` | any authenticated user | List/filter timesheets |
| GET | `/api/timesheets/{id}` | any authenticated user | Get one timesheet |
| PATCH | `/api/timesheets/{id}/approve` | service_desk, admin, accounts | Compute billing, advance ticket |
| PATCH | `/api/timesheets/{id}/reject` | service_desk, admin, accounts | Reject a timesheet |
| POST | `/api/entities` | admin, accounts | Add a billing entity (multi-company) |
| GET | `/api/entities` | any authenticated user | List entities |
| POST | `/api/invoices` | admin, accounts | Generate invoice from approved timesheets |
| GET | `/api/invoices` | any authenticated user | List/filter invoices |
| GET | `/api/invoices/{id}` | any authenticated user | Get one invoice |
| PATCH | `/api/invoices/{id}/send` | admin, accounts | Mark invoice sent |
| PATCH | `/api/invoices/{id}/paid` | admin, accounts | Mark invoice paid, advance tickets |
| POST | `/api/applicants` | recruiter, admin | Add a candidate |
| GET | `/api/applicants` | recruiter, admin | List/filter candidates |
| GET | `/api/applicants/{id}` | recruiter, admin | Get one candidate |
| PATCH | `/api/applicants/{id}/stage` | recruiter, admin | Move pipeline stage |
| POST | `/api/applicants/{id}/notes` | recruiter, admin | Add a note ("Chat with Recruiter") |
| GET | `/healthz` | — | Health check |

### New endpoints (SOW-depth additions)

| Method | Path | Role required | Purpose |
|---|---|---|---|
| GET | `/api/dashboard` | any authenticated user | Client/project ticket counts, upcoming visits, timesheet-missing list, engineer availability, leave calendar |
| PATCH | `/api/invoices/{id}/payment` | admin, accounts | Record a partial or full payment |
| PATCH | `/api/invoices/{id}/cancel` | admin, accounts | Cancel an invoice |
| PATCH | `/api/invoices/{id}/overdue` | admin, accounts | Mark an invoice overdue |
| POST | `/api/vendor-bills` | admin, accounts | Record what's owed to an engineer/vendor for a ticket |
| GET | `/api/vendor-bills` | any authenticated user | List/filter vendor bills |
| PATCH | `/api/vendor-bills/{id}/approve` | admin, accounts | Approve a vendor bill |
| PATCH | `/api/vendor-bills/{id}/pay` | admin, accounts | Mark a vendor bill paid |
| GET | `/api/tickets/{id}/profitability` | admin, accounts, service_desk | Billed amount vs. vendor cost vs. profit for one ticket |
| POST | `/api/adjustment-notes` | admin, accounts | Create a credit/debit note against an invoice |
| GET | `/api/invoices/{id}/adjustment-notes` | any authenticated user | List an invoice's credit/debit notes |
| POST | `/api/contracts` | admin, accounts | Create a client contract / SOW record |
| GET | `/api/contracts` | any authenticated user | List contracts, or `?expiring_within_days=N` for renewal reminders |
| GET | `/api/contracts/{id}` | any authenticated user | Get one contract |
| POST | `/api/engineers/{id}/assignments` | recruiter, admin, service_desk | Add a project assignment (re-hire — an engineer can hold several at once) |
| GET | `/api/engineers/{id}/assignments` | any authenticated user | List an engineer's active/past assignments |
| PATCH | `/api/assignments/{id}/end` | recruiter, admin | End a project assignment |
| GET | `/api/applicants/dashboard` | recruiter, admin | Recruitment pipeline counts grouped by project |
| GET | `/api/recruiters/performance` | recruiter, admin | Candidates handled / hired / hire rate per recruiter |

> **Note on `/api/auth/register`:** left open so the first admin account
> can be created. Before real deployment, protect it with
> `RequireRole(admin)` and seed the first admin through a one-off script.

## Run it

```bash
go run ./cmd/server
# or
go build -o server ./cmd/server && ./server
```

Server starts on `:8080` (override with `PORT` env var). Set
`SERVICEDESK_AUTH_SECRET` to a strong random value in any real deployment
— it defaults to an insecure dev value otherwise.

### Example: full dispatch-to-payment flow

```bash
# 1. Register + login as admin
curl -X POST localhost:8080/api/auth/register -H "Content-Type: application/json" \
  -d '{"name":"Admin","email":"admin@example.com","password":"AdminPass1","role":"admin"}'
TOKEN=$(curl -s -X POST localhost:8080/api/auth/login -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"AdminPass1"}' | python3 -c "import json,sys;print(json.load(sys.stdin)['token'])")

# 2. Create a billing entity, an engineer, and a ticket
curl -X POST localhost:8080/api/entities -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"ServiceDesk GmbH","country":"Germany","default_currency":"EUR"}'
curl -X POST localhost:8080/api/engineers -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Ali Raza","email":"ali@eng.com","skills":["Windows"],"location":"Berlin","hourly_rate":40,"currency":"EUR"}'
curl -X POST localhost:8080/api/tickets -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"title":"Windows login issue","client_name":"Acme GmbH","priority":"High"}'

# 3. Assign -> Onsite -> upload timesheet -> approve -> invoice -> send -> paid
#    (each step advances the ticket automatically — see the module list above)
```

## What's next (not built yet)

- Swap in-memory stores for PostgreSQL — interfaces are already in place.
- Link `User` accounts to `Engineer` records (so `/checkin` and
  `/assign` can derive the engineer from the logged-in user instead of a
  raw ID in the request body).
- Microsoft Graph API poller/webhook that calls `POST /api/tickets/import/email`.
- Currency conversion when an invoice's entity currency differs from a
  timesheet's — currently the two must match exactly.
- Dashboard/reporting endpoints (aggregate views over tickets, billing, ATS).
- WhatsApp intake, GPS attendance, OCR timesheet reading, job-board
  imports — all future-scope items per the original SOW.
- Swap the stdlib password hash for `bcrypt`/`argon2id` once the dev
  machine has normal internet access for `go get` (see comment in
  `internal/auth/password.go`).
- A real frontend — this is the API only.

## Deploying (Railway + Vercel)

This backend now supports two storage modes:

- **No `DATABASE_URL` set** — in-memory (resets on every restart). Good
  for local development.
- **`DATABASE_URL` set** — PostgreSQL. Data persists across restarts and
  deploys. The schema (`internal/db/schema.sql`) is applied automatically
  on every boot — no separate migration step.

### Backend → Railway (free tier)

1. Push this `service-desk/` folder to a GitHub repo.
2. On [railway.app](https://railway.app), create a new project → "Deploy
   from GitHub repo" → pick the repo.
3. Add a **PostgreSQL** plugin from Railway's "New" menu — it
   automatically sets a `DATABASE_URL` variable on your service.
4. Set two more environment variables on the Go service:
   - `SERVICEDESK_AUTH_SECRET` — any long random string (signs login tokens).
   - `CORS_ALLOWED_ORIGINS` — your Vercel frontend URL once you have it,
     e.g. `https://your-app.vercel.app` (comma-separate multiple origins).
5. Railway auto-detects Go (via Nixpacks) and builds/runs `./cmd/server`.
   It also sets `PORT` automatically — the server already reads it.
6. Note the public URL Railway gives you
   (`https://your-service.up.railway.app`) — the frontend needs it.

### Frontend → Vercel (free tier)

1. Push `service-desk-web/` to a GitHub repo (same or separate).
2. On [vercel.com](https://vercel.com), "Add New Project" → import the repo.
3. Framework preset: Vite (auto-detected). Build command `npm run build`,
   output directory `dist` — filled in automatically.
4. Add an environment variable: `VITE_API_URL` = your Railway backend URL.
5. Deploy. Vercel gives you a URL like `https://your-app.vercel.app`.
6. Go back to Railway and set `CORS_ALLOWED_ORIGINS` to that exact URL,
   then redeploy the backend so it takes effect.

`vercel.json` in the frontend folder already handles client-side routing
(so refreshing `/tickets/TCK-0001` doesn't 404).

## Outlook email ticket intake (optional)

If you want unread emails in a shared mailbox to become tickets
automatically, set 4 environment variables on the Railway service:

- `AZURE_TENANT_ID`
- `AZURE_CLIENT_ID`
- `AZURE_CLIENT_SECRET`
- `AZURE_MAILBOX` — the mailbox address to watch, e.g. `support@yourcompany.com`

These come from an Azure App Registration on the mailbox's Microsoft 365
tenant:

1. [portal.azure.com](https://portal.azure.com) → **App registrations** → **New registration**.
2. **API permissions** → **Add a permission** → **Microsoft Graph** →
   **Application permissions** → add `Mail.Read` → **Grant admin consent**.
3. **Certificates & secrets** → **New client secret** — copy its value
   immediately (it's shown once).
4. From the app's **Overview** page: copy the **Tenant ID** and
   **Client ID** (Application ID).

Without these 4 variables set, the server runs exactly as before —
nothing else is affected. With them set, it polls the mailbox's inbox
every 2 minutes, turns each unread email into a ticket (subject →
title, sender → client name, preview text → description), and marks
it read so it isn't imported twice.

## File uploads (images, videos, PDFs)

`POST /api/files/upload` (multipart/form-data, field `file`) accepts
images, videos, and PDFs up to 20MB, stores them in PostgreSQL (no
external storage service needed), and returns a URL like
`/api/files/FILE-000001`. `GET /api/files/{id}` serves the file back —
this endpoint is intentionally left open (no auth) since browser
`<img>`/`<video>` tags can't send an Authorization header; the random
ID acts as a private link.

Wired into the frontend for: ticket photos, engineer resumes, client
requirement files, social media task assets, and timesheet proof —
each of those forms has an upload button alongside the option to paste
an external URL instead.
