package config

type PostgresDB struct {
	Host     string
	Port     int
	User     string
	Pass     string
	DBName   string
	SSLMode  string
	TimeZone string
}
