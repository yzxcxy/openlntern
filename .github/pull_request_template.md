## Style Guard Checklist

- [ ] I ran `pnpm lint:style-guard` locally.
- [ ] New page or feature code uses `app/components/ui/*` and `app/components/layout/*` before adding raw controls.
- [ ] No new hard-coded Tailwind palette classes or raw `rgba` / `#hex` / gradient values were introduced in `className`.
- [ ] Any temporary exception is explained below and captured in `docs/residual-backlog.md`.

## Exception Notes

<!-- Required only when one of the checklist items above is intentionally not met. -->
