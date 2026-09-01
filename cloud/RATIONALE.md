## The renderer zip URL comes from `frontend.app_url`

**Context** — This module carried its own hardcoded copy of the webpage-renderer artifact URL,
duplicated across three Go modules that cannot import each other.

**Decision** — The literal is gone; the URL is `frontend.webpage_renderer_url`, or
`<frontend.app_url>/webpage-renderer.zip` when that key is empty.

**Rationale** — See the full entry in `backend/RATIONALE.md`.

