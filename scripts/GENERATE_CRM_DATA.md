# generate_crm_data

Fills the CRM of the demo company (company 1) through the production handlers of `app/crm`, with the
clock frozen on each simulated moment. Code: `backend/tests/sample_records/generate_crm_data.go`.

```bash
cd scripts && go run . generate_crm_data
cd backend && go run . fn-generate-crm-data
```

What it writes:

- 3 custom fields (a select, a checkbox, a text) and 4 lost reasons.
- 2 pipelines: *Ventas mayoristas* (4 stages, the contact origin is required from the first stage; its
  first two stages propose an activity task to the card's seller, so open cards leave pending tasks
  behind and lost ones leave cancelled tasks) and *Proyectos
  especiales* (3 stages). The first company user is Full on both; the next two users are Edit and View.
- 8 prospect clients (`IsProspect`), plus up to 10 existing clients.
- 300 opportunities created on past days (last 365). Each lives a random 2 to 30 calendar days from
  creation to its last change: it moves stage by stage inside that span and, on its last day, is
  won, lost, or (one in ten wholesale cards) moved to the projects pipeline. A span that reaches past
  today leaves the card open. Along the way some cards step back one stage and forward again, some
  get their amount renegotiated, a few are handed to another seller, and a few are lost and reopened
  a day or two later — the cases the CRM reports have to count correctly. After most forward moves
  the card's seller logs a call, meeting or follow-up (the scorecard's activities). The stage log and the
  per-pipeline status rows (the timeline report) come out of the production handlers.

A company that already has a pipeline is left untouched, so a second run writes nothing. It needs the
CRM tables to exist (`fn-homologate`, or "Recrear Tablas" in the deployer). It is not part of
`generate_demo_data`.
