# Promptfoo Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a minimal `promptfoo` evaluation workspace that can authenticate with openIntern, call `/v1/chat/sse`, parse AG-UI SSE output, and document how to run local evals safely.

**Architecture:** Keep prompt evaluation as an isolated workspace under `evals/promptfoo` so it does not affect frontend or backend runtime dependencies. Use Promptfoo's HTTP provider plus a file-based auth helper to log in and inject a JWT, and a local response parser to reconstruct assistant text from AG-UI SSE events.

**Tech Stack:** Promptfoo config YAML, Node.js file auth helper, Node.js SSE parser, Node built-in test runner, Markdown docs

---

### Task 1: Scaffold isolated eval workspace

**Files:**
- Create: `evals/promptfoo/package.json`
- Create: `evals/promptfoo/README.md`
- Create: `evals/promptfoo/.env.promptfoo.example`

- [ ] Add an isolated workspace with only promptfoo-related scripts and documentation.
- [ ] Keep secrets out of git by using an example env file instead of checked-in credentials.

### Task 2: Lock parser behavior with tests first

**Files:**
- Create: `evals/promptfoo/tests/agui-sse-parser.test.mjs`
- Create: `evals/promptfoo/src/agui-sse-parser.mjs`

- [ ] Write a failing test for reconstructing assistant text from AG-UI `TEXT_MESSAGE_CONTENT` events.
- [ ] Write a failing test for ignoring unrelated AG-UI events and malformed lines.
- [ ] Implement the minimal parser to make both tests pass.

### Task 3: Lock request building with tests first

**Files:**
- Create: `evals/promptfoo/tests/request-payload.test.mjs`
- Create: `evals/promptfoo/src/request-payload.mjs`

- [ ] Write a failing test for building a chat-mode `/v1/chat/sse` payload with unique `threadId`, one user message, and `forwardedProps.agentConfig`.
- [ ] Write a failing test for optional agent-mode overrides.
- [ ] Implement the minimal payload builder to make both tests pass.

### Task 4: Wire promptfoo runtime files

**Files:**
- Create: `evals/promptfoo/auth/get-token.mjs`
- Create: `evals/promptfoo/parsers/agui-sse-response.mjs`
- Create: `evals/promptfoo/promptfooconfig.yaml`

- [ ] Reuse the tested parser and payload builder from `src/`.
- [ ] Configure Promptfoo HTTP provider to log in through file auth, send JWT to `/v1/chat/sse`, and parse SSE output into final assistant text.
- [ ] Keep the first version focused on chat-mode evals, with model selection provided through test vars.

### Task 5: Document local usage and repo hygiene

**Files:**
- Modify: `.gitignore`
- Modify: `README.md`

- [ ] Ignore promptfoo local env files and output directories without disturbing existing user edits.
- [ ] Add a short README section pointing to the isolated eval workspace and the local run command.

### Task 6: Verify locally

**Files:**
- Verify: `evals/promptfoo/tests/agui-sse-parser.test.mjs`
- Verify: `evals/promptfoo/tests/request-payload.test.mjs`
- Verify: `evals/promptfoo/promptfooconfig.yaml`

- [ ] Run the Node test suite for the new workspace.
- [ ] Run a lightweight config validation path if possible without downloading dependencies; otherwise document the blocker clearly.
