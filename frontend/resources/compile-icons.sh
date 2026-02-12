#!/bin/bash
# This script generates PNG and ICO files from an SVG icon for use in various platforms.
# It creates PNG files of different sizes, an ICO file for Windows, and an ICNS file for macOS.
# It also prepares a hicolor structure for Linux and creates a square icon for Linux.
# Ensure the script is run from the correct directory

rm -rf icons
rm -rf icons.iconset
rm -rf icon.icns
rm -rf icon-square.png
rm -rf icon.ico
rm -rf icon.png

mkdir -p icons/png
mkdir -p icons/hicolor
mkdir -p icons.iconset

# Create PNG files of different sizes
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 16x16 -gravity center -extent 16x16 icons/png/16x16.png
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 32x32 -gravity center -extent 32x32 icons/png/32x32.png
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 48x48 -gravity center -extent 48x48 icons/png/48x48.png
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 64x64 -gravity center -extent 64x64 icons/png/64x64.png
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 128x128 -gravity center -extent 128x128 icons/png/128x128.png
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 256x256 -gravity center -extent 256x256 icons/png/256x256.png
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 512x512 -gravity center -extent 512x512 icons/png/512x512.png
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 1024x1024 -gravity center -extent 1024x1024 icons/png/1024x1024.png

# Create icon.png (512x512) in the resources directory for main use
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 512x512 -gravity center -extent 512x512 icon.png

echo "PNG files created successfully"

# Create ICO file for Windows using multiple PNG sizes
magick icons/png/16x16.png icons/png/32x32.png icons/png/48x48.png icons/png/64x64.png icons/png/128x128.png icons/png/256x256.png icon.ico

echo "ICO file created successfully"

# Create iconset directory for macOS
mkdir -p icons.iconset

# Copy PNG files with mac-compatible filenames
cp icons/png/16x16.png icons.iconset/icon_16x16.png
cp icons/png/32x32.png icons.iconset/icon_16x16@2x.png
cp icons/png/32x32.png icons.iconset/icon_32x32.png
cp icons/png/64x64.png icons.iconset/icon_32x32@2x.png
cp icons/png/128x128.png icons.iconset/icon_128x128.png
cp icons/png/256x256.png icons.iconset/icon_128x128@2x.png
cp icons/png/256x256.png icons.iconset/icon_256x256.png
cp icons/png/512x512.png icons.iconset/icon_256x256@2x.png
cp icons/png/512x512.png icons.iconset/icon_512x512.png
cp icons/png/1024x1024.png icons.iconset/icon_512x512@2x.png

echo "macOS iconset prepared"

# Try using iconutil if available (for Mac environments)
if command -v iconutil &> /dev/null; then
    iconutil -c icns icons.iconset -o icon.icns
    echo "ICNS file created using iconutil"
else
    # If iconutil is not available, use ImageMagick (this is more of a fallback, quality may vary)
    echo "iconutil not found, using ImageMagick as fallback for ICNS creation"
    
    # Use png2icns if available
    if command -v png2icns &> /dev/null; then
        png2icns icon.icns icons.iconset/icon_*
        echo "ICNS file created using png2icns"
    else
        # Create a single-size ICNS as fallback (not ideal but better than nothing)
        magick icons/png/512x512.png icon.icns
        echo "ICNS file created using ImageMagick (basic conversion)"
    fi
fi

# Create proper hicolor structure
mkdir -p icons/hicolor/16x16/apps
mkdir -p icons/hicolor/32x32/apps
mkdir -p icons/hicolor/48x48/apps
mkdir -p icons/hicolor/64x64/apps
mkdir -p icons/hicolor/128x128/apps
mkdir -p icons/hicolor/256x256/apps
mkdir -p icons/hicolor/512x512/apps

# Copy the icons to their respective directories
cp icons/png/16x16.png icons/hicolor/16x16/apps/dropz-electron.png
cp icons/png/32x32.png icons/hicolor/32x32/apps/dropz-electron.png
cp icons/png/48x48.png icons/hicolor/48x48/apps/dropz-electron.png
cp icons/png/64x64.png icons/hicolor/64x64/apps/dropz-electron.png
cp icons/png/128x128.png icons/hicolor/128x128/apps/dropz-electron.png
cp icons/png/256x256.png icons/hicolor/256x256/apps/dropz-electron.png
cp icons/png/512x512.png icons/hicolor/512x512/apps/dropz-electron.png

echo "Linux hicolor structure created"

# Create icon-square.png (512x512) - ensuring square dimensions
magick -density 1000 -background '#000616' svg/icon.svg -flatten -resize 512x512 -gravity center -extent 512x512 icon-square.png

echo "Square icon created for Linux"
