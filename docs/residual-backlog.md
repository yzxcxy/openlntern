# Residual Backlog

## P0
- Replace raw controls in `app/(workspace)/layout.tsx` with shared primitives (`UiButton` / `UiInput`) so shell-level interactions stop emitting style-guard warnings.
- Replace the raw close/action button in `app/(workspace)/a2ui/components/Modal.tsx` with the shared button layer or a dedicated modal action primitive.

## P1
- Migrate the remaining raw `input` fields in `app/(workspace)/skills/page.tsx` and `app/(workspace)/user/page.tsx` to `UiInput`.
- Tokenize the icon accent colors in `app/(workspace)/skills/detail/page.tsx` so they stop depending on direct Tailwind palette classes.

## P2
- Move hard-coded `rgba(...)`, `#hex`, and `linear-gradient(...)` arbitrary values in the following files into semantic tokens or shared variants:
  - `app/(workspace)/kb/page.tsx`
  - `app/(workspace)/chat/page.semi.tsx`
  - `app/(workspace)/layout.tsx`
  - `app/(workspace)/a2ui/components/Modal.tsx`
  - `app/(workspace)/a2ui/components/ConfirmDialog.tsx`

## P3
- Add CI wiring so every PR runs `pnpm lint:style-guard` automatically instead of relying on local invocation only.
- Add a lightweight metrics script to snapshot style-guard hit counts per PR and track the downward trend over time.
- Revisit warning-to-error promotion after raw controls and hard-coded color hits reach zero.
