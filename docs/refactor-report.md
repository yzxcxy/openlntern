# Frontend Refactor Report

## Phase 5 Baseline
- Scope: `openIntern/openIntern_forentend` as of the Phase 5 closeout pass.
- Goal: freeze style guardrails after the Phase 1-4 token/component rollout without rewriting in-flight feature files.

## Inventory Snapshot
- Route entry files: 10 total (`page.tsx` and `page.semi.tsx`), with 9 interactive pages after excluding the root redirect.
- Shared frontend foundation: 7 reusable files in `app/components` (`3` layout primitives, `4` UI adapters).
- Shared UI adoption: 7 of 9 interactive pages already import `app/components/ui/*`.
- Workspace shell centralization: `app/(workspace)/layout.tsx` is the single shared shell entry for workspace routes.

## Measurable Outcomes
- Style-guard warnings currently emitted by `pnpm lint:style-guard`: `63` (`14` raw controls, `44` arbitrary color values, `5` palette classes).
- Hard-coded Tailwind palette hits currently visible in app code: `5`.
- Arbitrary raw color or gradient class hits currently visible in app code: `44`.
- Raw native control usage outside `app/components/ui/*`: `14` (`12` `button`, `2` `input`).
- Files still producing style-debt signals: `6` for hard-coded color classes, `4` for raw native controls.

## Phase 5 Deliverables
- Added warning-level ESLint style guards for:
  - raw `button` / `input` / `select` / `textarea` usage outside `app/components/ui/*`
  - hard-coded Tailwind palette classes in `className`
  - raw `rgba` / `hex` / gradient arbitrary values in `className`
- Added `pnpm lint:style-guard` as the dedicated command for PR checks.
- Added `docs/style-guard-rules.md` and `.github/pull_request_template.md` so new pages inherit the same guardrails.

## Current Readout
- The component layer is established and already reused by the majority of interactive pages.
- Remaining debt is concentrated in legacy workspace shell code and the two most complex surfaces (`chat`, `kb`), which matches the planned Phase 4/5 tail.
- Guardrails are intentionally `warning` first so the current codebase stays mergeable while residual debt is burned down.
- Current verification status: `pnpm lint:style-guard` exits successfully with warnings only (`97` total warnings, `63` from the new Phase 5 style guards).
