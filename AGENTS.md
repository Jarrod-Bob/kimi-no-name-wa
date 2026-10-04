# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

- Add durable project-specific notes here as they are discovered through real work.
- Go toolchain: `go` may not be on PATH; use `~/sdk/go1.27/bin/go` (or `export PATH=$HOME/sdk/go1.27/bin:$PATH`). The `-race` flag can't run in this sandboxed environment (no C toolchain for cgo); run plain `go test ./...`. Builds are `CGO_ENABLED=0`; the Windows binary cross-compiles with `GOOS=windows`.
- Build order matters: the frontend builds into `internal/web/dist` (`web/vite.config.ts`), which `go:embed` bakes into the binary, so run `./build.sh` (or `cd web && npm run build`) before `go build`. A Go-only build still runs and serves the API; `/` then says the UI isn't built.
- `web/go.mod` exists only to keep `web/node_modules` (some packages ship Go code) out of the root module's `./...`.
- Web checks: `npx tsc --noEmit -p tsconfig.app.json` (the root tsconfig has `files: []`), `npm test` (vitest; DOM tests opt in with `// @vitest-environment jsdom`), `npm run lint`.
- Tests must never call a real model or network service. Ollama is faked with `internal/ollama/ollamatest` (shared by package tests); the name check takes injectable GitHub/npm base URLs (`internal/namecheck`).
- Generation pipeline: `internal/namegen` owns the prompt, the JSON schema sent as Ollama's `format`, validation (`Parse` drops bad/duplicate entries individually) and the one top-up retry; `internal/ollama` is the only HTTP client for Ollama and owns its settings keys (`config.go`). Tone IDs are defined in `namegen.Tones`; `web/src/lib/tones.ts` mirrors them and adds the kanji decoration.
- The UI's own kanji come from a generated subset font file: after adding or changing any kanji/kana in `web/src`, rerun `node web/scripts/kanji-fonts.mjs` so `web/src/styles/kanji-fonts.css` covers it.
- Security posture of the API (keep it when adding endpoints): binds 127.0.0.1, `httpapi.LocalOnly` rejects non-loopback Host headers (DNS rebinding), bodies must be `application/json` (`decodeJSON`) so cross-site pages can't post without a CORS preflight, and no CORS headers are ever sent.
- No env-var config by design: CLI flags in `cmd/kimi-no-name-wa/main.go`, everything else in the `settings` table.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
