package testutils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	TestDBHost = "localhost"
	TestDBName = "test"
	TestDBUser = "test"
	TestDBPass = "test"
)

func SetupTestDatabase() (*gorm.DB, testcontainers.Container, error) {
	ctx := context.Background()

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:17.2-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       TestDBName,
				"POSTGRES_USER":     TestDBUser,
				"POSTGRES_PASSWORD": TestDBPass,
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	}

	container, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	p, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, container, err
	}

	pgAddr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		TestDBUser,
		TestDBPass,
		TestDBHost,
		p.Port(),
		TestDBName,
	)
	db, err := gorm.Open(postgres.Open(pgAddr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, container, err
	}

	log.Println("SUCCESS: database ping")

	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return nil, container, err
	}

	migrationPath := filepath.Dir(path) + "/../../../knowledge-base/migrations"

	m, err := migrate.New(fmt.Sprintf("file:%s", migrationPath), pgAddr)
	if err != nil {
		return nil, container, err
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, container, err
	}

	log.Println("SUCCESS: migration")

	return db, container, nil
}
