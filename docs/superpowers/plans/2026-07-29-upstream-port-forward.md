# Upstream Port-Forward + Audio Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge `stashcoder42-main` forward onto upstream `develop` (161 commits, v0.31.1), then exhaustively port every upstream scene fix into our audio twin so audio reaches behavioural parity.

**Architecture:** Two phases with a hard gate. Phase 1 (Tasks 1–6) rebuilds the composite branch on the new upstream base and resolves textual merge conflicts — mechanical, verifiable by build+test. Phase 2 (Tasks 7–20) walks the 25-commit ledger in `docs/AUDIO_SCENE_PARITY.md`, porting scene changes into audio files that git merged clean and therefore silently left stale. Phase 2 cannot start until Phase 1 builds and tests green.

**Tech Stack:** Go 1.25.0, SQLite/goqu, gqlgen, React 18 + TypeScript, biome 2.5.0, pnpm.

## Global Constraints

- Go version floor is **1.25.0** (upstream raised it from 1.24.3 in `2b29207f1`).
- Frontend formatter/linter is **biome 2.5.0**. prettier and eslint are gone; `ui/v2.5/.eslintrc.json` was deleted upstream.
- Never run `go generate ./...` — dataloaden emits duplicate imports. Use `make generate-backend`.
- Never hand-edit generated files (`*_gen.go`, `pkg/models/mocks/*`, `src/core/generated-graphql.ts`). Regenerate them.
- Never use `git add .` — always add explicit filenames.
- Use `pnpm`, never `npm`.
- Do not create feature branches. Work proceeds on `stashcoder42-main` and the three component branches.
- `cd` into a directory before running git; do not use `git -C`.
- Component branch composition is fixed: `origin/develop` + `feat/audios` + `feat/file-watcher` + `stashcoder42/workflow`.
- Ledger authority: `docs/AUDIO_SCENE_PARITY.md`. Every Phase 2 task updates its ledger row status in the same commit as the code change.

## Verification Commands

Used throughout; memorise these.

```bash
cd /Users/myers/x/stash
make generate-backend     # regenerate resolvers, models, mocks, loaders
make generate-ui          # regenerate frontend graphql types
make build                # build stash binary
make test                 # backend unit tests
make lint                 # backend linter
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check        # tsc --noEmit
cd /Users/myers/x/stash/ui/v2.5 && pnpm run lint:js      # biome lint
cd /Users/myers/x/stash/ui/v2.5 && pnpm run format-check # biome format check
```

Single Go test:

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run TestAudioFindByPath -v
```

## File Structure

**Phase 1 touches:**
- `go.mod`, `go.sum` — dependency reconciliation
- `internal/manager/config/config.go` — config key conflict
- `pkg/models/tag.go`, `pkg/models/mocks/TagReaderWriter.go` — regenerate mocks
- `ui/v2.5/package.json` — dependency conflict
- `ui/v2.5/src/index.scss`, `ui/v2.5/src/locales/en-GB.json` — additive conflicts
- `ui/v2.5/src/components/{List/Filters/LabeledIdFilter,Performers/PerformerDetails/Performer,Shared/TagLink,Tags/TagDetails/Tag}.tsx`

**Phase 2 touches** the 29 audio twin files enumerated in `docs/AUDIO_SCENE_PARITY.md`. Each task modifies only the audio side; the scene side is upstream's and must not be edited.

**No new files** are created except where a ported upstream commit adds one.

---

## Phase 1 — Merge Forward

### Task 1: Fast-forward develop and re-establish the base

**Files:**
- Modify: none (branch pointers only)

**Interfaces:**
- Produces: local `develop` at `afdaa082b`; a clean `stashcoder42-main` reset to that base, ready for component merges.

- [ ] **Step 1: Confirm no local divergence before moving anything**

```bash
cd /Users/myers/x/stash
git fetch upstream
git rev-list --count upstream/develop..origin/develop
```

Expected: `0`. If this prints anything other than `0`, STOP — our develop has commits upstream lacks and this plan's fast-forward assumption is void. Report to the user before continuing.

- [ ] **Step 2: Fast-forward develop**

```bash
cd /Users/myers/x/stash
git checkout develop
git merge --ff-only upstream/develop
```

Expected: fast-forward to `afdaa082b`.

- [ ] **Step 3: Push develop**

```bash
cd /Users/myers/x/stash
git push origin develop
```

- [ ] **Step 4: Reset the composite branch to the new base**

```bash
cd /Users/myers/x/stash
git checkout -B stashcoder42-main origin/develop
git branch --set-upstream-to=origin/stashcoder42-main stashcoder42-main
```

- [ ] **Step 5: Verify the base builds before adding our patches**

```bash
cd /Users/myers/x/stash && make build
```

Expected: builds clean. This is upstream's own code — if it fails, the problem is toolchain (Go version), not our patches. Confirm `go version` reports 1.25.0 or later.

### Task 2: Merge stashcoder42/workflow

**Files:**
- Modify: `.github/workflows/` (single file, additive)

**Interfaces:**
- Consumes: `stashcoder42-main` at upstream base from Task 1.
- Produces: workflow branch merged, zero conflicts.

- [ ] **Step 1: Merge**

```bash
cd /Users/myers/x/stash
git merge stashcoder42/workflow --no-edit
```

Expected: clean merge, zero conflicts (verified during scoping).

- [ ] **Step 2: Verify no conflict markers landed**

```bash
cd /Users/myers/x/stash
git diff --name-only --diff-filter=U
```

Expected: empty output.

### Task 3: Merge feat/file-watcher and resolve go.mod

**Files:**
- Modify: `go.mod`, `go.sum`

**Interfaces:**
- Consumes: `stashcoder42-main` from Task 2.
- Produces: `github.com/fsnotify/fsnotify v1.9.0` present as a **direct** requirement, file-watcher code compiling.

**Context:** The only conflict is `fsnotify`. Upstream lists it as an indirect dependency; our file-watcher promotes it to direct. Resolution is to keep it in the direct `require` block and remove the `// indirect` entry.

- [ ] **Step 1: Merge and observe the conflict**

