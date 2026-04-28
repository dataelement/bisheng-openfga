# DaMeng (达梦) Storage Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add DaMeng (达梦) as a fifth storage engine in OpenFGA, wired into the server and migration commands via `--datastore-engine dm`.

**Architecture:** New `pkg/storage/dm/` package mirrors the MySQL backend: uses `gitee.com/chunanyong/dm` as the `database/sql` driver, squirrel with `?` placeholders, and delegates all shared query logic to `pkg/storage/sqlcommon`. Seven goose migration files in `assets/migrations/dm/` are adapted from the MySQL equivalents with DM-specific syntax fixes.

**Tech Stack:** Go `database/sql`, `gitee.com/chunanyong/dm` driver, `github.com/Masterminds/squirrel`, `github.com/pressly/goose/v3`, `pkg/storage/sqlcommon`

---

## File Map

| Action | Path | Responsibility |
|---|---|---|
| Create | `pkg/storage/dm/doc.go` | Package doc comment |
| Create | `pkg/storage/dm/dm.go` | `Datastore` struct + all `OpenFGADatastore` methods |
| Create | `pkg/storage/dm/dm_test.go` | Integration tests (skipped without DM URI) |
| Create | `assets/migrations/dm/001_initialize_schema.sql` | Base tables |
| Create | `assets/migrations/dm/002_add_authorization_model_version.sql` | Schema version column |
| Create | `assets/migrations/dm/003_add_reverse_lookup_index.sql` | Reverse lookup index |
| Create | `assets/migrations/dm/004_add_authorization_model_serialized_protobuf.sql` | Protobuf column |
| Create | `assets/migrations/dm/005_add_conditions_to_tuples.sql` | Conditions columns |
| Create | `assets/migrations/dm/006_extend_object_id.sql` | Widen object_id |
| Create | `assets/migrations/dm/007_collate_object_id.sql` | Index swap (DM-adapted) |
| Modify | `assets/assets.go` | Add `DMMigrationDir` constant |
| Modify | `cmd/run/run.go` | Add `"dm"` case to engine switch |
| Modify | `pkg/storage/migrate/migrate.go` | Add `"dm"` case to migration switch |

---

## Task 1: Add the DaMeng driver dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Fetch the driver**

```bash
cd /Users/zhangguoqing/works/openfga
go get gitee.com/chunanyong/dm
```

Expected: `go.mod` updated with `gitee.com/chunanyong/dm vX.X.X`

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: no errors (the new package has no import yet, so it just resolves)

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add gitee.com/chunanyong/dm driver dependency"
```

---

## Task 2: Write the migration SQL files

**Files:**
- Create: `assets/migrations/dm/001_initialize_schema.sql` through `007_collate_object_id.sql`

- [ ] **Step 1: Create the migration directory**

```bash
mkdir -p /Users/zhangguoqing/works/openfga/assets/migrations/dm
```

- [ ] **Step 2: Write `001_initialize_schema.sql`**

```sql
-- +goose Up
CREATE TABLE tuple (
    store VARCHAR(26) NOT NULL,
    object_type VARCHAR(128) NOT NULL,
    object_id VARCHAR(128) NOT NULL,
    relation VARCHAR(50) NOT NULL,
    _user VARCHAR(256) NOT NULL,
    user_type VARCHAR(7) NOT NULL,
    ulid VARCHAR(26) NOT NULL,
    inserted_at TIMESTAMP NOT NULL,
    PRIMARY KEY (store, object_type, object_id, relation, _user)
);

CREATE UNIQUE INDEX idx_tuple_ulid ON tuple (ulid);

CREATE TABLE authorization_model (
    store VARCHAR(26) NOT NULL,
    authorization_model_id VARCHAR(26) NOT NULL,
    type VARCHAR(256) NOT NULL,
    type_definition BLOB,
    PRIMARY KEY (store, authorization_model_id, type)
);

