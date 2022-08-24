package main

import (
	"flag"
	"log"
	"sshwordle/src/sshwordle"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {

	migrateDb()

	var apiFlag = flag.Bool("api", false,
		"Use WordAPI to generate and verify words. "+
			"If not specified uses hardcoded list of words "+
			"from original Wordle game")
	var host = flag.String("host", "127.0.0.1", "Host address for SSH server to listen")
	var port = flag.Int("port", 1337, "Port for SSH server to listen")

	flag.Parse()

	sshwordle.StartServer(*host, *port, *apiFlag)
}

func migrateDb() {
	m, err := migrate.New("file://./migrations/", "sqlite:///app/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}
}
