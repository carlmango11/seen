package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	"github.com/carlmango11/seen/server/serve"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	var storage = flag.String("storage", "", "")
	var dbUser = flag.String("dbuser", "", "")
	var dbPw = flag.String("dbpw", "", "")
	flag.Parse()
	storageDir := *storage

	log.Println(*storage, *dbUser, *dbPw)

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/seen", *dbUser, *dbPw))
	if err != nil {
		panic(err)
	}
	if err := db.Ping(); err != nil {
		panic(err)
	}

	s := serve.New(storageDir, db)
	s.Start()
}
