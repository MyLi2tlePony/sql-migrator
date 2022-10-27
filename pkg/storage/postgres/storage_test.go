//go:build integration
// +build integration

package postgres

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/MyLi2tlePony/sql-migrator/pkg/storage/entity"
	"github.com/stretchr/testify/require"
)

func TestMigrator(t *testing.T) {
	t.Run("necessary case", func(t *testing.T) {
		m := &sqlStorage{
			connString: "postgresql://postgres:1234512345@postgres:/postgres?sslmode=disable",
		}

		ctx := context.Background()

		require.Nil(t, m.Connect(ctx))

		migrations := []entity.Migration{
			entity.NewMigration("init", StatusSuccess, 1, time.Date(2000, 12, 10, 10, 0, 5, 0, time.UTC)),
			entity.NewMigration("add", StatusSuccess, 2, time.Date(2060, 12, 10, 10, 0, 5, 0, time.UTC)),
			entity.NewMigration("delete", StatusCancel, 3, time.Date(2100, 12, 10, 10, 0, 5, 0, time.UTC)),
		}

		for _, migration := range migrations {
			err := m.InsertMigration(ctx, migration)
			require.Nil(t, err)
		}

		selectMigrations, err := m.SelectMigrations(ctx)
		require.Nil(t, err)

		for _, selectMigration := range selectMigrations {
			require.True(t, Contains(migrations, selectMigration))
		}

		migration, err := m.SelectLastMigrationByStatus(ctx, StatusCancel)
		require.Nil(t, err)
		reflect.DeepEqual(migration, migrations[2])

		migration, err = m.SelectLastMigrationByStatus(ctx, StatusSuccess)
		require.Nil(t, err)
		reflect.DeepEqual(migration, migrations[1])

		migrations[2].SetStatus(StatusCancel)
		require.Nil(t, m.InsertMigration(ctx, migrations[2]))

		selectMigrations, err = m.SelectMigrations(ctx)
		require.Nil(t, err)

		for _, selectMigration := range selectMigrations {
			require.True(t, Contains(migrations, selectMigration))
		}

		require.Nil(t, m.DeleteMigrations(ctx))
		require.Nil(t, m.Close(ctx))
	})
}

func Contains(migrations []entity.Migration, migration entity.Migration) bool {
	for _, m := range migrations {
		if m.GetName() == migration.GetName() &&
			m.GetStatus() == migration.GetStatus() &&
			m.GetVersion() == migration.GetVersion() &&
			m.GetStatusChangeTime() == migration.GetStatusChangeTime() {
			return true
		}
	}

	return false
}
