package main

import (
	"database/sql"
	"fmt"
	"log"
)

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

func insertOrUpdateObjectLocations(db *sql.DB, objectLocations map[uint64][]uint64) error {
	// Prepare the insert statement
	insertSQL := `INSERT OR IGNORE INTO object_locations (object_uid, node_id) VALUES (?, ?);`
	statement, err := db.Prepare(insertSQL)
	if err != nil {
		return err
	}
	defer statement.Close()

	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Iterate through the map and insert each pair
	for objectUID, nodeIDs := range objectLocations {
		for _, nodeID := range nodeIDs {
			_, err := tx.Stmt(statement).Exec(objectUID, nodeID)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return err
	}

	//log.Println("Object locations inserted/updated successfully")
	return nil
}

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
