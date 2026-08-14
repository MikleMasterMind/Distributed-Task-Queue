# Git Commit Convention

Для проекта используется стиль **Conventional Commits**.

Цель — сделать историю Git понятной, предсказуемой и пригодной для автоматической генерации changelog.

## Формат

```text
<type>(<scope>): <description>
```

Пример:

```text
feat(worker): add heartbeat mechanism
```

Где:

* `type` — тип изменения;
* `scope` — компонент проекта;
* `description` — краткое описание изменения.

---

## Types

Используются следующие типы:

| Type       | Назначение                                       | Пример                                      |
| ---------- | ------------------------------------------------ | ------------------------------------------- |
| `feat`     | Новая функциональность                           | `feat(queue): add priority queue`           |
| `fix`      | Исправление ошибки                               | `fix(worker): handle redis disconnect`      |
| `refactor` | Изменение структуры кода без изменения поведения | `refactor(api): extract task service`       |
| `test`     | Добавление или изменение тестов                  | `test(worker): add retry tests`             |
| `docs`     | Изменение документации                           | `docs(readme): add setup instructions`      |
| `perf`     | Улучшение производительности                     | `perf(queue): reduce task polling overhead` |
| `build`    | Изменение системы сборки                         | `build(go): update dependencies`            |
| `ci`       | Изменение CI/CD                                  | `ci: add github actions workflow`           |
| `chore`    | Прочие технические изменения                     | `chore: update development dependencies`    |
| `revert`   | Откат предыдущего изменения                      | `revert: revert worker heartbeat`           |

---

## Scope

`scope` должен обозначать компонент проекта, которого касается изменение.

Основные scopes:

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

Примеры:

```text
feat(api): add task creation endpoint
feat(worker): add concurrent task execution
feat(queue): implement visibility timeout
feat(db): add task repository
feat(scheduler): add delayed tasks
feat(observability): add prometheus metrics
```

Если изменение затрагивает проект в целом и конкретный scope определить сложно, `scope` можно не указывать:

```text
chore: update project dependencies
```

---

## Description

Описание пишется на английском языке в **imperative mood** — как команда.

Использовать:

```text
add
implement
fix
remove
update
refactor
```

Хорошо:

```text
feat(worker): add graceful shutdown
fix(queue): handle empty queue
refactor(api): extract task service
docs(readme): add development setup
```

Плохо:

```text
feat(worker): added graceful shutdown
fix(queue): fixed empty queue
feat(api): changes
fix: bug
update
```

---

## Размер коммита

Один коммит должен содержать **одно логическое изменение**.

Хорошо:

```text
feat(queue): implement redis task queue
feat(worker): add task polling
feat(worker): add concurrent task execution
feat(worker): add graceful shutdown
test(worker): add task execution tests
```

Плохо:

```text
feat: implement worker, queue, tests, docker and api
```

Если изменение состоит из нескольких независимых частей, их следует разделить на несколько коммитов.

---

## Commit Body

Для небольших изменений достаточно одной строки:

```text
feat(worker): add graceful shutdown
```

Для сложных изменений можно добавить подробное описание:

```text
feat(queue): implement visibility timeout

Tasks are no longer removed immediately after
being acquired by a worker.

A task remains invisible for a configurable
period and is returned to the queue if the
worker does not acknowledge it.
```

Body следует использовать, когда из subject недостаточно понятно:

* зачем было сделано изменение;
* какое поведение изменилось;
* какое архитектурное решение было принято;
* какие ограничения появились.

---

## Breaking Changes

Если изменение ломает существующий API или контракт, это должно быть явно обозначено.

Можно использовать `!`:

```text
feat(api)!: change task creation response
```

и добавить `BREAKING CHANGE` в body:

```text
feat(api)!: change task status format

BREAKING CHANGE: task status is now returned as
an object instead of a string.
```
