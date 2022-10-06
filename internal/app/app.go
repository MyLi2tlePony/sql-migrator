package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/MyLi2tlePony/sql-migrator/pkg/migration"
)

type App interface {
	Create(name, path string)
	Up(path, connString string)
	Down(path, connString string)
	Redo(path, connString string)
	Status(connString string)
	DbVersion(connString string)
}

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

type application struct {
	logger Logger
}

type localMigration struct {
	Version int
	Name    string

	Up   string
	Down string
}

var (
	ErrInvalidMigrationName = errors.New("invalid migration name")

	regGetVersion       = regexp.MustCompile(`^\d+`)
	regGetUpMigration   = regexp.MustCompile(`^.+_up\.sql$`)
	regGetDownMigration = regexp.MustCompile(`^.+_down\.sql$`)
)

func New(logger Logger) App {
	return &application{
		logger: logger,
	}
}

func (app *application) Create(name, filePath string) {
	files, err := os.ReadDir(filePath)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	lastVersion := 0

	for _, file := range files {
		strVersion := regGetVersion.FindString(file.Name())

		if strVersion != "" {
			version, err := strconv.Atoi(strVersion)
			if err != nil {
				app.logger.Error(err.Error())
				return
			}

			if version > lastVersion {
				lastVersion = version
			}
		}
	}

	lastVersion++

	upFile := path.Join(filePath, fmt.Sprintf("%05d_%s_up.sql", lastVersion, name))
	err = os.WriteFile(upFile, nil, 0777)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	app.logger.Info(upFile + " created")

	downFile := path.Join(filePath, fmt.Sprintf("%05d_%s_down.sql", lastVersion, name))
	err = os.WriteFile(downFile, nil, 0777)
	if err != nil {
		return
	}
	app.logger.Info(downFile + " created")
}

func (app *application) Up(filePath, connString string) {
	migrator := migration.New(connString, app.logger)
	migrations, err := getMigrations(filePath)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	for i := 1; ; i++ {
		if _, ok := migrations[i]; !ok {
			break
		}

		migrator.Create(migrations[i].Name, migrations[i].Up, migrations[i].Down)
	}

	ctx := context.Background()
	if err = migrator.Connect(ctx); err != nil {
		return
	}

	if err = migrator.Up(ctx); err != nil {
		return
	}

	if err = migrator.Close(ctx); err != nil {
		return
	}
}

func (app *application) Down(filePath, connString string) {
	migrator := migration.New(connString, app.logger)
	ctx := context.Background()
	migrations, err := getMigrations(filePath)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	for i := 1; ; i++ {
		if _, ok := migrations[i]; !ok {
			break
		}

		migrator.Create(migrations[i].Name, migrations[i].Up, migrations[i].Down)
	}

	if err = migrator.Connect(ctx); err != nil {
		return
	}

	if err = migrator.Down(ctx); err != nil {
		return
	}

	if err = migrator.Close(ctx); err != nil {
		return
	}
}

func (app *application) Redo(filePath, connString string) {
	migrator := migration.New(connString, app.logger)
	ctx := context.Background()
	migrations, err := getMigrations(filePath)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	for i := 1; ; i++ {
		if _, ok := migrations[i]; !ok {
			break
		}

		migrator.Create(migrations[i].Name, migrations[i].Up, migrations[i].Down)
	}

	if err = migrator.Connect(ctx); err != nil {
		return
	}

	if err = migrator.Redo(ctx); err != nil {
		return
	}

	if err = migrator.Close(ctx); err != nil {
		return
	}
}

func (app *application) Status(connString string) {
	migrator := migration.New(connString, app.logger)
	ctx := context.Background()
	var err error

	if err = migrator.Connect(ctx); err != nil {
		return
	}

	if err = migrator.Status(ctx); err != nil {
		return
	}

	if err = migrator.Close(ctx); err != nil {
		return
	}
}

func (app *application) DbVersion(connString string) {
	migrator := migration.New(connString, app.logger)
	ctx := context.Background()
	var err error

	if err = migrator.Connect(ctx); err != nil {
		return
	}

	if err = migrator.DbVersion(ctx); err != nil {
		return
	}

	if err = migrator.Close(ctx); err != nil {
		return
	}
}

func getMigrations(filePath string) (map[int]*localMigration, error) {
	files, err := os.ReadDir(filePath)
	if err != nil {
		return nil, err
	}

	migrations := make(map[int]*localMigration)

	for _, file := range files {
		strVersion := regGetVersion.FindString(file.Name())

		if strVersion != "" {
			version, err := strconv.Atoi(strVersion)
			if err != nil {
				return nil, err
			}

			parts := strings.Split(file.Name(), "_")
			if len(parts) != 3 {
				return nil, ErrInvalidMigrationName
			}

			sql, err := os.ReadFile(path.Join(filePath, file.Name()))
			if err != nil {
				return nil, err
			}

			if regGetUpMigration.MatchString(file.Name()) {
				if _, ok := migrations[version]; ok {
					migrations[version].Up = string(sql)
				} else {
					migrations[version] = &localMigration{
						Version: version,
						Name:    parts[1],
						Up:      string(sql),
					}
				}
			} else if regGetDownMigration.MatchString(file.Name()) {
				if _, ok := migrations[version]; ok {
					migrations[version].Down = string(sql)
				} else {
					migrations[version] = &localMigration{
						Version: version,
						Name:    parts[1],
						Down:    string(sql),
					}
				}
			} else {
				return nil, ErrInvalidMigrationName
			}
		}
	}

	return migrations, nil
}
