package main

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateObjectLocationsTable(t *testing.T) {
	// Use an in-memory SQLite database for testing
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Call the function to create the object_locations table and index
	if err := createObjectLocationsTable(database); err != nil {
		t.Fatalf("Failed to create object_locations table: %v", err)
	}

	// Check if object_locations table exists
	var tableName string
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='object_locations';").Scan(&tableName)
	if err != nil {
		t.Fatalf("object_locations table was not created: %v", err)
	}

	// Check if index on object_uid for object_locations table exists
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_object_uid_object_locations';").Scan(&tableName)
	if err != nil {
		t.Fatalf("Index on object_uid for object_locations table was not created: %v", err)
	}
}

func TestInsertOrUpdateObjectLocations(t *testing.T) {
	// Use an in-memory SQLite database for testing
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Call the function to create the object_locations table
	if err := createObjectLocationsTable(database); err != nil {
		t.Fatalf("Failed to create object_locations table: %v", err)
	}

	// Example in-memory objectLocations map
	objectLocations := map[uint64][]uint64{
		1: {100, 101},
		2: {102, 103},
	}

	// Insert or update objectLocations
	if err := insertOrUpdateObjectLocations(database, objectLocations); err != nil {
		t.Fatalf("Failed to insert/update object locations: %v", err)
	}

	// Verify that data was inserted correctly
	rows, err := database.Query("SELECT object_uid, node_id FROM object_locations")
	if err != nil {
		t.Fatalf("Failed to query object_locations: %v", err)
	}
	defer rows.Close()

	result := make(map[uint64][]uint64)
	for rows.Next() {
		var objectUID uint64
		var nodeID uint64
		if err := rows.Scan(&objectUID, &nodeID); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		result[objectUID] = append(result[objectUID], nodeID)
	}

	expected := objectLocations
	if len(result) != len(expected) {
		t.Fatalf("Expected %d entries, got %d", len(expected), len(result))
	}

	for objectUID, nodeIDs := range expected {
		if len(result[objectUID]) != len(nodeIDs) {
			t.Fatalf("For objectUID %d, expected %d nodeIDs, got %d", objectUID, len(nodeIDs), len(result[objectUID]))
		}
		for i, nodeID := range nodeIDs {
			if result[objectUID][i] != nodeID {
				t.Fatalf("For objectUID %d, at index %d, expected nodeID %d, got %d", objectUID, i, nodeID, result[objectUID][i])
			}
		}
	}
}

func TestCreateWaitlistTable(t *testing.T) {
	// Use an in-memory SQLite database for testing
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Call the function to create the waitlist table and index
	if err := createWaitlistTable(database); err != nil {
		t.Fatalf("Failed to create waitlist table: %v", err)
	}

	// Check if waitlist table exists
	var tableName string
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='waitlist';").Scan(&tableName)
	if err != nil {
		t.Fatalf("waitlist table was not created: %v", err)
	}

	// Check if index on object_uid for waitlist table exists
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_object_uid_waitlist';").Scan(&tableName)
	if err != nil {
		t.Fatalf("Index on object_uid for waitlist table was not created: %v", err)
	}
}
