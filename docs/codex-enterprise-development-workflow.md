# Enterprise development workflow for Codex

This repository uses project-scoped Codex agents under `.codex/agents/`.
They are registered by `.codex/config.toml` and apply only when Codex is
working in this repository.

## Roles

| Agent | Purpose | Edit access |
| --- | --- | --- |
| `explorer` | Maps architecture, dependencies, entry points, tests, and risks before a change. | Read-only |
| `architect` | Converts findings into an incremental plan, decisions, risks, and acceptance criteria. | Read-only |
| `developer` | Implements an approved, scoped change and runs relevant validation. | Workspace write |
| `reviewer` | Audits the completed change and produces the required developer tutorial. | Read-only |

## Operating sequence

For non-trivial work, ask Codex to use the roles in this order:

1. `explorer` when the responsible code path is unknown.
2. `architect` to create the plan; it does not modify code.
3. `developer` to implement only after the plan is approved or implementation
   was explicitly requested.
4. `reviewer` to report production risks, test gaps, and a usage tutorial.

The explorer is optional if the architecture and affected path are already
known. The reviewer is mandatory for work that requires the full enterprise
workflow.

## Example prompt

```text
Implement feature X using the enterprise workflow. Ask explorer to map the
current path, architect to propose an incremental plan, developer to implement
the approved plan with tests, and reviewer to perform the final audit and write
the developer tutorial.
```

## Why there is no `workflow.toml`

Codex supports custom agent files in `.codex/agents/` and global agent settings
in `.codex/config.toml`. It does not use a `workflow.toml` file to schedule
agents. `AGENTS.md` therefore records the durable sequencing rule that Codex
loads for this repository, while each task explicitly determines whether
delegation is warranted.
