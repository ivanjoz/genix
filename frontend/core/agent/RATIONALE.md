## Chat: replies with charts or tables are wider than text bubbles
**Context** — Asked to give the chat's answers more room (width `calc(94% - 24px)` given): the agent's bubble was capped at 80% of the panel, leaving an empty strip beside charts.
**Decision** — Only a bubble that carries charts or tables gets the wider width; plain-text bubbles (the agent's and the user's) keep `max-w-[80%]`.
**Rationale** — Charts and tables gain from every pixel; a short text reply stretched across the panel would read worse and lose the left/right conversation shape.
