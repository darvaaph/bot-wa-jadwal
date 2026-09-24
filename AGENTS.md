# Repository Guidelines

## Project Structure & Module Organization

`cmd/bot/main.go` owns CLI flags and startup wiring. Domain code lives under `internal/`: HTTP handlers in `api`, WhatsApp integration in `bot`, SQLite setup in `database`, and business logic in `schedule`, `task`, `link`, `chat`, and `reminder`. Keep helpers in `internal/util` and configuration in `internal/config`.

Schedule data belongs in `data/jadwal/`. The dashboard is served from `web/` and embedded through `web/embed.go`; keep it as plain HTML with CDN-hosted Tailwind and Alpine.js. Do not add npm, `package.json`, or a frontend build step. Runtime files belong in ignored `storage/`. Design and operational notes live in `docs/`.

## Build, Test, and Development Commands

- `go mod download` downloads the module dependencies.
- `go run ./cmd/bot -web-only` starts the dashboard and REST API at `http://localhost:8080` without connecting to WhatsApp.
- `go run ./cmd/bot -session storage/sesi_dev.db` runs the full bot with an isolated development session.
- `go build -o bin/bot ./cmd/bot` builds the single application binary.
- `go test -v ./...` runs all package tests before a pull request.
- `go vet ./...` checks common Go correctness issues.
- `gofmt -w <files>` formats changed Go files before review.

## Coding Style & Naming Conventions

Use tabs as produced by `gofmt`. Package names stay short and lowercase. Exported identifiers use `PascalCase`; local variables and functions use `camelCase`. Keep HTTP handlers in `internal/api` and use parameterized SQLite queries with `?` placeholders. Name branches by purpose, such as `feat/be-1-tasks-api`, `fix/deadline-parser`, or `docs/api-spec`.

## Testing Guidelines

Use Go's standard `testing` package. Place tests beside their implementation as `*_test.go`, and name cases `TestFunction_Scenario`. Add tests for new handlers, validation paths, database behavior, concurrency-sensitive code, and regressions. Use temporary or isolated databases, never production session files.

## Commit & Pull Request Guidelines

History follows Conventional Commits: `feat(api): ...`, `fix(schedule): ...`, `test(link): ...`, `docs: ...`, and `chore: ...`. Write imperative, focused subjects. Do not push directly to `main`.

Complete the repository PR template with a change summary, linked issue when applicable, change type, and local verification results. Include screenshots for dashboard changes or logs for behavior that is difficult to review from code. A PR needs passing tests, a clean `go vet ./...`, and at least one reviewer approval.

## Security & Local State

Never commit `.env` files, SQLite databases, WhatsApp sessions, or compiled binaries. Use `storage/sesi_dev.db` for local bot testing and leave `storage/sesi_bot.db` untouched.
