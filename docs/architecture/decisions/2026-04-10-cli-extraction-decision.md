# CLI Extraction Decisions

## Decision 1: Combine qlik-parser or Keep Separate

**Context:** Issue mattiasthalen/qlik-plugin#14 proposes extracting sync/setup into a standalone CLI. qlik-parser already exists as a separate Go CLI for QVF/QVW extraction. Both tools serve the Qlik developer workflow.

**Options considered:**
- **A) Separate repos, separate binaries** — qlik-sync is bash-to-Go rewrite, qlik-parser stays independent. Two installs, no shared code. *Rejected.*
- **B) Absorb qlik-parser into qlik-sync** — Copy parser code, archive old repo. Single binary. Feels like a hostile takeover of the parser project. *Rejected.*
- **C) Internal parser inspired by qlik-parser** *(chosen)* — qlik-sync implements its own QVF/QVW parsing as an internal package, inspired by qlik-parser's approach. Parser logic is an implementation detail of sync (on-prem flow), not an exposed command. qlik-parser stays alive as standalone tool.

**Decision:** Option C. QVF extraction is a step within `qs sync`, not a user-facing command. No `qs parse` command. Internal `internal/parser/` package designed for streaming (accepts `io.Reader`). qlik-parser remains independent for users who only need local file extraction.

## Decision 2: CLI Naming

**Context:** Need a binary name that doesn't conflict with Qlik's official `qlik` CLI (qlik-oss/qlik-cli), which the tool shells out to.

**Options considered:**
- **A) `qlik`** — Clean but PATH conflict with qlik-cli. Would need to replace it entirely. *Rejected.*
- **B) `qs`** *(chosen)* — Short acronym for qlik-sync. 2 chars, fast to type. No conflict.
- **C) `qsync`** — Clear but 5 chars, `qsync sync` reads redundant. *Rejected.*
- **D) `qx`** — Ultra-short but too generic, no obvious meaning. *Rejected.*

**Decision:** `qs` — acronym for qlik-sync. Repo stays `mattiasthalen/qlik-sync`, binary is `qs`.

## Decision 3: API Access Strategy

**Context:** Sync needs Qlik Cloud/on-prem API access. `qlik-cli` (qlik-oss/qlik-cli) is closed-source, binary-only — cannot embed as Go library.

**Options considered:**
- **A) Shell out to qlik-cli permanently** — Simple but permanent external dependency. *Rejected.*
- **B) Direct REST API from day one** — Zero deps but large upfront effort, delays v1. *Rejected.*
- **C) Phased: shell out first, REST later** *(chosen)* — Ship v1 shelling out to qlik-cli. Migrate to direct Qlik REST API in v2. Get value fast, remove dependency later.

**Decision:** Phased approach. v1 uses `qlik-cli` via `os/exec`. v2 replaces with direct `net/http` calls to Qlik REST APIs.

## Decision 4: Config Location

**Context:** Need to store tenant configuration and synced app artifacts.

**Options considered:**
- **A) XDG config dir** (`~/.config/qs/`) — Standard for CLI tools, global across projects. *Rejected* — sync output is project-specific.
- **B) Dot-prefixed project-local** (`.qlik/`) — Hidden directory, common convention. *Rejected* — no reason to hide it.
- **C) Visible project-local** (`qlik/`) *(chosen)* — Visible, fully tracked in git. Config, index, and synced artifacts all in one place.

**Decision:** `qlik/` directory at project root. Fully tracked in git — no `.gitignore` entries. Config contains no secrets (auth lives in qlik-cli contexts).

## Decision 5: Plugin Role

**Context:** Current qlik-plugin orchestrates sync via Claude Code skills. With a standalone CLI, what role does the plugin keep?

**Options considered:**
- **A) Thin wrapper only** — Plugin just translates NL to commands. *Rejected* — too limited.
- **B) Plugin dies** — No plugin at all. *Rejected* — AI-powered inspect has real value.
- **C) Plugin optional** *(chosen)* — CLI works standalone. Plugin adds natural language convenience and AI-powered inspect on top.

**Decision:** Plugin is optional. `qs` is the primary interface. Plugin becomes a thin NL layer that calls `qs` commands, with AI-powered inspect as its unique value-add.

## Decision 6: On-Prem QVF Handling

**Context:** On-prem sync downloads QVF files then extracts artifacts. Current flow writes temp file to disk.

**Options considered:**
- **A) Keep temp QVF files** — Download to `/tmp`, parse, delete. Same as bash but internal function call. *Rejected* — unnecessary I/O.
- **B) Stream parse** *(chosen)* — Download QVF bytes into memory, stream directly into parser. No temp file. Cleaner, faster.
- **C) Defer on-prem** — Ship cloud-only first. *Rejected* — on-prem is a key use case.

**Decision:** Stream parse. Internal parser accepts `io.Reader`, processes QVF bytes without touching disk. More memory but eliminates temp file lifecycle.
