package main

import (
	"database/sql"
	"strings"
)

func setCacheSizeToZero(db *sql.DB) error {
	_, err := db.Exec("PRAGMA cache_size = 0;")
	return err
}

func createObjectLocationsTable(db *sql.DB) error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS object_locations (
        "object_uid" INTEGER NOT NULL,
        "node_id" INTEGER NOT NULL,
        PRIMARY KEY (object_uid, node_id)
    );`

	createIndexSQL := `CREATE INDEX IF NOT EXISTS idx_object_uid_object_locations ON object_locations (object_uid);`

	// Create object_locations table
	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	if err != nil {
		return err
	}

	// Create index on object_uid for object_locations table
	statement, err = db.Prepare(createIndexSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	if err != nil {
		return err
	}

	return nil
}

// We assume that, for any object_uid in the possession of memory, it has the latest state
// on all of the locations that it can be found, and hence it is imperate to overwrite the
// corresponding "state" in the database. That is, the database might have a location which in fact
// no longer exists for this object_uid, so we need to eliminate that from the database.
func insertOrUpdateObjectLocations(db *sql.DB, objectLocations map[uint64][]uint64) error {
	// NOTE ON IMPLEMENTATION DECISION:
	// We DELETE all rows that have object_uid and then INSERT anew because this is a lot more
	// efficient than having to query the db repeatedly to do comparisons about which rows
	// no longer exist in the state, and trying to selectively delete those rows.

	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Prepare the bulk delete statement
	objectUIDs := make([]string, 0, len(objectLocations))
	deleteArgs := make([]interface{}, 0, len(objectLocations))
	for objectUID := range objectLocations {
		objectUIDs = append(objectUIDs, "?")
		deleteArgs = append(deleteArgs, objectUID)
	}
	deleteSQL := `DELETE FROM object_locations WHERE object_uid IN (` + strings.Join(objectUIDs, ",") + `);`

	// Prepare the bulk insert statement
	insertSQL := `INSERT INTO object_locations (object_uid, node_id) VALUES `
	valueStrings := []string{}
	valueArgs := []interface{}{}

	for objectUID, nodeIDs := range objectLocations {
		for _, nodeID := range nodeIDs {
			valueStrings = append(valueStrings, "(?, ?)")
			valueArgs = append(valueArgs, objectUID, nodeID)
		}
	}

	insertSQL = insertSQL + strings.Join(valueStrings, ",")

	// Execute the bulk delete and insert at the end of preparation
	if _, err := tx.Exec(deleteSQL, deleteArgs...); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec(insertSQL, valueArgs...); err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// returns an empty []uint64 if nothing found
func getObjectLocations(db *sql.DB, objectUID uint64) ([]uint64, error) {
	querySQL := `SELECT node_id FROM object_locations WHERE object_uid = ?;`

	rows, err := db.Query(querySQL, objectUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodeIDs []uint64
	for rows.Next() {
		var nodeID uint64
		if err := rows.Scan(&nodeID); err != nil {
			return nil, err
		}
		nodeIDs = append(nodeIDs, nodeID)
	}

	// Check for errors from iterating over rows.
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return nodeIDs, nil
}
