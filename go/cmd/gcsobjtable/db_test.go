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

	// Initial example data
	objectLocations := map[uint64][]uint64{
		1: {100, 101},
		2: {102, 103},
	}

	// Insert initial data
	if err := insertOrUpdateObjectLocations(database, objectLocations); err != nil {
		t.Fatalf("Failed to insert/update object locations: %v", err)
	}

	// Verify initial data
	verifyObjectLocations(t, database, objectLocations)

	// Update example data
	objectLocations = map[uint64][]uint64{
		1: {100, 102}, // Change node_ids for object_uid 1
		3: {104},      // Add new object_uid 3
	}

	// Insert updated data
	if err := insertOrUpdateObjectLocations(database, objectLocations); err != nil {
		t.Fatalf("Failed to insert/update object locations: %v", err)
	}

	// Verify updated data
	verifyObjectLocations(t, database, objectLocations)
}

func verifyObjectLocations(t *testing.T, db *sql.DB, expected map[uint64][]uint64) {
	// Extract the keys from the expected map
	keys := make([]uint64, 0, len(expected))
	for k := range expected {
		keys = append(keys, k)
	}

	// Build the query with placeholders
	query := "SELECT object_uid, node_id FROM object_locations WHERE object_uid IN ("
	for i := range keys {
		if i > 0 {
			query += ","
		}
		query += "?"
	}
	query += ")"

	// Convert keys to a slice of interface{} for the query arguments
	args := make([]interface{}, len(keys))
	for i, v := range keys {
		args[i] = v
	}

	// Query the database
	rows, err := db.Query(query, args...)
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

	// Compare results
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
