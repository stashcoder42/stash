# Audio ↔ Scene Parity Checklist

The `audio` media type is a structural copy of `scene`. Upstream changes to
scene code do **not** conflict with our audio code — git merges cleanly and the
audio twin silently keeps the old behaviour. Every upstream scene fix is a bug
we still have until it is ported by hand.

This document is the resync procedure and the running ledger of what has been
ported.

## Twin file map

| Scene (upstream) | Audio (ours) |
| --- | --- |
| `internal/api/resolver_model_scene.go` | `internal/api/resolver_model_audio.go` |
| `internal/api/resolver_mutation_scene.go` | `internal/api/resolver_mutation_audio.go` |
| `internal/api/resolver_query_find_scene.go` | `internal/api/resolver_query_find_audio.go` |
| `internal/api/urlbuilders/scene.go` | `internal/api/urlbuilders/audio.go` |
| `pkg/models/mocks/SceneReaderWriter.go` | `pkg/models/mocks/AudioReaderWriter.go` |
| `pkg/models/repository_scene.go` | `pkg/models/repository_audio.go` |
| `pkg/scene/scan.go` | `pkg/audio/scan.go` |
| `pkg/sqlite/scene.go` | `pkg/sqlite/audio.go` |
| `pkg/sqlite/scene_filter.go` | `pkg/sqlite/audio_filter.go` |
| `pkg/sqlite/scene_marker_filter.go` | `pkg/sqlite/audio_marker_filter.go` |
| `pkg/sqlite/scene_test.go` | `pkg/sqlite/audio_test.go` |
| `ui/v2.5/src/components/ScenePlayer/PlaylistButtons.ts` | `ui/v2.5/src/components/AudioPlayer/PlaylistButtons.ts` |
| `ui/v2.5/src/components/ScenePlayer/ScenePlayer.tsx` | `ui/v2.5/src/components/AudioPlayer/AudioPlayer.tsx` |
| `ui/v2.5/src/components/ScenePlayer/ScenePlayerScrubber.tsx` | `ui/v2.5/src/components/AudioPlayer/AudioPlayerScrubber.tsx` |
| `ui/v2.5/src/components/ScenePlayer/big-buttons.ts` | `ui/v2.5/src/components/AudioPlayer/big-buttons.ts` |
| `ui/v2.5/src/components/ScenePlayer/live.ts` | `ui/v2.5/src/components/AudioPlayer/live.ts` |
| `ui/v2.5/src/components/Scenes/SceneCard.tsx` | `ui/v2.5/src/components/Audios/AudioCard.tsx` |
| `ui/v2.5/src/components/Scenes/SceneList.tsx` | `ui/v2.5/src/components/Audios/AudioList.tsx` |
| `ui/v2.5/src/components/Scenes/SceneMarkerList.tsx` | `ui/v2.5/src/components/Audios/AudioMarkerList.tsx` |
| `ui/v2.5/src/components/Scenes/SceneMarkerWallPanel.tsx` | `ui/v2.5/src/components/Audios/AudioMarkerWallPanel.tsx` ⚠️ *not a structural twin — see below* |
| `ui/v2.5/src/components/Scenes/SceneWallPanel.tsx` | `ui/v2.5/src/components/Audios/AudioWallPanel.tsx` |
| `ui/v2.5/src/components/Scenes/SceneDetails/Scene.tsx` | `ui/v2.5/src/components/Audios/AudioDetails/Audio.tsx` |
| `ui/v2.5/src/components/Scenes/SceneDetails/SceneEditPanel.tsx` | `ui/v2.5/src/components/Audios/AudioDetails/AudioEditPanel.tsx` |
| `ui/v2.5/src/components/Scenes/SceneDetails/SceneMarkersPanel.tsx` | `ui/v2.5/src/components/Audios/AudioDetails/AudioMarkersPanel.tsx` |
| `ui/v2.5/src/components/Scenes/SceneDetails/SceneScrapeDialog.tsx` | `ui/v2.5/src/components/Audios/AudioDetails/AudioScrapeDialog.tsx` |
| `ui/v2.5/src/components/Scenes/SceneDetails/ExternalPlayerButton.tsx` | `ui/v2.5/src/components/Audios/AudioDetails/ExternalPlayerButton.tsx` |
| `ui/v2.5/src/components/Scenes/SceneDetails/OCounterButton.tsx` | `ui/v2.5/src/components/Audios/AudioDetails/OCounterButton.tsx` |
| `ui/v2.5/src/components/Scenes/SceneDetails/QueueViewer.tsx` | `ui/v2.5/src/components/Audios/AudioDetails/QueueViewer.tsx` |
| `ui/v2.5/src/models/sceneQueue.ts` | `ui/v2.5/src/models/audioQueue.ts` |

