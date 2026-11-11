#!/bin/sh
# Prepares the web app's media from the sources in ../audio and ../graphics.
# Audio: re-encodes ../audio into src/audio as MP3 for the viewer:
# the sound effects at 128 kbps, the fight_music_* tracks and the menu music at 96 kbps (they are
# embedded in the pf binary, so size matters). Needs ffmpeg.
# Images: scales the cover art down to 800 px wide (JPEG) into src/images.
# Needs sips (macOS) or ImageMagick. Run it after adding or changing files in
# ../audio or ../graphics, then `npm run build`.
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

mkdir -p src/images
if command -v sips >/dev/null 2>&1; then
  sips -s format jpeg -s formatOptions 80 --resampleWidth 800 ../graphics/prompt_fighter_cover.jpeg --out src/images/cover.jpg >/dev/null 2>&1
else
  magick ../graphics/prompt_fighter_cover.jpeg -resize 800x -quality 80 src/images/cover.jpg
fi
ls -l src/images
