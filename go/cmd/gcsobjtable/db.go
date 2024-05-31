package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// TODO: Address the fact that SQLite does not hold uint64 that exceeds positive int64
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
	//log.Println("object_locations table created successfully")

	// Create index on object_uid for object_locations table
	statement, err = db.Prepare(createIndexSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	if err != nil {
		return err
	}
	//log.Println("Index on object_uid for object_locations table created successfully")

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

	log.Println("Object locations inserted/updated successfully 🌟👾🦄")
	return nil
}

// TODO: Address the fact that SQLite does not hold uint64 that exceeds positive int64
func createWaitlistTable(db *sql.DB) error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS waitlist (
        "object_uid" INTEGER NOT NULL,
        "ip_address" TEXT NOT NULL,
        PRIMARY KEY (object_uid, ip_address)
    );`

	createIndexSQL := `CREATE INDEX IF NOT EXISTS idx_object_uid_waitlist ON waitlist (object_uid);`

	// Create waitlist table
	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	if err != nil {
		return err
	}
	//log.Println("waitlist table created successfully")

	// Create index on object_uid for waitlist table
	statement, err = db.Prepare(createIndexSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	if err != nil {
		return err
	}
	//log.Println("Index on object_uid for waitlist table created successfully")

	return nil
}

func insertUser(db *sql.DB, name string, age int) {
	insertUserSQL := `INSERT INTO users (name, age) VALUES (?, ?)`
	statement, err := db.Prepare(insertUserSQL)
	if err != nil {
		log.Fatal(err)
	}
	_, err = statement.Exec(name, age)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted user successfully")
}

func queryUsers(db *sql.DB) {
	queryUserSQL := `SELECT id, name, age FROM users`
	rows, err := db.Query(queryUserSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		var age int
		err = rows.Scan(&id, &name, &age)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %d, Name: %s, Age: %d\n", id, name, age)
	}
}
