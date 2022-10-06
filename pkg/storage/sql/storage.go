package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MyLi2tlePony/sql-migrator/pkg/storage/entity"
	"github.com/jackc/pgx/v4"
)

type Storage interface {
	SelectMigrations(context.Context) ([]entity.Migration, error)
	SelectLastMigrationByStatus(context.Context, string) (entity.Migration, error)
	Connect(context.Context) error
	Close(context.Context) error
	InsertMigration(context.Context, entity.Migration) error
	Migrate(context.Context, string) error
}

type sqlStorage struct {
	connString string
	conn       *pgx.Conn
}

const (
	StatusProcess      = "применение"
	StatusSuccess      = "применена"
	StatusError        = "ошибка"
	StatusCancellation = "отмена"
	StatusCancel       = "отменена"
)

var (
	ErrUnexpectedStatus  = errors.New("err unexpected status")
	ErrMigrationNotFound = errors.New("migration not found")
)

func New(connString string) Storage {
	return &sqlStorage{
		connString: connString,
	}
}

func (storage *sqlStorage) Connect(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, storage.connString)
	if err != nil {
		return err
	}

	sql := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			Version INTEGER,
			Name CHARACTER VARYING(100),
			Status CHARACTER VARYING(20),
			StatusChangeTime TIMESTAMP
		);`

	_, err = conn.Exec(ctx, sql)
	if err != nil {
		return err
	}

	storage.conn = conn
	return nil
}

func (storage *sqlStorage) Close(ctx context.Context) error {
	return storage.conn.Close(ctx)
}

func (storage *sqlStorage) SelectMigrations(ctx context.Context) (migrations []entity.Migration, err error) {
	sql := `SELECT Name, Status, Version, StatusChangeTime FROM schema_migrations ORDER BY Version DESC;`

	rows, err := storage.conn.Query(ctx, sql)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		var (
			name             string
			version          int
			status           string
			statusChangeTime time.Time
		)

		err = rows.Scan(&name, &status, &version, &statusChangeTime)
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, entity.NewMigration(name, status, version, statusChangeTime))
	} else {
		return nil, ErrMigrationNotFound
	}

	return migrations, nil
}

func (storage *sqlStorage) SelectLastMigrationByStatus(ctx context.Context, status string) (migration entity.Migration, err error) {
	switch status {
	case StatusSuccess:
	case StatusError:
	case StatusProcess:
	case StatusCancellation:
	case StatusCancel:
	default:
		return nil, ErrUnexpectedStatus
	}

	sql := `SELECT Name, Status, Version, StatusChangeTime FROM schema_migrations WHERE Status = $1 ORDER BY Version DESC LIMIT 1;`

	rows, err := storage.conn.Query(ctx, sql, status)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		var (
			name             string
			version          int
			status           string
			statusChangeTime time.Time
		)

		err = rows.Scan(&name, &status, &version, &statusChangeTime)
		if err != nil {
			return nil, err
		}

		migration = entity.NewMigration(name, status, version, statusChangeTime)
	} else {
		return nil, ErrMigrationNotFound
	}

	return migration, nil
}

func (storage *sqlStorage) InsertMigration(ctx context.Context, migration entity.Migration) (err error) {
	sql := fmt.Sprintf(`
		DO $$ BEGIN
			IF EXISTS (SELECT * FROM schema_migrations WHERE Version = %d AND Name = '%s') THEN
				UPDATE schema_migrations SET Status = '%s', StatusChangeTime = '%s' WHERE Version = %d AND Name = '%s';
			ELSE
				INSERT INTO schema_migrations (Version, Name, Status, StatusChangeTime)
				VALUES (%d, '%s', '%s', '%s');
			END IF;
		END $$;`,
		migration.GetVersion(), migration.GetName(),
		migration.GetStatus(), migration.GetStatusChangeTime().Format("2006-01-02 15:04:05"), migration.GetVersion(), migration.GetName(),
		migration.GetVersion(), migration.GetName(), migration.GetStatus(), migration.GetStatusChangeTime().Format("2006-01-02 15:04:05"))

	_, err = storage.conn.Exec(ctx, sql)
	if err != nil {
		return err
	}

	return nil
}

func (storage *sqlStorage) Migrate(ctx context.Context, sql string) (err error) {
	_, err = storage.conn.Exec(ctx, sql)
	return err
}
