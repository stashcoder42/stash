#!/bin/sh
# simulate-sabnzbd.sh — Run inside the stash-watcher-test container
# Simulates SABnzbd-style unpacking behavior in /data/library
#
# Based on actual SABnzbd source code analysis (sabnzbd/postproc.py):
#   1. Create _UNPACK_ directory
#   2. PAR2 repair (create + delete .par2 files)
#   3. Extract archives (media files appear in _UNPACK_ dir)
#   4. Delete .rar files
#   5. Rename _UNPACK_ dir to final name
#
# Usage: docker exec stash-watcher-test /usr/local/bin/simulate-sabnzbd.sh

set -e

LIBRARY="/data/library"

ts() { date '+%H:%M:%S'; }

query() {
    wget -qO- --post-data="$1" \
        --header='Content-Type: application/json' \
        http://localhost:9999/graphql 2>/dev/null || echo "(query failed)"
}

watcher_status() {
    echo "$(ts) Watcher status:"
    query '{"query":"{ watcherStatus { running watchCount pendingEvents processedEvents triggeredScans triggeredCleans lastError } }"}'
    echo ""
}

echo "=== Watcher Test Suite ==="
echo "$(ts) Library: $LIBRARY"
watcher_status

# ─── TEST 1: Basic media file drop ─────────────────────────────
echo ""
echo "=== TEST 1: Basic media file drop ==="
echo "$(ts) Creating test.mp4 (100KB)..."
dd if=/dev/urandom of="$LIBRARY/test.mp4" bs=1024 count=100 2>/dev/null
echo "$(ts) File created. Expect: CREATE -> queued -> scan after 5s debounce."
sleep 8
watcher_status

# ─── TEST 2: Non-media noise (should be ignored) ───────────────
echo ""
echo "=== TEST 2: Non-media noise ==="
echo "$(ts) Creating non-media files..."
echo "nfo content" > "$LIBRARY/movie.nfo"
echo "par2 content" > "$LIBRARY/movie.par2"
echo "sfv content" > "$LIBRARY/movie.sfv"
echo "txt content" > "$LIBRARY/readme.txt"
dd if=/dev/urandom of="$LIBRARY/movie.r00" bs=1024 count=10 2>/dev/null
dd if=/dev/urandom of="$LIBRARY/movie.r01" bs=1024 count=10 2>/dev/null
echo "$(ts) Non-media files created. Expect: events logged but NOT queued."
sleep 8
watcher_status

# ─── TEST 3: SABnzbd-style unpacking ──────────────────────────
echo ""
echo "=== TEST 3: SABnzbd-style post-processing ==="

# Phase 1: SABnzbd creates _UNPACK_ directory (prepare_extraction_path)
echo "$(ts) Phase 1: mkdir _UNPACK_Movie.2024/"
mkdir -p "$LIBRARY/_UNPACK_Movie.2024"
echo "$(ts) Expect: inotify watch added immediately, quiesce timer started."
sleep 1

# Phase 2: PAR2 repair — create then delete .par2 files
echo "$(ts) Phase 2: PAR2 repair simulation..."
echo "par2" > "$LIBRARY/_UNPACK_Movie.2024/movie.par2"
echo "par2.1" > "$LIBRARY/_UNPACK_Movie.2024/movie.vol00+01.par2"
echo "par2.2" > "$LIBRARY/_UNPACK_Movie.2024/movie.vol01+02.par2"
sleep 0.5
rm "$LIBRARY/_UNPACK_Movie.2024/movie.par2"
rm "$LIBRARY/_UNPACK_Movie.2024/movie.vol00+01.par2"
rm "$LIBRARY/_UNPACK_Movie.2024/movie.vol01+02.par2"
echo "$(ts) PAR2 files created and deleted. Expect: activity resets quiesce timer."
sleep 1

