# Generate Menu Descriptions

Creates `tmp/menu_description.json` from route markdown files under `frontend/routes`.

## Usage

```bash
./app.sh generate_menu_descriptions
```

## Markdown Format

Each route markdown file can include Spanish and English description blocks:

```md
## DESCRIPTION::ES
Texto en espanol.

## DESCRIPTION::EN
English text.
```

Files without description blocks are skipped. Files with only one language fail validation so the generated menu data stays complete.
