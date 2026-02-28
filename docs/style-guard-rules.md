# Style Guard Rules

## Purpose
Phase 5 freezes the refactor constraints so new pages cannot quietly reintroduce the same style debt that Phase 1-4 removed.

## Active Rules
- `raw-controls`: warns on direct `button`, `input`, `select`, and `textarea` usage anywhere under `app/**`, except `app/components/ui/**`.
- `hard-coded-palette`: warns on direct Tailwind palette classes such as `text-amber-500` or `bg-blue-600` in `className`.
- `arbitrary-color-values`: warns on raw `rgba(...)`, `rgb(...)`, `hsl(...)`, `#hex`, and `linear-gradient(...)` values embedded inside `className`.

## Why Warning First
- The current repository still contains residual legacy usage.
- Warning mode keeps the repo mergeable while making every new violation visible during lint.
- Once the backlog in `docs/residual-backlog.md` is cleared, the same rules can be promoted to `error` or enforced with `--max-warnings 0`.

## PR Check Command
```bash
pnpm lint:style-guard
```

## New Page Acceptance
- Reuse `app/components/ui/*` and `app/components/layout/*` before introducing any new control or layout primitive.
- If a new visual treatment is reusable, add or extend a shared component variant instead of embedding literal colors in page code.
- If an exception is unavoidable, document it in the PR and add a follow-up item to `docs/residual-backlog.md`.
