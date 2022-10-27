package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

type logg struct{}

func (l *logg) Error(msg string) {}

func (l *logg) Info(msg string) {}

func TestCreateMigrations(t *testing.T) {
	t.Run("necessary case", func(t *testing.T) {
		var err error

		dir, err := os.MkdirTemp("", "migrations")
		require.Nil(t, err)

		err = os.MkdirAll(dir, 0777)
		require.Nil(t, err)

		app := application{
			logger: &logg{},
		}

		file1 := "init"
		app.Create(file1, dir)

		file2 := "new"
		app.Create(file2, dir)

		file3 := "add"
		app.Create(file3, dir)

		files, err := os.ReadDir(dir)
		require.Nil(t, err)

		expectedFiles := []string{
			fmt.Sprintf("%05d_%s_up.sql", 1, file1),
			fmt.Sprintf("%05d_%s_down.sql", 1, file1),
			fmt.Sprintf("%05d_%s_up.sql", 2, file2),
			fmt.Sprintf("%05d_%s_down.sql", 2, file2),
			fmt.Sprintf("%05d_%s_up.sql", 3, file3),
			fmt.Sprintf("%05d_%s_down.sql", 3, file3),
		}

		for _, file := range files {
			contain := slices.Contains(expectedFiles, file.Name())
			require.True(t, contain)
		}

		require.Nil(t, os.RemoveAll(dir))
	})
}

func TestGetMigrations(t *testing.T) {
	t.Run("necessary case", func(t *testing.T) {
		var err error

		dir, err := os.MkdirTemp("", "migrations")
		require.Nil(t, err)

		err = os.MkdirAll(dir, 0777)
		require.Nil(t, err)

		fileName := "init"
		app := application{
			logger: &logg{},
		}
		app.Create(fileName, dir)

		up := "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
		upFile := fmt.Sprintf("%05d_%s_up.sql", 1, fileName)
		err = os.WriteFile(filepath.Join(dir, upFile), []byte(up), 0777)
		require.Nil(t, err)

		down := "DROP EXTENSION IF EXISTS \"uuid-ossp\";\n"
		dowFile := fmt.Sprintf("%05d_%s_down.sql", 1, fileName)
		err = os.WriteFile(filepath.Join(dir, dowFile), []byte(down), 0777)
		require.Nil(t, err)

		migrations, err := getMigrations(dir)
		require.Nil(t, err)

		m, ok := migrations[1]
		require.True(t, ok)

		require.Equal(t, up, m.Up)
		require.Equal(t, down, m.Down)
		require.Equal(t, 1, m.Version)
		require.Equal(t, fileName, m.Name)
	})
}
