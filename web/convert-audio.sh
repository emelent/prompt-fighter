#!/bin/sh
# Re-encodes the sources in ../audio into src/audio as MP3 for the viewer:
# the sound effects at 128 kbps, the fight_music_* tracks and the menu music at 96 kbps (they are
# embedded in the pf binary, so size matters). Needs ffmpeg. Run it after
# adding or changing files in ../audio, then `npm run build`.
set -eu
cd "$(dirname "$0")"
rm -f src/audio/*.mp3
mkdir -p src/audio
for name in punch_hit punch_miss kick_hit kick_miss block jump; do
  src=$(ls ../audio/"$name".* | head -n 1)
  ffmpeg -hide_banner -loglevel error -y -i "$src" -c:a libmp3lame -b:a 128k "src/audio/$name.mp3"
done
for src in ../audio/fight_music_*.* ../audio/menu.*; do
  name=$(basename "$src"); name=${name%.*}
  ffmpeg -hide_banner -loglevel error -y -i "$src" -c:a libmp3lame -b:a 96k "src/audio/$name.mp3"
done
ls -l src/audio
