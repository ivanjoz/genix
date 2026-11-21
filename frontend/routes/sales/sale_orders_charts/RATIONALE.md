# RATIONALE — sale_orders_charts

## The charts read the frame files through a pure per-day transform; sub-units become fractions
**Context** — `sale-summary` is now a DataFrame: day files with one row per ProductID, whose
quantities already hold sub-units as a fraction of a unit (backend `SaleSummaryFrame`, kept × 1e6).
The components read a per-day columnar record and only use Date, ProductIDs, the sold and pending
quantities and the two amounts.
**Decision** — `sale_orders_charts.ts` turns the files into one `ISaleSummaryRecord` per day,
dividing the quantity sums by the frame's float scale (3 units and 2 sixths = 3.333333). The
interface drops CompanyID, `upd`, SubDivisor and the Sub* arrays. The service is a
plain class over `GETFrameFiles` reading the last 56 days (today included), the old default range;
it replaces the `GetHandler` delta cache.
**Rationale** — The old reader showed whole units only, ignoring sub-units; fractions are exact and
need no shared divisor. The components are unchanged except for the type import. The range has no
picker, as before.
