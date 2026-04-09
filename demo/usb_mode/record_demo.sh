#!/usr/bin/env bash
# Record wrap_demo.sh in xfce4-terminal + ffmpeg x11grab → MP4 + GIFs.
set -euo pipefail

export DISPLAY="${DISPLAY:-:1}"
ART="$(cd "$(dirname "$0")" && pwd)"
_REPO="$(cd "$ART/../.." && pwd)"
DEMO_HOME="${DEMO_HOME:-/tmp/git-fire-usb-demo-record}"
export DEMO_HOME
export GIT_FIRE_BIN="${GIT_FIRE_BIN:-$_REPO/git-fire}"

CAP_W="${CAP_W:-1200}"
CAP_H="${CAP_H:-700}"
CAP_X="${CAP_X:-40}"
CAP_Y="${CAP_Y:-50}"
FPS="${FPS:-12}"

MP4_OUT="$ART/usb_mode_demo_full.mp4"
PAL="$ART/palette.png"

rm -f "$MP4_OUT" "$PAL" "$ART"/usb_demo_part*.gif "$ART"/usb_mode_demo_full.gif "$ART/ffmpeg.log"
rm -rf "$DEMO_HOME"

ffmpeg -y -f x11grab -video_size "${CAP_W}x${CAP_H}" -framerate "$FPS" \
  -draw_mouse 0 -i "${DISPLAY}.0+${CAP_X},${CAP_Y}" \
  -codec:v libx264 -pix_fmt yuv420p -preset veryfast -crf 22 \
  "$MP4_OUT" 2>"$ART/ffmpeg.log" &
FFPID=$!

sleep 1.2

xfce4-terminal \
  --geometry="100x34+${CAP_X}+${CAP_Y}" \
  --hide-menubar \
  --font="Monospace 10" \
  -T "git-fire USB mode demo" \
  -x bash "$ART/wrap_demo.sh" &

for _ in $(seq 1 400); do
  if pgrep -f "usb_mode_demo_run.sh" >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done
while pgrep -f "usb_mode_demo_run.sh" >/dev/null 2>&1; do
  sleep 0.4
done
sleep 2

kill -INT "$FFPID" 2>/dev/null || true
wait "$FFPID" 2>/dev/null || true

ffmpeg -y -i "$MP4_OUT" -vf "fps=${FPS},scale=780:-1:flags=lanczos,palettegen" "$PAL" 2>>"$ART/ffmpeg.log"
ffmpeg -y -i "$MP4_OUT" -i "$PAL" -lavfi "fps=${FPS},scale=780:-1:flags=lanczos[x];[x][1:v]paletteuse" \
  "$ART/usb_mode_demo_full.gif" 2>>"$ART/ffmpeg.log"

DUR=$(ffprobe -v error -show_entries format=duration -of default=nw=1:nk=1 "$MP4_OUT")
Q=$(awk -v d="$DUR" 'BEGIN{printf "%.2f", d/4}')
for i in 1 2 3 4; do
  START=$(awk -v q="$Q" -v i="$i" 'BEGIN{printf "%.2f", q*(i-1)}')
  ffmpeg -y -ss "$START" -i "$MP4_OUT" -t "$Q" -an \
    -vf "fps=10,scale=720:-1:flags=lanczos,split[s0][s1];[s0]palettegen=reserve_transparent=0[p];[s1][p]paletteuse" \
    "$ART/usb_demo_part${i}.gif" 2>>"$ART/ffmpeg.log"
done

ls -la "$MP4_OUT" "$ART"/*.gif
