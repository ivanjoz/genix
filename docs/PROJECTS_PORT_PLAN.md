# Projects module — port from Berryapps (DynamoDB) to Genix (ScyllaDB)

Source: `/home/ivanjoz/projects/berryapps` — `backend/projects/**` (~3.9k Go lines, 28 routes),
`frontend/routes/projects/**` (~9.3k lines, 85 files), `db/projects.access.toml`.

Both repos share the same `genix-orm`. Berryapps' `app/db` re-exports the **dynamo** driver
(`db.Model` / `db.Schema` / `db.NewRepo` / `Entity` / `Slot G1`), and Genix's `app/db` re-exports
the **scylla** driver (`db.TableStruct` / `db.TableSchema` / `db.Query(&recs)`). So "converting from
DynamoDB to Scylla" means rewriting **the table declarations and every repo call**. The business rules,
the tests and most of the frontend carry over almost as they are.

---

## 1. The real gap: Berryapps relies on optimistic concurrency, which Genix's Scylla driver lacks

| Berryapps (dynamo driver) | Used for | Genix today |
|---|---|---|
| `Repo.Modify(key, fn)`: consistent read, then a write conditioned on `ver`, retried ×8 | every edit; in-row counters (`Story.LastActivityID`, `CommentsCount`, `Feedback.LastCommentID`); the `Sprint.Stories` nested list | none |
| `GetManyForUpdate` + `PutManyIfVersion` → 409 | the sprint board card move (`POST.projects-sprint-move`) | none |
| `Query().Consistent()` | reading every sprint of a project before it recomputes each story's final state | Scylla reads at QUORUM, so this is effectively already covered |
| `db.ReserveIDs("projects-story-number-<pid>", n)` | per-project story `Number` | no named sequences; fareward reserves per-table IDs |
| Entity-wide autoincrement (story IDs unique across projects) | `StoryAssignee` keyed by StoryID alone | fareward autoincrement (per table, effectively global) ✔ |
| `Query().Contains(Users.ModuleIDs, 6)` | team candidates | `GET.users` + access check |
| GSI `Slot G1` on Stories (AssigneeID, Updated) | nothing; no code queries it | drop it |

**Proposed replacement (Decision A):** serialize writes per project with the existing fareward
lock: `core.AcquireLock(ctx, core.ActionProjectWrite, projectID, maxWaiters)`, the pattern in
`backend/sales/sale_order_annul.go:57`. Each `Modify` becomes **lock → `db.Query` read → apply the
pure rule → `db.Update`/`Insert` → unlock**. The board no longer needs the 409 path, because moves
are serialized rather than rejected.
- Alternative: add `Modify` to `genix-orm/scylla` with an LWT (`UPDATE … IF version = ?`) and a
  retry loop. That keeps Berryapps' semantics but touches the shared ORM submodule, and LWT costs
  about 4 round trips per write. Not recommended for a write rate of a few people on one board.

**Story `Number` (Decision B):** the story ID is a per-project autoincrement: `KeyIntPacking`
`[ProjectID.Size(40), Autoincrement]` with `AutoincrementPart: ProjectID`. The low 23 bits are the
story's number in its project (`types.StoryNumber`), so there is no `Number` column and no `ReserveIDs`.

## 2. Table conversion

Every table: `db.TableStruct[XTable, X]`, columns `db.Col[*XTable, T]` / `db.ColSlice`, a
hand-assigned `ID` (57+ looks free; confirm with `cd scripts && go run . check_tables`),
`Partition: CompanyID` (tenancy), `FixedValues` for `Status {0,1}` and for every pinned delta key
(a `Min/Max` that sets its bit slot), `Version`/`Metadata`/`Updated`/`Status` fields. The `cb` tags
are kept. `UpdatedMillis` is dropped, since it is Berryapps migration debt.

| Berryapps entity | Berryapps keys | Genix proposal |
|---|---|---|
| `projects_project` | no partition, `ID` | part `CompanyID`, key `ID.Autoincrement`, delta `(Status)` |
| `projects_epic` | part `ProjectID`, key `ID` | part `CompanyID`, key `ID.Autoincrement`, `ProjectID` column, delta pinned `(ProjectID, Status)` |
| `projects_story` | part `ProjectID`, key `ID`, CacheByIDs, G1 | same as epic + `CacheByIDs: true`; G1 dropped |
| `projects_story_assignee` | key `StoryID`, delta pinned `(AssigneeID, Status)` | part `CompanyID`, key `StoryID`, delta pinned `(AssigneeID, Status)`. The ORM must emit the tombstone when AssigneeID changes: **verify the scylla delta log does this the way dynamo does** |
| `projects_story_detail` | part `ProjectID`, key `StoryID`, delta pinned StoryID | part `CompanyID`, key `StoryID`, delta pinned `(StoryID, Status)` |
| `projects_story_activity` | part `ProjectID`, keys `(StoryID, ID)` | part `CompanyID`, keys `(StoryID, ID)`, delta pinned `(StoryID, Status)`; ID from `Story.LastActivityID` under the lock |
| `projects_sprint` | part `ProjectID`, key `ID` | like epic; `Stories []SprintStory` stays a struct-slice column |
| `projects_document` | part `ProjectID`, key `ID` | like epic |
| `projects_feedback` / `_comment` | part `ProjectID` | **out of scope** (section 7) |

Open technical check: a pinned `ProjectID`/`StoryID` takes a bit slot in the packed delta view key.
Size it (for example `Max: 1<<24`) and confirm the packing fits next to `Status`.

## 3. Backend layout (`backend/projects/`)

