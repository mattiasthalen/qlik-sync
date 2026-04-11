# Version Check Decisions

## Decision 1: Version Check Implementation Approach

**Context:** `qs` shells out to `qlik-cli` but doesn't verify the installed version is compatible. Need to add version validation. Three approaches considered during brainstorming.

**Options considered:**
- **A) Extend CheckPrerequisites() with stdlib semver parsing** *(chosen)* — Add version check directly after PATH lookup. Parse `qlik version` output, compare against `>= 3.0.0, < 3.1.0`. Pure stdlib, ~20 lines. Keeps all prereq logic together.
- **B) Use `golang.org/x/mod/semver` package** — Same structure but with external semver library. More robust parsing but adds a dependency for something simple. Project currently has only one dep (cobra). *Rejected.*
- **C) Config-driven version bounds** — Store min/max version in config file so users can adjust without recompiling. Overengineered — version bounds are a build-time decision, not user config. *Rejected.*

**Decision:** Option A. Minimal dependencies, fits project style. Semver parsing for `major.minor.patch` format is trivial with stdlib `strings` and `strconv`.

## Decision 2: Version Compatibility Range

**Context:** Need to define what qlik-cli versions are compatible. Developed against qlik-cli 3.0.0. v3.0.0 changed API surface from v2.x.

**Options considered:**
- **A) Floor only (`>= 3.0.0`)** — Simple but allows untested future minor versions that may break API. *Rejected.*
- **B) Pin major.minor, allow patches (`>= 3.0.0, < 3.1.0`)** *(chosen)* — Standard semver approach. Patches are bug fixes, minor versions may change API.
- **C) Exact version match (`== 3.0.0`)** — Too restrictive, would reject harmless patch releases. *Rejected.*

**Decision:** Option B. `major == 3 && minor == 0`, any patch allowed. Common practice for CLI tool compatibility.

## Decision 3: Skip Flag Scope

**Context:** Issue proposes `--skip-version-check` flag. Could be placed on root command (all subcommands) or only on `setup`/`sync` where qlik-cli is actually called.

**Options considered:**
- **A) Root command flag** *(chosen)* — One flag, works everywhere. If future commands shell out to qlik-cli, no changes needed. Simple.
- **B) Per-command flag on setup/sync only** — More targeted but duplicated flag definition. Maintenance burden grows with new commands. *Rejected.*

**Decision:** Option A. Global flag on root command. `qs version` and `qs help` don't call prerequisites anyway, so the flag is harmless there.
