// dm-probe: minimal viability probe for the DaMeng storage backend.
// Run: go run ./cmd/dm-probe --uri "dm://SYSDBA:password@host:5236"
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	_ "gitee.com/chunanyong/dm"
)

func main() {
	uri := flag.String("uri", "", "DaMeng connection URI, e.g. dm://SYSDBA:pass@192.168.107.9:5236")
	flag.Parse()
	if *uri == "" {
		log.Fatal("--uri is required")
	}

	db, err := sql.Open("dm", *uri)
	if err != nil {
		log.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Ping
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}
	fmt.Println("✓ ping OK")

	// 2. Server version
	var version string
	if err := db.QueryRowContext(ctx, "SELECT BANNER FROM V$VERSION WHERE ROWNUM = 1").Scan(&version); err != nil {
		// Some DM versions expose version differently; fall back gracefully.
		fmt.Printf("  (version query failed: %v)\n", err)
	} else {
		fmt.Printf("✓ server version: %s\n", version)
	}

	// 3. DDL: create a throwaway table (drop first in case previous run left it behind)
	_, _ = db.ExecContext(ctx, `DROP TABLE openfga_probe`)
	_, err = db.ExecContext(ctx, `CREATE TABLE openfga_probe (id VARCHAR(64) PRIMARY KEY, val VARCHAR(64) NOT NULL)`)
	if err != nil {
		log.Fatalf("CREATE TABLE: %v", err)
	}
	fmt.Println("✓ CREATE TABLE OK")
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DROP TABLE openfga_probe`)
		fmt.Println("✓ DROP TABLE (cleanup) OK")
	}()

	// 4. INSERT
	_, err = db.ExecContext(ctx, `INSERT INTO openfga_probe (id, val) VALUES (?, ?)`, "01PROBE00000000000000000000", "hello-dm")
	if err != nil {
		log.Fatalf("INSERT: %v", err)
	}
	fmt.Println("✓ INSERT OK")

	// 5. SELECT
	var id, val string
	err = db.QueryRowContext(ctx, `SELECT id, val FROM openfga_probe WHERE id = ?`, "01PROBE00000000000000000000").Scan(&id, &val)
	if err != nil {
		log.Fatalf("SELECT: %v", err)
	}
	fmt.Printf("✓ SELECT OK: id=%s val=%s\n", id, val)

	// 6. MERGE INTO (upsert — used by WriteAssertions)
	mergeSQL := `MERGE INTO openfga_probe t
USING (SELECT ? AS id, ? AS val FROM dual) s
ON (t.id = s.id)
WHEN MATCHED THEN UPDATE SET t.val = s.val
WHEN NOT MATCHED THEN INSERT (id, val) VALUES (s.id, s.val)`
	_, err = db.ExecContext(ctx, mergeSQL, "01PROBE00000000000000000000", "hello-dm-updated")
	if err != nil {
		log.Fatalf("MERGE INTO: %v", err)
	}
	fmt.Println("✓ MERGE INTO (upsert) OK")

	// 7. DATEADD (used by ReadChanges)
	var ts time.Time
	err = db.QueryRowContext(ctx, `SELECT DATEADD(MICROSECOND, -1000000, NOW()) FROM dual`).Scan(&ts)
	if err != nil {
		log.Fatalf("DATEADD: %v", err)
	}
	fmt.Printf("✓ DATEADD OK: %s\n", ts.Format(time.RFC3339))

	fmt.Println("\nAll checks passed — DaMeng backend is viable.")
}