- `projects/types/`: the tables, constants, `CanGiveFeedback`, `IsProjectFileKey`, `NormalizeFeedback`.
- `projects/*_rules.go` + tests: copied almost verbatim (they are pure). Only the imports change
  (`helpers.SUnixTime` → `core.SUnixTime`, `core.BusinessUnixDay` → the Genix equivalent).
- Handlers (`projects.go`, `stories.go`, `sprints.go`, `documents.go`, `feedback.go`, `attachments.go`):
  rewritten onto `db.Query(...).CompanyID.Equals(req.User.CompanyID)…Delta(req.GetDeltaCursor(), 1)`,
  `req.GetCachedIDsRequest`, the project lock, and `json.Unmarshal(*req.Body)`. Error messages in Spanish.
- `main.go`: `ModuleHandlers`, registered in `backend/main-handlers.go`, plus `moduleBodies` in
  `scripts/boundaries/check_module_imports.go`. Then `./app.sh generate_controllers` and
  `generate_route_ids`.
- Attachments: upload through the backend (`cloud.SaveFile`, cap ~4 MB), see section 7. The key
  becomes `projects/c<companyID>/p<projectID>/<token>/<name>`, and `IsProjectFileKey` validates it.
  `StoryAttachments.svelte` / `documents` change from "presign + PUT" to "POST the file".
- Team candidates: active users from `GET.users` who hold the projects Edit access.
- Not ported: `documents_link_cli.go`, `story_state_dates_cli.go` (one-off Berryapps migrations),
  `UpdatedMillis`.

## 4. Access catalog and menu

- `backend/access.toml`: a new `[[groups]]` "Proyectos" and accesses 39+ (`projects`, `drafts` if in
  scope) carrying the 28 `backend_apis` and the `frontend_routes`.
- `db/pages.toml` + `db/modules.toml`: a `projects` module with the menus Project list / Backlog /
  Sprints / Documents / Feedback / Reports (Gantt, Calendar, Sprints, By person).
  `frontend/core/modules.test.ts` must stay green.
- Berryapps' `module.ts` (menu shape) and `getAccessIdByKey` are dropped: Genix gates by route via
  `security.canAccessRoute`, and by access ID for the "can create" checks.

## 5. Frontend

- `genix-ui`: Berryapps is one commit ahead on the same `main` (`b52e8b3`: Badge, Avatar,
  AvatarToggleGroup, CollapsibleSection, FileDropZone multiple mode). **A `git pull` in the
  submodule brings them in, with no copying** (the Genix submodule has local changes; check them first).
- Copy into Genix `libs/`: `dates.ts` (merging with the existing `formatTime`, no duplicates),
  `drag-drop.ts`, `back-steps.svelte.ts`. Into `domain-components/`: `MonthInlineSelector.svelte`,
  `charts/ChartLegend.svelte`. Add `formatFileSize` to `libs/helpers.ts`.
- `domain-components/Page.svelte`: Berryapps uses `selectedOption/onSelectOption`, `showBackButton/onBack`,
  `isLoading`, `header`, `subtitle`, `useMenuSearchBox`. Port the props the project picker needs.
- `GetHandler`: Berryapps reads `hasFetched` / `isFirstLoading` (5 uses). Add them to genix-ui or
  replace them locally.
- `Env.makeCdnRoute` → `Env.makeCDNRoute`.
- Route folders copied as they are (`project-list`, `backlog`, `story`, `sprints`, `documents`,
  `feedback`, `reports/*`): they already follow the Genix split (`x.ts` / `x.svelte.ts` / views). Each
  `GUIDE.md` (Spanish) is turned into a bilingual `DOCUMENTATION.md`. The `bun:test` tests run as they are.

## 6. Phases

1. **Base**: types + tables + rules/tests + the project lock; Project/Epic/Story CRUD + backlog
   (rank) + access/menu + project-list and backlog pages. genix-ui pull + libs.
2. **Story detail**: detail, activity/comments, attachments (presign) + documents.
3. **Sprints**: plan/start/move/close + board + `syncStoriesFinalState` + StoryAssignee.
4. **Reports**: gantt, calendar, burndown, by-person (`projects-assigned-stories`, `stories-ids`).

Each phase ends with `go build`, the rule tests, `check_tables`, `modules.test.ts`, and a check in the
real app (`agent-browser` skill).

## 7. Decisions (answered 2026-10-31)

- **A. Concurrency:** fareward lock per project (`core.AcquireLock`). No changes to genix-orm.
- **C. Partition:** `CompanyID` + delta pinned on `ProjectID`.
- **Scope:** **core only**: projects, epics, backlog, story detail/comments/attachments, documents,
  sprints/board, reports. Left out: Feedback (tables, page, `projects-feedback*` routes,
  `CanGiveFeedback`/`NormalizeFeedback`), Drafts (`POST.projects-token`, `POST.projects-from-draft`,
  `Project.DraftID`, `normalizeProjectFromDraft`, `projectOfDraft`), CLIs. Phase 5 is dropped.
- **Uploads:** through the backend with `cloud.SaveFile`, with no presign. The browser sends the file
  to `POST.projects-attachment-upload`, which saves it and returns the `Attachment` (Key, Name,
  ContentType, Size). Because of the Lambda payload limit (~6 MB, base64 +33%), the cap drops from
  10 MB to ~4 MB. Deleting objects needs `cloud.DeleteFile(s)` if it doesn't exist yet.
- **B. Story ID:** a per-project custom autoincrement (packed `ProjectID` + sequence); the sequence
  is the story's number.
