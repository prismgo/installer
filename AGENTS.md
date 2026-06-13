<karpathy-guidelines>

## Karpathy Guidelines

These behavioral rules reduce common LLM coding mistakes. They favor caution over speed; use judgment for trivial tasks.

### 1. Think Before Coding

Do not assume or hide uncertainty. Before implementing:
- State assumptions; ask when uncertain.
- Surface multiple interpretations instead of silently choosing.
- Mention simpler approaches and push back when appropriate.
- If unclear, stop, name the confusion, and ask.

### 2. Simplicity First

Write the minimum code that solves the request:
- No unrequested features, abstractions, flexibility, configurability, or impossible-scenario error handling.
- If a solution is much longer than necessary, simplify it.

### 3. Surgical Changes

Touch only what the request requires:
- Do not improve, refactor, reformat, or delete adjacent code unless needed.
- Match existing style.
- Remove unused imports, variables, or functions created by your change.
- Report unrelated dead code; do not delete it unless asked.

Every changed line must trace directly to the request.

### 4. Goal-Driven Execution

Turn tasks into verifiable goals and loop until verified:
- Validation change: test invalid inputs, then pass.
- Bug fix: reproduce with a test, then pass.
- Refactor: ensure tests pass before and after.

For multi-step work, state a brief plan:
```text
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]
3. [Step] -> verify: [check]
```

These rules work when diffs shrink, rewrites decrease, and clarifying questions come before implementation mistakes.

</karpathy-guidelines>

<general-guidelines>

## General Agent Rules

### Response Rules

- Reply in the same language as the user's question.
- When clarifying requirements or boundaries, ask one question at a time and offer 3-5 concrete options with enough detail to choose, including example code when useful.
- Keep responses concise and avoid filler.

### Hard Rules

1. Never modify production code solely for tests.
   - No test-only logic, APIs, helpers, fallbacks, or workarounds.
   - Production code may change only to fix genuine production bugs.
2. Never keep compatibility code after feature changes without approval.
3. Never ignore errors.
   - Return errors whenever possible.
   - Otherwise report them with `exception.Report(...)`; if no `Report` API exists, log the error.
4. Never change public APIs unless explicitly requested.

### Security

Do not commit secrets, local credentials, coverage files, or temporary runtime data.

### Completion Checklist

After completing a feature:

1. Check for orphaned or dead code. Report findings first and confirm before deleting.
2. Check for compatibility or fallback code. Report findings first and confirm before deleting.
3. Run static analysis only for changed packages, for example `golangci-lint run --verbose ./cache/...`.
4. Run `gofmt`.
5. Create `docs/changes/v{next}-{function-description}.md` unless `docs/changes` is ignored. Match the document language to the agent response language, increment `{next}` numerically, and include:
   - Feature overview and implementation goals.
   - Requirements / business background.
   - Impact scope.
   - Modified files.
   - Behavioral changes.
   - Checks executed and result summary.
   - Unit-test coverage details, with complex logic explained.
   - Risks and optimization suggestions.
   - Orphaned/dead code.
   - Compatibility/fallback code.
   - Outstanding/incomplete items.
6. For every Go code change, the final response must report:
   - Actual coverage command executed.
   - Whether coverage came from unit tests, integration tests, or both.
   - Total statement coverage.
   - Packages or functions with significantly low coverage.
   - If coverage was skipped or partial, the exact reason.

</general-guidelines>

<go-guidelines>

## Go Project Rules

### Coding Style

Follow standard Go conventions: `gofmt`, tabs, short package names, exported `PascalCase`, unexported `camelCase`, idiomatic APIs, and naming consistent with nearby components. Prefer package-level tests and local naming patterns such as `facade_registry_test.go`, `service_provider_test.go`, and focused behavior names like `redis_lifecycle_test.go`.

#### Principles

- Prefer reuse over reimplementation; refactor only when boundaries are unclear or existing code does not fit.
- Keep functions, structs, interfaces, files, and packages single-purpose with clear boundaries.
- Prefer the Go standard library; adding a third-party dependency requires user approval.
- Favor explicit logic over implicit behavior.
- Use clear, consistent names for classes, functions, variables, tables, and fields.

#### Comments

Modified and newly added code, including tests, must include useful comments that explain logic, design rationale, complex function internals, and parameter purposes.

### Testing

Add or update colocated `*_test.go` files for behavior changes. Use focused unit tests for package behavior and integration-style tests when behavior spans components.

### Test Failure Boundary

When fixing tests, first decide whether the failure is bad test setup or a real production bug.

Do not change production code to tolerate incomplete tests. Forbidden without explicit approval:
- Adding fallback config or service behavior.
- Making required services optional.
- Replacing required resolution with silent defaults.
- Adding nil checks only to avoid test panics.
- Adding test-only branches, helpers, or alternate runtime paths.
- Weakening validation, lifecycle, or provider requirements.

Production code may change only when a failure proves a real production bug under valid runtime setup. If unsure, stop and ask. A panic from missing required test setup is a test bug, not a production bug.

</go-guidelines>

<project-guidelines>

## Framework Project Rules

### Commands

- `go test ./...`: run the full Go test suite.
- `make test`: run verbose tests with count coverage and write `coverage.out`.
- `make covdata`: run `./.github/scripts/coverage.sh` with `PACKAGES` support and write coverage artifacts under `.coverage/`.
- `make vet`: run `go vet ./...`.
- `make fmt`: format all Go files with `gofmt`, excluding `./tmp`.
- `make fmt-check`: verify formatting without modifying files.
- `make lint`: run `golangci-lint run`; pass extra options with `LINT_ARGS`, for example `make lint LINT_ARGS=--verbose`.
- `make ci`: run the local CI gate.

### Testing

Add or update colocated `*_test.go` files for behavior changes. Use focused unit tests for package contracts and integration-style tests for cross-component behavior such as queue, Redis, RabbitMQ, Horizon, or filesystem flows. Run `make test` before submitting; coverage is uploaded from `coverage.out`, so do not bypass it for final verification.

### Coverage

- Any change to Go code, `go.mod`, `go.sum`, tests, or code generation logic must run tests and compute coverage.
- Collect coverage through the project script:
  - Linux/macOS/Git Bash: `make covdata`
  - Narrow scope: `make covdata PACKAGES=./cache` or `./.github/scripts/coverage.sh ./cache`
- Coverage output goes to `.coverage/`; the script fixes Go build cache to `tmp/gocache` to avoid the user's global cache.
- Before final delivery, run the appropriate coverage command for the changed scope.
- Required coverage for the changed scope is greater than `90%`; add tests if it is lower.
- If full coverage is blocked by existing flaky tests, rerun failing packages in isolation and explain which tests failed and whether they relate to the current change. A failed full coverage run cannot be treated as passing.

</project-guidelines>