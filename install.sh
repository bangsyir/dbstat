#!/bin/bash

APP_NAME="dbstat"
BUILD_DIR="build"
USER_HOME="$HOME"
LOCAL_BIN="$USER_HOME/.local/bin"
LOCAL_APPS="$USER_HOME/.local/share/applications"
LOCAL_ICONS="$USER_HOME/.local/share/icons"

echo "Installing DB Manager..."

# Create directories if they don't exist
mkdir -p $LOCAL_BIN
mkdir -p $LOCAL_APPS
mkdir -p $LOCAL_ICONS

# Check if build exists
if [ ! -f "$BUILD_DIR/$APP_NAME" ]; then
    echo "Error: Build not found. Run ./build.sh first."
    exit 1
fi

# Copy binary
echo "Installing binary..."
cp "$BUILD_DIR/$APP_NAME" "$LOCAL_BIN/"
chmod +x "$LOCAL_BIN/$APP_NAME"

# Copy icon if exists
if [ -f "$BUILD_DIR/icon.png" ]; then
    echo "Installing icon..."
    cp "$BUILD_DIR/icon.png" "$LOCAL_ICONS/$APP_NAME.png"
fi

# Create desktop file
echo "Creating desktop entry..."
cat > "$LOCAL_APPS/$APP_NAME.desktop" << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=DB Stat Manager
Comment=Manage database services
Exec=env HOME=$USER_HOME $LOCAL_BIN/$APP_NAME
Icon=$APP_NAME
Terminal=false
Categories=Utility;Database;
Keywords=database;postgresql;mysql;redis;
StartupWMClass=db-stat
StartupNotify=true
EOF

# Make desktop file executable
chmod +x "$LOCAL_APPS/$APP_NAME.desktop"

# Update desktop database
echo "Updating desktop database..."
update-desktop-database $LOCAL_APPS

echo "Installation complete!"
echo "You should now find 'DB Manager' in your application menu."
echo "If not, try logging out and back in, or run:"
echo "  update-desktop-database ~/.local/share/applications"