```bash
cd /Users/myers/x/stash
git merge feat/file-watcher --no-edit
git diff --name-only --diff-filter=U
```

Expected: exactly `go.mod`.

- [ ] **Step 2: Resolve go.mod**

Open `go.mod`. In the first (direct) `require` block, ensure this line is present, alphabetically after `github.com/doug-martin/goqu/v9`:

```
	github.com/fsnotify/fsnotify v1.9.0
```

In the second (indirect) `require` block, delete this line if present:

```
	github.com/fsnotify/fsnotify v1.9.0 // indirect
```

Keep **all** other upstream additions in both blocks — `github.com/enetx/g`, `github.com/enetx/surf`, `github.com/feederbox826/gosx-notifier`, and the `go 1.25.0` directive. Remove every conflict marker (`<<<<<<<`, `=======`, `>>>>>>>`).

- [ ] **Step 3: Let Go reconcile the module graph**

```bash
cd /Users/myers/x/stash
go mod tidy
```

- [ ] **Step 4: Verify fsnotify is direct and no markers remain**

```bash
cd /Users/myers/x/stash
grep -n 'fsnotify' go.mod
grep -c '<<<<<<<\|>>>>>>>' go.mod
```

Expected: one `fsnotify` line with **no** `// indirect` suffix; marker count `0`.

- [ ] **Step 5: Build**

```bash
cd /Users/myers/x/stash && make build
```

Expected: success.

- [ ] **Step 6: Complete the merge commit**

```bash
cd /Users/myers/x/stash
git add go.mod go.sum
git commit --no-edit
```

### Task 4: Merge feat/audios — backend conflicts

**Files:**
- Modify: `internal/manager/config/config.go`, `pkg/models/tag.go`
- Regenerate: `pkg/models/mocks/TagReaderWriter.go`

**Interfaces:**
- Consumes: `stashcoder42-main` from Task 3.
- Produces: audio backend merged; mocks regenerated to match upstream's `TagReaderWriter` interface.

**Context:** `TagReaderWriter.go` is generated. Do not hand-resolve it — take any side to clear the conflict, then regenerate. Upstream commits `9cb2ffd56` (tag count depth) and `5ed738558` changed the tag interface.

- [ ] **Step 1: Merge and list conflicts**

```bash
cd /Users/myers/x/stash
git merge feat/audios --no-edit
git diff --name-only --diff-filter=U
```

Expected: 10 files — `internal/manager/config/config.go`, `pkg/models/mocks/TagReaderWriter.go`, `pkg/models/tag.go`, `ui/v2.5/package.json`, and 6 UI files. This task handles the three backend ones; Task 5 handles UI.

- [ ] **Step 2: Resolve `internal/manager/config/config.go`**

One conflict hunk, ~6 lines. Both sides add config keys. Keep **both** — upstream's new keys (from `d4e02f754` security tripwire, `058356004` marker ceiling, `8abbac98d` JXL, `d04ecc4f8` airplay) and our audio keys. Preserve upstream's ordering and place our audio keys adjacent to the other media-type keys. Remove all conflict markers.

- [ ] **Step 3: Resolve `pkg/models/tag.go`**

One hunk, ~7 lines. Upstream added depth-aware tag count fields; our side added audio-related tag fields. Keep both sets. Remove all conflict markers.

- [ ] **Step 4: Clear the generated mock conflict, then regenerate**

```bash
cd /Users/myers/x/stash
git checkout --theirs pkg/models/mocks/TagReaderWriter.go
make generate-backend
```

The `--theirs` is only to clear markers so the generator can parse the tree; `make generate-backend` overwrites the file from the interface definition, which is the real source of truth.

- [ ] **Step 5: Verify no markers survive in backend files**

```bash
cd /Users/myers/x/stash
grep -rn '<<<<<<<\|>>>>>>>' internal/ pkg/ --include='*.go' | head
```

Expected: no output.

- [ ] **Step 6: Build and run backend tests**

```bash
cd /Users/myers/x/stash && make build && make test
```

Expected: build succeeds, tests pass. Do not proceed with failures — record any failure verbatim and report it.

### Task 5: Merge feat/audios — frontend conflicts

**Files:**
- Modify: `ui/v2.5/package.json`, `ui/v2.5/src/index.scss`, `ui/v2.5/src/locales/en-GB.json`, `ui/v2.5/src/components/List/Filters/LabeledIdFilter.tsx`, `ui/v2.5/src/components/Performers/PerformerDetails/Performer.tsx`, `ui/v2.5/src/components/Shared/TagLink.tsx`, `ui/v2.5/src/components/Tags/TagDetails/Tag.tsx`

**Interfaces:**
- Consumes: in-progress `feat/audios` merge from Task 4.
- Produces: merge commit complete; frontend typechecks.

**Context:** `Performer.tsx` is the hairiest — 7 hunks, ~55 lines, because upstream `b044005fc` added recursive performer_count/o_counter sort UI while our side added audio counts. `index.scss` is the largest (~57 lines) but purely additive on both sides.

- [ ] **Step 1: Resolve `ui/v2.5/package.json`**

One hunk, ~8 lines. Upstream replaced prettier/eslint deps with `"@biomejs/biome": "2.5.0"` and changed the `lint`/`format` scripts. **Take upstream's side entirely** for the scripts and devDependencies — our branch has no package.json changes worth preserving here beyond any audio-specific runtime dependency. If our side added a runtime dependency (check the conflict body), re-add it to `dependencies` on top of upstream's version.

- [ ] **Step 2: Resolve `ui/v2.5/src/index.scss` and `en-GB.json`**

Both are additive-only. Keep both sides in full. For `en-GB.json`, ensure the result is valid JSON — no duplicate keys, no trailing comma. Verify:

```bash
cd /Users/myers/x/stash && python3 -m json.tool ui/v2.5/src/locales/en-GB.json > /dev/null && echo "valid json"
```

Expected: `valid json`.

- [ ] **Step 3: Resolve the four component files**

