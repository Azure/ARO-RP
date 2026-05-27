# Agentic Hints for Debugging ARO Classic

- IMPORTANT: this document is referenced by agentic workflows, DO NOT REMOVE IT.
- Start with this file before loading any specialized ARO Classic debugging hints.
- This file is intentionally short. Load only the specialized doc(s) needed for the current task.
- If any of the info below turns out not to be accurate, suggest an update PR at the end of the session.

## Scope

- Use this file to choose which ARO Classic debugging hints to load.
- Do not pull specialized docs into context unless the current task actually needs them.
- Add future specialized docs here instead of expanding this file into a catch-all runbook.
- Prefer the current workspace for checked-in ARO-RP code and docs.
- You may use the full contents of this repository, not just files under `docs/ai/`.
- If the current workspace lacks needed context, you may use another local checkout of this repo or clone it if necessary.
- When drawing code-specific conclusions from another checkout, prefer one whose revision you can identify and match to the workspace or target deployment; otherwise call out the uncertainty.
- You may use relevant `eng.ms` documentation when available through an MCP server, or a locally cloned/mirrored copy of that documentation when available, as supporting context. Some external docs may require access. Verify them against the checked-out code and current environment.

## Available Specialized Docs

- `docs/ai/classic-log-search.md` - Kusto/Geneva log-search guidance. Load this only when the task is primarily about searching logs, tracing a request through logs, or investigating a failure from log evidence.
- Add future specialized docs here with one-line trigger descriptions.

## Core ARO Classic Context

- Do not assume ARO HCP debugging docs, Kusto tables, or Grafana datasource names apply to ARO Classic.
- Customer cluster mutations (`PUT` / `PATCH` / `DELETE`) are async: the frontend accepts the request and persists state, then the backend completes the work later. Many investigations span both frontend and backend evidence.
- Cross-cutting pivots that are useful across multiple debugging modes: `request_id`, `resource_id`, `subscription_id`, `resource_group`, `resource_name`, and sometimes `correlation_id` / `client_request_id`.

## Loading Rules

- Always start with this file.
- If the task is log-only or Kusto/Geneva-centric, then load `docs/ai/classic-log-search.md`.
- If the task is not about logs, do not load the log-search doc unless logs become relevant.
- When more specialized docs exist, load the smallest set that matches the current task.
- Use specialized docs to minimize unnecessary context, but consult the wider repo and supporting external docs when the task requires them.
