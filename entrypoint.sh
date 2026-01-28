#!/bin/sh
set -e

# Fix permissions on .gemini folder if mounted from host
# This ensures the node user can write to it for token renewal
if [ -d "/app/.gemini" ]; then
    echo "Checking .gemini folder permissions..."
    
    # Check if we're running as root (shouldn't be, but just in case)
    if [ "$(id -u)" = "0" ]; then
        echo "Running as root, fixing permissions..."
        chown -R node:node /app/.gemini
        chmod 700 /app/.gemini
        echo "✓ Permissions fixed"
        
        # Now switch to node user and run the app
        exec su-exec node /app/gemini-wrapper
    else
        # Already running as node user
        echo "Running as node user (UID $(id -u))"
        
        # Check if we can write
        if [ -w "/app/.gemini" ]; then
            echo "✓ Write access OK"
        else
            echo "⚠ Warning: No write access to /app/.gemini"
            echo "  Token renewal may fail!"
            echo "  Fix: sudo chown -R \$USER:\$USER ~/.gemini on host"
        fi
    fi
fi

# Start the application
exec /app/gemini-wrapper
