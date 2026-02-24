package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := flag.String("db", "", "Postgres connection string")
	projectID := flag.String("project", "", "Project ID to verify")
	flag.Parse()

	if *dbURL == "" || *projectID == "" {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Printf("Starting verification for project %s...\n", *projectID)

	// Fetch all logs ordered by timestamp/sequence
	rows, err := db.Query(ctx, `
		SELECT id, previous_hash, payload, created_at 
		FROM logs 
		WHERE project_id = $1 
		ORDER BY created_at ASC, id ASC
	`, *projectID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var (
		lastHash    string
		count       int
		broken      bool
		lastID      int64
		lastCreated time.Time
	)

	// Initial hash for the chain start (genesis block equivalent, often empty or specific constant)
	lastHash = ""

	for rows.Next() {
		var (
			id        int64
			prevHash  string
			payload   []byte
			createdAt time.Time
		)

		if err := rows.Scan(&id, &prevHash, &payload, &createdAt); err != nil {
			log.Fatal(err)
		}

		// 1. Verify links
		if count > 0 && prevHash != lastHash {
			fmt.Printf("❌ BROKEN CHAIN at ID %d!\n", id)
			fmt.Printf("   Expected Previous: %s\n", lastHash)
			fmt.Printf("   Actual Previous:   %s\n", prevHash)
			broken = true
			break
		}

		// 2. Re-calculate hash (consistency check)
		// This assumes the 'hash' column in DB acts as the link.
		// If the schema stores the hash of the current record, we would verify against that.
		// For this tool, we replicate the logic: Hash = SHA256(prevHash + payload)

		hash := sha256.New()
		hash.Write([]byte(prevHash))
		hash.Write(payload)
		currentHash := hex.EncodeToString(hash.Sum(nil))

		// In a real WORM implementation, we would also verify the 'current_hash' column if it existed.
		// Here we update our tracking hash for the next iteration.
		lastHash = currentHash
		lastID = id
		lastCreated = createdAt
		count++

		if count%1000 == 0 {
			fmt.Printf("Verified %d logs...\r", count)
		}
	}

	if !broken {
		fmt.Printf("\n✅ Verification Complete. Chain is INTACT.\n")
		fmt.Printf("   Total Logs: %d\n", count)
		fmt.Printf("   Last Log ID: %d (%s)\n", lastID, lastCreated)
		fmt.Printf("   Final Hash:  %s\n", lastHash)
	} else {
		os.Exit(1)
	}
}
