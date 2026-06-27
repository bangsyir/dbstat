#!/bin/bash

APP_NAME="dbstat"
USER_HOME="$HOME"
LOCAL_BIN="$USER_HOME/.local/bin"
LOCAL_APPS="$USER_HOME/.local/share/applications"
LOCAL_ICONS="$USER_HOME/.local/share/icons"

echo "Uninstalling DB Manager..."

# Remove binary
if [ -f "$LOCAL_BIN/$APP_NAME" ]; then
  echo "Removing binary..."
  rm "$LOCAL_BIN/$APP_NAME"
fi

# Remove desktop entry
if [ -f "$LOCAL_APPS/$APP_NAME.desktop" ]; then
  echo "Removing desktop entry..."
  rm "$LOCAL_APPS/$APP_NAME.desktop"
fi

# Remove icon
if [ -f "$LOCAL_ICONS/$APP_NAME.png" ]; then
  echo "Removing icon..."
  rm "$LOCAL_ICONS/$APP_NAME.png"
fi

# Update desktop database
echo "Updating desktop database..."
update-desktop-database $LOCAL_APPS 2>/dev/null

echo "Uninstall complete!"