For `LabeledIdFilter.tsx`, `TagLink.tsx`, `Tag.tsx` — one hunk each (7, 7, 9 lines). Upstream's changes come from `9cb2ffd56` (tag count depth) and biome reformatting. Keep upstream's structural changes and re-apply our audio additions on top.

For `Performer.tsx` — 7 hunks. Upstream `b044005fc` added recursive sort by `performer_count`/`o_counter` and display updates. Our side added audio counts to the performer detail tabs. Keep both: upstream's new sort/display code AND our audio tab entries. Work hunk by hunk; do not take one side wholesale.

- [ ] **Step 4: Verify no markers survive**

```bash
cd /Users/myers/x/stash
grep -rn '<<<<<<<\|>>>>>>>' ui/v2.5/src ui/v2.5/package.json | head
```

Expected: no output.

- [ ] **Step 5: Install deps and regenerate frontend types**

```bash
cd /Users/myers/x/stash && make pre-ui && make generate-ui
```

- [ ] **Step 6: Typecheck**

```bash
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check
```

Expected: no errors. Biome lint is deliberately deferred to Task 6.

- [ ] **Step 7: Complete the merge commit**

```bash
cd /Users/myers/x/stash
git add go.mod go.sum internal/manager/config/config.go pkg/models/tag.go pkg/models/mocks/TagReaderWriter.go ui/v2.5/package.json ui/v2.5/src/index.scss ui/v2.5/src/locales/en-GB.json ui/v2.5/src/components/List/Filters/LabeledIdFilter.tsx ui/v2.5/src/components/Performers/PerformerDetails/Performer.tsx ui/v2.5/src/components/Shared/TagLink.tsx ui/v2.5/src/components/Tags/TagDetails/Tag.tsx
git add ui/v2.5/src/core/generated-graphql.ts
git commit --no-edit
```

### Task 6: Biome sweep over audio files

**Files:**
- Modify: all audio-owned UI files (~120), formatting only

**Interfaces:**
- Consumes: merged tree from Task 5.
- Produces: audio UI files biome-clean, isolated in one commit so future diffs stay readable.

**Context:** Our ~120 audio UI files were written under prettier and never biome-formatted. Upstream's `biome.jsonc` uses `"includes": ["**"]`, so audio files are already in scope — no config change needed. Keeping this as its own commit is the point: mixing reformatting into logic commits poisons every future `git blame` and resync diff.

- [ ] **Step 1: Confirm the scale of the reformat before applying**

```bash
cd /Users/myers/x/stash/ui/v2.5 && pnpm run format-check 2>&1 | tail -20
```

Expected: a list of files needing formatting. This is informational.

- [ ] **Step 2: Apply formatting**

```bash
cd /Users/myers/x/stash/ui/v2.5 && pnpm run format
```

- [ ] **Step 3: Apply safe lint fixes**

```bash
cd /Users/myers/x/stash/ui/v2.5 && pnpm run lint:js:fix
```

- [ ] **Step 4: Verify formatting is now clean and types still pass**

```bash
cd /Users/myers/x/stash/ui/v2.5 && pnpm run format-check && pnpm run check
```

Expected: both clean. If `lint:js` still reports unfixable errors, note them — some biome rules need manual edits. Fix only errors in audio files; leave upstream files alone.

- [ ] **Step 5: Confirm this commit is formatting-only**

```bash
cd /Users/myers/x/stash && git diff --stat | tail -3
```

Sanity-check that the diff is whitespace/quote churn, not logic changes. If you see behavioural edits, biome's `lint:js:fix` did something unsafe — review those hunks individually.

- [ ] **Step 6: Commit**

```bash
cd /Users/myers/x/stash
git add ui/v2.5/src
git commit -m "style: apply biome formatting to audio UI files

Audio UI files predate the upstream prettier-to-biome migration (#6996).
Formatting-only; no behavioural changes."
```

- [ ] **Step 7: PHASE 1 GATE — full verification**

```bash
cd /Users/myers/x/stash && make build && make test && make lint
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check && pnpm run lint:js
```

All five must pass. **Do not start Phase 2 until they do.** Report any failure verbatim rather than working around it.

- [ ] **Step 8: Push the merged composite branch**

```bash
cd /Users/myers/x/stash
git push origin stashcoder42-main --force-with-lease
```

---

## Phase 2 — Scene→Audio Parity Port

**Method for every Phase 2 task:**

1. `git show <sha> -- <scene-file>` to read exactly what upstream changed.
2. Locate the corresponding construct in the audio twin (`docs/AUDIO_SCENE_PARITY.md` twin map).
3. Write a failing test against the audio behaviour where a test is meaningful.
4. Apply the equivalent change to the audio file.
5. Verify the test passes.
6. Flip the ledger row to ✅ (or ⏭️/➖ with a one-line reason) and commit code + ledger together.

**On skipping:** If inspection shows a change is genuinely video-only, mark ⏭️ or ➖ with the reason and move on. Recording *why* is mandatory — that reason is what prevents the next resync re-litigating the same commit.

### Task 7: Case-insensitive scan file lookup (#7098, `267c7ad34`)

**Files:**
- Modify: `pkg/sqlite/audio.go`
- Test: `pkg/sqlite/audio_test.go`

**Interfaces:**
- Consumes: shared helpers `pathHasWildcard(string) bool`, `pathLike(exp.IdentifierExpression, string) exp.Expression`, `pathEqNoCase(exp.IdentifierExpression, string) exp.Expression` from `pkg/sqlite/path_match.go` — these arrive free from upstream, already merged in Phase 1.
- Produces: `AudioStore.FindByPath` matching case-insensitively.

**Context:** Upstream rewrote `SceneStore.FindByPath` to branch on wildcards: literal paths use case-insensitive equality, wildcard paths use LIKE. Our `AudioStore.FindByPath` still does the old unconditional `.Like()` with manual `*`→`%` replacement, so audio files whose on-disk case differs from the DB record fail to match on rescan.

- [ ] **Step 1: Confirm the helpers exist from Phase 1**

```bash
cd /Users/myers/x/stash && grep -n 'func pathEqNoCase\|func pathLike\|func pathHasWildcard' pkg/sqlite/path_match.go
```

