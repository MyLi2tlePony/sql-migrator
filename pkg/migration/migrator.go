package migration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MyLi2tlePony/sql-migrator/pkg/storage/entity"
	"github.com/MyLi2tlePony/sql-migrator/pkg/storage/postgres"
)

type Migration interface {
	Connect(context.Context) error
	Close(context.Context) error
	Create(name, up, down string)
	Up(context.Context) error
	Down(context.Context) error
	Redo(context.Context) error
	Status(context.Context) error
	DbVersion(context.Context) error
}

type Logger interface {
	Error(string)
	Info(string)
}

type migrator struct {
	logger Logger

	storage    postgres.Storage
	migrations []migration
}

var (
	ErrConnect       = errors.New("error connect")
	ErrClose         = errors.New("error close")
	ErrMigrationUp   = errors.New("error migration up")
	ErrMigrationDown = errors.New("error migration Down")
	ErrMigrationRedo = errors.New("error migration redo")
	ErrGetStatus     = errors.New("error db status")
	ErrGetVersion    = errors.New("error db version")

	ErrUnexpectedMigrationVersion = errors.New("unexpected migration version")
)

func New(connString string, logger Logger) Migration {
	return &migrator{
		storage:    postgres.New(connString),
		logger:     logger,
		migrations: make([]migration, 0),
	}
}

func (m *migrator) logError(method, err error) {
	m.logger.Error(method.Error())
	m.logger.Error(err.Error())
}

func (m *migrator) Connect(ctx context.Context) error {
	m.logger.Info("Db connect")

	err := m.storage.Connect(ctx)
	if err != nil {
		m.logError(ErrConnect, err)
		return err
	}

	return nil
}

func (m *migrator) Close(ctx context.Context) error {
	m.logger.Info("Db close")

	err := m.storage.Close(ctx)
	if err != nil {
		m.logError(ErrClose, err)
		return err
	}

	return nil
}

func (m *migrator) Create(name, up, down string) {
	m.migrations = append(m.migrations, migration{
		version: len(m.migrations) + 1,
		name:    name,
		up:      up,
		down:    down,
	})
}

func (m *migrator) Up(ctx context.Context) (err error) {
	m.logger.Info("Up migrations start")

	lastVersion := 0

	lastMigration, err := m.storage.SelectLastMigrationByStatus(ctx, postgres.StatusSuccess)
	if err == nil {
		lastVersion = lastMigration.GetVersion()
	} else if err != postgres.ErrMigrationNotFound {
		m.logError(ErrMigrationUp, err)
		return err
	}

	if lastMigration != nil && lastMigration.GetVersion()-1 > len(m.migrations) {
		m.logError(ErrMigrationUp, ErrUnexpectedMigrationVersion)
		return ErrUnexpectedMigrationVersion
	}

	for i := lastVersion; i < len(m.migrations); i++ {
		err = m.upMigration(ctx, &m.migrations[i], m.migrations[i].up)
		if err != nil {
			m.logError(ErrMigrationUp, err)
			return err
		}
	}

	m.logger.Info("Up migrations end")
	return nil
}

func (m *migrator) upMigration(ctx context.Context, migration entity.Migration, sql string) (err error) {
	migration.SetStatus(postgres.StatusProcess)
	migration.SetStatusChangeTime(time.Now())

	if err = m.storage.InsertMigration(ctx, migration); err != nil {
		return err
	}

	if err = m.storage.Migrate(ctx, sql); err != nil {
		migration.SetStatus(postgres.StatusError)
		migration.SetStatusChangeTime(time.Now())

		if errStatus := m.storage.InsertMigration(ctx, migration); errStatus != nil {
			return errStatus
		}

		return err
	}

	migration.SetStatus(postgres.StatusSuccess)
	migration.SetStatusChangeTime(time.Now())

	if err = m.storage.InsertMigration(ctx, migration); err != nil {
		return err
	}

	return nil
}

