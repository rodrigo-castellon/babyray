package main

import (
	"database/sql"
	"fmt"
	"log"
)

func createTable(db *sql.DB) {
	createTableSQL := `CREATE TABLE IF NOT EXISTS users (
        "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "name" TEXT,
        "age" INTEGER
    );`

	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()
	fmt.Println("Table created successfully")
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