Expected: all three found. If missing, Phase 1 did not merge cleanly — stop and report.

- [ ] **Step 2: Read the reference change**

```bash
cd /Users/myers/x/stash && git show 267c7ad34 -- pkg/sqlite/scene.go pkg/sqlite/path_match.go
```

- [ ] **Step 3: Write the failing test**

Add to `pkg/sqlite/audio_test.go`, mirroring upstream's scene test added in the same commit (read it via `git show 267c7ad34 -- pkg/sqlite/scene_test.go` and adapt names):

```go
func TestAudioStoreFindByPathCaseInsensitive(t *testing.T) {
	withTxn(func(ctx context.Context) error {
		aqb := db.Audio

		const path = audioFilePath // an existing fixture audio path
		upper := strings.ToUpper(path)

		audios, err := aqb.FindByPath(ctx, upper)
		if err != nil {
			t.Errorf("Error finding audio: %s", err.Error())
		}

		if len(audios) == 0 {
			t.Errorf("expected audio to be found with case-differing path %q", upper)
		}

		return nil
	})
}
```

Adjust `audioFilePath` to whatever fixture constant `audio_test.go` already uses for a known audio path.

- [ ] **Step 4: Run it and confirm it fails**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run TestAudioStoreFindByPathCaseInsensitive -v
```

Expected: FAIL — "expected audio to be found". This proves the bug is real in audio before fixing it.

- [ ] **Step 5: Apply the fix**

In `pkg/sqlite/audio.go`, find `func (qb *AudioStore) FindByPath`. Delete the two manual wildcard-replacement lines:

```go
	// replace wildcards
	basename = strings.ReplaceAll(basename, "*", "%")
	dir = strings.ReplaceAll(dir, "*", "%")
```

Then split the query construction so the `Where` is conditional, exactly mirroring the scene version:

```go
	sq := dialect.From(audiosFilesJoinTable).InnerJoin(
		filesTable,
		goqu.On(filesTable.Col(idColumn).Eq(audiosFilesJoinTable.Col(fileIDColumn))),
	).InnerJoin(
		foldersTable,
		goqu.On(foldersTable.Col(idColumn).Eq(filesTable.Col("parent_folder_id"))),
	).Select(audiosFilesJoinTable.Col(audioIDColumn))

	if pathHasWildcard(basename) || pathHasWildcard(dir) {
		sq = sq.Where(
			pathLike(foldersTable.Col("path"), dir),
			pathLike(filesTable.Col("basename"), basename),
		)
	} else {
		sq = sq.Where(
			pathEqNoCase(foldersTable.Col("path"), dir),
			pathEqNoCase(filesTable.Col("basename"), basename),
		)
	}
```

Use whatever the actual audio table/column identifiers are in our file (`audiosFilesJoinTable`, `audioIDColumn` — confirm exact names by reading the surrounding function). If `strings` becomes unused, remove the import.

- [ ] **Step 6: Verify the test passes**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run TestAudioStoreFindByPathCaseInsensitive -v
```

Expected: PASS.

- [ ] **Step 7: Run the full sqlite suite for regressions**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/
```

Expected: PASS.

- [ ] **Step 8: Update the ledger and commit**

In `docs/AUDIO_SCENE_PARITY.md`, change row 12's status from ⬜ to ✅.

```bash
cd /Users/myers/x/stash
git add pkg/sqlite/audio.go pkg/sqlite/audio_test.go docs/AUDIO_SCENE_PARITY.md
git commit -m "fix(audio): case-insensitive scan file lookup

