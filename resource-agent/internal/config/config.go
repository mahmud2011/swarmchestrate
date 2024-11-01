package config

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App         *App
	HTTPServer  *HTTPServer
	RegisterSvc *RegisterSvc
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
				Type:              viper.GetString("app.type"),
				HeartBeatInterval: viper.GetDuration("app.heartbeatinterval") * time.Second,
				DeploymentTimeout: viper.GetDuration("app.deploymenttimeout") * time.Second,
			},
			HTTPServer: &HTTPServer{
				Host:                    viper.GetString("httpserver.host"),
				Port:                    viper.GetInt("httpserver.port"),
				GracefulShutdownTimeout: viper.GetDuration("httpserver.gracefulshutdowntimeout") * time.Second,
			},
			RegisterSvc: &RegisterSvc{
				Host:    viper.GetString("registersvc.host"),
				Port:    viper.GetInt("registersvc.port"),
				Timeout: viper.GetDuration("registersvc.timeout") * time.Second,
			},
		}
	})

	return instance
}
