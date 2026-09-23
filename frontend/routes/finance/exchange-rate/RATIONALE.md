# RATIONALE — finance/exchange-rate

Design decisions for the exchange rate maintainer, newest first.

## The BCRP default is a colour, not a record

**Context** — Every empty day now shows the BCRP interbank rate published by
public-business-data. Gray means "this is the default", blue means "the company loaded this one",
and the two have to be told apart on sight.

**Decision** — The default is display-only. `BcrpDefaultRates` holds the fetched series in its own
`$state` map and nothing about it ever reaches a draft: a cell falls back to it when the stored
rate is 0, `setCellCss` returns `_rate-bcrp` or `_rate-own` from that same test, and
`dirtyDraftsToRecords` keeps sending only what was typed. Editing opens on the value that is on
screen, defaults included, so accepting a BCRP rate unchanged writes nothing — `CellInput` only
fires `onChange` when the value moved — and typing over it is what turns the cell blue.

**Rationale** — "Which of these did we load ourselves?" is answered by the absence of a stored
value, with no extra column and no flag to keep in sync. The cost is that the default lives only
on this page: nothing else in the ERP can read a rate the company never loaded, and a consumer
that needs one — invoicing, exchange difference — will have to either fetch the same series or
get the rates loaded here first. That is also why the ceiling on it is low: none of this data is
valid for tax purposes.

**Why the BCRP and not SUNAT** — asked for explicitly. Worth repeating where it matters: the
interbank rate is the market one, it lags SUNAT by a couple of business days, and Peruvian tax
rules point at SUNAT/SBS. The tail of the series also carries up to three filled days flagged
`provisional` (±0.5%), and the page shows them like any other default — no mark, by decision.

## The published series is fetched by the browser, not through the backend

**Context** — public-business-data is static gzipped files on GitHub Pages with
`Access-Control-Allow-Origin: *`, and its client caches each year in IndexedDB under the content
hash the manifest gives it.

**Decision** — `BcrpDefaultRates` calls the client directly from the page, on mount, for the exact
span the calendar shows (`calendarDateRange`). It does not extend `GetHandler`, and a failure is a
`console.warn` and nothing else.

**Rationale** — Routing it through our backend would add an endpoint, a cache and a failure mode to
re-serve bytes that are already public, immutable and cached closer to the user. It is data no
user of ours owns, so there is nothing to authorize. Staying off `GetHandler` also keeps the delta
cache holding only what the company actually saved. The rates are a convenience, so the page has
to work with the source down — hence the warn rather than a `Notify`.

## The calendar is one flat row list with a header flag

**Context** — `TableGrid` takes a single `data` array and asks `useRowRenderer(record)` which rows
to paint full width. The calendar has two kinds of row — the month separator and a week — and they
have to live in the same array, in order.

**Decision** — One `ICalendarRow` type with an `IsMonthHeader` flag: a month row carries only its
`MonthKey` and an empty `Days`, a week row carries the day number per weekday (Monday first, `0`
where the week runs outside the month). `buildCalendarRows` emits them month after month, and the
rates stay where they were — one draft per month, indexed by day — so a cell resolves as
`draft[Buy|Sell][Days[weekday] - 1]`.

**Rationale** — A discriminated union would have forced a cast in every `getValue`, and two arrays
would have forced the page to interleave them anyway. The flag keeps the columns readable and lets
`disableCellInteractions` be the same test the layout already makes: `!row.Days[weekday]` is both
"this cell is outside the month" and "nothing to edit here".

Every row keeps the same height on purpose, month separators included: `TableGrid`'s virtualizer is
the fixed-height one (`createFixedTableVirtualizer`), so a taller header row would desync the scroll
offsets from the rendered window.

## A delta sync mid-edit leaves the edited months alone

**Context** — The service keeps delta-syncing while the page is open. A sync that lands after the
user typed into March but before they saved would otherwise overwrite March with the stored row.

**Decision** — Each month draft carries `isDirty`. `mergeFetchedRates` re-seeds every clean month
from the sync and skips the dirty ones; saving clears the flags. The Save button is disabled until
something is dirty and shows how many months it will write.

**Rationale** — Same shape the shipping costs page uses for the same reason. The alternative —
freezing the sync while the page is open — would leave a second tab's saves invisible for as long as
the page stays open.

## Anything that is not a rate clears the day

**Context** — Cells accept free text before they are parsed, and 0 already means "no rate published
this day" — a weekend, a holiday, or a day nobody has loaded.

**Decision** — `parseRateInput` returns 0 for empty, non-numeric, negative, and anything above
1000.000; everything else is stored as the rate x 1000. The backend re-checks the same ceiling and
rejects the payload.

**Rationale** — There is no third state to represent: a day either has a rate or it does not, so a
bad value falls back to "does not" instead of inventing an error state per cell. The ceiling catches
the realistic typo — a misplaced decimal separator — rather than pretending to validate the rate
against a real market.