Ports upstream 267c7ad34 (#7098) to AudioStore.FindByPath. Literal paths
now match case-insensitively; wildcard paths keep LIKE semantics."
```

### Task 8: OR sub-filter join type (#6920, `fc0b2a5d9`)

**Files:**
- Modify: `pkg/sqlite/audio_filter.go`
- Test: `pkg/sqlite/audio_test.go`

**Interfaces:**
- Consumes: nothing from prior Phase 2 tasks.
- Produces: audio OR sub-filters using the correct join type.

**Context:** A real query-correctness bug: OR sub-filters used INNER JOIN where LEFT JOIN is required, silently dropping rows that should match.

- [ ] **Step 1: Read the reference change**

```bash
cd /Users/myers/x/stash && git show fc0b2a5d9 -- pkg/sqlite/scene_filter.go pkg/sqlite/scene_test.go
```

- [ ] **Step 2: Locate the audio equivalent**

```bash
cd /Users/myers/x/stash && grep -n 'Or\b\|orFilter\|SubFilter' pkg/sqlite/audio_filter.go | head -20
```

- [ ] **Step 3: Write the failing test**

Adapt upstream's scene test from `git show fc0b2a5d9 -- pkg/sqlite/scene_test.go` to audio: build a filter with an OR sub-filter whose right branch matches rows the left does not, and assert every expected row comes back. Name it `TestAudioQueryOrSubFilterJoinType`.

- [ ] **Step 4: Run it and confirm it fails**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run TestAudioQueryOrSubFilterJoinType -v
```

Expected: FAIL — rows missing from results.

- [ ] **Step 5: Apply the same join-type correction to `audio_filter.go`**

Mirror upstream's change exactly, substituting audio table/column identifiers.

- [ ] **Step 6: Verify**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run TestAudioQueryOrSubFilterJoinType -v && go test ./pkg/sqlite/
```

Expected: both PASS.

- [ ] **Step 7: Update ledger row 4 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add pkg/sqlite/audio_filter.go pkg/sqlite/audio_test.go docs/AUDIO_SCENE_PARITY.md
git commit -m "fix(audio): use correct join type for OR sub-filters

Ports upstream fc0b2a5d9 (#6920) to audio filters."
```

### Task 9: Size summary should represent all files (#7006, `db4b33f53`)

**Files:**
- Modify: `pkg/sqlite/audio.go`
- Test: `pkg/sqlite/audio_test.go`

**Interfaces:**
- Produces: audio size summary aggregating every associated file, not just the primary.

- [ ] **Step 1: Read the reference change**

```bash
cd /Users/myers/x/stash && git show db4b33f53 -- pkg/sqlite/scene.go pkg/sqlite/scene_test.go
```

- [ ] **Step 2: Write the failing test**

Adapt upstream's test: create an audio with two associated files, assert the reported size equals the sum of both. Name it `TestAudioSizeSummaryAllFiles`.

- [ ] **Step 3: Run it and confirm it fails**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run TestAudioSizeSummaryAllFiles -v
```

Expected: FAIL — size reflects only one file.

- [ ] **Step 4: Apply the aggregation fix to the audio size query**

- [ ] **Step 5: Verify**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run TestAudioSizeSummaryAllFiles -v && go test ./pkg/sqlite/
```

- [ ] **Step 6: Update ledger row 8 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add pkg/sqlite/audio.go pkg/sqlite/audio_test.go docs/AUDIO_SCENE_PARITY.md
git commit -m "fix(audio): size summary represents all files

Ports upstream db4b33f53 (#7006)."
```

### Task 10: Optimise table joins (#6648, `8e070717e`)

**Files:**
- Modify: `pkg/sqlite/audio_filter.go`, `pkg/sqlite/audio_marker_filter.go`

**Interfaces:**
- Produces: audio filters using the optimised join strategy (117 lines on the scene side).

- [ ] **Step 1: Read the reference change**

```bash
cd /Users/myers/x/stash && git show 8e070717e -- pkg/sqlite/scene_filter.go pkg/sqlite/scene_marker_filter.go
```

- [ ] **Step 2: Port the join restructuring to both audio filter files**

This is a performance refactor, not a behaviour change — existing tests are the safety net. Apply the same structural pattern, substituting audio identifiers.

- [ ] **Step 3: Verify no behavioural regression**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/
```

Expected: PASS. Any failure means the port changed semantics — fix before committing.

- [ ] **Step 4: Update ledger row 1 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add pkg/sqlite/audio_filter.go pkg/sqlite/audio_marker_filter.go docs/AUDIO_SCENE_PARITY.md
git commit -m "perf(audio): optimise table joins in audio filters

Ports upstream 8e070717e (#6648)."
```

### Task 11: Related object resolvers on file graphql types (#6938, `bb67152f9`)

**Files:**
- Modify: `pkg/models/repository_audio.go`, `pkg/sqlite/audio.go`
- Regenerate: `pkg/models/mocks/AudioReaderWriter.go`

**Interfaces:**
- Produces: audio file graphql types exposing related-object resolvers, matching scene.

- [ ] **Step 1: Read the reference change**

```bash
cd /Users/myers/x/stash && git show bb67152f9 -- pkg/models/repository_scene.go pkg/sqlite/scene.go
```

- [ ] **Step 2: Port the repository interface additions to `repository_audio.go`**

- [ ] **Step 3: Implement the corresponding store methods in `pkg/sqlite/audio.go`**

- [ ] **Step 4: Regenerate mocks — never hand-edit them**

```bash
cd /Users/myers/x/stash && make generate-backend
```

- [ ] **Step 5: Build and test**

```bash
cd /Users/myers/x/stash && make build && go test ./pkg/sqlite/ ./pkg/models/
```

- [ ] **Step 6: Update ledger row 5 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add pkg/models/repository_audio.go pkg/sqlite/audio.go pkg/models/mocks/AudioReaderWriter.go docs/AUDIO_SCENE_PARITY.md
git commit -m "feat(audio): add related object resolvers to audio file types

Ports upstream bb67152f9 (#6938)."
```

### Task 12: Recursive sort by performer_count and o_counter (#6933, `b044005fc`)

**Files:**
- Modify: `pkg/sqlite/audio.go`, `pkg/models/repository_audio.go`
- Regenerate: `pkg/models/mocks/AudioReaderWriter.go`

**Interfaces:**
- Produces: audio sortable by `performer_count` and `o_counter` with depth semantics.

- [ ] **Step 1: Read the reference change**

```bash
cd /Users/myers/x/stash && git show b044005fc -- pkg/sqlite/scene.go pkg/models/repository_scene.go
```

- [ ] **Step 2: Port the sort options to the audio store's sort handling**

- [ ] **Step 3: Regenerate and build**

```bash
cd /Users/myers/x/stash && make generate-backend && make build && go test ./pkg/sqlite/
```

- [ ] **Step 4: Update ledger row 10 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add pkg/sqlite/audio.go pkg/models/repository_audio.go pkg/models/mocks/AudioReaderWriter.go docs/AUDIO_SCENE_PARITY.md
git commit -m "feat(audio): recursive sort by performer_count and o_counter

Ports upstream b044005fc (#6933)."
```

### Task 13: json.Number custom field filters (#7040, `8a98b72c1`)

**Files:**
- Modify: `pkg/sqlite/audio_filter.go` (or wherever audio custom-field filtering lives)
- Test: `pkg/sqlite/audio_test.go`

**Interfaces:**
- Produces: audio custom-field filters handling `json.Number` correctly.

**Context:** Cross-check against our own `appSchemaVersion` 89 custom-fields migration (commit `14bcb2a6`). Our audio custom fields were added independently of upstream's — verify the fix applies and does not conflict with our migration's storage format.

- [ ] **Step 1: Read the reference change**

```bash
cd /Users/myers/x/stash && git show 8a98b72c1
```

- [ ] **Step 2: Confirm how audio stores custom fields**

```bash
cd /Users/myers/x/stash && grep -rn 'custom_field\|CustomField' pkg/sqlite/audio*.go | head -20
```

- [ ] **Step 3: Write a failing test**

Filter audio by a numeric custom field value and assert correct matching. Name it `TestAudioCustomFieldNumberFilter`.

- [ ] **Step 4: Run it and confirm it fails**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run TestAudioCustomFieldNumberFilter -v
```

If it unexpectedly PASSES, our implementation may already be correct — mark the ledger row ➖ with reason "audio custom fields already handle json.Number; verified by test" and still commit the test as a regression guard.

- [ ] **Step 5: Apply the fix if the test failed**

- [ ] **Step 6: Verify**

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/
```

- [ ] **Step 7: Update ledger row 9 and commit**

```bash
cd /Users/myers/x/stash
git add pkg/sqlite/ docs/AUDIO_SCENE_PARITY.md
git commit -m "fix(audio): handle json.Number in custom field filters

Ports upstream 8a98b72c1 (#7040)."
```

### Task 14: Signed URLs for authenticated streaming (#6529 `d04ecc4f8`, #7069 `7b5e84d69`)

**Files:**
- Modify: `internal/api/urlbuilders/audio.go`, `internal/api/resolver_model_audio.go`, `ui/v2.5/src/components/AudioPlayer/AudioPlayer.tsx`
- Test: `internal/api/urlbuilders/audio_test.go`

**Interfaces:**
- Consumes: HMAC signing helpers added by upstream in `d04ecc4f8` (merged in Phase 1).
- Produces: `AudioURLBuilder.GetCaptionPath() string` alongside existing `GetCaptionURL() string`; signed stream URLs for authenticated clients.

**Context:** These two commits are paired — `#6529` adds backend signing, `#7069` fixes the caption URL case in the player. Not video-only: the mechanism exists because AirPlay/Chromecast clients cannot pass auth cookies, and it applies to any authenticated stream. Our `urlbuilders/audio_test.go` already exists, so there is a test harness to extend.

- [ ] **Step 1: Read both reference changes**

```bash
cd /Users/myers/x/stash && git show d04ecc4f8 -- internal/api/urlbuilders/scene.go internal/api/resolver_model_scene.go
cd /Users/myers/x/stash && git show 7b5e84d69
```

- [ ] **Step 2: Write the failing test for the path/URL split**

Add to `internal/api/urlbuilders/audio_test.go`:

```go
func TestAudioURLBuilderGetCaptionPath(t *testing.T) {
	b := AudioURLBuilder{
		BaseURL: "http://localhost:9999",
		AudioID: "123",
	}

	const want = "/audio/123/caption"
	if got := b.GetCaptionPath(); got != want {
		t.Errorf("GetCaptionPath() = %q, want %q", got, want)
	}

	const wantURL = "http://localhost:9999/audio/123/caption"
	if got := b.GetCaptionURL(); got != wantURL {
		t.Errorf("GetCaptionURL() = %q, want %q", got, wantURL)
	}
}
```

Match the existing field names in our `AudioURLBuilder` — read the struct first and adjust `AudioID` if it differs.

- [ ] **Step 3: Run it and confirm it fails**

```bash
cd /Users/myers/x/stash && go test ./internal/api/urlbuilders/ -run TestAudioURLBuilderGetCaptionPath -v
```

Expected: FAIL — `GetCaptionPath` undefined.

- [ ] **Step 4: Add `GetCaptionPath` and refactor `GetCaptionURL` to use it**

```go
func (b AudioURLBuilder) GetCaptionPath() string {
	return "/audio/" + b.AudioID + "/caption"
}

func (b AudioURLBuilder) GetCaptionURL() string {
	return b.BaseURL + b.GetCaptionPath()
}
```

- [ ] **Step 5: Port the signing logic**

Apply the same HMAC-signing approach upstream added for scene streams to the audio stream URLs in `urlbuilders/audio.go` and `resolver_model_audio.go`. Reuse upstream's signing helper — do not reimplement it. When credentials are disabled, signing must be bypassed entirely; an API key takes precedence over signed params when both are present.

- [ ] **Step 6: Port the player-side caption fix**

Apply `7b5e84d69`'s change to `AudioPlayer.tsx`, mirroring `ScenePlayer.tsx`.

- [ ] **Step 7: Verify**

```bash
cd /Users/myers/x/stash && go test ./internal/api/... && make build
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check
```

- [ ] **Step 8: Update ledger rows 6 and 20 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add internal/api/urlbuilders/audio.go internal/api/urlbuilders/audio_test.go internal/api/resolver_model_audio.go ui/v2.5/src/components/AudioPlayer/AudioPlayer.tsx docs/AUDIO_SCENE_PARITY.md
git commit -m "feat(audio): signed URLs for authenticated audio streaming

Ports upstream d04ecc4f8 (#6529) and 7b5e84d69 (#7069) to audio."
```

### Task 15: Adjudicate the three video-specific candidates

**Files:**
- Modify: `docs/AUDIO_SCENE_PARITY.md`; audio files only if a port turns out to apply

**Interfaces:**
- Produces: ledger rows 2, 7, 11, 18 resolved to ✅, ⏭️, or ➖ — each with a recorded reason.

**Context:** These four were deliberately left undecided during scoping because deciding correctly requires reading our audio code, not upstream's. Resolve each with evidence.

- [ ] **Step 1: Row 11 — LEFT JOIN for NULL phash (#7121, `3d333a22a`)**

```bash
cd /Users/myers/x/stash && grep -rn 'phash' pkg/sqlite/audio_filter.go pkg/models/audio.go | head
```

If audio has no phash concept: mark ➖ "audio has no phash; filter does not exist". If it does: read `git show 3d333a22a -- pkg/sqlite/scene_filter.go` and port, with a test.

- [ ] **Step 2: Row 7 — scene_filter on findDuplicateScenes (#6884, `f3bfd8db7`)**

```bash
cd /Users/myers/x/stash && grep -rn 'findDuplicate\|FindDuplicate' internal/api/ pkg/sqlite/ graphql/schema/ | grep -i audio | head
```

If no audio duplicate-finder exists: mark ➖ "audio exposes no duplicate finder API". If one exists: port the filter parameter (333 lines on the scene side — the largest single item; treat it as its own commit).

- [ ] **Step 3: Row 18 — Safari auto-start on transcode (#7016, `1534587cb`)**

```bash
cd /Users/myers/x/stash && grep -rn 'transcode\|Transcode' ui/v2.5/src/components/AudioPlayer/*.tsx | head
```

If audio never transcodes in the player: mark ➖ with that reason. Otherwise port to `AudioPlayer.tsx`.

- [ ] **Step 4: Row 2 — api key in funscript url (#6760, `103181a6d`)**

Funscript is an interactive-toy format tied to video. Verify audio has no funscript support:

```bash
cd /Users/myers/x/stash && grep -rn 'funscript\|Funscript' internal/api/urlbuilders/audio.go pkg/models/audio.go | head
```

If absent, mark ➖ "audio has no funscript support". If our audio URL builder has a comparable interactive/asset URL that should carry the API key, port the pattern.

- [ ] **Step 5: Commit the adjudications**

```bash
cd /Users/myers/x/stash
git add docs/AUDIO_SCENE_PARITY.md
git commit -m "docs(audio): adjudicate video-specific upstream commits in parity ledger"
```

If any step turned into an actual port, commit the code separately with its own message before this one.

### Task 16: Wall panel fixes (#6803 `98074e3b5`, #7017 `5bbd821e5`)

**Files:**
- Modify: `ui/v2.5/src/components/Audios/AudioWallPanel.tsx`, `ui/v2.5/src/components/Audios/AudioMarkerWallPanel.tsx`

**Interfaces:**
- Produces: wall items pushing history once, and respecting the configured wall preview type.

**Context:** Two genuine user-visible bugs in the same two files — port together.

- [ ] **Step 1: Read both reference changes**

```bash
cd /Users/myers/x/stash && git show 98074e3b5 -- ui/v2.5/src/components/Scenes/SceneWallPanel.tsx ui/v2.5/src/components/Scenes/SceneMarkerWallPanel.tsx
cd /Users/myers/x/stash && git show 5bbd821e5 -- ui/v2.5/src/components/Scenes/SceneWallPanel.tsx ui/v2.5/src/components/Scenes/SceneMarkerWallPanel.tsx
```

- [ ] **Step 2: Port the double-history-push fix to both audio wall panels**

- [ ] **Step 3: Port the wall-preview-type fix to both audio wall panels**

- [ ] **Step 4: Verify**

```bash
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check && pnpm run lint:js
```

- [ ] **Step 5: Manual verification in the running app**

```bash
cd /Users/myers/x/stash && .local/stash-dev.sh start
```

Open `http://localhost:3000`, navigate to the audio wall, click an item, then press Back once. Expected: you return to the wall in a single Back press (previously took two). Confirm the wall honours the configured preview type.

- [ ] **Step 6: Update ledger rows 15 and 21 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add ui/v2.5/src/components/Audios/AudioWallPanel.tsx ui/v2.5/src/components/Audios/AudioMarkerWallPanel.tsx docs/AUDIO_SCENE_PARITY.md
git commit -m "fix(audio): wall panel history push and preview type

Ports upstream 98074e3b5 (#6803) and 5bbd821e5 (#7017) to audio walls."
```

### Task 17: Memoize AudioCard and fix IntersectionObserver churn (#6935, `f222bddf9`)

**Files:**
- Modify: `ui/v2.5/src/components/Audios/AudioCard.tsx`

**Interfaces:**
- Produces: `AudioCard` memoized, IntersectionObserver no longer re-registering per render.

**Context:** Largest single UI port (179 lines on the scene side). Pure performance work — visible as scroll jank on long audio lists.

- [ ] **Step 1: Read the reference change**

```bash
cd /Users/myers/x/stash && git show f222bddf9 -- ui/v2.5/src/components/Scenes/SceneCard.tsx
```

- [ ] **Step 2: Port the memoization and observer lifecycle changes to `AudioCard.tsx`**

Match upstream's structure: memo wrapper, stable callback identities, observer created once rather than per render. Keep our audio-specific props and fields intact.

- [ ] **Step 3: Verify**

```bash
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check && pnpm run lint:js
```

- [ ] **Step 4: Manual verification**

With the dev server running, open the audio list and scroll. Expected: no visual regression; cards render correctly with thumbnails and metadata.

- [ ] **Step 5: Update ledger row 19 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add ui/v2.5/src/components/Audios/AudioCard.tsx docs/AUDIO_SCENE_PARITY.md
git commit -m "perf(audio): memoize AudioCard and fix IntersectionObserver churn

Ports upstream f222bddf9 (#6935)."
```

### Task 18: Consolidate cover buttons (#6924, `634f567e3`)

**Files:**
- Modify: `internal/api/resolver_mutation_audio.go`, `ui/v2.5/src/components/Audios/AudioDetails/Audio.tsx`, `ui/v2.5/src/components/Audios/AudioDetails/AudioEditPanel.tsx`

**Interfaces:**
- Produces: consolidated cover-management UI and the matching mutation resolver for audio.

**Context:** Spans backend and frontend (ledger rows 13 and 23) — port as one unit since the UI depends on the resolver.

- [ ] **Step 1: Read the reference change**

```bash
cd /Users/myers/x/stash && git show 634f567e3
```

- [ ] **Step 2: Port the mutation resolver change to `resolver_mutation_audio.go`**

- [ ] **Step 3: Regenerate backend and build**

```bash
cd /Users/myers/x/stash && make generate-backend && make build
```

- [ ] **Step 4: Port the UI consolidation to `Audio.tsx` and `AudioEditPanel.tsx`**

- [ ] **Step 5: Verify**

```bash
cd /Users/myers/x/stash && go test ./internal/api/...
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check && pnpm run lint:js
```

- [ ] **Step 6: Manual verification**

Open an audio detail page, edit it, and exercise the cover buttons. Expected: cover set/clear works through the consolidated control.

- [ ] **Step 7: Update ledger rows 13 and 23 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add internal/api/resolver_mutation_audio.go ui/v2.5/src/components/Audios/AudioDetails/Audio.tsx ui/v2.5/src/components/Audios/AudioDetails/AudioEditPanel.tsx docs/AUDIO_SCENE_PARITY.md
git commit -m "feat(audio): consolidate audio cover buttons

Ports upstream 634f567e3 (#6924)."
```

### Task 19: Remaining small UI ports (#7117, #6977, #6777)

**Files:**
- Modify: `ui/v2.5/src/components/Audios/AudioList.tsx`, `ui/v2.5/src/components/Audios/AudioMarkerList.tsx`, `ui/v2.5/src/components/Audios/AudioDetails/OCounterButton.tsx`, `ui/v2.5/src/components/Audios/AudioDetails/AudioEditPanel.tsx`

**Interfaces:**
- Produces: plugin UI extension hooks, `data-action` attributes, and the ui-package-update fix present in audio components.

**Context:** Three small commits (2, 2, and 2 lines on the twin side) batched into one task — each is too small to warrant its own review gate.

- [ ] **Step 1: Read all three reference changes**

```bash
cd /Users/myers/x/stash && git show afdaa082b -- ui/v2.5/src/components/Scenes/SceneList.tsx ui/v2.5/src/components/Scenes/SceneMarkerList.tsx
cd /Users/myers/x/stash && git show c637b2931 -- ui/v2.5/src/components/Scenes/SceneDetails/OCounterButton.tsx
cd /Users/myers/x/stash && git show 083ba25d0 -- ui/v2.5/src/components/Scenes/SceneDetails/SceneEditPanel.tsx
```

- [ ] **Step 2: Port `#7117` plugin hooks to `AudioList.tsx` and `AudioMarkerList.tsx`**

Register the audio equivalents of upstream's new plugin extension hook points. Use audio-specific hook names consistent with our existing naming.

- [ ] **Step 3: Port `#6977` `data-action` attributes to audio `OCounterButton.tsx`**

- [ ] **Step 4: Port `#6777` to `AudioEditPanel.tsx`**

- [ ] **Step 5: Verify**

```bash
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check && pnpm run lint:js
```

- [ ] **Step 6: Update ledger rows 16, 17, 22 to ✅ and commit**

```bash
cd /Users/myers/x/stash
git add ui/v2.5/src/components/Audios/AudioList.tsx ui/v2.5/src/components/Audios/AudioMarkerList.tsx ui/v2.5/src/components/Audios/AudioDetails/OCounterButton.tsx ui/v2.5/src/components/Audios/AudioDetails/AudioEditPanel.tsx docs/AUDIO_SCENE_PARITY.md
git commit -m "feat(audio): plugin hooks, data-action attrs, ui package fixes

Ports upstream afdaa082b (#7117), c637b2931 (#6977), 083ba25d0 (#6777)."
```

### Task 20: Close out remaining rows and seal the ledger

**Files:**
- Modify: `docs/AUDIO_SCENE_PARITY.md`, `pkg/audio/scan.go` (lint only, if flagged)

**Interfaces:**
- Consumes: all prior Phase 2 tasks complete.
- Produces: every ledger row non-⬜; `LAST_SYNCED` bumped to `afdaa082b`.

- [ ] **Step 1: Row 3 — Go 1.25 lint fixes (#6869)**

```bash
cd /Users/myers/x/stash && make lint
```

If the linter flags anything in `pkg/audio/scan.go` or other audio files, fix it. If clean, mark row 3 ➖ "no audio lint violations under golangci-lint config".

- [ ] **Step 2: Row 14 — tag count filter depth (#6929)**

Upstream's change was test-only on the twin side. Verify audio tag filters behave consistently:

```bash
cd /Users/myers/x/stash && go test ./pkg/sqlite/ -run 'Tag' -v 2>&1 | tail -20
```

Mark ✅ if audio tag filtering already inherits the shared behaviour, ➖ otherwise — with the reason.

- [ ] **Step 3: Rows 24–26 — formatting sweeps**

Mark all three ⏭️ "covered by the biome sweep in Task 6".

- [ ] **Step 4: Confirm zero rows remain unresolved**

```bash
cd /Users/myers/x/stash && grep -c '⬜' docs/AUDIO_SCENE_PARITY.md
```

Expected: `0`. If non-zero, unresolved rows remain — resolve them before proceeding.

- [ ] **Step 5: Bump LAST_SYNCED**

In `docs/AUDIO_SCENE_PARITY.md`, change the LAST_SYNCED line to:

```markdown
**LAST_SYNCED:** `afdaa082b`
```

Remove the parenthetical "(pre-port baseline — bump to ...)" note.

- [ ] **Step 6: Verify the resync procedure now reports nothing outstanding**

```bash
cd /Users/myers/x/stash
TWINS=$(awk '/^## Twin file map/{t=1} t && /^## Resync/{exit} t' docs/AUDIO_SCENE_PARITY.md | awk -F'|' '/^\| `/ {gsub(/[` ]/,"",$2); if ($2 ~ /\//) print $2}')
git log --reverse --format='%h %s' afdaa082b..upstream/develop -- $TWINS
```

Expected: empty (assuming no new upstream fetch). This proves the ledger procedure is self-consistent.

- [ ] **Step 7: FINAL GATE — full verification**

```bash
cd /Users/myers/x/stash && make build && make test && make lint
cd /Users/myers/x/stash/ui/v2.5 && pnpm run check && pnpm run lint:js && pnpm run format-check
```

All six must pass.

- [ ] **Step 8: Commit and push**

```bash
cd /Users/myers/x/stash
git add docs/AUDIO_SCENE_PARITY.md
git commit -m "docs(audio): seal parity ledger at upstream afdaa082b"
git push origin stashcoder42-main --force-with-lease
```

---

## Self-Review Notes

**Spec coverage:** All 25 ledger commits are assigned. Backend rows 1,4,5,8,9,10,12 → Tasks 7–13; rows 2,7,11,18 → Task 15; row 6 → Task 14; rows 13,23 → Task 18; rows 3,14 → Task 20; UI rows 15,21 → Task 16; row 19 → Task 17; row 20 → Task 14; rows 16,17,22 → Task 19; formatting rows 24–26 → Tasks 6 and 20.

**Known gap, deliberately excluded:** upstream `058356004` (#6855, marker end times) is a *feature gap*, not a port — audio has no end-time handling at all, so there is no twin to update. It is recorded in the ledger Notes and is out of scope for this plan. Adding audio marker end-time support should be brainstormed and planned separately.

**Ordering rationale:** Phase 2 runs backend-before-frontend, and within backend, bug fixes (Tasks 7–9) before refactors (Task 10) before features (Tasks 11–12), so that the highest-value, most-testable changes land while the tree is most stable.
