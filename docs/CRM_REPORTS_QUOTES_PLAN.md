# CRM Plan II — Reports, Teams, Goals, Forecast, Approvals, Tasks, Quotes

**Status: in progress.** Done: §10 steps 1–4 — stage log v2, report scope, teams, goals, forecast
category per stage, forecast submissions, the Equipos, Metas and Forecast pages, reports R1–R7
(R6 on the Forecast page), step 5, required fields per stage, and step 6, approvals and tasks
(see "As built" in §5), and step 7, email templates and sending (see "As built" in §6.2, task kind 3
included). Steps 8–10 are still a proposal. None of it has been run in the real app
yet. Builds on `docs/CRM_PLAN.md` (pipelines,
opportunities, stage log, access), which is implemented.

## 0. Summary

This phase adds what Odoo, Salesforce, HubSpot, Pipedrive and Zoho use to manage a sales team, kept
flexible enough for companies with very different processes:

| Feature | Taken from | Section |
|---|---|---|
| **Teams** of users, with a leader | Odoo sales teams | §3.1 |
| **Goals** per user × pipeline × month | Odoo targets, Pipedrive goals | §3.2 |
| **Forecast category** per stage + **forecast submissions** compared with goals and actuals | Salesforce forecasting | §3.3, §3.4 |
| **Required fields per stage** | HubSpot, Zoho Blueprint | §4.1 |
| **Stage approval** by a named approver | Zoho Blueprint, Dynamics process flows | §4.2 |
| **Tasks**: stage automations that are *proposed*, never auto-run; the user executes them from a Tasks page | Bitrix24 robots, Pipedrive activities | §5 |
| **Email templates** and email sending (virtual mailbox outside production) | HubSpot / Odoo templates | §6 |
| **Quotes** owned by `sales`, shown and created from the CRM, PDF + email | Odoo quotations, Salesforce quotes | §7 |
| **Reports R1–R7** built from a self-contained stage log | all of the above | §8 |

New CRM menu layout:

| Group | Pages |
|---|---|
| Clientes | Clientes · Oportunidades · **Tareas** |
| **Reportes** | **Reportes** · **Forecast** |
| Gestión | Flujos · Flujos: Accesos · Diccionarios · **Equipos** · **Metas** |

Sales module gets **Cotizaciones** (`sales/quotes`).

## 1. Decisions taken

| # | Decision |
|---|---|
| E1 | Reports attribute movements to the **opportunity's seller** (`OwnerUserID`), not to the user who clicked. |
| E2 | The **stage position** (funnel depth) is stored on every log row. Reordering stages later does not rewrite history. |
| E3 | Moving to another pipeline counts as **transferred out (−)** in the source pipeline and **entered (+)** in the target. |
| E4 | **Lost subtracts the stages reached.** Net-progress reports (R1) show a lost deal as −; where it died is still visible in the raw log, which the funnel report (R4) reads. |
| E5 | Each stage carries a **forecast category** so pipelines named freely can be compared. |
| E6 | **Quotes belong to `sales`** (`sales/types`), and the CRM shows and creates them through the same component. |
| E7 | Quotes are sent as **PDF download + email**. |
| E8 | Quote lines come from the **product/service catalog only**, with no free-text lines. |
| E9 | First delivery: R1, R2 and reports 2–6 of the earlier proposal (waterfall, funnel/time in stage, seller scorecard, forecast vs goal, lost reasons). |
| E10 | Goals are per **user × pipeline**. Users are grouped in **teams**; a team's goal is the sum of its members' goals, never stored. |
| E11 | A stage can **require approval** from one user chosen per stage. |
| E12 | Stage automations **never run on their own.** Entering a stage creates pending **tasks**; a user opens the Tasks page and executes them (send a quote, send a template email, ...). |
| E13 | Email goes out for real **only in production.** In local and QAS, the send function stores the message in a virtual mailbox table instead. That mailbox is built separately by another agent (§6.1). |
| E14 | Report pages are gated by their own access, as every page. A separate access, **CRM Reporte completo**, widens every CRM report to **all pipelines** (§8.4). |
| E15 | Quote PDFs are generated in the backend with **`github.com/gpdf-dev/gpdf`** (pure Go, MIT, no dependencies; tables, images, 12-column grid, repeated header/footer). |
| E16 | Converting an accepted quote **prefills** the sale creation screen; the user picks warehouse and payment there. |
| E17 | **Won is a hidden final column, not "reached everything".** A deal can be won from any stage; it keeps credit only for the stages it actually reached, plus +1 Won, and reports show the stage it was won from (seller A wins at stage 2, seller B at stage 4). Won stays an outcome usable from any stage (D2), not configurable. |

## 2. The stage log becomes self-contained

Today `CrmOpportunityStageLog` stores from/to pipeline and stage, the outcome after, the time and the
acting user. That is not enough for E1–E4: the seller, the amount and the position are missing,
and amount / close-date / owner edits are not logged at all.

