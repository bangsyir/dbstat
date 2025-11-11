#!/bin/bash

# Build script for DB Manager
APP_NAME="dbstat"
ICON_PATH="assets/icon.svg"
BUILD_DIR="build"
BINARY_NAME="$BUILD_DIR/$APP_NAME"

echo "Building DB Manager for Linux..."

# Create build directory
mkdir -p $BUILD_DIR

# Build the application
echo "Compiling..."
go build -ldflags="-s -w" -o $BINARY_NAME

if [ $? -eq 0 ]; then
    echo "Build successful!"
    chmod +x $BINARY_NAME

    # Copy icon
    if [ -f "$ICON_PATH" ]; then
        cp "$ICON_PATH" "$BUILD_DIR/"
        echo "Icon copied"
    fi

    # Create desktop file
    cat > "$BUILD_DIR/$APP_NAME.desktop" << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=DB Stat Manager
Comment=Manage database services
Exec=$BINARY_NAME
Icon=$APP_NAME
Terminal=false
StartupWMClass=DB Stat
Categories=Utility;Database;
Keywords=database;postgresql;mysql;redis;
EOF

    echo "Desktop file created"
    echo "Application built in: $BUILD_DIR/"
else
    echo "Build failed!"
    exit 1
fi
