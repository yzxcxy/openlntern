# Repository Guidelines

## Project Structure & Module Organization
This workspace contains multiple codebases:
- `openIntern/openIntern_backend/`: Gin-based backend (`internal/controllers`, `internal/services`, `internal/dao`, `internal/models`).
- `openIntern/openIntern_forentend/`: Next.js 16 + TypeScript frontend (`app/`, `app/components/ui`, `app/components/layout`, `docs/`).
- `openIntern/go/`: Go SDK (`pkg/`) and runnable examples (`example/client`, `example/server`).

Keep changes scoped to one module when possible; run commands from that module root.

## Build, Test, and Development Commands
- `cd openIntern/openIntern_backend && go run .`: start backend (reads `config.yaml`).
- `cd openIntern/openIntern_backend && go test ./...`: run backend unit tests.
- `cd openIntern/openIntern_forentend && pnpm dev`: start frontend locally.
- `cd openIntern/openIntern_forentend && pnpm build`: production build.
- `cd openIntern/openIntern_forentend && pnpm lint && pnpm lint:style-guard`: ESLint + style guard checks.
- `cd openIntern/go && go test ./...`: run SDK tests.

## Coding Style & Naming Conventions
- Go: always format with `gofmt` (and `goimports` for import grouping). Use exported GoDoc comments where required.
- Frontend: TypeScript strict mode is enabled; use `PascalCase` for React components (e.g., `UiButton.tsx`) and Next.js route conventions (`page.tsx`, `layout.tsx`).
- In frontend pages/features, prefer shared primitives in `app/components/ui/*` and `app/components/layout/*` over raw controls.
- 

## Testing Guidelines
- Go tests live beside source as `*_test.go`, with `TestXxx` naming; table-driven tests are preferred.
- Frontend currently relies on lint/style-guard gates; add tests with new test tooling when introducing complex UI logic.
- No fixed coverage threshold is configured; new features should include meaningful regression tests.

## Commit & Pull Request Guidelines
- Follow Conventional Commits as used in history: `feat(scope): ...`, `fix(scope): ...`, `refactor(scope): ...`.
- Keep commits focused and small; include scope when it clarifies the area (`kb`, `plugins`, `agent`, `ui`).
- Frontend PRs should satisfy the style-guard checklist in `.github/pull_request_template.md`, and document any exception in `docs/residual-backlog.md`.

## Security & Configuration Tips
- Do not commit secrets or local configs. `openIntern/openIntern_backend/config.yaml` and frontend `.env*` are local-only.
- Use sanitized example values in docs and test fixtures.

# Agent Development Rules

## Code Documentation
- Every function and method must include clear comments or docstrings.
- The comment should explain the purpose

## Code Organization
- Avoid putting too much logic into a single file.
- Split code into multiple modules when the file grows large.
- Separate responsibilities into different files (e.g., service, utils, controllers).

## Maintainability
- Write readable and maintainable code.
- Prefer small functions with clear responsibilities.

# Development Phase Rules

## No Legacy Compatibility
The project is currently in the active development phase.

- Do NOT add compatibility layers for old data structures or APIs.
- Prefer updating the implementation directly instead of supporting legacy formats.
- If a schema or interface changes, update all related code accordingly.

## Avoid Redundant Code
- Do not introduce temporary adapters or compatibility wrappers.
- Remove unused or obsolete logic immediately.
- Prefer refactoring over adding workaround code.

## Clean Architecture
- Keep the codebase simple and maintainable.
- Avoid premature abstractions meant for backward compatibility.
- Implement the simplest correct solution for the current design.

## Refactoring Policy
When changing data structures, APIs, or modules:
- Update all usages in the repository.
- Do not keep deprecated fields or functions unless explicitly required.