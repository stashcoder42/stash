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
| `ui/v2.5/src/components/Scenes/SceneMarkerWallPanel.tsx` | `ui/v2.5/src/components/Audios/AudioMarkerWallPanel.tsx` |
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

After porting, regenerate rather than hand-editing generated files:

```bash
make generate-backend   # mocks, loaders, resolvers
make generate-ui        # graphql types
```

## Ledger

**LAST_SYNCED:** `2da807431` (pre-port baseline — bump to `afdaa082b` when the
table below is fully worked through)

Status: ⬜ todo · ✅ ported · ⏭️ skipped · ➖ n/a

### Backend

| # | Upstream | Change | Audio action | Status |
| --- | --- | --- | --- | --- |
| 1 | `8e070717e` | Optimise table joins (#6648) | Port to `audio_filter.go`, `audio_marker_filter.go` | ⬜ |
| 2 | `103181a6d` | Include api key in funscript url (#6760) | Review — funscript is video-domain; the URL-builder api-key pattern may still apply | ⬜ |
| 3 | `2b29207f1` | Upgrade go 1.25 / golangci-lint (#6869) | Lint fixes in `pkg/audio/scan.go` if linter flags them | ⬜ |
| 4 | `fc0b2a5d9` | Fix OR sub-filter join type (#6920) | Port to `audio_filter.go` — real bug | ⬜ |
| 5 | `bb67152f9` | Related object resolvers on file graphql types (#6938) | Port to `repository_audio.go`, `pkg/sqlite/audio.go`; regen mocks | ⬜ |
| 6 | `d04ecc4f8` | Signed urls for airplay (#6529) | Port — refactors `GetCaptionPath`/`GetCaptionURL` + HMAC signing; applies to any authenticated stream, not just video | ⬜ |
| 7 | `f3bfd8db7` | `scene_filter` param on findDuplicateScenes (#6884) | Largest backend change (333 lines). Port if audio exposes a duplicate finder; else mark n/a with reason | ⬜ |
| 8 | `db4b33f53` | Size summary should represent all files (#7006) | Port to `pkg/sqlite/audio.go` — real bug | ⬜ |
| 9 | `8a98b72c1` | json.Number custom field filters (#7040) | Port. Cross-check against our `appSchemaVersion` 89 custom-fields migration | ⬜ |
| 10 | `b044005fc` | Recursive sort performer_count / o_counter (#6933) | Port sort options to `pkg/sqlite/audio.go`, `repository_audio.go` | ⬜ |
| 11 | `3d333a22a` | LEFT JOIN for NULL phash (#7121) | Review — phash is video-specific; port only if audio filters on phash | ⬜ |
| 12 | `267c7ad34` | Case-insensitive scan file lookup (#7098) | Port to `AudioStore.FindByPath`. Helpers (`pathLike`, `pathEqNoCase`, `path_match.go`) come free from shared code | ⬜ |
| 13 | `634f567e3` | Consolidate scene cover buttons (#6924) | Backend half in `resolver_mutation_scene.go`; port with UI item 24 | ⬜ |
| 14 | `9cb2ffd56` | Tag count filter depth (#6929) | Test-only on twin side; verify audio tag filters behave the same | ⬜ |

### Frontend

| # | Upstream | Change | Audio action | Status |
| --- | --- | --- | --- | --- |
| 15 | `98074e3b5` | Wall item double history push (#6803) | Port to `AudioWallPanel.tsx`, `AudioMarkerWallPanel.tsx` — real bug | ⬜ |
| 16 | `083ba25d0` | ui package updates sprint 1 (#6777) | Port to `AudioEditPanel.tsx` (2 lines) | ⬜ |
| 17 | `c637b2931` | `data-action` attributes (#6977) | Port to `OCounterButton.tsx` | ⬜ |
| 18 | `1534587cb` | Safari auto-start on transcode (#7016) | Review — transcode path is video-specific | ⬜ |
| 19 | `f222bddf9` | Memoize cards / IntersectionObserver churn (#6935) | Port to `AudioCard.tsx` — perf fix, 179 lines | ⬜ |
| 20 | `7b5e84d69` | Signed caption URLs with auth (#7069) | Port to `AudioPlayer.tsx` alongside item 6 | ⬜ |
| 21 | `5bbd821e5` | Respect wall preview type (#7017) | Port to `AudioWallPanel.tsx`, `AudioMarkerWallPanel.tsx` | ⬜ |
| 22 | `afdaa082b` | Plugin UI extension hooks (#7117) | Port to `AudioList.tsx`, `AudioMarkerList.tsx` | ⬜ |
| 23 | `634f567e3` | Consolidate cover buttons — UI half (#6924) | Port to `Audio.tsx`, `AudioEditPanel.tsx` | ⬜ |

### Formatting sweeps — do not hand-port

| # | Upstream | Change | Action | Status |
| --- | --- | --- | --- | --- |
| 24 | `b8c17f780` | Replace prettier/eslint with biome (#6996) | Run biome over audio UI files as one isolated commit | ⬜ |
| 25 | `5ed738558` | Biome fixes (#7004) | Covered by the sweep above | ⬜ |
| 26 | `4bd16c29b` | Biome format (#7005) | Covered by the sweep above | ⬜ |

## Notes

- Our audio has **no marker end-time support**. Upstream #6855 (honor marker end
  times, configurable ceiling) is a *feature gap*, not a port — it does not
  appear in the twin table because there is no corresponding audio code yet.
- Generated files (`*_gen.go`, mocks) must be regenerated, never hand-merged.
  Use `make generate-backend`; `go generate ./...` has a dataloaden bug that
  emits duplicate imports.
