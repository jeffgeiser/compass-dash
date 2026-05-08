# compass-dash

A local dashboard for reviewing and managing [compass-md](https://github.com/jeffgeiser/compass-md) refinements.

compass-dash runs as a small HTTP server on your machine (`localhost:7174`) and serves a single-page web UI. No cloud, no accounts, no telemetry.

---

## What it does

When AI tools propose refinements to your Compass, they land in `refinements/pending/`. compass-dash is the review interface for that folder:

- **Review screen** — read pending refinements one at a time; accept, reject, or edit before accepting
- **Accept** — automatically applies the proposed change to the target file and archives the refinement
- **Edit & Accept** — opens the target file in an editor so you can apply the change manually (required when auto-apply is ambiguous)
- **Reject** — archives the refinement with a reason
- **Files screen** — browse and read your Compass substrate files
- **Activity screen** — read your `log.md` in chronological order
- **Stats** — pending count, oldest pending age, accepted/rejected this month, last review date

---

## Install

**Pre-built binaries (no Go required):**

```sh
# macOS Apple Silicon
curl -L https://github.com/jeffgeiser/compass-dash/releases/latest/download/compass-dash-darwin-arm64 -o /usr/local/bin/compass-dash
chmod +x /usr/local/bin/compass-dash
compass-dash --compass-path ~/compass

# macOS Intel
curl -L https://github.com/jeffgeiser/compass-dash/releases/latest/download/compass-dash-darwin-amd64 -o /usr/local/bin/compass-dash
chmod +x /usr/local/bin/compass-dash
compass-dash --compass-path ~/compass

# Linux
curl -L https://github.com/jeffgeiser/compass-dash/releases/latest/download/compass-dash-linux-amd64 -o /usr/local/bin/compass-dash
chmod +x /usr/local/bin/compass-dash
compass-dash --compass-path ~/compass
```

> If `/usr/local/bin` requires sudo, either prefix the `curl` and `chmod` commands with `sudo`, or replace `/usr/local/bin/compass-dash` with `~/compass-dash` to install in your home directory instead.

**From source** (requires [Go 1.21+](https://go.dev/dl/)):

```sh
git clone https://github.com/jeffgeiser/compass-dash
cd compass-dash
make build
./compass-dash --compass-path ~/compass
```

---

## Usage

```
compass-dash --compass-path /path/to/your/compass
```

Opens `http://127.0.0.1:7174` in your browser (or visit it manually).

The `--compass-path` flag sets the path and saves it to `~/.compass-dash/config.json` for future runs. After the first run you can launch without the flag.

---

## The review workflow

1. Open the Review screen
2. Click a pending refinement to read it
3. Review the Observation, Proposed change, Reasoning, and Evidence sections
4. Click **Accept** to auto-apply the change, or **Edit & Accept** to modify it first
5. If the auto-apply fails (ambiguous change), the dashboard shows why and opens the editor automatically
6. Click **Reject** and enter a reason if the refinement isn't right

The dashboard writes to your Compass files directly. Changes take effect immediately in the files.

---

## Auto-apply scope

The auto-apply logic (`compass/target.go`) is deliberately conservative. It will apply a change automatically only when:

- For **addition**: the named section exists exactly once
- For **modification**: the "before" text exists exactly once in the target file
- For **removal**: the target text exists exactly once

If any of these conditions fail, the change is surfaced as ambiguous and the Edit & Accept flow is required.

---

## Build targets

```sh
make build    # build for current platform
make test     # run tests
make release  # cross-compile for macOS (amd64/arm64), Linux, Windows
make clean    # remove build artifacts
```

---

## Requirements

- Go 1.21+ (for building)
- A Compass folder with `COMPASS.md`, `refinements/pending/`, `refinements/accepted/`, `refinements/rejected/`

---

## Non-goals

- No authentication (localhost only by design)
- No cloud sync
- No multi-user support
- No modification of refinements other than archiving (accept/reject)
