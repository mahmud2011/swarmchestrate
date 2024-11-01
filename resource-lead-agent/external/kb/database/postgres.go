package database

import (
	"fmt"
	"log"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/internal/config"
)

type PostgresDB struct {
	*gorm.DB
}

var (
	once     sync.Once
	instance *PostgresDB
)

func NewPostgresDB(conf *config.Config) *PostgresDB {
	once.Do(func() {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
			conf.PostgresDB.Host,
			conf.PostgresDB.User,
			conf.PostgresDB.Pass,
			conf.PostgresDB.DBName,
			conf.PostgresDB.Port,
			conf.PostgresDB.SSLMode,
			conf.PostgresDB.TimeZone,
		)

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			log.Fatalln(err)
		}

		instance = &PostgresDB{db}
	})

	return instance
}

func (p *PostgresDB) Get() *gorm.DB {
	return instance.DB
}
