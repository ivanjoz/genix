## The per-company SMTP panel is removed; outgoing mail stays platform-wide

**Context** — The Parameters page showed an "SMTP parameters for notifications" card writing
`Company.SmtpConfig` (Host / Port / User / Password / Email). Nothing ever read it: every caller of
`core.SendEmail` — sign-up verification, the contact form — takes its credentials from
`core.Env.SMTP_*`, loaded from the `[smtp]` section of `config.toml`. The panel had also never
worked end to end: the Go struct field was named `Post` while the form bound `save="Port"`, so the
port never even deserialized. Worse, the record is served by the standard `company-parametros`
delta-cache endpoint, so a saved SMTP password shipped in plaintext to any client allowed to read
company parameters.

**Decision** — Deleted the panel, the `ICompanySmtp` interface in both `configuration/parameters`
and `system/companies`, and the `SmtpConfig` field and type from `config/types/empresas.go`. The
two-column page grid collapses to the single form grid, since the right column existed only to hold
that card.

**Rationale** — A control that stores a secret it never uses is worse than no control: it invites an
operator to enter real mail credentials, then leaks them over an endpoint whose read permission is
much broader than "may configure mail". Wiring it up for real is the alternative, but per-tenant
SMTP needs a write-only password field, a fallback chain to the global config and a send-test path —
work that only pays off once tenants actually want mail from their own domain. Cost: the Scylla
column `smtp_config` is now orphaned. Pre-alpha, so it is left in place rather than migrated; the
ORM only reads declared columns and `check_tables` passes.