## Resync procedure

Run this after every upstream merge. `LAST_SYNCED` below is the last upstream
commit whose scene changes have been adjudicated.

```bash
# 1. list scene-side twin files (only rows between the twin-map header and the
#    next heading, so ledger tables are not scraped)
TWINS=$(awk '/^## Twin file map/{t=1} t && /^## Resync/{exit} t' \
          docs/AUDIO_SCENE_PARITY.md \
        | awk -F'|' '/^\| `/ {gsub(/[` ]/,"",$2); if ($2 ~ /\//) print $2}')

# 2. upstream commits touching them since last sync, oldest first
git log --reverse --format='%h %s' <LAST_SYNCED>..upstream/develop -- $TWINS
```

For each commit: `git show <sha>` and decide **Port**, **Skip**, or **N/A**.
Record the decision in the ledger. Skip/N/A still get a one-line reason —
that reason is what stops the next resync from re-litigating the same commit.

Pure-formatting sweeps (biome/prettier reformats) are **Skip** — handle them by
running the formatter over audio files, not by hand-porting hunks.

The `<LAST_SYNCED>..upstream/develop` range only catches commits *newer* than
the last sync. It does not catch scene fixes that predate it and were never
ported in the first place — audio was branched from an older scene, so some
gaps are arbitrarily old. Upstream `74a8f2e5d` (#6649, disable wall links while
selecting) was found this way: it sat outside every sync range and only
surfaced because a reviewer diffed the audio and scene files side by side.
When porting a fix, diff the whole twin function against its scene counterpart
rather than applying the upstream hunk in isolation.

After porting, regenerate rather than hand-editing generated files:

```bash
make generate-backend   # mocks, loaders, resolvers
make generate-ui        # graphql types
```

**Running the audio tests.** `pkg/sqlite/audio_test.go` and the other audio
store tests are `//go:build integration` gated. Plain `go test ./pkg/sqlite/`
runs only the non-integration tests and silently exercises none of them — it
will report success without having run a single audio test. Use:

```bash
make it                 # supplies the full GO_BUILD_TAGS set
```

Do not hand-roll `go test -tags integration`: it omits `sqlite_stat4`, which
changes the SQLite query planner and makes unrelated tests
(e.g. `TestStudioQueryFast`) fail spuriously.

## Ledger

**LAST_SYNCED:** `2da807431` (pre-port baseline — bump to `afdaa082b` when the
table below is fully worked through)

Status: ⬜ todo · ✅ ported · ⏭️ skipped · ➖ n/a

### Backend

| # | Upstream | Change | Audio action | Status |
| --- | --- | --- | --- | --- |
| 1 | `8e070717e` | Optimise table joins (#6648) | Ported to `audio_filter.go` (commit `9dfb466db`). Six criteria now use `joinTypeInner` unless the modifier is `IsNull`. `audio_marker_filter.go` already matched scene and needed no change | ✅ |
| 2 | `103181a6d` | Include api key in funscript url (#6760) | **N/A — audio has no funscript/interactive support.** `AudioPathsType` (`types/audio.graphql`) is `stream`/`preview`/`cover`/`caption` only, and `urlbuilders/audio.go` has no `GetFunscriptURL`. The commit is confined to funscript URL building; its api-key mechanism is not a general pattern audio is missing — audio's `GetStreamURL` already has the identical `(apiKey string) *url.URL` shape, and `resolver_model_audio.go` already branches on `config.HasCredentials()` (row 6) | ➖ |
| 3 | `2b29207f1` | Upgrade go 1.25 / golangci-lint (#6869) | Lint fixes in `pkg/audio/scan.go` if linter flags them | ⬜ |
| 4 | `fc0b2a5d9` | Fix OR sub-filter join type (#6920) | Ported to `audio_filter.go` AND `audio_marker_filter.go` (commit `ad5fa21f9`). Fix is a statement reorder — `handleCriterion` must run before `handleSubFilter` — not a join-type edit | ✅ |
| 5 | `bb67152f9` | Related object resolvers on file graphql types (#6938) | Ported (commit `8abbc2f66`). Adds `AudioFile.audios`, `AudioStore.GetManyIDsByFileIDs`, and an `AudioIDsByFileID` loader — distinct from the pre-existing `AudioFileIDsLoader`, which maps the opposite direction | ✅ |
| 6 | `d04ecc4f8` | Signed urls for airplay (#6529) | Ported (commits `6e6e1e286`, `320159b74`). Signs both `audioResolver.Paths` and `audioResolver.AudioStreams`; added `GetCaptionPath`. Marker endpoints stay unsigned, matching scene. Audio has no `Query.audioStreams`, so scene's third signing site has no counterpart | ✅ |
| 7 | `f3bfd8db7` | `scene_filter` param on findDuplicateScenes (#6884) | **N/A — no audio duplicate finder exists.** `findDuplicateScenes` (`schema.graphql:47`) is the only `findDuplicate*` query in the schema; there is no `findDuplicateAudios` resolver or store method. Duplicate finding is phash-driven, and audio has no phash (row 11), so there is no substrate for one either | ➖ |
| 8 | `db4b33f53` | Size summary should represent all files (#7006) | Ported to `pkg/sqlite/audio.go` (commit `833fc2450`). Also fixed a worse audio-only defect found alongside it: totals ignored the query filter entirely. Audio now has `queryGroupedFields` mirroring scene's | ✅ |
| 9 | `8a98b72c1` | json.Number custom field filters (#7040) | **No code to port** — the fix lives in shared `pkg/sqlite/custom_fields.go:105` and arrived with the merge; `AudioStore` embeds the same `customFieldsStore`. Audio had no custom-field tests at all, so regression tests were added instead (commit `88646ad83`). Verified decisive: removing the `json.Number` branch makes them fail | ✅ |
| 10 | `b044005fc` | Recursive sort performer_count / o_counter (#6933) | **N/A — no audio surface.** The commit is studio-scoped: it adds a `depth` parameter to `OCountByStudioID` and recursive studio sorts. Audio has no studio relationship at all (no `StudioID` on the model, no studio methods on the store), and `AudioCounter` declares only `OCountByPerformerID` — no `OCountByStudioID`/`OCountByGroupID`. The new sort entries went to studios, and audio already supports `performer_count` and `o_counter` sorting (`pkg/sqlite/audio.go:875-977`) | ➖ |
| 11 | `3d333a22a` | LEFT JOIN for NULL phash (#7121) | **N/A — audio has no phash at all.** `AudioFilterType` (`types/filters.graphql:803-866`) has `checksum` but no `phash`/`phash_distance`/`duplicated`; `audio_filter.go` has no phash criterion handler; no phash reference exists in any audio Go file. The `IsMissing: "phash"` path is absent too — audio's `missingCriterionHandler` (`audio_filter.go:234-260`) accepts only performers/tags/cover/url plus the `validateIsMissing` set. Perceptual hashing is frame-based; nothing computes one for audio | ➖ |
| 12 | `267c7ad34` | Case-insensitive scan file lookup (#7098) | Ported to `AudioStore.FindByPath` (commit `864bc3c99`). Signature kept as single-return; wildcard branch currently has no production caller | ✅ |
| 13 | `634f567e3` | Consolidate scene cover buttons (#6924) | **N/A — backend half is screenshot generation.** The change makes `sceneGenerateScreenshot` return a job ID; there is no `audioGenerateScreenshot` mutation and audio has no video frames to capture. UI half ported under row 23 | ➖ |
| 14 | `9cb2ffd56` | Tag count filter depth (#6929) | Test-only on twin side; verify audio tag filters behave the same | ⬜ |

### Frontend

| # | Upstream | Change | Audio action | Status |
| --- | --- | --- | --- | --- |
| 15 | `98074e3b5` | Wall item double history push (#6803) | Ported to `AudioWallPanel.tsx` (commit `3f485b9d0`). Real bug — `handleClick` was on both the item div and the nested img. **Not applicable to `AudioMarkerWallPanel.tsx`**, which is a plain card grid with no gallery, no `handleClick` and no `history.push` | ✅ |
| 16 | `083ba25d0` | ui package updates sprint 1 (#6777) | **Already satisfied — nothing to do.** The only twin-file hunk swaps `cloneDeep` from `@apollo/client/utilities` to `lodash-es/cloneDeep`. `AudioEditPanel.tsx` has no `cloneDeep` at all; `AudioList.tsx:2` and `AudioMarkerList.tsx:1` already import the `lodash-es` form. Remaining `@apollo/client/utilities` imports in the tree are `getMainDefinition`, untouched by this commit | ➖ |
| 17 | `c637b2931` | `data-action` attributes (#6977) | Ported to `Audios/AudioDetails/OCounterButton.tsx` (commit `44bb9ce1c`). One line — `data-action="o-counter"` on the button group. The commit's other targets (`MainNavbar`, `SceneDuplicateChecker`, shared `CountButton`) are shared or scene-only and arrived with the merge | ✅ |
| 18 | `1534587cb` | Safari auto-start on transcode (#7016) | **N/A while audio has no source-selector** (conditional — see note below). The fix guards two unconditional `player.play()` calls inside `SourceSelectorPlugin`; `AudioPlayer/` has no `source-selector.ts` and nothing calls `setShouldAutoplay`. Audited every `.play()` in the audio player: hotkey toggle, external-seek handler, and the one-shot autostart effect — none is a failover/preload path. `AudioPlayer.tsx` sets a single source with no `player.on("error")` handler and no next-source retry | ➖ |
| 19 | `f222bddf9` | Memoize cards / IntersectionObserver churn (#6935) | Ported to `AudioCard.tsx` (commit `a3c5108eb`). `React.memo` on `AudioPreview` and the five `PatchComponent` sub-components, matching scene's `React.memo(PatchComponent(...))` ordering. **The IntersectionObserver half is N/A** — scene's observer lives in `ScenePreview` to drive video preview playback; `AudioCard` has no observer and no video preview. Scene's `useCallback` is on `onScrubberClick`, which has no `AudioCard` counterpart (audio's scrubber is in `AudioPlayer`) | ✅ |
| 20 | `7b5e84d69` | Signed caption URLs with auth (#7069) | Ported to `AudioPlayer.tsx` (commit `6e6e1e286`). Builds the caption URL via the `URL` API so appending `lang`/`type` cannot corrupt a signed query string | ✅ |
| 21 | `5bbd821e5` | Respect wall preview type (#7017) | **Mostly N/A** (commit `3f485b9d0`). Audio has no video or animated preview, so there is no preview type to respect — `wallPlayback` has nothing to select. Adopted the shared `getFirstValidPreviewSource` helper anyway to match scene's structure and kill a dead ternary (both branches returned `paths.cover`). `paths.preview` is deliberately not a fallback: its `/thumbnail` route reads the same blob via the same `GetCover` call as `/cover`. Not applicable to `AudioMarkerWallPanel.tsx` — no preview-source selection exists there | ✅ |
| 22 | `afdaa082b` | Plugin UI extension hooks (#7117) | Ported to `AudioList.tsx`, `AudioMarkerList.tsx` (commit `a3c5108eb`). One line each — `view={view}` on `<FilterTags>`, so the new filter-tag-extras hook can target audio. The hook implementations live in shared `List/FilterTags.tsx` and `List/SavedFilterList.tsx` and arrived with the merge | ✅ |
| 23 | `634f567e3` | Consolidate cover buttons — UI half (#6924) | Ported to `AudioEditPanel.tsx` (commit `44bb9ce1c`). Only the cover **reset** applies: audio passed no `onReset`, so a cover could never be cleared from the edit panel even though the backend supported removal and `coverImagePreview` already had unreachable handling for a null form value. The two `extraActions` (generate thumb from playback position / default) are N/A — see row 13. `Audio.tsx` needed no change: its `useMonitorJob` screenshot tracking exists only to poll the screenshot job audio has no counterpart for | ✅ |

### Formatting sweeps — do not hand-port

| # | Upstream | Change | Action | Status |
| --- | --- | --- | --- | --- |
| 24 | `b8c17f780` | Replace prettier/eslint with biome (#6996) | Run biome over audio UI files as one isolated commit | ⬜ |
| 25 | `5ed738558` | Biome fixes (#7004) | Covered by the sweep above | ⬜ |
| 26 | `4bd16c29b` | Biome format (#7005) | Covered by the sweep above | ⬜ |

## Known parity gaps (not upstream ports — audio was built without these)

These are places where audio diverges from scene in our own code, independent of
any upstream commit. They surfaced during the 2026-07-29 merge-forward. Each
needs its own fix; none is a port from upstream.

| Gap | Evidence | Impact |
| --- | --- | --- |
| `ExportObjectsInput` has no `audios` field | `graphql/schema/types/metadata.graphql:313-323`; UI passes `{audios: …}` through a cast at `ui/v2.5/src/components/Audios/AudioList.tsx:582` | Clicking Export on the Audios list sends an unknown input field; gqlgen rejects it and the UI shows a GraphQL validation error toast. Shipped broken in `feat/audios`; the merge only added a cast to keep it compiling. |
| `audioResetActivity` lacks partial-reset params | `graphql/schema/schema.graphql:407` is `audioResetActivity(id: ID!)`; scene's at line 340 takes `reset_resume` / `reset_duration`. Resolver hardcodes `ResetActivity(ctx, audioID, true, true)` | Audio cannot reset resume-time and play-duration independently the way scene can. |
| Audio's `OCounterButton` ignores `sfwContentMode` | `Audios/AudioDetails/OCounterButton.tsx` has no `useConfigurationContext`; it hardcodes the `SweatDrops` icon and the `o_count` message. Scene's reads `sfwContentMode` and swaps to `faThumbsUp` / `o_count_sfw` | SFW mode is honoured in 11 components but leaks on audio detail pages — the non-SFW icon and wording still show. The `o_count_sfw` locale key already exists |
| `AudioEditPanel` has no custom-fields UI | `AudioUpdateInput.custom_fields` exists in the generated types and `resolver_mutation_audio.go` handles `SetCustomFields`, but the edit panel renders no `<CustomFieldsInput>` — scene's does | Audio custom fields can be set through the API but not through the UI. Plan exists at `docs/superpowers/plans/2026-04-09-audio-custom-fields-last-o-at.md` |
| `AudioEditPanel` has no Save-and-New | Audio's `onSave(input)` (`AudioEditPanel.tsx:143`); scene's is `onSave(input, andNew?)` with an `onSaveAndNewClick` handler and a matching button | Minor UX divergence — cannot chain audio creation the way scene allows |
| `AudioWallPanel` uses a flat errored-src list | `AudioWallPanel.tsx` tracks `erroredImgs: string[]`; scene uses a per-id `FailedSrcMap` (`SceneWallPanel.tsx:217,257-267`) | Harmless today because cover URLs are id-scoped, but one item's errored URL would suppress the same URL for another. Structural divergence from upstream |
| `AudioWallItem.getDimensions` ignores its argument | `AudioWallPanel.tsx:141` returns a hardcoded 300×300 for every audio, so `zoomFactor` is always 1 | Wall zoom levels have no effect on audio item sizing |
| ~~Checksum filter always LEFT-joins~~ | ~~`pkg/sqlite/audio_filter.go:60`~~ | **Resolved** in commit `9dfb466db` (ledger row 1). Audio now matches scene: `joinTypeInner` unless the modifier is `IsNull`. |

## Notes

- Our audio has **no marker end-time support**. Upstream #6855 (honor marker end
  times, configurable ceiling) is a *feature gap*, not a port — it does not
  appear in the twin table because there is no corresponding audio code yet.
- Tab panels: when a detail page gates a panel on the active tab, use the
  resolved `activeTabKey` from `useTabKey()`, never the raw `tabKey` route
  param. Audio tabs have twice been added using `tabKey`, which breaks the
  panel whenever audio is the *resolved default* tab (raw param is `undefined`).
- `AudioMarkerWallPanel.tsx` is listed in the twin map for routing purposes but
  is **not** a structural copy of `SceneMarkerWallPanel.tsx`. Scene's is a
  `react-photo-gallery` wall; audio's is a plain card grid that delegates to
  `AudioMarkerCard`. It has no `handleClick`, no `history.push`, and no
  preview-source selection. Upstream fixes to scene's marker wall usually do
  **not** apply — check the actual file before assuming a row covers it. Two
  ledger rows (15, 21) initially overclaimed on this and were corrected.
- Ledger row 18 (#7016, Safari auto-start) is N/A for the *current* tree, not
  permanently. The backend already returns multiple endpoints from
  `audioResolver.AudioStreams`; the frontend just uses index 0 and never fails
  over. If a source-selector is ever added to `AudioPlayer.tsx`, that commit
  becomes portable and its two guarded `player.play()` call sites must be
  carried across at the same time.
- Generated files (`*_gen.go`, mocks) must be regenerated, never hand-merged.
  Use `make generate-backend`; `go generate ./...` has a dataloaden bug that
  emits duplicate imports.
