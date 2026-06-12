# projx-store

ProjX's **declared-knowledge** layer — the second deterministic root (core = facts
from code; store = facts *you* declare). **No AI, no UI, no logic** beyond records
and a read/write API.

## Two files, three scopes

| Scope | File | Holds |
|---|---|---|
| `Global` | **your store** (travels with you) | recipes, conventions, style — how you work everywhere |
| `Workspace` | **your store** | this machine's repo list, default gate posture |
| `Project` | **project store** (stays with the repo) | ADRs, declared architecture, history, this project's gate rules |

> *My file travels with me; the project's file stays with the project.*
> Global recipes live **only** in your store. Finest gate rule wins.

## Interface-first

`Store` (Put / Get / List / Delete) is the contract context, workflow, graph, and
verify depend on. The concrete **SQLite schema is deferred** until real records
teach its shape — define the contract now, learn the implementation later. `Mem`
(in-memory) backs it today; a SQLite `Store` swaps in without touching callers.

## Status

P1, fresh build, zero external deps. Clean slate.

```sh
go test ./...
```