CREATE TABLE store (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE assertion (
    store VARCHAR(26) NOT NULL,
    authorization_model_id VARCHAR(26) NOT NULL,
    assertions BLOB,
    PRIMARY KEY (store, authorization_model_id)
);

CREATE TABLE changelog (
    store VARCHAR(26) NOT NULL,
    object_type VARCHAR(256) NOT NULL,
    object_id VARCHAR(256) NOT NULL,
    relation VARCHAR(50) NOT NULL,
    _user VARCHAR(512) NOT NULL,
    operation INTEGER NOT NULL,
    ulid VARCHAR(26) NOT NULL,
    inserted_at TIMESTAMP NOT NULL,
    PRIMARY KEY (store, ulid, object_type)
);

-- +goose Down
DROP TABLE tuple;
DROP TABLE authorization_model;
DROP TABLE store;
DROP TABLE assertion;
DROP TABLE changelog;
```

- [ ] **Step 3: Write `002_add_authorization_model_version.sql`**

```sql
-- +goose Up
ALTER TABLE authorization_model ADD schema_version VARCHAR(5) NOT NULL DEFAULT '1.0';

-- +goose Down
ALTER TABLE authorization_model DROP schema_version;
```

- [ ] **Step 4: Write `003_add_reverse_lookup_index.sql`**

```sql
-- +goose Up
CREATE INDEX idx_reverse_lookup_user ON tuple (store, object_type, relation, _user);

-- +goose Down
DROP INDEX idx_reverse_lookup_user;
```

Note: DM drops indexes with `DROP INDEX <name>` (no `ON table` clause).

- [ ] **Step 5: Write `004_add_authorization_model_serialized_protobuf.sql`**

```sql
-- +goose Up
ALTER TABLE authorization_model ADD serialized_protobuf BLOB;

-- +goose Down
ALTER TABLE authorization_model DROP serialized_protobuf;
```

Note: `LONGBLOB` → `BLOB` (DM has no `LONGBLOB` type).

- [ ] **Step 6: Write `005_add_conditions_to_tuples.sql`**

```sql
-- +goose Up
ALTER TABLE tuple ADD condition_name VARCHAR(256);
ALTER TABLE tuple ADD condition_context BLOB;
ALTER TABLE changelog ADD condition_name VARCHAR(256);
ALTER TABLE changelog ADD condition_context BLOB;

-- +goose Down
ALTER TABLE tuple DROP condition_name;
ALTER TABLE tuple DROP condition_context;
ALTER TABLE changelog DROP condition_name;
ALTER TABLE changelog DROP condition_context;
```

Note: Each `ADD` is its own `ALTER TABLE` statement — DM may not support multiple `ADD COLUMN` in one statement.

- [ ] **Step 7: Write `006_extend_object_id.sql`**

```sql
-- +goose Up
ALTER TABLE tuple MODIFY object_id VARCHAR(255);

-- +goose Down
ALTER TABLE tuple MODIFY object_id VARCHAR(128);
```

Note: DM uses `MODIFY col TYPE` (no `COLUMN` keyword after `MODIFY`).

- [ ] **Step 8: Write `007_collate_object_id.sql`**

```sql
-- +goose Up
CREATE INDEX idx_user_lookup ON tuple (store, _user, relation, object_type, object_id);
DROP INDEX idx_reverse_lookup_user;

-- +goose Down
DROP INDEX idx_user_lookup;
CREATE INDEX idx_reverse_lookup_user ON tuple (store, object_type, relation, _user);
```

Note: MySQL `LOCK=` hints and `COLLATE utf8mb4_bin` are omitted — DM does not support them.

- [ ] **Step 9: Commit**

```bash
git add assets/migrations/dm/
git commit -m "feat(dm): add migration SQL files"
```

---

## Task 3: Add DMMigrationDir to assets.go

**Files:**
- Modify: `assets/assets.go`

- [ ] **Step 1: Add the constant**

In `assets/assets.go`, after the `SqliteMigrationDir` line, add:

```go
DMMigrationDir       = "migrations/dm"
```

The file should look like:

```go
const (
	MySQLMigrationDir    = "migrations/mysql"
	PostgresMigrationDir = "migrations/postgres"
	SqliteMigrationDir   = "migrations/sqlite"
	DMMigrationDir       = "migrations/dm"
)
```

- [ ] **Step 2: Verify the embed picks up the new directory**

```bash
go build ./assets/...
```

Expected: no errors (the `//go:embed migrations/*` glob includes `migrations/dm/` automatically)

- [ ] **Step 3: Commit**

```bash
git add assets/assets.go
git commit -m "feat(dm): add DMMigrationDir constant to assets"
```

---

## Task 4: Write pkg/storage/dm/doc.go

**Files:**
- Create: `pkg/storage/dm/doc.go`

- [ ] **Step 1: Create the file**

```go
// Package dm contains an implementation of the storage interface that works with DaMeng (达梦) database.
package dm
```

- [ ] **Step 2: Verify**

```bash
go build ./pkg/storage/dm/...
```

Expected: no errors

---

## Task 5: Write the Datastore implementation

**Files:**
- Create: `pkg/storage/dm/dm.go`

This is the largest task. The implementation mirrors `pkg/storage/mysql/mysql.go` with these DM-specific differences:
1. **Credential override in `New()`**: the DM `parseDSN` does NOT URL-decode credentials — it uses raw strings. Therefore `net/url.Parse` + `url.UserPassword` must NOT be used (they URL-encode on `String()`). Instead, parse/reconstruct the URI manually with string operations.
2. `WriteAssertions` uses `MERGE INTO` instead of `ON DUPLICATE KEY UPDATE`
3. `ReadChanges` interval expression uses `DATEADD` instead of `NOW() - INTERVAL`
4. `HandleSQLError` maps DM error codes (not MySQL codes)

- [ ] **Step 1: Write the full dm.go**

Create `/Users/zhangguoqing/works/openfga/pkg/storage/dm/dm.go`:

```go
package dm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/cenkalti/backoff/v4"
	dmdriver "gitee.com/chunanyong/dm"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"

	"github.com/openfga/openfga/pkg/logger"
	"github.com/openfga/openfga/pkg/storage"
	"github.com/openfga/openfga/pkg/storage/sqlcommon"
	tupleUtils "github.com/openfga/openfga/pkg/tuple"
)

var tracer = otel.Tracer("openfga/pkg/storage/dm")

func startTrace(ctx context.Context, name string) (context.Context, trace.Span) {
	return tracer.Start(ctx, "dm."+name)
}

// Datastore provides a DaMeng based implementation of [storage.OpenFGADatastore].
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

// Ensures that Datastore implements the OpenFGADatastore interface.
var _ storage.OpenFGADatastore = (*Datastore)(nil)

// New creates a new [Datastore] storage.
func New(uri string, cfg *sqlcommon.Config) (*Datastore, error) {
	if cfg.Username != "" || cfg.Password != "" {
		// DM parseDSN uses raw strings without URL-decoding, so net/url must NOT
		// be used here (url.UserPassword would URL-encode the password on .String()).
		// Manual parse: dm://user:password@host:port
		rest := strings.TrimPrefix(uri, "dm://")
		atIdx := strings.LastIndex(rest, "@")
		var hostPart, username, password string
		if atIdx >= 0 {
			userPart := rest[:atIdx]
			hostPart = rest[atIdx+1:]
			if colonIdx := strings.Index(userPart, ":"); colonIdx >= 0 {
				username = userPart[:colonIdx]
				password = userPart[colonIdx+1:]
			} else {
				username = userPart
			}
		} else {
			hostPart = rest
		}
		if cfg.Username != "" {
			username = cfg.Username
		}
		if cfg.Password != "" {
			password = cfg.Password
		}
		uri = fmt.Sprintf("dm://%s:%s@%s", username, password, hostPart)
	}

	db, err := sql.Open("dm", uri)
	if err != nil {
		return nil, fmt.Errorf("initialize dm connection: %w", err)
	}
	return NewWithDB(db, cfg)
}

// NewWithDB creates a new [Datastore] storage with the provided database connection.
func NewWithDB(db *sql.DB, cfg *sqlcommon.Config) (*Datastore, error) {
	if cfg.MaxIdleConns != 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	policy := backoff.NewExponentialBackOff()
	policy.MaxElapsedTime = 1 * time.Minute
	attempt := 1
	err := backoff.Retry(func() error {
		err := db.PingContext(context.Background())
		if err != nil {
			cfg.Logger.Info("waiting for database", zap.Int("attempt", attempt))
			attempt++
			return err
		}
		return nil
	}, policy)
	if err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	var collector prometheus.Collector
	if cfg.ExportMetrics {
		collector = collectors.NewDBStatsCollector(db, "openfga")
		if err := prometheus.Register(collector); err != nil {
			return nil, fmt.Errorf("initialize metrics: %w", err)
		}
	}

	stbl := sq.StatementBuilder.RunWith(db)
	dbInfo := sqlcommon.NewDBInfo(stbl, HandleSQLError, "dm")

	return &Datastore{
		stbl:                   stbl,
		db:                     db,
		dbInfo:                 dbInfo,
		logger:                 cfg.Logger,
		dbStatsCollector:       collector,
		maxTuplesPerWriteField: cfg.MaxTuplesPerWriteField,
		maxTypesPerModelField:  cfg.MaxTypesPerModelField,
		versionReady:           false,
	}, nil
}

// Close see [storage.OpenFGADatastore].Close.
func (s *Datastore) Close() {
	if s.dbStatsCollector != nil {
		prometheus.Unregister(s.dbStatsCollector)
	}
	s.db.Close()
}

// Read see [storage.RelationshipTupleReader].Read.
func (s *Datastore) Read(
	ctx context.Context,
	store string,
	filter storage.ReadFilter,
	_ storage.ReadOptions,
) (storage.TupleIterator, error) {
	ctx, span := startTrace(ctx, "Read")
	defer span.End()
	return s.read(ctx, store, filter, nil)
}

// ReadPage see [storage.RelationshipTupleReader].ReadPage.
func (s *Datastore) ReadPage(ctx context.Context, store string, filter storage.ReadFilter, options storage.ReadPageOptions) ([]*openfgav1.Tuple, string, error) {
	ctx, span := startTrace(ctx, "ReadPage")
	defer span.End()
	iter, err := s.read(ctx, store, filter, &options)
	if err != nil {
		return nil, "", err
	}
	defer iter.Stop()
	return iter.ToArray(ctx, options.Pagination)
}

func (s *Datastore) read(ctx context.Context, store string, filter storage.ReadFilter, options *storage.ReadPageOptions) (*sqlcommon.SQLTupleIterator, error) {
	_, span := startTrace(ctx, "read")
	defer span.End()

	sb := s.stbl.
		Select(
			"store", "object_type", "object_id", "relation",
			"_user",
			"condition_name", "condition_context", "ulid", "inserted_at",
		).
		From("tuple").
		Where(sq.Eq{"store": store})
	if options != nil {
		sb = sb.OrderBy("ulid")
	}

	objectType, objectID := tupleUtils.SplitObject(filter.Object)
	if objectType != "" {
		sb = sb.Where(sq.Eq{"object_type": objectType})
	}
	if objectID != "" {
		sb = sb.Where(sq.Eq{"object_id": objectID})
	}
	if filter.Relation != "" {
		sb = sb.Where(sq.Eq{"relation": filter.Relation})
	}
	if filter.User != "" {
		userType, userID, _ := tupleUtils.ToUserParts(filter.User)
		if userID != "" {
			sb = sb.Where(sq.Eq{"_user": filter.User})
		} else {
			sb = sb.Where(sq.Like{"_user": userType + ":%"})
		}
	}
	if len(filter.Conditions) > 0 {
		sb = sb.Where(sq.Eq{"COALESCE(condition_name, '')": filter.Conditions})
	}
	if options != nil && options.Pagination.From != "" {
		sb = sb.Where(sq.GtOrEq{"ulid": options.Pagination.From})
	}
	if options != nil && options.Pagination.PageSize != 0 {
		sb = sb.Limit(uint64(options.Pagination.PageSize + 1))
	}

	return sqlcommon.NewSQLTupleIterator(sqlcommon.NewSBIteratorQuery(sb), HandleSQLError), nil
}

// Write see [storage.RelationshipTupleWriter].Write.
func (s *Datastore) Write(
	ctx context.Context,
	store string,
	deletes storage.Deletes,
	writes storage.Writes,
	opts ...storage.TupleWriteOption,
) error {
	ctx, span := startTrace(ctx, "Write")
	defer span.End()
	return sqlcommon.Write(ctx, s.dbInfo, s.db, store,
		sqlcommon.WriteData{
			Deletes: deletes,
			Writes:  writes,
			Opts:    storage.NewTupleWriteOptions(opts...),
			Now:     time.Now().UTC(),
		})
}

// ReadUserTuple see [storage.RelationshipTupleReader].ReadUserTuple.
func (s *Datastore) ReadUserTuple(ctx context.Context, store string, filter storage.ReadUserTupleFilter, _ storage.ReadUserTupleOptions) (*openfgav1.Tuple, error) {
	ctx, span := startTrace(ctx, "ReadUserTuple")
	defer span.End()

	objectType, objectID := tupleUtils.SplitObject(filter.Object)
	userType := tupleUtils.GetUserTypeFromUser(filter.User)

	var conditionName sql.NullString
	var conditionContext []byte
	var record storage.TupleRecord

	sb := s.stbl.
		Select(
			"object_type", "object_id", "relation",
			"_user",
			"condition_name", "condition_context",
		).
		From("tuple").
		Where(sq.Eq{
			"store":       store,
			"object_type": objectType,
			"object_id":   objectID,
			"relation":    filter.Relation,
			"_user":       filter.User,
			"user_type":   userType,
		})
	if len(filter.Conditions) > 0 {
		sb = sb.Where(sq.Eq{"COALESCE(condition_name, '')": filter.Conditions})
	}

	err := sb.QueryRowContext(ctx).
		Scan(
			&record.ObjectType,
			&record.ObjectID,
			&record.Relation,
			&record.User,
			&conditionName,
			&conditionContext,
		)
	if err != nil {
		return nil, HandleSQLError(err)
	}

	if conditionName.String != "" {
		record.ConditionName = conditionName.String
		if conditionContext != nil {
			var conditionContextStruct structpb.Struct
			if err := proto.Unmarshal(conditionContext, &conditionContextStruct); err != nil {
				return nil, err
			}
			record.ConditionContext = &conditionContextStruct
		}
	}

	return record.AsTuple(), nil
}

// ReadUsersetTuples see [storage.RelationshipTupleReader].ReadUsersetTuples.
func (s *Datastore) ReadUsersetTuples(
	ctx context.Context,
	store string,
	filter storage.ReadUsersetTuplesFilter,
	_ storage.ReadUsersetTuplesOptions,
) (storage.TupleIterator, error) {
	_, span := startTrace(ctx, "ReadUsersetTuples")
	defer span.End()

	sb := s.stbl.
		Select(
			"store", "object_type", "object_id", "relation",
			"_user",
			"condition_name", "condition_context", "ulid", "inserted_at",
		).
		From("tuple").
		Where(sq.Eq{"store": store}).
		Where(sq.Eq{"user_type": tupleUtils.UserSet})

	objectType, objectID := tupleUtils.SplitObject(filter.Object)
	if objectType != "" {
		sb = sb.Where(sq.Eq{"object_type": objectType})
	}
	if objectID != "" {
		sb = sb.Where(sq.Eq{"object_id": objectID})
	}
	if filter.Relation != "" {
		sb = sb.Where(sq.Eq{"relation": filter.Relation})
	}
	if len(filter.AllowedUserTypeRestrictions) > 0 {
		orConditions := sq.Or{}
		for _, userset := range filter.AllowedUserTypeRestrictions {
			if _, ok := userset.GetRelationOrWildcard().(*openfgav1.RelationReference_Relation); ok {
				orConditions = append(orConditions, sq.Like{
					"_user": userset.GetType() + ":%#" + userset.GetRelation(),
				})
			}
			if _, ok := userset.GetRelationOrWildcard().(*openfgav1.RelationReference_Wildcard); ok {
				orConditions = append(orConditions, sq.Eq{
					"_user": userset.GetType() + ":*",
				})
			}
		}
		sb = sb.Where(orConditions)
	}
	if len(filter.Conditions) > 0 {
		sb = sb.Where(sq.Eq{"COALESCE(condition_name, '')": filter.Conditions})
	}

	return sqlcommon.NewSQLTupleIterator(sqlcommon.NewSBIteratorQuery(sb), HandleSQLError), nil
}

// ReadStartingWithUser see [storage.RelationshipTupleReader].ReadStartingWithUser.
func (s *Datastore) ReadStartingWithUser(
	ctx context.Context,
	store string,
	filter storage.ReadStartingWithUserFilter,
	_ storage.ReadStartingWithUserOptions,
) (storage.TupleIterator, error) {
	_, span := startTrace(ctx, "ReadStartingWithUser")
	defer span.End()

	var targetUsersArg []string
	for _, u := range filter.UserFilter {
		targetUser := u.GetObject()
		if u.GetRelation() != "" {
			targetUser = strings.Join([]string{u.GetObject(), u.GetRelation()}, "#")
		}
		targetUsersArg = append(targetUsersArg, targetUser)
	}

	builder := s.stbl.
		Select(
			"store", "object_type", "object_id", "relation",
			"_user",
			"condition_name", "condition_context", "ulid", "inserted_at",
		).
		From("tuple").
		Where(sq.Eq{
			"store":       store,
			"object_type": filter.ObjectType,
			"relation":    filter.Relation,
			"_user":       targetUsersArg,
		}).OrderBy("object_id")

	if filter.ObjectIDs != nil && filter.ObjectIDs.Size() > 0 {
		builder = builder.Where(sq.Eq{"object_id": filter.ObjectIDs.Values()})
	}
	if len(filter.Conditions) > 0 {
		builder = builder.Where(sq.Eq{"COALESCE(condition_name, '')": filter.Conditions})
	}

	return sqlcommon.NewSQLTupleIterator(sqlcommon.NewSBIteratorQuery(builder), HandleSQLError), nil
}

// MaxTuplesPerWrite see [storage.RelationshipTupleWriter].MaxTuplesPerWrite.
func (s *Datastore) MaxTuplesPerWrite() int {
	return s.maxTuplesPerWriteField
}

// ReadAuthorizationModel see [storage.AuthorizationModelReadBackend].ReadAuthorizationModel.
func (s *Datastore) ReadAuthorizationModel(ctx context.Context, store string, modelID string) (*openfgav1.AuthorizationModel, error) {
	ctx, span := startTrace(ctx, "ReadAuthorizationModel")
	defer span.End()
	return sqlcommon.ReadAuthorizationModel(ctx, s.dbInfo, store, modelID)
}

// ReadAuthorizationModels see [storage.AuthorizationModelReadBackend].ReadAuthorizationModels.
func (s *Datastore) ReadAuthorizationModels(ctx context.Context, store string, options storage.ReadAuthorizationModelsOptions) ([]*openfgav1.AuthorizationModel, string, error) {
	ctx, span := startTrace(ctx, "ReadAuthorizationModels")
	defer span.End()

	sb := s.stbl.
		Select("authorization_model_id").
		Distinct().
		From("authorization_model").
		Where(sq.Eq{"store": store}).
		OrderBy("authorization_model_id desc")

	if options.Pagination.From != "" {
		sb = sb.Where(sq.LtOrEq{"authorization_model_id": options.Pagination.From})
	}
	if options.Pagination.PageSize > 0 {
		sb = sb.Limit(uint64(options.Pagination.PageSize + 1))
	}

	rows, err := sb.QueryContext(ctx)
	if err != nil {
		return nil, "", HandleSQLError(err)
	}
	defer rows.Close()

	var modelIDs []string
	var modelID string
	for rows.Next() {
		err = rows.Scan(&modelID)
		if err != nil {
			return nil, "", HandleSQLError(err)
		}
		modelIDs = append(modelIDs, modelID)
	}
	if err := rows.Err(); err != nil {
		return nil, "", HandleSQLError(err)
	}

	var token string
	numModelIDs := len(modelIDs)
	if len(modelIDs) > options.Pagination.PageSize {
		numModelIDs = options.Pagination.PageSize
		token = modelID
	}

	models := make([]*openfgav1.AuthorizationModel, 0, numModelIDs)
	for i := 0; i < numModelIDs; i++ {
		model, err := s.ReadAuthorizationModel(ctx, store, modelIDs[i])
		if err != nil {
			return nil, "", err
		}
		models = append(models, model)
	}

	return models, token, nil
}

// FindLatestAuthorizationModel see [storage.AuthorizationModelReadBackend].FindLatestAuthorizationModel.
func (s *Datastore) FindLatestAuthorizationModel(ctx context.Context, store string) (*openfgav1.AuthorizationModel, error) {
	ctx, span := startTrace(ctx, "FindLatestAuthorizationModel")
	defer span.End()
	return sqlcommon.FindLatestAuthorizationModel(ctx, s.dbInfo, store)
}

// MaxTypesPerAuthorizationModel see [storage.TypeDefinitionWriteBackend].MaxTypesPerAuthorizationModel.
func (s *Datastore) MaxTypesPerAuthorizationModel() int {
	return s.maxTypesPerModelField
}

// WriteAuthorizationModel see [storage.TypeDefinitionWriteBackend].WriteAuthorizationModel.
func (s *Datastore) WriteAuthorizationModel(ctx context.Context, store string, model *openfgav1.AuthorizationModel) error {
	ctx, span := startTrace(ctx, "WriteAuthorizationModel")
	defer span.End()
	return sqlcommon.WriteAuthorizationModel(ctx, s.dbInfo, store, model)
}

// CreateStore adds a new store to storage.
func (s *Datastore) CreateStore(ctx context.Context, store *openfgav1.Store) (*openfgav1.Store, error) {
	ctx, span := startTrace(ctx, "CreateStore")
	defer span.End()

	var id, name string
	var createdAt, updatedAt time.Time

	txn, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, HandleSQLError(err)
	}
	defer func() {
		_ = txn.Rollback()
	}()

	_, err = s.stbl.
		Insert("store").
		Columns("id", "name", "created_at", "updated_at").
		Values(store.GetId(), store.GetName(), sq.Expr("NOW()"), sq.Expr("NOW()")).
		RunWith(txn).
		ExecContext(ctx)
	if err != nil {
		return nil, HandleSQLError(err)
	}

	err = s.stbl.
		Select("id", "name", "created_at", "updated_at").
		From("store").
		Where(sq.Eq{"id": store.GetId()}).
		RunWith(txn).
		QueryRowContext(ctx).
		Scan(&id, &name, &createdAt, &updatedAt)
	if err != nil {
		return nil, HandleSQLError(err)
	}

	err = txn.Commit()
	if err != nil {
		return nil, HandleSQLError(err)
	}

	return &openfgav1.Store{
		Id:        id,
		Name:      name,
		CreatedAt: timestamppb.New(createdAt),
		UpdatedAt: timestamppb.New(updatedAt),
	}, nil
}

// GetStore retrieves the details of a specific store using its storeID.
func (s *Datastore) GetStore(ctx context.Context, id string) (*openfgav1.Store, error) {
	ctx, span := startTrace(ctx, "GetStore")
	defer span.End()

	row := s.stbl.
		Select("id", "name", "created_at", "updated_at").
		From("store").
		Where(sq.Eq{
			"id":         id,
			"deleted_at": nil,
		}).
		QueryRowContext(ctx)

	var storeID, name string
	var createdAt, updatedAt time.Time
	err := row.Scan(&storeID, &name, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, HandleSQLError(err)
	}

	return &openfgav1.Store{
		Id:        storeID,
		Name:      name,
		CreatedAt: timestamppb.New(createdAt),
		UpdatedAt: timestamppb.New(updatedAt),
	}, nil
}

// ListStores provides a paginated list of all stores present in the storage.
func (s *Datastore) ListStores(ctx context.Context, options storage.ListStoresOptions) ([]*openfgav1.Store, string, error) {
	ctx, span := startTrace(ctx, "ListStores")
	defer span.End()

	whereClause := sq.And{sq.Eq{"deleted_at": nil}}
	if len(options.IDs) > 0 {
		whereClause = append(whereClause, sq.Eq{"id": options.IDs})
	}
	if options.Name != "" {
		whereClause = append(whereClause, sq.Eq{"name": options.Name})
	}
	if options.Pagination.From != "" {
		whereClause = append(whereClause, sq.GtOrEq{"id": options.Pagination.From})
	}

	sb := s.stbl.
		Select("id", "name", "created_at", "updated_at").
		From("store").
		Where(whereClause).
		OrderBy("id")

	if options.Pagination.PageSize > 0 {
		sb = sb.Limit(uint64(options.Pagination.PageSize + 1))
	}

	rows, err := sb.QueryContext(ctx)
	if err != nil {
		return nil, "", HandleSQLError(err)
	}
	defer rows.Close()

	var stores []*openfgav1.Store
	var id string
	for rows.Next() {
		var name string
		var createdAt, updatedAt time.Time
		err := rows.Scan(&id, &name, &createdAt, &updatedAt)
		if err != nil {
			return nil, "", HandleSQLError(err)
		}
		stores = append(stores, &openfgav1.Store{
			Id:        id,
			Name:      name,
			CreatedAt: timestamppb.New(createdAt),
			UpdatedAt: timestamppb.New(updatedAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, "", HandleSQLError(err)
	}

	if len(stores) > options.Pagination.PageSize {
		return stores[:options.Pagination.PageSize], id, nil
	}
	return stores, "", nil
}

// DeleteStore removes a store from storage.
func (s *Datastore) DeleteStore(ctx context.Context, id string) error {
	ctx, span := startTrace(ctx, "DeleteStore")
	defer span.End()

	_, err := s.stbl.
		Update("store").
		Set("deleted_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		ExecContext(ctx)
	if err != nil {
		return HandleSQLError(err)
	}
	return nil
}

// WriteAssertions see [storage.AssertionsBackend].WriteAssertions.
// DaMeng does not support ON DUPLICATE KEY UPDATE; use MERGE INTO instead.
func (s *Datastore) WriteAssertions(ctx context.Context, store, modelID string, assertions []*openfgav1.Assertion) error {
	ctx, span := startTrace(ctx, "WriteAssertions")
	defer span.End()

	marshalledAssertions, err := proto.Marshal(&openfgav1.Assertions{Assertions: assertions})
	if err != nil {
		return err
	}

	mergeSQL := `MERGE INTO assertion t
USING (SELECT ? AS store, ? AS authorization_model_id, ? AS assertions FROM dual) s
ON (t.store = s.store AND t.authorization_model_id = s.authorization_model_id)
WHEN MATCHED THEN UPDATE SET t.assertions = s.assertions
WHEN NOT MATCHED THEN INSERT (store, authorization_model_id, assertions)
  VALUES (s.store, s.authorization_model_id, s.assertions)`

	_, err = s.db.ExecContext(ctx, mergeSQL, store, modelID, marshalledAssertions)
	if err != nil {
		return HandleSQLError(err)
	}
	return nil
}

// ReadAssertions see [storage.AssertionsBackend].ReadAssertions.
func (s *Datastore) ReadAssertions(ctx context.Context, store, modelID string) ([]*openfgav1.Assertion, error) {
	ctx, span := startTrace(ctx, "ReadAssertions")
	defer span.End()

	var marshalledAssertions []byte
	err := s.stbl.
		Select("assertions").
		From("assertion").
		Where(sq.Eq{
			"store":                  store,
			"authorization_model_id": modelID,
		}).
		QueryRowContext(ctx).
		Scan(&marshalledAssertions)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*openfgav1.Assertion{}, nil
		}
		return nil, HandleSQLError(err)
	}

	var assertions openfgav1.Assertions
	if err = proto.Unmarshal(marshalledAssertions, &assertions); err != nil {
		return nil, err
	}
	return assertions.GetAssertions(), nil
}

// ReadChanges see [storage.ChangelogBackend].ReadChanges.
func (s *Datastore) ReadChanges(ctx context.Context, store string, filter storage.ReadChangesFilter, options storage.ReadChangesOptions) ([]*openfgav1.TupleChange, string, error) {
	ctx, span := startTrace(ctx, "ReadChanges")
	defer span.End()

	objectTypeFilter := filter.ObjectType
	horizonOffset := filter.HorizonOffset

	orderBy := "ulid asc"
	if options.SortDesc {
		orderBy = "ulid desc"
	}

	// DaMeng uses DATEADD for interval arithmetic instead of MySQL's NOW() - INTERVAL x MICROSECOND.
	sb := s.stbl.
		Select(
			"ulid", "object_type", "object_id", "relation",
			"_user",
			"operation",
			"condition_name", "condition_context", "inserted_at",
		).
		From("changelog").
		Where(sq.Eq{"store": store}).
		Where(fmt.Sprintf("inserted_at <= DATEADD(MICROSECOND, -%d, NOW())", horizonOffset.Microseconds())).
		OrderBy(orderBy)

	if objectTypeFilter != "" {
		sb = sb.Where(sq.Eq{"object_type": objectTypeFilter})
	}
	if options.Pagination.From != "" {
		sb = sqlcommon.AddFromUlid(sb, options.Pagination.From, options.SortDesc)
	}
	if options.Pagination.PageSize > 0 {
		sb = sb.Limit(uint64(options.Pagination.PageSize))
	}

	rows, err := sb.QueryContext(ctx)
	if err != nil {
		return nil, "", HandleSQLError(err)
	}
	defer rows.Close()

	var changes []*openfgav1.TupleChange
	var ulid string
	for rows.Next() {
		var objectType, objectID, relation, user string
		var operation int
		var insertedAt time.Time
		var conditionName sql.NullString
		var conditionContext []byte

		err = rows.Scan(
			&ulid,
			&objectType,
			&objectID,
			&relation,
			&user,
			&operation,
			&conditionName,
			&conditionContext,
			&insertedAt,
		)
		if err != nil {
			return nil, "", HandleSQLError(err)
		}

		var conditionContextStruct structpb.Struct
		if conditionName.String != "" {
			if conditionContext != nil {
				if err := proto.Unmarshal(conditionContext, &conditionContextStruct); err != nil {
					return nil, "", err
				}
			}
		}

		tk := tupleUtils.NewTupleKeyWithCondition(
			tupleUtils.BuildObject(objectType, objectID),
			relation,
			user,
			conditionName.String,
			&conditionContextStruct,
		)

		changes = append(changes, &openfgav1.TupleChange{
			TupleKey:  tk,
			Operation: openfgav1.TupleOperation(operation),
			Timestamp: timestamppb.New(insertedAt.UTC()),
		})
	}

	if len(changes) == 0 {
		return nil, "", storage.ErrNotFound
	}
	return changes, ulid, nil
}

// IsReady see [sqlcommon.IsReady].
func (s *Datastore) IsReady(ctx context.Context) (storage.ReadinessStatus, error) {
	versionReady, err := sqlcommon.IsReady(ctx, s.versionReady, s.db)
	if err != nil {
		return versionReady, err
	}
	s.versionReady = versionReady.IsReady
	return versionReady, nil
}

// HandleSQLError processes a SQL error and converts it into a more specific error type.
func HandleSQLError(err error, args ...interface{}) error {
	if errors.Is(err, sql.ErrNoRows) {
		return storage.ErrNotFound
	}

	var dmErr *dmdriver.DMError
	if errors.As(err, &dmErr) && dmErr.ErrCode == -6602 {
		if len(args) > 0 {
			if tk, ok := args[0].(*openfgav1.TupleKey); ok {
				return storage.InvalidWriteInputError(tk, openfgav1.TupleOperation_TUPLE_OPERATION_WRITE)
			}
		}
		return storage.ErrCollision
	}

	return fmt.Errorf("sql error: %w", err)
}
```

- [ ] **Step 2: Build to catch import and type errors**

```bash
go build ./pkg/storage/dm/...
```

Expected: no errors. If the DM error type name or error code is wrong (the driver's exported type may differ from `dm.DMError`/`-6602`), you will see a compile error — adjust to match the driver's actual exported error struct. Run `go doc gitee.com/chunanyong/dm` to inspect exported types.

- [ ] **Step 3: Commit**

```bash
git add pkg/storage/dm/
git commit -m "feat(dm): implement DaMeng storage backend"
```

---

## Task 6: Wire DM into cmd/run/run.go

**Files:**
- Modify: `cmd/run/run.go`

- [ ] **Step 1: Add the import**

In `cmd/run/run.go`, find the existing storage imports block (around line 81-84):

```go
"github.com/openfga/openfga/pkg/storage/mysql"
"github.com/openfga/openfga/pkg/storage/postgres"
...
"github.com/openfga/openfga/pkg/storage/sqlite"
```

Add after `mysql`:

```go
"github.com/openfga/openfga/pkg/storage/dm"
```

- [ ] **Step 2: Add the "dm" case to the engine switch**

Find the switch at line ~502 that looks like:

```go
case "sqlite":
    datastore, err = sqlite.New(config.Datastore.URI, dsCfg)
    if err != nil {
        return nil, nil, fmt.Errorf("initialize sqlite datastore: %w", err)
    }
default:
```

Add before `default:`:

```go
case "dm":
    datastore, err = dm.New(config.Datastore.URI, dsCfg)
    if err != nil {
        return nil, nil, fmt.Errorf("initialize dm datastore: %w", err)
    }
```

- [ ] **Step 3: Build**

```bash
go build ./cmd/...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add cmd/run/run.go
git commit -m "feat(dm): wire DaMeng engine into server startup"
```

---

## Task 7: Wire DM into migrate.go

**Files:**
- Modify: `pkg/storage/migrate/migrate.go`

- [ ] **Step 1: Add the "dm" case**

Find the switch in `RunMigrations` that ends with `case "sqlite":` (around line 97). After that block, before `case "":`, add:

```go
case "dm":
    driver = "dm"
    migrationsPath = assets.DMMigrationDir

    // DM parseDSN uses raw strings without URL-decoding; use string ops, not net/url.
    if cfg.Username != "" || cfg.Password != "" {
        rest := strings.TrimPrefix(uri, "dm://")
        atIdx := strings.LastIndex(rest, "@")
        var hostPart, username, password string
        if atIdx >= 0 {
            userPart := rest[:atIdx]
            hostPart = rest[atIdx+1:]
            if colonIdx := strings.Index(userPart, ":"); colonIdx >= 0 {
                username = userPart[:colonIdx]
                password = userPart[colonIdx+1:]
            } else {
                username = userPart
            }
        } else {
            hostPart = rest
        }
        if cfg.Username != "" {
            username = cfg.Username
        }
        if cfg.Password != "" {
            password = cfg.Password
        }
        uri = fmt.Sprintf("dm://%s:%s@%s", username, password, hostPart)
    }
```

- [ ] **Step 2: Add the blank import for the DM driver**

The `migrate.go` file needs: (a) the DM driver registered, and (b) `"strings"` imported if not already present. Add the blank driver import:

```go
import (
    ...
    _ "gitee.com/chunanyong/dm"
    ...
)
```

- [ ] **Step 3: Build**

```bash
go build ./pkg/storage/migrate/...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add pkg/storage/migrate/migrate.go
git commit -m "feat(dm): wire DaMeng engine into migration command"
```

---

## Task 8: Write smoke-test integration test

**Files:**
- Create: `pkg/storage/dm/dm_test.go`

This test is skipped automatically if the env var `OPENFGA_DM_URI` is not set, so it never breaks CI.

- [ ] **Step 1: Write the test file**

```go
package dm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openfga/openfga/pkg/storage/sqlcommon"
	"github.com/openfga/openfga/pkg/storage/test"
)

func TestDMDatastore(t *testing.T) {
	uri := os.Getenv("OPENFGA_DM_URI")
	if uri == "" {
		t.Skip("OPENFGA_DM_URI not set — skipping DaMeng integration tests")
	}

	cfg := sqlcommon.NewConfig()
	ds, err := New(uri, cfg)
	require.NoError(t, err)
	defer ds.Close()

	test.RunAllTests(t, ds)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/storage/dm/...
```

Expected: no errors

- [ ] **Step 3: Run the test against your DM instance**

First run migrations:

```bash
go run ./cmd/openfga migrate \
  --datastore-engine dm \
  --datastore-uri "dm://SYSDBA:6o%2B%25s3z2NK7J@192.168.107.9:5236"
```

Expected: goose migrations run successfully, tables created.

Then run the integration test:

```bash
OPENFGA_DM_URI="dm://SYSDBA:6o%2B%25s3z2NK7J@192.168.107.9:5236" \
  go test ./pkg/storage/dm/... -v -count=1 -timeout 120s
```

Expected: all tests pass. If `DATEADD` or `MERGE INTO` syntax fails, see the troubleshooting notes below.

- [ ] **Step 4: Commit**

```bash
git add pkg/storage/dm/dm_test.go
git commit -m "test(dm): add DaMeng integration test (skips without OPENFGA_DM_URI)"
```

---

## Task 9: Smoke-test the running server

- [ ] **Step 1: Run migrations (if not done in Task 8)**

```bash
go run ./cmd/openfga migrate \
  --datastore-engine dm \
  --datastore-uri "dm://SYSDBA:6o%2B%25s3z2NK7J@192.168.107.9:5236"
```

- [ ] **Step 2: Start the server**

```bash
go run ./cmd/openfga run \
  --datastore-engine dm \
  --datastore-uri "dm://SYSDBA:6o%2B%25s3z2NK7J@192.168.107.9:5236"
```

Expected output includes: `using 'dm' storage engine`

- [ ] **Step 3: Verify readiness**

```bash
curl -s http://localhost:8080/healthz | jq .
```

Expected: `{"status":"SERVING"}`

---

## Viability Probe Results (2026-04-28)

Ran `cmd/dm-probe` against `192.168.107.9:5236` (DM V8):

| Check | Result |
|---|---|
| Ping | ✓ |
| Server version | `DM Database Server 64 V8` |
| `CREATE TABLE` (VARCHAR) | ✓ |
| `INSERT` with `?` placeholder | ✓ |
| `SELECT` | ✓ |
| `MERGE INTO ... FROM dual` (upsert) | ✓ |
| `DATEADD(MICROSECOND, -N, NOW())` | ✓ |

**Key findings that update the design:**
- Password must NOT be URL-encoded — DM `parseDSN` uses raw strings
- `CHAR(n)` fails for ASCII strings ≥ n/2 chars (server uses double-byte charset); use `VARCHAR(n)` throughout migrations
- `MERGE INTO ... FROM dual` and `DATEADD` work as specified

---

## Troubleshooting

**`DATEADD` not recognized:** DaMeng versions before 2.0 may use `TIMESTAMPADD(MICROSECOND, -N, NOW())` instead. Replace `DATEADD(MICROSECOND, -%d, NOW())` with `TIMESTAMPADD(MICROSECOND, -%d, NOW())` in `ReadChanges`.

**`MERGE INTO ... FROM dual` fails:** Some DM versions need `FROM SYS.DUAL` explicitly. Replace `FROM dual` with `FROM SYS.DUAL` in `WriteAssertions`.

**DM driver error type is not `dm.DMError`:** Run `go doc gitee.com/chunanyong/dm` and search for the exported error struct. Update `HandleSQLError` to match the actual type name and unique-constraint error code.

**`DROP INDEX idx_name` fails:** Some DM versions require `DROP INDEX table_name.idx_name`. Update the `DOWN` migrations accordingly.

**`ALTER TABLE ... DROP column` fails:** DaMeng may require `ALTER TABLE t DROP COLUMN col` (with the `COLUMN` keyword). Add `COLUMN` to the Down migrations for 005 and 006.