**Rule: every log row carries the full reportable state *before* and *after* the change.** A report
for any period then aggregates that period's rows alone, without reading earlier rows of the same
opportunity (which may sit in another month's cache group).

```go
type CrmOpportunityStageLog struct {
	db.TableStruct[CrmOpportunityStageLogTable, CrmOpportunityStageLog]
	CompanyID     int32 `cb:"1"`
	ID            int32 `cb:"2"`
	OpportunityID int32 `cb:"3"`
	Kind          int8  `cb:"4"`  // 1 created, 2 moved, 3 closed/reopened, 4 amount, 5 close date, 6 owner
	Time          int32 `cb:"5"`  // SUnixTime
	Period        int16 `cb:"6"`  // YYMM of Time: the cache-group key of the reports
	UserID        int32 `cb:"7"`  // who acted (audit only, never used for attribution)
	// State before the change. All zero on Kind = created.
	FromPipelineID        int16 `cb:"8"`
	FromStageID           int16 `cb:"9"`
	FromPosition          int8  `cb:"10"` // 1-based depth of the stage in its pipeline at that time
	FromOutcome           int8  `cb:"11"`
	FromOwnerUserID       int32 `cb:"12"`
	FromAmount            int32 `cb:"13"`
	FromExpectedCloseDate int16 `cb:"14"`
	// State after the change.
	ToPipelineID        int16 `cb:"15"`
	ToStageID           int16 `cb:"16"`
	ToPosition          int8  `cb:"17"`
	ToOutcome           int8  `cb:"18"`
	ToOwnerUserID       int32 `cb:"19"`
	ToAmount            int32 `cb:"20"`
	ToExpectedCloseDate int16 `cb:"21"`
}
```

Indexes: the existing local index by `OpportunityID` (history panel), plus a **cache-groups index
by `Period`** (skill `cache-groups-api`). The log is append-only and always written with the current
time, so a closed month never changes again: its group is downloaded once.

Write path: `POST.crm-opportunity` (when amount, close date or owner change),
`POST.crm-opportunity-move`, `POST.crm-opportunity-close`, quote acceptance (§7.4) and approval
decisions (§4.2) all build the row with one pure function,
`types.BuildStageLogRow(before, after CrmOpportunity, pipelines, kind, userID, now)`, in the same
transaction as the opportunity update. Kinds are 1 created, 2 moved, 3 closed/reopened, 4 edited;
an edit is logged only when it changes amount, owner or expected close date, as **one** row whose
From/To fields show what changed (`EditChangesReportedState`).

**As built:** the struct keeps the original field names where they fit (`Outcome` became
`ToOutcome`), adds `LostReasonID` (the lost-reasons report reads it from the row), and drops the
stage-count fields once E17 made "won" stop reaching every stage. The cache group is
`(Period, PipelineIDs)`, where `PipelineIDs` is a computed slice of the row's one or two pipelines,
so a transfer reaches both pipelines' reports.

Pre-alpha: the existing log rows and sample data are regenerated (wipe + reseed), no migration.

## 3. Teams, goals, forecast

### 3.1 `CrmTeam` — `crm_teams`

```go
type CrmTeam struct {
	CompanyID     int32   `cb:"1"`
	ID            int16   `cb:"2"`
	Name          string  `cb:"3"`
	LeaderUserID  int32   `cb:"4"`
	MemberUserIDs []int32 `cb:"5"`
	Status        int8    `json:"ss" cb:"ss,6"`
	Updated       int32   `json:"upd" cb:"upd,7"`
	Version       int16   `json:"ver" cb:"ver,8"`
	UpdatedBy     int32   `cb:"9"`
	Metadata      *string `json:"mdt" cb:"mdt,10"`
}
```

Small table, delta-synced whole. Validation: name required, users exist and are active, no
duplicates, leader is a member, **a user belongs to at most one active team** (open decision P5).

### 3.2 `CrmGoal` — `crm_goals`

```go
type CrmGoal struct {
	CompanyID  int32 `cb:"1"`
	ID         int32 `cb:"2"`
	PipelineID int16 `cb:"3"`
	UserID     int32 `cb:"4"`
	Period     int16 `cb:"5"` // YYMM
	Metric     int8  `cb:"6"` // CrmGoalMetric*, see below
	Target     int32 `cb:"7"` // amount scale for money metrics, plain count otherwise
	Status, Updated, Version, UpdatedBy, Metadata
}
```

Unique on (PipelineID, UserID, Period, Metric), enforced on save. Delta-synced by `Period`'s year
(one subset per year). Metrics (open decision P4 — proposed set):

| ID | Metric | Actual measured from |
|---|---|---|
| 1 | Won amount | log rows reaching Outcome = won in the period, minus reopenings |
| 2 | Won count | same, counted |
| 3 | New opportunities | log rows Kind = created + entered by transfer |
| 4 | Activities | `crm_activity` rows by the seller in the period |

The **Metas** page (`crm/goals`): a grid of users (grouped by team) × 12 months for one pipeline,
metric and year; cells editable inline; "copy previous year" and "fill row" helpers. Team rows show
the sum of their members, read-only.

### 3.3 Forecast category per stage

`CrmPipelineStage` gets `ForecastCategory int8`:

| ID | Category | Meaning |
|---|---|---|
| 1 | Pipeline | early, low confidence |
| 2 | Best case | could close this period |
| 3 | Commit | the seller commits to it |

Closed won is always *Closed*; lost is always *Omitted*. These two are outcomes, not stages, so
they are not selectable.

### 3.4 `CrmForecastSubmission` — `crm_forecast_submissions`

The computed forecast of a seller for a month is: open opportunities with `ExpectedCloseDate` in
that month, summed by category (raw and weighted by probability), plus what is already won in that
month. That number changes every day, so to compare "what the seller said" with "what happened",
the seller **submits** it:

```go
type CrmForecastSubmission struct {
	CompanyID  int32 `cb:"1"`
	ID         int32 `cb:"2"`
	PipelineID int16 `cb:"3"`
	UserID     int32 `cb:"4"` // the seller, or the team leader for a team submission
	TeamID     int16 `cb:"5"` // 0 = individual submission
	Period     int16 `cb:"6"` // YYMM being forecast
	// Computed at submission time by the server, never sent by the client.
	ComputedCommit, ComputedBestCase, ComputedPipeline, ComputedClosed int32 `cb:"7..10"`
	// What the seller (or leader) declares; defaults to the computed values.
	CommitAmount   int32  `cb:"11"`
	BestCaseAmount int32  `cb:"12"`
	Note           string `cb:"13"`
	Created        int32  `cb:"14"`
	Status, Updated, Version, UpdatedBy, Metadata
}
```

Append-only: every submission is a new row, so the report can show how the forecast evolved
during the month. The **Forecast** page (`crm/forecast`): pick pipeline and month; one row per
seller (grouped by team) with goal, computed categories, last submission, won so far; the user
submits their own row; the leader submits the team row (open decision P7).

## 4. Stage rules

### 4.1 Required fields per stage

Replace `CrmPipelineField.Required bool` with `RequiredFromStageID int16`: 0 = optional; otherwise
the field is required for the opportunity to **enter that stage or any later one** (by position).
Setting it to the first stage means "required from creation", which is what `Required` meant.

Rules live in `crm/types/custom_field_rules.go`: `MissingRequiredFields(pipeline, targetStageID,
values)`. The move endpoint already rejects with the list of missing fields and the UI opens the
form; it now passes the target stage. The pipeline editor shows a "required from" stage select per
field.

Standard fields (amount, expected close date) can also be required per stage:
`CrmPipelineStage.RequiredStandardFields []int8` (1 amount, 2 expected close date) — the field
catalog only holds custom fields, so these two need their own list.

**As built:**
- `MissingRequiredFields(pipeline, stageID, opportunity)` takes the whole opportunity (the standard
  fields are read from it) and returns `MissingRequirements{FieldIDs, StandardFields}`. The same
  check runs on **create** (stage = the chosen one), **edit** (stage = the stored one, so an edit
  cannot empty a field the current stage requires) and **move** (stage = the target).
- A standard field listed on a stage is required there **and in every later stage** by position,
  like the custom fields, so skipping the stage does not skip the rule. An empty amount or close
  date is zero.
- Pipeline save refuses a field whose required-from stage does not exist or is deleted (so a stage
  cannot be removed while a field still names it), an unknown or repeated standard field. A stage
  added in the same edit has no ID yet; the editor lists only saved stages in "required from"
  (a new pipeline's stages get IDs 1..n, which the seed uses).
- The board asks for the missing data when a card is **dropped** on a stage it cannot enter: the
  move form opens with that stage chosen, with inputs for the missing amount / close date and the
  missing custom fields; the card is saved with them, then moved.
- Reseed after this change: `Required bool` was replaced by an `int16` under the same column number.

### 4.2 Stage approval

`CrmPipelineStage` gets `ApproverUserID int32` (0 = no approval). Moving an opportunity **forward
into** a stage with an approver does not move it. Instead:

1. The move endpoint creates an approval **task** (§5, Kind = approval) assigned to the approver,
   carrying the target stage, and sets `CrmOpportunity.PendingApprovalTaskID`.
2. While that is set, the card shows "pending approval of <user>" and cannot be moved or closed.
   The requester can cancel the request.
3. The approver approves or rejects from the Tasks page, with an optional note.
   - Approve → the move runs (same code as `POST.crm-opportunity-move`, required fields checked
     again), the log row's `UserID` is the approver. Stage tasks of the new stage are created.
   - Reject → the opportunity stays; the note goes to its activity timeline.

Proposed rules (open decision P1, P2):
- A forward move **cannot skip** a stage that requires approval: going from 1 to 4 when 3 has an
  approver is rejected ("must pass through <stage 3>"). Otherwise a user could jump over every gate.
- Moving **backward** into a gated stage needs no approval.
- Moving from another pipeline into a gated stage needs approval.
- The approver must be an active user; not necessarily a pipeline member (approving does not give
  board access).

## 5. Tasks

A stage can define **task templates**, created when an opportunity enters it. They are never run
automatically: they wait on the **Tareas** page (`crm/tasks`) until a user executes or dismisses
them. Approvals use the same table, so the approver has one inbox.

```go
// On CrmPipelineStage
TaskTemplates []CrmStageTaskTemplate `cb:"10"`

type CrmStageTaskTemplate struct {
	ID              int16  `cb:"1"` // stable inside the stage
	Kind            int8   `cb:"2"` // 2 send quote, 3 send email template, 4 activity
	Title           string `cb:"3"`
	EmailTemplateID int16  `cb:"4"` // Kind 3
	ActivityTypeID  int16  `cb:"5"` // Kind 4: call, visit, ... (crm activity types)
	DueDays         int16  `cb:"6"` // due = entered + DueDays; 0 = same day
	AssignTo        int8   `cb:"7"` // 1 opportunity owner, 2 a fixed user
	AssignUserID    int32  `cb:"8"`
}
```

```go
type CrmTask struct {
	CompanyID      int32 `cb:"1"`
	ID             int32 `cb:"2"`
	OpportunityID  int32 `cb:"3"`
	PipelineID     int16 `cb:"4"`
	StageID        int16 `cb:"5"` // stage that generated it (approval: the target stage)
	Kind           int8  `cb:"6"` // 1 approval, 2 send quote, 3 send email, 4 activity
	TemplateID     int16 `cb:"7"`
	Title          string `cb:"8"`
	AssignedUserID int32 `cb:"9"`
	DueDate        int16 `cb:"10"` // UnixDay
	// 1 pending, 2 done, 3 dismissed, 4 approved, 5 rejected, 6 cancelled
	TaskStatus     int8   `cb:"11"`
	ResultNote     string `cb:"12"`
	ResultRefID    int64  `cb:"13"` // quote ID, sent email ID, activity ID
	Created        int32  `cb:"14"`
	DoneTime       int32  `cb:"15"`
	DoneBy         int32  `cb:"16"`
	Status, Updated, Version, UpdatedBy, Metadata
}
```

Index: delta subset pinned on `AssignedUserID` + an open/closed status key, so each user syncs
only their open tasks (closed ones leave the subset as removed IDs, like closed opportunities).

**Tasks page:** tabs *Mine* / *Team* (leader) / *Done*; grouped by due date (overdue, today,
upcoming). Executing a task opens the matching action, prefilled:
- **Approval** → approve / reject with note.
- **Send quote** → the quote editor for that opportunity (§7); the task is done when a quote is
  sent.
- **Send email** → the template rendered for that opportunity and client, editable before sending
  (§6); done when sent.
- **Activity** → the activity form; done when saved.

Executing a task always goes through the same endpoint as doing the action by hand, plus
`TaskID` in the payload, so the task is closed in the same transaction.

**As built (step 6, §4.2 and §5).** Task kinds 1 (approval), 3 (send email, added in step 7, see §6.2)
and 4 (activity) ship; kind 2 (send quote) waits for step 8, and `normalizeTaskTemplates` refuses it
until then. Code: `backend/crm/task.go`, `types/task.go`, `types/task_rules.go`;
`frontend/routes/crm/tasks/`. Differences from the text above:
- `CrmTask.CreatedBy` (cb 17) holds the requester, so only they can cancel an approval request.
- `GET.crm-task-ids` (records-by-ID cache) resolves the pending approval of a card from its
  `PendingApprovalTaskID` (the requester is not the assignee, so the delta list does not carry it);
  access 47 also gets it and `POST.crm-task`.
- Route names: `GET.crm-tasks` (`?user-id=`, `?closed=1`), `POST.crm-task` (dismiss / cancel request),
  `POST.crm-task-decide`. New access 51 "CRM Tareas" (`crm/tasks`); the activity endpoint accepts `TaskID`.
- Team tab: a leader reads their members' open tasks, read-only; only the assignee acts.
- A cross-pipeline move into a gated stage needs approval and has no skip rule. An approver who
  requests a move still goes through the task (no self-approval shortcut).
- Approving runs the move as the approver and does not need pipeline membership; rejecting writes a
  timeline note (approving only when a note is given).
- The pipeline editor (Settings) sets the approver and the activity templates of each stage.
- Not run in the real app. Needs a reseed (new `crm_tasks` table, new stage and opportunity columns).

When the opportunity **leaves the stage backward, is lost, or moves to another pipeline**, its
pending tasks from that stage become *cancelled* (open decision P3). Re-entering creates them again.

## 6. Email templates and sending

### 6.1 Sending (dependency on the virtual mailbox work)

`core.SendEmail` today sends plain HTML through SMTP with "Genix" as sender. The CRM needs a richer
message, and must not send real mail outside production. The CRM calls one function:

```go
core.SendEmail(message core.EmailMessage) error

type EmailMessage struct {
	To, Cc      []string
	ReplyTo     string // the seller's email
	FromName    string // the company's name
	Subject     string
	HTMLBody    string
	Attachments []EmailAttachment // {Name, ContentType, Content []byte}
	CompanyID   int32
	SourceTable string // "sale_quotes", "crm_opportunities", ...
	SourceID    int64
}
```

- **Production:** SMTP, as today.
- **Local / QAS:** the message is stored in the virtual mailbox table and nothing leaves the server.

The environment switch and the mailbox table are **built by another agent**. This plan only fixes
the message contract above; CRM work can start against it and the sign-up email moves to the same
signature. Every sent CRM email is also written to the opportunity's activity timeline
(`ActivityTypeEmail`), with the subject and the attachment names.

### 6.2 `CrmEmailTemplate` — `crm_email_templates`

`CompanyID, ID int16, Name, Subject, BodyHTML, Status, Updated, Version, UpdatedBy, Metadata`.
Edited in a new tab of **Diccionarios**. Placeholders, replaced by a pure function
`RenderEmailTemplate(template, context)` in `crm/types`, unit-tested:

`{{client.name}} {{client.email}} {{opportunity.title}} {{opportunity.amount}} {{owner.name}}
{{owner.email}} {{company.name}} {{quote.number}} {{quote.total}} {{quote.valid_until}}`

An unknown placeholder is a save-time validation error, never a silent blank in a sent email.

**As built (step 7).** Code: `backend/crm/email.go`, `types/email_template.go`, `types/email_template_rules.go`;
`frontend/routes/crm/dictionaries/EmailTemplatesTable.svelte`, `frontend/domain-components/crm/OpportunityEmailModal.svelte`.
- The virtual mailbox already existed (`core.SendEmail(companyID, EmailMessage)`, table `sent_emails`,
  `[smtp].store_only`, page `system/email-inbox`). The contract got `FromName` (the company name; default
  "Genix") and `ReplyTo` (the sending user's email); `sent_emails` stores `ReplyTo` (cb 13). No
  `SourceTable/SourceID`: the opportunity's Email activity is the link. Attachments wait for quotes (step 8).
- `crm_email_templates` (schema ID 90). Body edited with the genix-ui `HTMLEditor`; placeholders are
  typed as text (the editor lists them, a click copies one). Quote placeholders are accepted on save;
  rendering one without a quote is an error ("only sent with a quote") until step 8 supplies it.
- Routes: `GET.crm-email-templates`, `POST.crm-email-template` (access 50),
  `GET.crm-opportunity-email-draft?opportunity-id=&template-id=` (renders on the server; To = the
  client's email), `POST.crm-opportunity-email` (access 47 and 51). Anyone who can see the pipeline may
  send. The send is validated (addresses, ≤ 20 recipients, no raw `{{…}}` left) and the task checked
  **before** the email leaves; the Email activity and the task close are written after it, and a
  failure there says the email was sent instead of inviting a retry.
- Task kind 3 (send email) ships: a stage template carries `EmailTemplateID` (cb 8); the Tasks page
  opens the send modal with that template, and only a sent email closes the task (a hand-written
  activity cannot).
- The send action lives in the opportunity layer's Activity tab.

## 7. Quotes (`sales`)

### 7.1 `SaleQuote` — `sale_quotes`

```go
type SaleQuote struct {
	CompanyID     int32 `cb:"1"`
	ID            int32 `cb:"2"`
	Number        int32 `cb:"3"`  // per-company sequence, printed "COT-000123"
	OpportunityID int32 `cb:"4"`  // 0 = quote made from Ventas without an opportunity
	ClientID      int32 `cb:"5"`
	OwnerUserID   int32 `cb:"6"`
	Date          int16 `cb:"7"`  // UnixDay
	ValidUntil    int16 `cb:"8"`  // UnixDay
	CurrencyType  int8  `cb:"9"`  // as SaleOrder (open decision P10)
	ExchangeRate  int32 `cb:"10"`
	Lines         []SaleQuoteLine `cb:"11"`
	SubtotalAmount, DiscountAmount, TaxAmount, TotalAmount int32 `cb:"12..15"`
	Terms         string `cb:"16"` // payment / delivery terms shown on the PDF
	// 1 draft, 2 sent, 3 accepted, 4 rejected, 5 expired, 6 cancelled
	QuoteStatus   int8  `cb:"17"`
	IsPrimary     bool  `cb:"18"` // the one whose total feeds the opportunity amount
	SentTime      int32 `cb:"19"`
	DecisionTime  int32 `cb:"20"`
	SaleOrderID   int64 `cb:"21"` // set when converted
	Created       int32 `cb:"22"`
	CreatedBy     int32 `cb:"23"`
	Status, Updated, Version, UpdatedBy, Metadata
}

type SaleQuoteLine struct {
	ProductID      int32   `cb:"1"`
	PresentationID int16   `cb:"2"`
	Quantity       int32   `cb:"3"` // packed like SaleOrderLine
	SubDivisor     int16   `cb:"4"`
	Price          int32   `cb:"5"` // unit price at quote time, never re-read from the catalog
	DiscountPct    int16   `cb:"6"` // × 100
}
```

Indexes: local by `OpportunityID` (the opportunity's Quotes tab), delta subset by status for the
Ventas list, cache groups by month of `Date` for history.

**Totals are computed by the server** (`types.ComputeQuoteTotals`, pure, unit-tested) with the
same tax rules as `SaleOrder`. Client-sent totals are ignored.

### 7.2 Rules (`POST.sale-quote`)

- Client required and active; at least one line; every product exists and is active; quantity > 0;
  discount 0–100 %.
- `ValidUntil ≥ Date`.
- Only drafts can be edited. A sent quote is changed by **"new revision"**: the old one becomes
  cancelled and a copy is created as a draft (same `Number`, a revision suffix is open decision P13).
- Linked to an opportunity: the user needs Edit level on its pipeline (owner/creator rule of D8)
  — checked with the shared rules in `crm/types`, which `sales` may import.
- Marking a quote primary un-marks the others of the same opportunity and sets the opportunity
  `Amount` to the quote total, which writes an amount log row (§2).

### 7.3 PDF and sending (`POST.sale-quote-send`)

- **PDF:** generated **in the backend** with `gpdf` (E15), because the email must attach it and the
  client is not trusted with document content. Layout: company logo and data, client, lines table,
  totals, terms, validity, seller contact; page numbers in the footer. The builder lives in
  `sales/quote_pdf.go`; the data it prints is prepared by a pure function in `sales/types` so the
  content is unit-tested without rendering. Spanish text needs an embedded TrueType font (gpdf
  subsets it); verify accents and `ñ` render in the first spike.
- `GET.sale-quote-pdf?id=` → download.
- `POST.sale-quote-send {ID, To, Cc, EmailTemplateID, Subject, HTMLBody, TaskID}` → renders, attaches
  the PDF, calls `core.SendEmail`, sets status sent + `SentTime`, writes the activity, closes the
  task if present.

### 7.4 Accept / reject / convert

- `POST.sale-quote-status {ID, QuoteStatus, Note}`: accepted / rejected / cancelled by the seller.
  Expiry is computed (status sent and `ValidUntil` < today) and shown, not written by a job.
- **Accepted** → proposes closing the opportunity as **won** (one confirm, then the existing close
  rule from `crm/types`, logged). Rejecting a quote does not close the opportunity.
- **Convert to sale order**: opens the sales creation screen prefilled with client and lines; the
  created order stores nothing back except `SaleQuote.SaleOrderID`. Warehouse, payment and invoicing
  series are chosen there, as for any sale (E16). Prices come from the quote lines, not the
  current catalog.

### 7.5 Where it is shown

- `domain-components/quotes/` — `QuoteList.svelte`, `QuoteEditor.svelte`, `QuoteSendModal.svelte`:
  one implementation, used by both screens.
- **Ventas → Cotizaciones** (`sales/quotes`): all quotes, filters by status, seller, client, date.
- **CRM → Oportunidades** → opportunity layer → new **Cotizaciones** tab (quotes of that
  opportunity, "New quote" prefilled with the client).

### 7.6 Catalog services

E8 says product **and service** catalog, but `Product` has no service flag today. Proposed: add
`Product.IsService bool` — services skip stock and lots, and the quote / sale line shows no
presentation or sub-unit (open decision P8).

## 8. Reports

**As built:** the server serves the **raw** stage log through cache groups
(`GET.crm-report-stage-log?pipeline-id&period-start&period-end`, one calendar year per call, one
group per month and pipeline), and every report is computed **in the browser** by pure functions
in `routes/crm/reports/reports.ts`. Cache groups return records, not aggregates, and each row is
self-contained, so this needs no server aggregation code and no aggregate table. Closed months stay
in the browser; only the current one is downloaded again. Other routes:
the reportable pipelines (§8.4) are derived in the browser from the pipelines delta cache, and
today's open cards of one pipeline (the anchor of R2) are the board's own delta list
(`GET.crm-opportunities`, which also admits the full-report access).

### 8.1 The signed "reach" rule (R1)

Define, for one opportunity state *s* in pipeline *p*:

- `Reach(s)` = +1 (and +amount) on every position 1…`s.Position`, if open;
  on every position 1…`s.Position` plus the **Won** column, if won (E17: the stage it was won
  from, never the stages after it);
  **only the Lost column**, if lost (E4).

A won or lost opportunity keeps the `StageID` / position it was closed from, so `s.Position` of a
closed state is that stage. The Won column is also split by that position ("won from"):

| Seller | Pos 1 | Pos 2 | Pos 3 | Pos 4 | Won | Won from pos 2 | Won from pos 4 |
|---|---|---|---|---|---|---|---|
| A — won at stage 2 | 1 | 1 | | | 1 | 1 | |
| B — won at stage 4 | 1 | 1 | 1 | 1 | 1 | | 1 |

Every log row contributes `Reach(after)` to the **after-owner** in the **after-pipeline**, and
`−Reach(before)` to the **before-owner** in the **before-pipeline**.

Summed over any period, this telescopes to *(state at the end) − (state at the start)* per seller and
pipeline, so an opportunity that goes back and forth contributes only its net progress:

| Day | Change | Pos 2 | Pos 3 | Pos 4 |
|---|---|---|---|---|
| 1 | 1 → 4 | +1 | +1 | +1 |
| 2 | 4 → 3 | | | −1 |
| 3 | 3 → 4 | | | +1 |
| | **Net** | **1** | **1** | **1** |

The same formula covers every case without special code:
- **Forward / backward move:** positions between the two get + / −.
- **Pipeline transfer (E3):** −all reached positions in the source, + in the target.
- **Owner change:** − for the old seller, + for the new one.
- **Amount change:** the amount columns shift, the counts don't.
- **Won / reopen:** + / − on the Won column (and its "won from" split) only; the reached
  positions don't change, because closing does not move the stage.
- **Lost (E4):** − on all reached positions, + on Lost.

Position 1 = "entered the pipeline", so column 1 is net new opportunities. Columns are positions
(funnel depth), labelled with the stage currently at that position (E2).

Pure function `stageReachDeltas(row, pipelineID)` in `routes/crm/reports/reports.ts`, with tests
for each case above in `reports.test.ts`.

### 8.2 Report list

| ID | Report | Shape | Source |
|---|---|---|---|
| **R1** | **Net progress by seller** — year × seller × position, counts and amounts; drill month → day | Pivot table, + / − colored; a stacked bar per month | §8.1 |
| **R2** | **Pipeline state by day** — how many opportunities (and how much) sit in each stage at the end of each day, per seller/team | Stacked bars per day + start/end table | Occupancy by **stage ID** (a reorder doesn't change who sits where): −1 at the before-stage if it was open, +1 at the after-stage if open. Rebuilt **backwards** from today's open opportunities, undoing each day's deltas, so it needs the log from the first shown day to today |
| **R3** | **Pipeline waterfall** — opening open amount + new + increased − decreased − won − lost ± transferred = closing, per month and seller | Waterfall chart + table | Kind and amount fields of each row |
| **R4** | **Funnel and time in stage** — of the opportunities created in a range, how many reached each position (max ever, from the raw log, so lost deals count where they died); median / average days per stage | Funnel bars + box plot or table | Stage log by opportunity, consecutive rows |
| **R5** | **Seller scorecard** — win rate, average won amount, cycle length (created → won), velocity = open count × win rate × avg won amount ÷ cycle days, activities, goal attainment | Leaderboard table, team subtotals | Log + activities + goals |
| **R6** | **Forecast vs goal vs actual** — per month, seller/team/pipeline: goal, last submitted forecast (and its evolution), computed forecast today, won | Bullet chart per seller + line of submissions | Goals + forecast submissions + log |
| **R7** | **Lost reasons** — by reason, seller, stage where lost | Pareto bar | Log rows with Outcome lost + opportunity `LostReasonID` |

Common filters: pipeline, team, seller, year/month range. Which pipelines a user can report on is
§8.4.

**As built (R3–R7), decided 2026-11-09:**
- **R3** adds a **Reopened** step, so the formula is opening + new + increased − decreased − won −
  lost + reopened ± transferred = closing. Each row is split into steps that sum exactly to its R2
  occupancy delta; a month's closing is today's open deals with every later row undone, its opening
  is the closing minus its steps. An owner change is transferred out/in between sellers.
- **R4** cohort = opportunities that **entered** the pipeline in the range (created in it or
  transferred in), so their whole history from entry is loaded. Time in stage counts **finished
  stays only**; a deal still sitting in a stage today is left out.
- **R5** cycle length uses only won deals whose creation row is loaded (created since the range's
  start year); the rest count in win rate but not in cycle. Win rate = won ÷ (won + lost), net of
  reopenings. Team rows add their members' raw sums and derive their own rates. Activities come
  from a new cache-groups index on `crm_activity` by `(Period, PipelineID)` — both columns are
  set by the server on insert — served by `GET.crm-report-activities` (access 52), with text and
  files stripped.
- **R6** lives on the Forecast page: a "submitted before start" column (P7) and, on row click, a
  line of every submission's commit and best case against the goal.
- **R7** counts a deal once when its **last state in the range** is lost, with the reason of the
  row that lost it (a reopened deal was not lost).

**Later (not in this delivery):** slippage (close date pushed — the log already carries
`From/ToExpectedCloseDate`, so this only needs a report), quote conversion and time to quote,
activity without next step.

### 8.3 Frontend

```
routes/crm/reports/
  +page.svelte            filters + top tabs: Avance · Estado diario · Cascada · Embudo ·
                          Vendedores · Motivos de pérdida
  ReportNetProgress.svelte  ReportDailyState.svelte  ReportWaterfall.svelte
  ReportFunnel.svelte       ReportScorecard.svelte   ReportLostReasons.svelte
  reports.svelte.ts       filters state + service calls
  reports.ts              pure pivots: month/day roll-ups, team roll-up, velocity, attainment
  reports.test.ts
routes/crm/forecast/      +page.svelte, forecast.svelte.ts, forecast.ts, forecast.test.ts (R6 lives here)
routes/crm/tasks/         +page.svelte, TaskExecuteLayer.svelte, tasks.svelte.ts, tasks.ts
routes/crm/teams/         +page.svelte, teams.svelte.ts
routes/crm/goals/         +page.svelte, goals.svelte.ts, goals.ts
routes/sales/quotes/      +page.svelte, quotes.svelte.ts
domain-components/quotes/ QuoteList, QuoteEditor, QuoteSendModal, quote.ts (totals mirror)
services/crm/             crm-team, crm-goal, crm-forecast, crm-task, crm-email-template, crm-reports
services/sales/           sale-quote.svelte.ts
```

Charts follow the `dataviz` skill.

### 8.4 Report visibility

Two layers, like the rest of the CRM:

1. **Page access** (`access.toml`): *CRM Reportes* (52) opens the three report pages
   `crm/reports/flow`, `crm/reports/conversion` and `crm/reports/performance`, *CRM Forecast* (53)
   opens `crm/forecast`. Without it the page is not in the menu and its routes are refused.
2. **Data scope**, decided inside the report handlers by one pure function,
   `types.ReportablePipelineIDs(user grants, pipelines)`:
   - The user holds **CRM Reporte completo** (57) → every active pipeline of the company. A deleted
     pipeline leaves the reports (it cannot be deleted while it holds open opportunities).
   - Otherwise → only the pipelines where the user is a member (any level, §2 of CRM_PLAN).

   A requested pipeline outside that set is an error, never an empty report. The pipeline filter
   on the page lists only the reportable pipelines. Access 57 has no route or API of its own: the
   handlers read it from the user's grants with `core.DecodeAccesosGrants`, as
   `projects/project_access.go` (`canWorkInProjects`) does.

The same scope applies to the Forecast page and to goals shown in R5/R6. Writing a goal still
needs *CRM Metas* (55).

## 9. Access (`backend/access.toml`, group 10 unless noted)

| id | name | frontend_routes | backend_apis |
|---|---|---|---|
| 51 | CRM Tareas | `crm/tasks` | `POST.crm-task`, `POST.crm-task-decide` |
| 52 | CRM Reportes | `crm/reports/flow`, `crm/reports/conversion`, `crm/reports/performance` | `GET.crm-report-stage-log`, `GET.crm-report-activities` |
| 53 | CRM Forecast | `crm/forecast` | `POST.crm-forecast-submission` |
| 54 | CRM Equipos | `crm/teams` | `POST.crm-team` |
| 55 | CRM Metas | `crm/goals` | `POST.crm-goal` |
| 56 | Cotizaciones (sales group) | `sales/quotes` | `POST.sale-quote`, `POST.sale-quote-send`, `POST.sale-quote-status` |
| 57 | CRM Reporte completo | — | — (read by the report handlers, §8.4) |

`POST.crm-email-template` joins access 50 (Diccionarios). Opportunity users (47) also need the
quote APIs to work from the opportunity layer: they get `POST.sale-quote*` in their list too.

Menus (`db/modules.toml`): Clientes = `crm/customers, crm/pipeline, crm/tasks`; new
**Reportes** = `crm/reports/flow, crm/reports/conversion, crm/reports/performance, crm/timeline, crm/forecast`
(Flujo, Conversión, Desempeño); Gestión adds `crm/teams, crm/goals`.

## 10. Implementation order

1. **Stage log v2** (§2): struct, `BuildStageLogRow`, write on every change kind, Period cache-group
   index, reseed sample data with a year of realistic moves (including back-and-forth, transfers,
   owner changes).
2. **Reach and occupancy functions** + tests (§8.1), then **R1, R2** endpoints and pages.
3. **Teams, goals, forecast category, forecast submissions** → **R5, R6**, Forecast page.
4. **R3, R4, R7.**
5. **Required fields per stage** (§4.1).
6. **Tasks table + stage templates + Tasks page**, then **approvals** on top (§4.2, §5).
7. **Email message contract + templates** (§6) — sending works against the virtual mailbox once
   the other agent delivers it.
8. **Quotes** (§7): table, totals, editor, PDF, send, accept/convert; CRM tab and Ventas page.
9. `agent-browser` checks: back-and-forth moves net to 1 in R1; an approval gate blocks the move
   until approved; a stage's tasks appear in the assignee's Tasks page; a quote sent in QAS lands in
   the virtual mailbox with its PDF.
10. `DOCUMENTATION.md` per route; `crm/RATIONALE.md` entries for the self-contained log, the
    signed reach rule and tasks-instead-of-automations.

## 11. Decisions P1–P13

All accepted as recommended on 2026-11-09; P6, P9, P11 and P14 were answered separately (E14–E17).

| # | Question | Recommendation |
|---|---|---|
| P1 | Can a forward move **skip** a stage that requires approval? | No — reject and require passing through it. |
| P2 | Does moving **backward** into a gated stage need approval? | No. |
| P3 | What happens to pending tasks when the opportunity leaves the stage (backward, lost, other pipeline)? | Cancel them; re-entering recreates them. |
| P4 | Goal metrics. | Won amount, won count, new opportunities, activities. |
| P5 | Can a user belong to more than one team? | No — one team per user keeps team roll-ups free of double counting. |
| P7 | Forecast: can the team leader adjust the team's forecast, and is there a cut-off date? | Leader submits a team row (adjustable); no cut-off — the report shows the last submission before the month started and the latest. |
| P8 | Services in the catalog: add `Product.IsService`? | Yes. |
| P10 | Quote currency and taxes: PEN + USD with the same IGV rules as `SaleOrder`? | Yes, same as sales. |
| P12 | Email sender: "Company name <platform address>" with Reply-To = seller? | Yes; per-company SMTP later. |
| P13 | Revisions: same number with a suffix (COT-000123-2) or a new number? | Same number + suffix. |
