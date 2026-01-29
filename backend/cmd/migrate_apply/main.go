package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	apply := flag.Bool("apply", false, "apply migration")
	flag.Parse()

	migDir := filepath.Join("internal", "migrations")
	files, err := os.ReadDir(migDir)
	if err != nil {
		log.Fatalf("read migrations dir: %v", err)
	}
	var hasErrors bool
	for _, f := range files {
		name := f.Name()
		if !*apply {
			fmt.Println(name)
			continue
		}
		b, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			log.Printf("ERROR: read file %s: %v", name, err)
			hasErrors = true
			continue
		}
		if _, err := db.Exec(context.Background(), string(b)); err != nil {
			log.Printf("ERROR: failed to apply %s: %v", name, err)
			hasErrors = true
			continue
		}
		fmt.Printf("applied %s\n", name)
	}
	if hasErrors {
		log.Println("Some migrations failed - check errors above")
	}
}
