# CRM Plan — Pipelines, Opportunities, Custom Fields, Access

**Status: proposal.** Nothing here is implemented.

## 0. Summary

The CRM is built around user-defined **pipelines**. A pipeline (`CrmPipeline`) holds ordered
**stages** (`CrmPipelineStage`), and each pipeline is shown as its own kanban board. An
**opportunity** (`CrmOpportunity`) sits in exactly one pipeline and one stage at a time. It can
move between stages and between pipelines, and every move is written to a stage log.

Everything that defines a pipeline is configured by the user:

The CRM menu has a **Gestión** (management) group with three pages:

- **Flujos** (`/crm/settings`): pipelines and their stages, custom fields.
- **Flujos: Accesos** (`/crm/pipeline-access`): which users can see or work on which pipeline,
  and at what level.
- **Diccionarios** (`/crm/dictionaries`): value lists the CRM picks from (today: lost reasons),
  edited inline in the table, no side layer.

Out of scope for this plan (later phases, §10): client contacts, activities/agenda, receivables,
quotes, campaigns.

## 1. Decisions taken

| # | Decision |
|---|---|
| D1 | Names: `CrmPipeline` / `CrmPipelineStage`. The `Crm` prefix avoids collisions with `projects`. |
| D2 | **Won and lost are an outcome on the opportunity, not stages.** The kanban shows only open stages; each card has Won / Lost actions. Lost asks for a reason from a configurable list. This keeps reports comparable across pipelines that users name freely. |
| D3 | A stage has: name, color, default probability (%), and optional "days before stale". |
| D4 | **Stages are embedded in the pipeline record** with a stable `ID` each. Reordering is one atomic save, and one delta cache carries the whole configuration. |
| D5 | **A stage that still holds opportunities cannot be deleted.** The same applies to a pipeline. Both are soft-deleted (`Status`) once empty, so history keeps its labels. |
| D6 | **Every stage or pipeline change goes to `CrmOpportunityStageLog`.** It feeds conversion and time-in-stage reports. |
| D7 | An opportunity **can move between pipelines.** |
| D8 | **Access per pipeline is assigned per user** on a dedicated page. **Only the opportunity's creator, or a user with Full level on that pipeline, can move it.** |
| D9 | **Users can define custom fields**, rendered from an array of field definitions with genix-ui inputs. |

## 2. Access model — two layers

**Layer 1 — the catalog (`backend/access.toml`)** decides who can open the pages and call the
routes. Three new accesses in group 10 (CRM):

| id | name | frontend_routes | backend_apis |
|---|---|---|---|
| 47 | CRM Oportunidades | `crm/pipeline` | `POST.crm-opportunity`, `POST.crm-opportunity-move`, `POST.crm-opportunity-close` |
| 48 | CRM Flujos | `crm/settings` | `POST.crm-pipeline`, `POST.crm-custom-field` |
| 49 | CRM Flujos: Accesos | `crm/pipeline-access` | `POST.crm-pipeline-members` |
| 50 | CRM Diccionarios | `crm/dictionaries` | `POST.crm-lost-reason` |

**Layer 2 — pipeline membership (data)** decides what a user can do *inside* each pipeline. It is
stored as `CrmPipeline.Members` (§3.1). Levels:

| Level | Value | Can do |
|---|---|---|
| View | 1 | See the board and open opportunities. |
| Edit | 2 | View, plus create opportunities, and edit / move / close **the ones they created**. |
| Full | 3 | Edit / move / close **any** opportunity in the pipeline. |

A user who is not a member does not see the pipeline: its opportunities are never returned to
them, and it is missing from the selector.

**Rules for moving an opportunity** (pure functions in `crm/types/pipeline_rules.go`, unit-tested):

- **Within a pipeline:** the user is the creator with level ≥ Edit, or has level Full.
- **To another pipeline:** the same rule on the source pipeline, **and** level ≥ Edit on the target
  pipeline. Without the target check, a user could move an opportunity into a pipeline they cannot
  work in.
- **Closing (won/lost) or reopening:** the same rule as moving within a pipeline.

