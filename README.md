# kimi-no-name-wa

Cool, witty project-name ideas from a description of your work. Local UI and API, powered by a local Ollama model.

The name is the idea: *Kimi no Na wa* (君の名は, "Your Name") is the anime film, and this is the app that names things. That is the bar for the names it suggests. It aims past "DocuReader Pro" toward portmanteaus (**acceleread** = ACCELErate + READ, for a tool that ingests documents faster), metaphors (**nuggets**, for little nuggets of information), puns, pop-culture riffs, and names drawn from whatever is around you.

## What it does

- **Generate names** from a description of what you're building. You can add:
  - **Surroundings and inspiration**: things you like or can see right now, such as a game, your desk or the weather. They become raw material for the names.
  - **Keywords** to work in.
  - **Tones**: witty, punny, cool/sleek, comical, melancholic, joking, mythic/epic and Japanese-flavoured. Pick any mix.
  - **How many** names you want (1–20).

  Each name comes back with the **technique** behind it (portmanteau, pun, cultural reference, metaphor, acronym, foreign word, alliteration, compound, respelling), a **one-line explanation** of why it fits, and its **tone**.
- **More like this** on any name asks for names in the same spirit.
- **Re-roll** asks again for the same brief and never repeats a name already shown this session.
- **Favourites**: star a name to keep it, and copy any name with one click.
- **History**: every batch you or another app generated, with its inputs. Deleting a batch keeps its starred names.
- **Taken?** On request, checks one name for a GitHub repository with exactly that name and an npm package. It is best-effort and needs no API key (details under [Name check](#name-check)).
- **An HTTP API** so other local apps, such as nuggets, can ask for names. See [API](#api).
- **Settings** for the Ollama URL and model, saved in the app's database.

The app starts and works whether or not Ollama is installed. Until Ollama is reachable and the model is pulled, the page and `GET /api/v1/health` say exactly what's missing.

## 1. Install Ollama and pull the model

Names come from a model running on your own machine through [Ollama](https://ollama.com). Nothing is sent to a cloud LLM.

1. Install Ollama for Windows from <https://ollama.com/download> and start it. It listens on `http://127.0.0.1:11434`.
2. Pull the default model, in PowerShell or cmd:

   ```
   ollama pull gemma4:31b
   ```

**Why `gemma4:31b`:** Google's Gemma 4 31B (dense) is strong at creative wordplay and at Japanese. It is about 20 GB at Ollama's default quantisation, so it fits an RTX 5090's 32 GB with room for context. The first request after Ollama starts loads the model into VRAM and can take a minute. Later requests take seconds.

Smaller GPUs can use another model. Pull it, then pick it in **Settings**:

| Model | Size | Notes |
|---|---|---|
| `gemma4:31b` | ~20 GB | Default. Best wordplay. Wants a 24 GB+ GPU. |
| `gemma4:26b` | ~18 GB | Mixture-of-experts: faster, slightly less inventive. |
| `gemma4:12b` | ~8 GB | For ~12 GB GPUs. |

Any Ollama chat model works. Models that support Ollama's structured outputs (JSON schema in `format`) give the cleanest results.

## 2. Build and run

You need [Go](https://go.dev/dl/) 1.25+ and [Node.js](https://nodejs.org/) 20+. Each script builds the frontend (installing its packages on the first run) and embeds it into one self-contained binary with no cgo.

```powershell
# Windows, PowerShell
./build.ps1
./kimi-no-name-wa.exe
```

```bat
:: Windows, cmd.exe. Use this if PowerShell refuses to run build.ps1
:: ("running scripts is disabled on this system"): batch files aren't
:: subject to PowerShell's execution policy.
build.cmd
kimi-no-name-wa.exe
```

```bash
# Linux / macOS
./build.sh
./kimi-no-name-wa
```

Running it opens `http://127.0.0.1:7799` in your browser. For a chromeless, app-like window instead:

```
msedge --app=http://127.0.0.1:7799
```

To cross-compile the Windows binary from Linux or WSL, build the frontend first (`cd web && npm ci && npm run build`), then run `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o kimi-no-name-wa.exe ./cmd/kimi-no-name-wa`.

## Configuration

**Command-line flags** (`kimi-no-name-wa -help`):

| Flag | Default | |
|---|---|---|
| `-addr` | `127.0.0.1:7799` | Address to listen on. Keep it on 127.0.0.1: `:7799` would expose the API to your network, and Windows asks for a firewall exception on every rebuild. |
| `-db` | `%AppData%\kimi-no-name-wa\kimi.db` | SQLite database file (`~/.config/kimi-no-name-wa/kimi.db` on Linux). Back up by copying it. |
| `-open` | `true` | Open a browser on start. |

**In the app** (**Settings**, or `PUT /api/v1/settings`), stored in the database:

| Setting | Default |
|---|---|
| Ollama URL | `http://127.0.0.1:11434` |
| Model | `gemma4:31b` |

There are no environment variables.

## API

Everything is JSON under `/api/v1` on `http://127.0.0.1:7799`. The web UI uses the same endpoints.

- **Requests with a body** must send `Content-Type: application/json`. A web page on another site can't send that without a CORS preflight, and this server grants none. That stops websites you visit from driving the API.
- **Callers:** another local app calls it from its own backend, server to server. A browser page served from a different origin can't call it directly.
- **Host check:** requests must be addressed to `localhost` or `127.0.0.1`. This blocks DNS-rebinding attacks.
- **Errors** always look like this:

  ```json
  { "error": { "message": "A sentence you can show a person.", "code": "ollama_unreachable" } }
  ```

### `POST /api/v1/names`: generate names

Request. Only `description` is required:

```json
{
  "description": "A bank for the little ideas I have while I'm out",
  "inspiration": ["maplestory", "cold brew"],
  "keywords": ["idea", "save"],
  "tones": ["witty", "japanese"],
  "count": 8,
  "avoid": ["nuggets"],
  "like": { "name": "acceleread", "technique": "portmanteau", "explanation": "ACCELErate + READ" },
  "client": "nuggets"
}
```

| Field | | |
|---|---|---|
| `description` | string, required | What's being named. Up to 2000 characters. |
| `inspiration` | string[] | Surroundings and things the person likes. Up to 20 entries of up to 80 characters each. |
| `keywords` | string[] | Words to work in. Same limits. |
| `tones` | string[] | Any of `witty`, `punny`, `sleek`, `comical`, `melancholic`, `joking`, `mythic`, `japanese`. Default `["witty","punny","sleek"]`. |
| `count` | number | 1–20, default 8. |
| `avoid` | string[] | Names not to suggest again, such as ones already shown. Matching ignores case, spaces and punctuation. Up to 500 entries. |
| `like` | object | "More like this": a name (`name` required) whose spirit to follow. |
| `client` | string | Who's asking. History shows it. Default `"api"` (the web UI sends `"web"`). |

Response `200`: the generation as saved in history.

```json
{
  "id": 12,
  "created_at": "2026-10-03T09:15:02Z",
  "client": "nuggets",
  "model": "gemma4:31b",
  "description": "A bank for the little ideas I have while I'm out",
  "inspiration": ["maplestory", "cold brew"],
  "keywords": ["idea", "save"],
  "tones": ["witty", "japanese"],
  "like_name": "acceleread",
  "names": [
    {
      "id": 87,
      "generation_id": 12,
      "name": "Ideanori",
      "technique": "portmanteau",
      "explanation": "IDEA + onigiri: small, portable, packed for later.",
      "tone": "japanese",
      "favourite": false,
      "favourited_at": null
    }
  ]
}
```

`names` may hold fewer than `count` names. The model is asked for structured JSON and every entry is validated. Entries that are malformed, use a tone that wasn't requested, or repeat a name already in the batch or in `avoid` are dropped, and the rest of the batch is kept. If a reply leaves fewer than half the names wanted, the model is asked once more and the results are merged.

Errors:

| Status | `code` | Meaning |
|---|---|---|
| 400 | `invalid_request`, `invalid_json` | Fix the request. The message says how. |
| 415 | `unsupported_media_type` | Send `Content-Type: application/json`. |
| 503 | `ollama_unreachable` | Nothing answered at the Ollama URL. |
| 503 | `model_missing` | Ollama is running but the model isn't pulled. The message includes the `ollama pull` command. |
| 504 | `model_timeout` | The model didn't answer in 5 minutes. |
| 502 | `no_names` | The model answered twice without one usable name. |
| 502 | `model_error` | Ollama failed some other way, such as not being able to load the model. The message has its error, and [health](#get-apiv1health-is-the-model-ready) reports `degraded` until a generation works again. |

Example:

```bash
curl -s http://127.0.0.1:7799/api/v1/names \
  -H "Content-Type: application/json" \
  -d '{"description":"a bank for little ideas","tones":["witty","japanese"],"count":5,"client":"my-app"}'
```

### `GET /api/v1/health`: is the model ready?

Always `200` while the app is running. `status` is `"ok"` when names can be generated, else `"degraded"`, with `ollama.error` saying why and what to do.

Health doesn't load the model itself, so it can't see a model that is pulled but won't run (say, the GPU on Ollama's host fails to initialise) until a generation tries it. After `POST /api/v1/names` fails with `model_error`, health reports `"degraded"` with an `error` saying the model failed to load, until a generation with the same URL and model succeeds.

```json
{
  "status": "degraded",
  "ollama": {
    "url": "http://127.0.0.1:11434",
    "reachable": true,
    "version": "0.12.3",
    "model": "gemma4:31b",
    "model_pulled": false,
    "models": ["llama3.2:latest"],
    "error": "Ollama is running but the model gemma4:31b isn't pulled yet. Run: ollama pull gemma4:31b"
  }
}
```

A caller such as nuggets can check this before showing a "Name this nugget" button, or show the `error` text.

### Other endpoints

| Method and path | |
|---|---|
| `GET /api/v1/tones` | The tones (`id`, `label`, `guidance`), the techniques, and the count limits. |
| `GET /api/v1/generations?limit=20&before=<id>` | History, newest first: `{ "generations": [...], "next_before": <id or null> }`. Pass `next_before` as `before` for the next page. |
| `DELETE /api/v1/generations/{id}` | Delete a batch (`204`). Its starred names stay in favourites. |
| `GET /api/v1/favourites` | `{ "favourites": [name, ...] }`, most recently starred first. Each name carries the `description` it was generated for, or none once its batch is deleted. |
| `PUT /api/v1/names/{id}/favourite` | Star a name. Returns the name. |
| `DELETE /api/v1/names/{id}/favourite` | Unstar a name. Returns the name. |
| `GET /api/v1/check?name=<name>` | [Name check](#name-check). |
| `GET /api/v1/settings` | `{ "ollama_url", "model", "defaults": { "ollama_url", "model" } }` |
| `PUT /api/v1/settings` | Body `{ "ollama_url", "model" }`. Returns the saved settings, or `400` if the URL isn't `http(s)://` or the model name has spaces. |

## Name check

**Taken?** on a name turns it into a repository or package slug (`Kimi no Name wa` becomes `kimi-no-name-wa`), then asks two services without a key:

- **GitHub**: searches repositories by name and reports **taken** when one is named exactly the slug, linking the most-starred match. It reports **free** otherwise, even if other repositories merely contain the slug.
- **npm**: looks the slug up in the registry. It reports **taken** if the package exists and **free** if it doesn't.

GitHub allows 10 unauthenticated searches a minute. Beyond that, a check says **unknown** and asks you to try again in a minute. Definite answers are cached for 10 minutes. These checks are the only time the app contacts anything besides Ollama. They run only when you press the button, and they send nothing but the name.

## Development

```bash
# Backend tests. These never call a real model or service: they use an
# httptest fake Ollama (internal/ollama/ollamatest) and fake GitHub/npm servers.
go test ./...

cd web
npm ci
npm test                                # vitest
npx tsc --noEmit -p tsconfig.app.json   # typecheck
npm run lint
npm run dev                             # Vite on :5173, proxying /api to a running binary on :7799
```

| | |
|---|---|
| Backend | Go: stdlib `net/http`, SQLite via `modernc.org/sqlite` (pure Go, `CGO_ENABLED=0`), `goose` migrations |
| Frontend | TypeScript, React, Vite, hand-written CSS tokens (`web/src/styles/index.css`) |
| Shape | One binary serving the JSON API and the embedded frontend on `127.0.0.1:7799` |
| Model | Ollama's `/api/chat` with a JSON-schema `format` (`internal/ollama`, `internal/namegen`) |

The code is laid out like its sibling app [nuggets](https://github.com/Jarrod-Bob/nuggets). `AGENTS.md` has build notes for agents.
