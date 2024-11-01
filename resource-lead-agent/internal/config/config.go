package config

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App        *App
	HTTPServer *HTTPServer
	PostgresDB *PostgresDB
	Raft       *Raft
}

var (
	once     sync.Once
	instance *Config
)

func Get() *Config {
	once.Do(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/config")

		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			log.Fatalln(err)
		}

		instance = &Config{
			App: &App{
				Type:                        viper.GetString("app.type"),
				SchedulerInterval:           viper.GetDuration("app.schedulerinterval") * time.Second,
				ApplicationHeartbeatTimeout: viper.GetDuration("app.applicationheartbeattimeout") * time.Second,
				NodeHeartbeatTimeout:        viper.GetDuration("app.nodeheartbeattimeout") * time.Second,
			},
			HTTPServer: &HTTPServer{
				Host:                    viper.GetString("httpserver.host"),
				Port:                    viper.GetInt("httpserver.port"),
				GracefulShutdownTimeout: viper.GetDuration("httpserver.gracefulshutdowntimeout") * time.Second,
			},
			PostgresDB: &PostgresDB{
				Host:     viper.GetString("postgresdb.host"),
				Port:     viper.GetInt("postgresdb.port"),
				User:     viper.GetString("postgresdb.user"),
				Pass:     viper.GetString("postgresdb.pass"),
				DBName:   viper.GetString("postgresdb.dbname"),
				SSLMode:  viper.GetString("postgresdb.sslmode"),
				TimeZone: viper.GetString("postgresdb.timezone"),
			},
			Raft: &Raft{
				ID:    viper.GetInt("raft.id"),
				Peers: viper.GetStringSlice("raft.peers"),
			},
		}
	})

	return instance
}
