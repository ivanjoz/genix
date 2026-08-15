---
name: create-vector-image
description: Generate real SVG artwork (icons, logos, spot illustrations, backgrounds) for the website with the Recraft vector models through OpenRouter, then visually check the result before it enters the repo. Use when a task needs a new vector asset rather than an existing one.
version: 0.1.0
---

# Create vector image (`generate_vector_image`)

Generates true SVG — paths, no embedded raster — from a text prompt:

```bash
cd scripts
go run . generate_vector_image -prompt "<description>"
```

Output lands in **`tmp/vector/<slug-of-prompt>.svg`** (gitignored). The script prints every file it
wrote and the real USD cost of the call.

## Each call costs real money — confirm first

These models bill per image token, not per text token. One image is roughly:

| Model (`-model`) | Cost per image | Use for |
| --- | --- | --- |
| `recraft/recraft-v4.1-vector` (default) | **~$0.08** | icons, spot illustrations, everything by default |
| `recraft/recraft-v4.1-pro-vector` (`-model pro`) | **~$0.30** | hero art or a logo, where resolution and finish matter |

`-n 3` costs three times as much. **Ask the user before generating**, tell them which model and how
many variants, and never silently retry a call that already succeeded.

Both entries live in the `[[image_models]]` array of `config.toml` — a table separate from
`[[models]]`, which is the agent's *chat* model registry and must not receive these. The script
reads `agent.openrouter_key` from the same file.

## Flags

| Flag | Default | Notes |
| --- | --- | --- |
| `-prompt` | — | required |
| `-model` | first `[[image_models]]` entry | substring match, so `-model pro` is enough |
| `-out` | `tmp/vector/<slug>.svg` | pass an explicit path to write straight into the site |
| `-aspect` | `1:1` | `16:9`, `3:4`, … (only `1:1` is verified against the live API) |
| `-n` | `1` | extra variants get `_2`, `_3` suffixes |

## Always look at the result

The model returns valid SVG that is often *not* what was asked for. Rasterize and actually view it
before using it anywhere:

```bash
rsvg-convert -w 400 tmp/vector/<file>.svg -o tmp/vector/<file>.png
```

Then Read the PNG — the Read tool renders it, so you can judge the artwork instead of guessing from
path data. Report to the user what you see, and don't spend another call without their say-so.

## Prompting

The models are tuned for aesthetics, not instruction-following. What works:

- Name the form first: `minimal flat line icon of …`, `isometric spot illustration of …`.
- Constrain the palette explicitly (`single accent color`, `two-tone blue and near-black`), or you
  get an arbitrary one.
- Say `clean geometric shapes`, `uniform stroke weight`, `no text` for UI icons — lettering comes
  out malformed.
- Ask for `on transparent background` for anything that overlays the page.

## What the SVG looks like

Verified against a real generation:

- Transparent background — no white rect is baked in.
- `viewBox="0 0 2048 2048"` with fixed `width`/`height` attributes. **Strip `width`/`height`** before
  using it responsively, or it will not scale with its container.
- Fills are literal `rgb(r,g,b)` per path, so recoloring to a theme token means editing each fill.
- Carries a small empty `<metadata><c2pa:manifest>` provenance block — harmless, removable.

## Placing the asset

The default output is a scratch dir on purpose: generated art gets reviewed, not auto-published.
Once the user approves one, move it to the directory that matches its use:

- `frontend/webpage/static/images/` — public website / storefront
- `frontend/static/images/` — main app (holds `genix_logo*.svg`)
- `frontend/libs/assets/` — shared UI icons consumed by the packages

## Limits

- Text-to-SVG only. The models accept image input, but this script does not send it.
- Generation is slow (~1 min per image); the client waits up to 5 minutes.
- If the model ever returns a non-SVG media type the script writes the bytes anyway and warns.

Script: `scripts/generate_vector_image.go` · reference: `scripts/GENERATE_VECTOR_IMAGE.md`
