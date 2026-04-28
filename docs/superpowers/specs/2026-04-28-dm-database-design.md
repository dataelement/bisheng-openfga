# DaMeng (达梦) Storage Backend Design

**Date:** 2026-04-28
**Status:** Approved
**Scope:** Private/internal use

## Overview

Add DaMeng (达梦, DM) as a fourth SQL-backed storage engine in OpenFGA, alongside the existing MySQL, PostgreSQL, and SQLite backends. The implementation follows the MySQL backend pattern closely, reusing `sqlcommon` for all shared query logic.

## Goals

- Full `storage.OpenFGADatastore` interface implementation
- Goose-based schema migrations
- Prometheus connection pool metrics
- Connection pool configuration via existing flags
- No secondary read replica support (out of scope)

## Architecture

### File Structure

**New files:**
```
pkg/storage/dm/
  doc.go                                       — package documentation
  dm.go                                        — Datastore struct + all OpenFGADatastore methods

assets/migrations/dm/
  001_initialize_schema.sql
  002_add_authorization_model_version.sql
  003_add_reverse_lookup_index.sql
  004_add_authorization_model_serialized_protobuf.sql
  005_add_conditions_to_tuples.sql
  006_extend_object_id.sql
  007_collate_object_id.sql
```

**Modified files:**
```
assets/assets.go                       — add DMMigrationDir constant
cmd/run/run.go                         — add "dm" case to engine switch + import
pkg/storage/migrate/migrate.go         — add "dm" case to migration switch
go.mod / go.sum                        — add gitee.com/chunanyong/dm dependency
```

### Dependencies

| Concern | Choice |
|---|---|
| Go driver | `gitee.com/chunanyong/dm` (registers as `"dm"` with `database/sql`) |
| Query builder | `github.com/Masterminds/squirrel` with `?` placeholder (MySQL-compatible) |
| Migrations | `github.com/pressly/goose/v3` |
| Shared query logic | `pkg/storage/sqlcommon` |

## Datastore Implementation

### Struct

```go
type Datastore struct {
    stbl                   sq.StatementBuilderType
    db                     *sql.DB
    dbInfo                 *sqlcommon.DBInfo
    logger                 logger.Logger
    dbStatsCollector       prometheus.Collector
    maxTuplesPerWriteField int
    maxTypesPerModelField  int
    versionReady           bool
}
```

Identical layout to the MySQL `Datastore`. All `OpenFGADatastore` methods delegate to `sqlcommon` the same way MySQL does.

### Connection Setup

- Connection string format: `dm://user:password@host:5236`
- Credential override (from `--datastore-username` / `--datastore-password` flags): parse URI with `net/url`, substitute user info, reconstruct URI
- Open connection: `sql.Open("dm", uri)`
- Connection pool: apply `MaxIdleConns`, `MaxOpenConns`, `ConnMaxIdleTime`, `ConnMaxLifetime` from `sqlcommon.Config`
- Readiness: exponential backoff ping for up to 1 minute

### Metrics

`collectors.NewDBStatsCollector(db, "openfga")` — same as MySQL/SQLite.

### Error Handling

`HandleSQLError` maps DM driver error codes to OpenFGA sentinel errors:
- Unique constraint violation → `storage.ErrInvalidWriteInput`
- Other DB errors → wrapped as-is

The DM driver exposes errors as a typed struct with a numeric code field.

## Migration SQL Adaptations

All 7 migrations are adapted from the MySQL equivalents with these changes:

| MySQL syntax | DM equivalent | Reason |
|---|---|---|
| `LONGBLOB` | `BLOB` | DM has no `LONGBLOB` type |
| `COLLATE utf8mb4_bin` | *(omit)* | DM collation is server-configured |
| `LOCK = SHARED` / `LOCK = NONE` | *(omit)* | MySQL-specific online DDL hints |
| `MODIFY COLUMN col TYPE` | `MODIFY col TYPE` | DM `ALTER TABLE` syntax |
| `DROP INDEX idx ON table` | `DROP INDEX idx` | DM drops indexes without `ON table` |
| Multiple `ADD COLUMN` in one `ALTER` | Separate `ALTER TABLE` per column | DM may not support multi-column `ALTER` |

## Wiring

### `cmd/run/run.go`

```go
case "dm":
    datastore, err = dm.New(config.Datastore.URI, dsCfg)
    if err != nil {
        return nil, nil, fmt.Errorf("initialize dm datastore: %w", err)
    }
```

### `pkg/storage/migrate/migrate.go`

```go
case "dm":
    driver = "dm"
    migrationsPath = assets.DMMigrationDir
    // URI credential override via net/url (same pattern as postgres case)
```

### `assets/assets.go`

```go
DMMigrationDir = "migrations/dm"
```

## Configuration

No new CLI flags are required. Existing flags work as-is:

| Flag | Usage for DM |
|---|---|
| `--datastore-engine dm` | Selects DM backend |
| `--datastore-uri dm://user:pass@host:5236` | Connection string |
| `--datastore-username` / `--datastore-password` | Override credentials |
| `--datastore-max-open-conns` etc. | Pool tuning |
| `--datastore-metrics-enabled` | Prometheus metrics |

## Out of Scope

- Secondary read replica (`--datastore-secondary-uri`) — not supported
- Docker-based integration tests in CI — DM instance is not available in CI
- Upstream contribution to openfga/openfga
