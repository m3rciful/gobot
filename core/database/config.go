package database

// Config holds database connection settings shared across bots.
type Config struct {
	Host           string `yaml:"host" envconfig:"DB_HOST"`
	Port           string `yaml:"port" envconfig:"DB_PORT"`
	User           string `yaml:"user" envconfig:"DB_USER"`
	Password       string `yaml:"password" envconfig:"DB_PASSWORD"`
	Name           string `yaml:"name" envconfig:"DB_NAME"`
	SSLMode        string `yaml:"sslmode" envconfig:"DB_SSLMODE"`
	MaxConnections int    `yaml:"max_connections" envconfig:"DB_MAX_CONNECTIONS"`
}