# Phase 3: unrar extraction — media files appear
echo "$(ts) Phase 3: Extracting media files (simulating unrar)..."
dd if=/dev/urandom of="$LIBRARY/_UNPACK_Movie.2024/Movie.2024.mp4" bs=1024 count=500 2>/dev/null
dd if=/dev/urandom of="$LIBRARY/_UNPACK_Movie.2024/Movie.2024.jpg" bs=1024 count=10 2>/dev/null
echo "subtitle content" > "$LIBRARY/_UNPACK_Movie.2024/Movie.2024.srt"
echo "$(ts) Media files extracted. Expect: activity resets quiesce timer, NO scan yet."
sleep 1

# Phase 4: delete .rar parts (SABnzbd deletes after successful extraction)
echo "$(ts) Phase 4: Deleting .rar parts..."
dd if=/dev/urandom of="$LIBRARY/_UNPACK_Movie.2024/movie.part01.rar" bs=1024 count=50 2>/dev/null
dd if=/dev/urandom of="$LIBRARY/_UNPACK_Movie.2024/movie.part02.rar" bs=1024 count=50 2>/dev/null
sleep 0.5
rm "$LIBRARY/_UNPACK_Movie.2024/movie.part01.rar"
rm "$LIBRARY/_UNPACK_Movie.2024/movie.part02.rar"
echo "$(ts) RAR parts deleted. Expect: activity resets quiesce timer."
sleep 1

# Phase 5: rename directory (remove _UNPACK_ prefix)
echo "$(ts) Phase 5: Renaming _UNPACK_Movie.2024 -> Movie.2024"
mv "$LIBRARY/_UNPACK_Movie.2024" "$LIBRARY/Movie.2024"
echo "$(ts) Directory renamed. Expect: old tracking canceled, new dir tracked, quiesce restarts."
echo "$(ts) Waiting for quiesce + debounce..."
sleep 12
watcher_status

# ─── TEST 4: Rapid file creation (batch test) ──────────────────
echo ""
echo "=== TEST 4: Rapid file creation (20 files) ==="
echo "$(ts) Creating 20 .mp4 files rapidly..."
mkdir -p "$LIBRARY/rapid-test"
# Give the dir watch time to set up
sleep 1
for i in $(seq 1 20); do
    dd if=/dev/urandom of="$LIBRARY/rapid-test/clip_${i}.mp4" bs=1024 count=10 2>/dev/null
done
echo "$(ts) All 20 files created. Expect: activity keeps timer resetting, single scan after quiesce."
sleep 8
watcher_status

# ─── TEST 5: Directory rename with contents ─────────────────────
echo ""
echo "=== TEST 5: Directory rename ==="
echo "$(ts) Renaming rapid-test -> renamed-collection..."
mv "$LIBRARY/rapid-test" "$LIBRARY/renamed-collection"
echo "$(ts) Directory renamed. Expect: removal batcher -> full scan for move detection + clean."
sleep 12
watcher_status

# ─── TEST 6: Delete after scan ──────────────────────────────────
echo ""
echo "=== TEST 6: Delete after scan ==="
echo "$(ts) Deleting test.mp4 (scanned in test 1)..."
rm "$LIBRARY/test.mp4"
echo "$(ts) File deleted. Expect: REMOVE -> removal batcher -> clean triggered."
sleep 8
watcher_status

# ─── TEST 7: Failed job rename ──────────────────────────────────
echo ""
echo "=== TEST 7: Failed job rename (_UNPACK_ -> _FAILED_) ==="
mkdir -p "$LIBRARY/_UNPACK_BadJob"
dd if=/dev/urandom of="$LIBRARY/_UNPACK_BadJob/corrupt.mp4" bs=1024 count=50 2>/dev/null
sleep 2
echo "$(ts) Renaming _UNPACK_BadJob -> _FAILED_BadJob..."
mv "$LIBRARY/_UNPACK_BadJob" "$LIBRARY/_FAILED_BadJob"
echo "$(ts) Expect: _UNPACK_ tracking canceled, _FAILED_ dir tracked, quiesce fires."
sleep 8
watcher_status

# ─── Final status ──────────────────────────────────────────────
echo ""
echo "=== FINAL STATUS ==="
watcher_status

echo ""
echo "=== Test suite complete ==="
echo "$(ts) Review full logs: docker logs stash-watcher-test"