The frontend repeats these checks in `pipeline.ts` only to disable dragging and buttons. The
backend check is the one that counts.

## 3. Tables

All tables are partitioned by `CompanyID`. Table IDs are claimed at implementation time
(see `docs/TABLE_ID_ASSIGNMENT.md` — don't pick one by searching the source with a regex).
Dates are `UnixDay int16`, datetimes `SUnixTime int32`, and amounts use the same scale as
`SaleOrder` amounts.

### 3.1 `CrmPipeline` — `crm_pipelines`

```go
type CrmPipeline struct {
	db.TableStruct[CrmPipelineTable, CrmPipeline]
	CompanyID   int32  `db:",pk" cb:"1"`
	ID          int16  `db:",pk" cb:"2"`
	Name        string `cb:"3"`
	Description string `cb:"4"`
	Color       string `cb:"5"`
	SortOrder   int16  `cb:"6"`
	// Board columns, in display order. Stage IDs are stable and allocated by the server
	// (max+1 inside the pipeline); opportunities and the stage log reference them.
	Stages []CrmPipelineStage `cb:"7"`
	// Custom fields this pipeline shows, in display order (§5).
	Fields []CrmPipelineField `cb:"8"`
	// Written ONLY by POST.crm-pipeline-members, never by POST.crm-pipeline, so the
	// settings page and the access page cannot overwrite each other.
	Members   []CrmPipelineMember `cb:"9"`
	Status    int8  `json:"ss" cb:"ss,10"`
	Updated   int32 `json:"upd" cb:"upd,11"`
	UpdatedBy int32 `cb:"12"`
}

type CrmPipelineStage struct {
	ID          int16  `cb:"1"`
	Name        string `cb:"2"`
	Color       string `cb:"3"`
	Probability int8   `cb:"4"` // default % for opportunities entering the stage
	StaleDays   int16  `cb:"5"` // 0 = never flagged
	Status      int8   `cb:"6"` // soft delete; blocked while the stage holds opportunities
}

type CrmPipelineField struct {
	FieldID  int16 `cb:"1"`
	Required bool  `cb:"2"`
}

type CrmPipelineMember struct {
	UserID int32 `cb:"1"`
	Level  int8  `cb:"2"` // 1 view, 2 edit, 3 full
}
```
Indexes: one delta view on `Updated`. The table is small (a handful of rows per company), so the
whole thing syncs to every CRM user.

### 3.2 `CrmOpportunity` — `crm_opportunities`

```go
type CrmOpportunity struct {
	db.TableStruct[CrmOpportunityTable, CrmOpportunity]
	CompanyID         int32  `db:",pk" cb:"1"`
	ID                int32  `db:",pk" cb:"2"`
	PipelineID        int16  `cb:"3"`
	StageID           int16  `cb:"4"`
	Title             string `cb:"5"`
	ClientID          int32  `cb:"6"`  // ClientProvider; see open decision O1
	OwnerUserID       int32  `cb:"7"`  // assigned sales rep
	Amount            int32  `cb:"8"`
	Probability       int8   `cb:"9"`  // starts at the stage default, user can override
	ExpectedCloseDate int16  `cb:"10"` // UnixDay
	Outcome           int8   `cb:"11"` // 0 open, 1 won, 2 lost
	LostReasonID      int16  `cb:"12"`
	LostNote          string `cb:"13"`
	ClosedTime        int32  `cb:"14"` // SUnixTime
	StageEnteredTime  int32  `cb:"15"` // SUnixTime; drives the "stale" flag
	// Where the opportunity was before its last move, denormalized from the stage log so the
	// card can show "from <pipeline> · <stage>" without reading the log. 0 = never moved.
	PreviousPipelineID int16 `cb:"16"`
	PreviousStageID    int16 `cb:"17"`
	CustomValues []CrmCustomFieldValue `cb:"18"`
	Created   int32 `cb:"19"`
	CreatedBy int32 `cb:"20"` // authority for the move rule (D8)
	Status    int8  `json:"ss" cb:"ss,21"`
	Updated   int32 `json:"upd" cb:"upd,22"`
	UpdatedBy int32 `cb:"23"`
}
```
Indexes:
- **Open board:** a delta view pinned on `PipelineID` (plus `Outcome = open`), so each board
  syncs only its own cards.
- **History:** closed opportunities queried by `ClosedTime` month, through cache groups.

> To verify at implementation (delta-cache skill): an opportunity that **leaves** a board's subset
> (closed, or moved to another pipeline) must be evicted from that board's cache. If the delta
> subset does not handle eviction natively, the pinned key must include `Outcome`, and a move
> must leave a removal in the source subset.

### 3.3 `CrmOpportunityStageLog` — `crm_opportunity_stage_log`

Append-only, one row per change: created, stage move, pipeline move, won, lost, reopened.

```go
type CrmOpportunityStageLog struct {
	CompanyID      int32 `db:",pk" cb:"1"`
	ID             int32 `db:",pk" cb:"2"`
	OpportunityID  int32 `cb:"3"`
	FromPipelineID int16 `cb:"4"`
	FromStageID    int16 `cb:"5"`
	ToPipelineID   int16 `cb:"6"`
	ToStageID      int16 `cb:"7"`
	Outcome        int8  `cb:"8"`  // outcome after the change
	Time           int32 `cb:"9"`  // SUnixTime
	UserID         int32 `cb:"10"`
}
```
Indexes: by `OpportunityID` (history in the opportunity panel) and by `Time` (reports, cache
groups by month). Written in the **same transaction** as the opportunity update.

### 3.4 `CrmCustomField` — `crm_custom_fields`

The custom-field catalog is **company-wide**. Each pipeline picks which fields it shows. This is
what makes moving between pipelines (D7) safe: a value is keyed by a company-unique `FieldID`, so
it survives the move. The value is shown again if the target pipeline includes that field, and
reappears if the opportunity comes back.

```go
type CrmCustomField struct {
	CompanyID int32  `db:",pk" cb:"1"`
	ID        int16  `db:",pk" cb:"2"`
	Name      string `cb:"3"`
	Type      int8   `cb:"4"` // §5
	Options   []CrmCustomFieldOption `cb:"5"` // select / multi-select only
	Status    int8  `json:"ss" cb:"ss,6"`
	Updated   int32 `json:"upd" cb:"upd,7"`
}

type CrmCustomFieldOption struct {
	ID     int16  `cb:"1"` // stable, server-allocated
	Name   string `cb:"2"`
	Color  string `cb:"3"`
	Status int8   `cb:"4"`
}

// One value per field on the opportunity. Only the column matching the field's type is set.
type CrmCustomFieldValue struct {
	FieldID   int16   `cb:"1"`
	Number    int64   `cb:"2"` // integer, decimal (scaled), money, date (UnixDay), checkbox (0/1), user ID, single option ID
	Text      string  `cb:"3"` // text, long text
	OptionIDs []int16 `cb:"4"` // multi-select
}
```

### 3.5 `CrmLostReason` — `crm_lost_reasons`

`CompanyID, ID int16, Name, Status, Updated`. It's a small catalog synced with the delta cache.
(Alternative: put it in the company config blob. It is a separate table here so the settings page
treats it like every other catalog.)

## 4. Backend

```
backend/crm/
  pipeline.go             GET.crm-pipelines (delta), POST.crm-pipeline, POST.crm-pipeline-members
  opportunity.go          GET.crm-opportunities (delta per pipeline), GET.crm-opportunities-closed,
                          POST.crm-opportunity, POST.crm-opportunity-move, POST.crm-opportunity-close,
                          GET.crm-opportunity-log
  custom_field.go         GET.crm-custom-fields, POST.crm-custom-field
  lost_reason.go          GET.crm-lost-reasons, POST.crm-lost-reason
  types/
    pipeline.go  opportunity.go  stage_log.go  custom_field.go  lost_reason.go
    pipeline_rules.go       member level lookup, CanEditOpportunity, CanMoveOpportunity
    custom_field_rules.go   ValidateCustomValues (type match, option exists, required-by-pipeline)
    *_test.go
```

Validations (never trust the client):

**`POST.crm-pipeline`**
- Name is required.
- At least one active stage.
- Stage probability is between 0 and 100.
- New stages get server IDs; client-sent IDs must already exist in the stored record.
- An existing stage that is missing or soft-deleted in the payload is rejected if open
  opportunities still sit on it (D5).
- Field IDs must exist in `CrmCustomField`.
- `Members` from the payload is ignored; the stored value is kept.

**`POST.crm-pipeline-members`**
- Payload is `{PipelineID, Members}`.
- Users must exist and be active.
- Levels must be 1–3.
- No duplicate users.

**`POST.crm-opportunity`** (create or edit)
- Required: title, client, pipeline, stage.
- The user needs level ≥ Edit; editing someone else's opportunity requires Full.
- `CustomValues` are checked against the field catalog, and the target pipeline's required fields
  must be present.
- The stage cannot be changed through this route. Stage changes go through `move`, so they are
  logged.

**`POST.crm-opportunity-move`**
- Payload is `{ID, PipelineID, StageID}`.
- The rule in §2 is applied.
- The target stage must be active.
- If the target pipeline requires fields the opportunity doesn't have, the move is rejected with
  the list of missing fields; the UI then opens the form.
- Copies the current `PipelineID` / `StageID` into `PreviousPipelineID` / `PreviousStageID`,
  sets the new ones and `StageEnteredTime`, resets `Probability` to the stage default, and
  writes the log, all in one transaction. Applies to stage-only moves too.
- A move to the same pipeline and stage is rejected, so `Previous*` never equals the current
  position.

**`POST.crm-opportunity-close`**
- Payload is `{ID, Outcome, LostReasonID, LostNote}`.
- A lost reason is required when the outcome is lost.
- Reopening (outcome 0) is allowed with the same permission.
- Writes the log.

**`POST.crm-custom-field`**
- A field's `Type` cannot change once any opportunity holds a value for it.
- Options are soft-deleted, never removed.

## 5. Custom fields — types and the generic form

| Type | Value | genix-ui input |
|---|---|---|
| 1 Text | `Text` | `Input` |
| 2 Long text | `Text` | `Input` (multiline) |
| 3 Integer | `Number` | `Input` type number |
| 4 Decimal | `Number` (×100) | `Input` type number |
| 5 Money | `Number` (amount scale) | `Input` type number |
| 6 Date | `Number` (UnixDay) | `DateInput` |
| 7 Select | `Number` (option ID) | `SelectBox` / `SearchSelect` |
| 8 Multi-select | `OptionIDs` | `SelectedTags` / `CheckboxOptions` |
| 9 Checkbox | `Number` 0/1 | `Checkbox` |
| 10 User | `Number` (user ID) | `SearchSelect` over users |

genix-ui has every input above, but **no component that renders a form from an array of field
definitions** (searched: nothing like it exists). This plan adds one:

- **`packages/genix-ui/form/FieldsForm.svelte`** — generic, no business logic. Props:
  `fields: { key, label, type, options?, required? }[]`, `values`, and `onchange`. It maps each
  `type` to the input in the table above. It goes in genix-ui because it is generic, so it must be
  committed and pushed inside the submodule.
- **`routes/crm/pipeline/pipeline.ts`** — `buildOpportunityFormFields(pipeline, customFields)`
  converts the CRM definitions into `FieldsForm` fields, and `readCustomValues` /
  `writeCustomValues` convert between the generic `values` map and `CrmCustomFieldValue[]`.

## 6. Kanban component

genix-ui has no kanban or board component either. This plan adds
**`packages/genix-ui/layers/Kanban.svelte`** (or `navigation/` — wherever the library puts boards):

- Columns and cards come in as props; each card is rendered with a snippet.
- An `onmove(cardID, toColumnID)` callback.
- A `canDrag(card)` predicate, so cards the user cannot move are not draggable.
- Pointer-based drag so it works on touch. On mobile it also shows a "move to…" menu, because
  dragging across horizontally scrolled columns is unreliable.

## 7. Frontend

```
services/crm/
  crm-pipeline.svelte.ts        pipelines, custom fields, lost reasons (delta caches)
  crm-opportunity.svelte.ts     open opportunities per pipeline (delta), closed (cache groups)

routes/crm/pipeline/            ← the board
  +page.svelte                  pipeline selector (remembers the last one) + Kanban
  OpportunityCard.svelte        title, client, amount, owner, stale flag, "from <pipeline> ·
                                <stage>" when Previous* is set
  OpportunityLayer.svelte       side Layer: main fields + FieldsForm + stage history
  OpportunityCloseModal.svelte  won / lost + reason
  OpportunityMoveModal.svelte   move to another pipeline + stage
  pipeline.svelte.ts            board state, service calls
  pipeline.ts                   pure rules: permissions mirror, form-field build, column totals,
                                stale detection, weighted amount
  pipeline.test.ts

routes/crm/settings/
  +page.svelte                  tabs: Flujos (pipelines) · Custom fields
  PipelineEditor.svelte         LayerStatic: name, color, stages (drag reorder, color,
                                probability, stale days), fields (pick + required + order)
  CustomFieldEditor.svelte      name, type, options
  settings.svelte.ts  settings.ts  settings.test.ts

routes/crm/pipeline-access/
  +page.svelte                  grid: users (rows) × pipelines (columns), each cell a level
                                select (none / view / edit / full); saves per pipeline
  pipeline-access.svelte.ts  pipeline-access.ts
```

Each route also gets a `DOCUMENTATION.md` (`document-user-routes` skill) once implemented.
Menu: the three routes are added under the **Clientes (CRM)** module, next to `/crm/customers`.

## 8. Implementation order

1. Tables + `crm/types` rules + tests; `access.toml` entries 47–49.
2. Pipeline and custom-field and lost-reason endpoints → settings page.
3. Pipeline members endpoint → pipeline-access page.
4. `FieldsForm` and `Kanban` in genix-ui (submodule commit + push).
5. Opportunity endpoints + stage log → pipeline board.
6. Sample data in `tests/sample_records` (two pipelines, a few dozen opportunities, logs).
7. `agent-browser` end-to-end check: create a pipeline, assign members, check that a View user
   cannot drag, that an Edit user can drag only their own cards, and that a move to a pipeline
   with required fields asks for them.
8. `crm/RATIONALE.md` entries for D2, D4, D8 and the company-wide field catalog.

## 9. Open decisions

| # | Question | Recommendation |
|---|---|---|
| O1 | **Prospects.** Is a lead with no purchase yet a `ClientProvider` row (with a prospect flag), or a free-text name on the opportunity? | A `ClientProvider` row with a prospect flag: `ClientID` stays mandatory, and winning needs no data move. Adds one column to `ClientProvider`. |
| O2 | **The assigned owner** (`OwnerUserID`) is often not the creator. Should the owner also be allowed to move the opportunity? D8 currently says creator only. | Allow creator **or** owner, so reassigning a lead actually hands it over. |
| O3 | **Who can change `OwnerUserID`?** | Level Full only. |
| O4 | **Users with no membership:** is it OK that a pipeline must be explicitly assigned before anyone, including admins, sees it? | Yes. The creator of a pipeline is added as Full automatically. |
| O5 | **Currency:** are opportunity amounts always in the company currency, or can they carry USD like sales? | Company currency only for now. |

## 10. Later phases (not designed here)

- **Contacts:** `ClientContact`, the people at a company client.
- **Activities:** `CrmActivity` for calls, visits, meetings and tasks, linked to a client and/or an
  opportunity; a "my agenda" page.
- **Client file:** tabs in `/crm/customers` for summary, sales (from `SaleClientProductFrame`),
  debt, contacts, opportunities and timeline.
- **Receivables and credit limit:** aging built from `SaleOrder.DebtAmount` / `PaymentDueDate`,
  and a credit check in the POS.
- **Won opportunity → quote / sale order.**
- **Reports:** funnel conversion, time in stage, win rate per rep, forecast (amount × probability by
  expected close month), all built from the stage log.
- **Agent:** client summary and next-best-action.
