# generate_crm_data

Fills the CRM of the demo company (company 1) through the production handlers of `app/crm`, with the
clock frozen on each simulated moment. Code: `backend/tests/sample_records/generate_crm_data.go`.

```bash
cd scripts && go run . generate_crm_data
cd backend && go run . fn-generate-crm-data
```

What it writes:

- 3 custom fields (a select, a checkbox, a text) and 4 lost reasons.
- 2 pipelines: *Ventas mayoristas* (4 stages, the contact origin is required) and *Proyectos
  especiales* (3 stages). The first company user is Full on both; the next two users are Edit and View.
- 8 prospect clients (`IsProspect`), plus up to 10 existing clients.
- 36 opportunities created on past days (last 120), moved stage by stage a few days apart, and then
  left open, won, lost, or lost and reopened; about one in ten wholesale cards crosses to the projects
  pipeline. The stage log comes out as the board's history reads it.

A company that already has a pipeline is left untouched, so a second run writes nothing. It needs the
CRM tables to exist (`fn-homologate`, or "Recrear Tablas" in the deployer). It is not part of
`generate_demo_data`.
