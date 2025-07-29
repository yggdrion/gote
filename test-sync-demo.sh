#!/bin/bash

# Demo script for testing Gote file synchronization
# This script demonstrates how external file changes are handled

echo "=== Gote File Synchronization Demo ==="
echo
echo "This script demonstrates how Gote handles external file changes"
echo "Make sure Gote is running on http://localhost:8080 before running this script"
echo

# Wait for user to confirm Gote is running
read -p "Is Gote running and do you have some notes created? (y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Please start Gote and create some notes first, then run this script again."
    exit 1
fi

DATA_DIR="./data"

# Check if data directory exists
if [ ! -d "$DATA_DIR" ]; then
    echo "Error: Data directory '$DATA_DIR' not found"
    echo "Make sure you're running this script from the Gote root directory"
    exit 1
fi

# List existing notes
echo "Current notes in $DATA_DIR:"
ls -la "$DATA_DIR"/*.json 2>/dev/null || echo "No notes found"
echo

# Test 1: Create a new note file externally
echo "=== Test 1: Creating a new note file externally ==="
NEW_NOTE_ID="synctest1"
NEW_NOTE_FILE="$DATA_DIR/${NEW_NOTE_ID}.json"

# Note: This creates an unencrypted note for demo purposes
# In real usage, notes would need to be properly encrypted
cat > "$NEW_NOTE_FILE" << 'EOF'
{
  "id": "synctest1",
  "encrypted_data": "demo-encrypted-content",
  "created_at": "2025-07-29T09:30:00Z",
  "updated_at": "2025-07-29T09:30:00Z"
}
EOF

echo "Created new note file: $NEW_NOTE_FILE"
echo "Check your browser - the new note should appear automatically!"
echo

read -p "Press Enter to continue to the next test..."

# Test 2: Modify an existing note file
echo "=== Test 2: Modifying an existing note file ==="
EXISTING_NOTE=$(ls "$DATA_DIR"/*.json | head -n 1)
if [ -n "$EXISTING_NOTE" ]; then
    echo "Modifying existing note: $EXISTING_NOTE"
    # Update the updated_at timestamp to simulate an external change
    sed -i.bak 's/"updated_at": "[^"]*"/"updated_at": "'$(date -u +%Y-%m-%dT%H:%M:%S)Z'"/' "$EXISTING_NOTE"
    echo "Modified note file timestamp"
    echo "Check your browser - you should see a file event in the server logs!"
else
    echo "No existing notes found to modify"
fi
echo

read -p "Press Enter to continue to the next test..."

# Test 3: Delete a note file
echo "=== Test 3: Deleting the test note file ==="
if [ -f "$NEW_NOTE_FILE" ]; then
    rm "$NEW_NOTE_FILE"
    echo "Deleted test note file: $NEW_NOTE_FILE"
    echo "Check your browser - the note should disappear automatically!"
else
    echo "Test note file not found"
fi
echo

# Test 4: Test manual sync
echo "=== Test 4: Manual sync test ==="
echo "Now test the manual sync functionality:"
echo "1. Go to your browser"
echo "2. Click the '🔄 Sync' button in the header"
echo "3. Or open Settings and click 'Sync from Disk'"
echo "4. You should see a success notification"
echo

echo "=== Demo Complete ==="
echo "Key points demonstrated:"
echo "✅ External file creation is automatically detected"
echo "✅ External file modification is automatically detected"
echo "✅ External file deletion is automatically detected"
echo "✅ Manual sync button provides on-demand refresh"
echo
echo "For production use with Syncthing:"
echo "1. Set up Syncthing to sync the ./data directory"
echo "2. Use the same password on all devices"
echo "3. Notes will sync automatically across devices"
echo "4. Use the sync button if you notice any inconsistencies"
