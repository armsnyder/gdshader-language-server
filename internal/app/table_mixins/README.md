# Table Mixins

This directory contains optional CSV overrides for `internal/tools/gentables`.

## Why

Generated tables are overwritten each time `gentables` runs. Mixins are kept in a
separate directory so custom additions and adjustments survive regeneration.

## How files map

Each mixin CSV path is relative to this directory and mirrors generated table
paths under `internal/app/tables`.

Example:

- Generated: `internal/app/tables/spatial_shader/render_modes.csv`
- Mixin: `internal/app/table_mixins/spatial_shader/render_modes.csv`

## Mixin formats

### Upsert rows

Use the exact same header as the generated CSV. Rows are matched by the first
column and replaced if present, appended if missing.

## Add custom tables

If no generated table exists for a mixin path, the mixin is used as the full
table.
