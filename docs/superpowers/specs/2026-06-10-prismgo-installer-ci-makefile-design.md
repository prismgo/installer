# PrismGo Installer CI and Makefile Design

## Goal

Adapt `.github` and `Makefile` from the current Prismgo Lens / generic Go template into a PrismGo Installer repository setup.

The installer will eventually behave like Laravel's `laravel new {yourapp}` command, with the expected user-facing command:

```shell
prismgo new tmp/app
```

The current repository does not yet contain `go.mod`, Go source code, or `cmd/prismgo`. Therefore this change must keep CI green while making the intended future checks explicit.

## Confirmed Requirements

- The project is the PrismGo installer, not Prismgo Lens.
- Go version remains `1.26`.
- CI and Makefile should ultimately cover both:
  - the installer itself;
  - generated PrismGo application skeletons.
- The generated application validation step is intentionally skipped for now.
- Current CI must stay green while CLI source code is absent.
- Go checks should skip with clear messages when `go.mod` is absent.
- CLI build / install checks should skip with clear messages when `cmd/prismgo` is absent.
- No production code, public API, or compatibility behavior should be introduced as part of this configuration-only change.

## Scope

In scope:

- `.github/workflows/ci.yml`
- `.github/workflows/codeql.yml`
- `.github/dependabot.yml`
- `.github/PULL_REQUEST_TEMPLATE.md`
- `.github/ISSUE_TEMPLATE/*.yaml`
- `Makefile`
- Optional project documentation updates if needed to describe available Make targets

Out of scope:

- Implementing `prismgo`
- Adding `go.mod`
- Adding installer CLI source code
- Running `prismgo new tmp/app` in CI
- Validating generated application files
- Adding release automation
- Adding package publishing

## Makefile Design

The Makefile should keep these standard targets:

- `fmt`
- `fmt-check`
- `vet`
- `test`
- `lint`
- `ci`

It should add or preserve these future-facing targets:

- `build`
- `install`
- `smoke-new`

Behavior:

- If `go.mod` is missing, `fmt`, `fmt-check`, `vet`, `test`, and `lint` should print a clear skip message and exit successfully.
- If `cmd/prismgo` is missing, `build` and `install` should print a clear skip message and exit successfully.
- `smoke-new` should remain explicit and should not be part of `ci` yet.
- `ci` should run only the checks that are valid for the current repository state.

Rationale:

- This keeps the repository usable before CLI source exists.
- The target names document the intended workflow.
- Once `go.mod` and `cmd/prismgo` are added, the same targets can start performing real work without changing developer muscle memory.

## GitHub Actions Design

### CI Workflow

`.github/workflows/ci.yml` should:

- be named `PrismGo Installer CI`;
- run on pushes and pull requests targeting `main`;
- use Go `1.26`;
- run `make ci`;
- avoid invoking `prismgo new tmp/app` for now.

The workflow should rely on Makefile skip behavior to remain green while no Go module exists.

### CodeQL Workflow

`.github/workflows/codeql.yml` should:

- remain enabled for Go;
- use a PrismGo Installer-oriented name;
- keep standard GitHub permissions for code scanning.

Although CodeQL has little value before Go source exists, keeping the workflow avoids a second configuration pass when the installer implementation is added.

### Dependabot

`.github/dependabot.yml` should:

- keep GitHub Actions dependency updates;
- remove the `gomod` ecosystem entry until `go.mod` exists.

This avoids a configuration entry that does not match the current repository shape.

## GitHub Templates Design

Pull request and issue templates should be renamed from Prismgo Lens to PrismGo Installer.

Installer-oriented areas should replace Lens-specific areas. Suitable values include:

- CLI
- Project generation
- Template rendering
- Dependency installation
- Configuration writing
- Error output
- Cross-platform behavior
- Documentation
- Other

Lens-specific references should be removed, including:

- MCP
- Agent configuration
- host project detection
- document search
- browser logs
- skill management
- `.prismgo-lens.json`

## Testing and Verification

For this configuration-only change, verification should include:

- `make ci`
- YAML syntax sanity through GitHub Actions workflow structure review
- checking that no Prismgo Lens references remain in `.github` or `Makefile`
- checking for orphaned or fallback compatibility logic introduced by the change

Coverage is not applicable until Go source or tests exist. If the repository still has no `go.mod`, coverage should be reported as skipped because there are no Go packages to test.

## Risks

- Skip behavior can hide missing implementation if kept too long.
- `smoke-new` is intentionally not in CI yet, so generated application correctness is not validated.
- CodeQL may not provide useful feedback until Go source files are added.

These risks are acceptable for the current repository state because the approved goal is to keep CI green while preparing the installer project skeleton.

## Future Work

When installer source code is added:

- add `go.mod`;
- add `cmd/prismgo`;
- make `build` produce `bin/prismgo`;
- make `install` run `go install ./cmd/prismgo`;
- enable `smoke-new` to run `prismgo new tmp/app`;
- decide whether generated projects should run `go test ./...`, `make test`, or `make ci`;
- restore Dependabot `gomod` updates.
