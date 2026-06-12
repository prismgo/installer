# Repository Guidelines

## Karpathy Guidelines

These guidelines reduce common LLM coding mistakes. Merge them with project-specific instructions. They intentionally favor caution over speed; use judgment for trivial tasks.

### 1. Think Before Coding

Do not assume or hide uncertainty.

- State assumptions before implementing; ask when unsure.
- If a request has multiple interpretations, surface them instead of choosing silently.
- Point out simpler approaches and push back when warranted.
- If something is unclear, stop, name the ambiguity, and ask.

### 2. Simplicity First

Write the minimum code that solves the request.

- Do not add unrequested features, abstractions, flexibility, configurability, or impossible-case error handling.
- If a solution is much larger than necessary, simplify it.

### 3. Surgical Changes

Touch only what the request requires, and clean up only issues introduced by your change.

- Do not improve, refactor, reformat, or delete adjacent unrelated code.
- Match existing style.
- Mention unrelated dead code instead of deleting it.
- Remove imports, variables, functions, or other orphans created by your own changes.
- Every changed line should trace directly to the user request.

### 4. Goal-Driven Execution

Turn work into verifiable goals and loop until verified.

- Validation changes: test invalid inputs, then make tests pass.
- Bug fixes: reproduce with a test, then make it pass.
- Refactors: ensure tests pass before and after.
- For multi-step work, state a short plan:

```text
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]
3. [Step] -> verify: [check]
```

These guidelines are working when diffs are smaller, rewrites are rarer, and clarifying questions happen before implementation mistakes.

## Hard Rules

1. Never modify production code solely for testing: no test-only logic, APIs, helpers, fallbacks, or workarounds. Production code may change only to fix real production bugs.
2. Never keep compatibility code after feature changes without approval.
3. Never ignore errors; return them whenever possible.
4. Never change public APIs unless explicitly requested.

## Build, Test, and Development Commands

- `go test ./...`: run the full Go test suite.
- `make test`: run verbose tests with count coverage and write `coverage.out`.
- `make covdata`: run `./.github/scripts/coverage.sh` with `PACKAGES` support and write coverage artifacts under `.coverage/`.
- `make vet`: run `go vet ./...`.
- `make fmt`: format all Go files with `gofmt`, excluding `./tmp`.
- `make fmt-check`: verify formatting without modifying files.
- `make lint`: run `golangci-lint run`; pass extra options with `LINT_ARGS`, for example `make lint LINT_ARGS=--verbose`.
- `make ci`: run the local CI gate.

## Coding Style and Standards

Use standard Go conventions: `gofmt`, tabs, short package names, exported `PascalCase`, unexported `camelCase`, and idiomatic APIs consistent with nearby code.

- Prefer reuse over reimplementation; refactor properly when existing code does not fit or boundaries are unclear.
- Follow single responsibility for functions, structs, interfaces, files, and packages.
- Prefer the Go standard library and minimize dependencies. New libraries require user approval.
- Favor explicit logic over implicit behavior.
- Use consistent, clear names for classes, functions, variables, tables, and fields.
- Modified and newly added code, including tests, must include useful comments explaining logic, rationale, complex flows, and parameter purposes.

## Testing Guidelines

Add or update colocated `*_test.go` files for behavior changes. Use focused unit tests for package contracts and integration-style tests for cross-component behavior such as queue, Redis, RabbitMQ, Horizon, or filesystem flows. Run `make test` before submitting; coverage is uploaded from `coverage.out`, so do not bypass it for final verification.

### Testing and Coverage

- Any change to Go code, `go.mod`, `go.sum`, tests, or code generation logic must run tests and compute coverage.
- Collect coverage through the project script:
  - Linux/macOS/Git Bash: `make covdata`
  - Narrow scope: `make covdata PACKAGES=./cache` or `./.github/scripts/coverage.sh ./cache`
- Coverage output goes to `.coverage/`; the script fixes Go build cache to `tmp/gocache` to avoid the user's global cache.
- Before final delivery, run the appropriate coverage command for the changed scope.
- Required coverage for the changed scope is greater than `90%`; add tests if it is lower.
- If full coverage is blocked by existing flaky tests, rerun failing packages in isolation and explain which tests failed and whether they relate to the current change. A failed full coverage run cannot be treated as passing.

## Security and Configuration

Do not commit secrets, local credentials, coverage files, or temporary runtime data.

## Completion Checklist

After completing a feature:

1. Check for orphaned/dead code. Report findings first and ask before deleting.
2. Check for compatibility/fallback code. Report findings first and ask before deleting.
3. Run static analysis only for packages containing changed code, for example `golangci-lint run --verbose ./cmd/...`.
4. Run `gofmt`.
5. Write `docs/changes/v{next}-{function-description}.md`. If `docs/changes` is ignored, do not commit the document. The document must be in Chinese, increment `{next}` numerically, and include:
   - Feature overview and implementation goals
   - Requirements / business background
   - Impact scope
   - Modified files
   - Behavioral changes
   - Checks executed and result summary
   - Unit-test coverage, with detailed explanation for complex logic
   - Risks and optimization suggestions
   - Orphaned/dead code
   - Compatibility/fallback code
   - Outstanding/incomplete items
6. For every Go code change task, the final response must report:
   - Actual coverage command executed
   - Whether coverage came from unit tests, integration tests, or both
   - Total statement coverage
   - Packages or functions with significantly low coverage
   - Exact reason when coverage is skipped or only partially run