func (m *migrator) Down(ctx context.Context) (err error) {
	m.logger.Info("Down migration start")

	lastMigration, err := m.storage.SelectLastMigrationByStatus(ctx, postgres.StatusSuccess)
	if err != nil {
		m.logError(ErrMigrationDown, err)
		return err
	}

	if lastMigration != nil && lastMigration.GetVersion()-1 > len(m.migrations) {
		m.logError(ErrMigrationDown, ErrUnexpectedMigrationVersion)
		return ErrUnexpectedMigrationVersion
	}

	downMigrationIndex := lastMigration.GetVersion() - 1
	err = m.downMigration(ctx, &m.migrations[downMigrationIndex], m.migrations[downMigrationIndex].down)
	if err != nil {
		m.logError(ErrMigrationDown, err)
		return err
	}

	m.logger.Info("Down migration end")
	return nil
}

func (m *migrator) downMigration(ctx context.Context, migration entity.Migration, sql string) (err error) {
	migration.SetStatus(postgres.StatusCancellation)
	migration.SetStatusChangeTime(time.Now())

	if err = m.storage.InsertMigration(ctx, migration); err != nil {
		return err
	}

	if err = m.storage.Migrate(ctx, sql); err != nil {
		migration.SetStatus(postgres.StatusError)
		migration.SetStatusChangeTime(time.Now())

		if errStatus := m.storage.InsertMigration(ctx, migration); errStatus != nil {
			return errStatus
		}

		return err
	}

	migration.SetStatus(postgres.StatusCancel)
	migration.SetStatusChangeTime(time.Now())

	if err = m.storage.InsertMigration(ctx, migration); err != nil {
		return err
	}

	return nil
}

func (m *migrator) Redo(ctx context.Context) error {
	m.logger.Info("Redo migration start")

	err := m.Down(ctx)
	if err != nil {
		m.logError(ErrMigrationRedo, err)
		return err
	}

	lastVersion := 0

	lastMigration, err := m.storage.SelectLastMigrationByStatus(ctx, postgres.StatusSuccess)
	if err == nil {
		lastVersion = lastMigration.GetVersion()
	} else if err != postgres.ErrMigrationNotFound {
		m.logError(ErrMigrationRedo, err)
		return err
	}

	if lastMigration != nil && lastMigration.GetVersion()-1 > len(m.migrations) {
		m.logError(ErrMigrationRedo, ErrUnexpectedMigrationVersion)
		return ErrUnexpectedMigrationVersion
	}

	err = m.upMigration(ctx, &m.migrations[lastVersion], m.migrations[lastVersion].up)
	if err != nil {
		m.logError(ErrMigrationRedo, err)
		return err
	}

	m.logger.Info("Redo migration end")
	return nil
}

func (m *migrator) Status(ctx context.Context) error {
	migrations, err := m.storage.SelectMigrations(ctx)
	if err != nil {
		m.logError(ErrGetStatus, err)
		return err
	}
	m.logger.Info("._____________________._____________________._____________________.")
	m.logger.Info(fmt.Sprintf("| %-19s | %-19s | %-19s |", "????????????????", "????????????", "??????????"))

	for _, migr := range migrations {
		formatMigration := fmt.Sprintf("| %-19s | %-19s | %s |",
			migr.GetName(), migr.GetStatus(), migr.GetStatusChangeTime().Format("2006-01-02 15:04:05"))

		m.logger.Info(formatMigration)
	}

	m.logger.Info("|_____________________|_____________________|_____________________|")
	return nil
}

func (m *migrator) DbVersion(ctx context.Context) error {
	lastVersion := 0

	lastMigration, err := m.storage.SelectLastMigrationByStatus(ctx, postgres.StatusSuccess)
	if err == nil {
		lastVersion = lastMigration.GetVersion()
	} else if err != postgres.ErrMigrationNotFound {
		m.logError(ErrGetVersion, err)
		return err
	}

	m.logger.Info(fmt.Sprintf("Version: %d", lastVersion))
	return nil
}
