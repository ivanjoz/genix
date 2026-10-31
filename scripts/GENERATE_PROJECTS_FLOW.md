# generate_projects_flow

Simulates the life of agile projects end to end, through the production handlers of the projects
module, and checks every step. It seeds realistic data for the projects pages and reports, and it
is a regression test of the module: the first wrong result stops the run and names the step.

```bash
cd scripts && FAREWARD_ADDRESS=127.0.0.1:14013 go run . generate_projects_flow
FAREWARD_ADDRESS=127.0.0.1:14013 ./app.sh generate_projects_flow --projects=1 --sprints=2 --stories=8
cd backend && FAREWARD_ADDRESS=127.0.0.1:14013 go run . fn-generate-projects-flow --seed=42
```

**`FAREWARD_ADDRESS` is required in dev.** Every write of the module takes the project's lock in
fareward. A backend process started without the `dev` argument (every `fn-…` process) resolves
fareward to the remote host of `config.toml`, which closes the connection, and the first lock
answers 503. The variable points the process at the dev stack's local daemon (`start.js`).

## Arguments

| Flag | Default | |
|---|---|---|
| `--projects` | 2 | Projects to simulate, cycling the catalog's 3 themes |
| `--sprints` | 4 | Sprints per project; the last one is active today |
| `--stories` | 20 | Stories per project; ~75% get planned into sprints |
| `--sprint-days` | 14 | Calendar days per sprint |
| `--seed` | time | Random seed. A failure prints it: rerun with it to replay the same flow |

## What it does

Content (projects, epics, stories, comments, reasons, files) is in Spanish, from
`backend/tests/sample_records/projects_flow_catalog.json`. The first run creates the catalog's 4
people (password `genix2026`) with the projects access at the write level; later runs reuse them.
Every run creates new projects: earlier ones stay.

The calendar runs backwards from today: the last sprint is half way through, the earlier ones are
closed, and the project kicks off 5 days before the first. Each write runs with the clock frozen on
its simulated moment (`core.SetHistoricalUnix`), so dates, `upd`, `StateDates`, `DoneAt` and the
burndown come out as real history.

| Step | Route |
|---|---|
| Project with its team (creator = user 1, a second admin, 3 members), then its dates | `POST.projects-projects` |
| Epics, plus a temporary one that is renamed and later deleted with a story in it | `POST.projects-epics` |
| Backlog over three days: some stories assigned, some with description and checklist | `POST.projects-stories` |
| Two files on a story, then one removed | `POST.projects-attachment-upload`, `POST.projects-stories` |
| Comments, one with a file | `POST.projects-story-comments` |
| A story dragged between two others, into another epic | `POST.projects-stories-rank` |
| Two project documents, one deleted | `POST.projects-documents` |
| A draft sprint planned and deleted | `POST.projects-sprints` |
| Each sprint: planned (assigning its stories), started, cards moved by their assignees on work days (some stop with a reason), closed carrying unfinished stories to the next sprint or to the backlog | `POST.projects-sprints`, `-sprint-start`, `-sprint-move`, `-sprint-close` |
| A story reassigned mid-sprint, a never-planned story deleted | `POST.projects-stories` |

## What it checks

After every write the project is read back through the GET routes the browser uses, as a first sync
(`GET.projects-projects`, `-epics`, `-stories`, `-sprints`, `-documents`), and checked twice:

- **The step's own outcome**: the story got the next number, the card is in its column after the
  previous one, the carry-over landed in the next sprint with its state, the file is linked to its
  document row, the history line names the move, the person's list holds the story, …
- **The invariants of any project** (`checkProjectInvariants`): records numbered inside the project,
  each story's final state, last sprint and sprint list equal to what its sprints derive, at most one
  active sprint, no story in two open sprints, `DoneAt` exactly on finished cards, epics, assignees
  and files of this project only.

At the end every story's detail and timeline is read: attachment and comment counters, every file
linked to an active document row and no orphan rows, and each member's assigned stories equal to the
stories assigned to them.

A failed check is read again up to 3 times before it fails, since the lists come from materialized
views that may trail an acknowledged write. A check that only passes on a later read is counted
(`viewRetries` in the result) and logged as `lectura desfasada`.

## Delta logs and the frozen clock

The ORM files each delta-log entry by the effective clock, so backdated writes land in the log files
of those past moments. A browser reads only the files from its cursor on: the new projects' own logs
are listed whole on their first sync, but the company's project list and each person's assigned
stories are not new. The run therefore ends each project by writing those rows again with the real
clock, which puts them in the current log file.
