# Git Commit Convention

This project uses the **Conventional Commits** style.

The goal is to make the Git history understandable, predictable, and suitable for automatic changelog generation.

## Format

```text
<type>(<scope>): <description>
```

Example:

```text
feat(worker): add heartbeat mechanism
```

Where:

* `type` — type of change;
* `scope` — project component;
* `description` — brief description of the change.

---

## Types

The following types are used:

| Type       | Purpose                                          | Example                                     |
| ---------- | ------------------------------------------------ | ------------------------------------------- |
| `feat`     | New functionality                                | `feat(queue): add priority queue`           |
| `fix`      | Bug fix                                          | `fix(worker): handle redis disconnect`      |
| `refactor` | Code restructuring without behavior change       | `refactor(api): extract task service`       |
| `test`     | Adding or modifying tests                        | `test(worker): add retry tests`             |
| `docs`     | Documentation changes                            | `docs(readme): add setup instructions`      |
| `perf`     | Performance improvement                          | `perf(queue): reduce task polling overhead` |
| `build`    | Build system changes                             | `build(go): update dependencies`            |
| `ci`       | CI/CD changes                                    | `ci: add github actions workflow`           |
| `chore`    | Other technical changes                          | `chore: update development dependencies`    |
| `revert`   | Reverting a previous change                      | `revert: revert worker heartbeat`           |

---

## Scope

`scope` should denote the project component affected by the change.

Main scopes:

```text
api
worker
queue
scheduler
db
config
observability
auth
docs
ci
```

Examples:

```text
feat(api): add task creation endpoint
feat(worker): add concurrent task execution
feat(queue): implement visibility timeout
feat(db): add task repository
feat(scheduler): add delayed tasks
feat(observability): add prometheus metrics
```

If the change affects the project as a whole and a specific scope is hard to determine, `scope` can be omitted:

```text
chore: update project dependencies
```

---

## Description

The description is written in English in **imperative mood** — like a command.

Use:

```text
add
implement
fix
remove
update
refactor
```

Good:

```text
feat(worker): add graceful shutdown
fix(queue): handle empty queue
refactor(api): extract task service
docs(readme): add development setup
```

Bad:

```text
feat(worker): added graceful shutdown
fix(queue): fixed empty queue
feat(api): changes
fix: bug
update
```

---

## Commit Size

A single commit should contain **one logical change**.

Good:

```text
feat(queue): implement redis task queue
feat(worker): add task polling
feat(worker): add concurrent task execution
feat(worker): add graceful shutdown
test(worker): add task execution tests
```

Bad:

```text
feat: implement worker, queue, tests, docker and api
```

If a change consists of multiple independent parts, they should be split into multiple commits.

---

## Commit Body

For small changes, a single line is sufficient:

```text
feat(worker): add graceful shutdown
```

For complex changes, a detailed description can be added:

```text
feat(queue): implement visibility timeout

Tasks are no longer removed immediately after
being acquired by a worker.

A task remains invisible for a configurable
period and is returned to the queue if the
worker does not acknowledge it.
```

The body should be used when the subject alone doesn't make it clear:

* why the change was made;
* what behavior changed;
* what architectural decision was made;
* what new constraints were introduced.

---

## Breaking Changes

If a change breaks an existing API or contract, this must be explicitly indicated.

You can use `!`:

```text
feat(api)!: change task creation response
```

and add `BREAKING CHANGE` to the body:

```text
feat(api)!: change task status format

BREAKING CHANGE: task status is now returned as
an object instead of a string.
```
