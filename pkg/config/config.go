package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Postgres struct {
		Host     string
		Port     int
		User     string
		Password string
		DBName   string
		SSLMode  string
	}
	Redis struct {
		Host     string
		Port     int
		Password string
		DB       int
	}
	Server struct {
		Port int
	}
	Replication struct {
		Slot        string
		Publication string
	}
}

func (c *Config) GetPostgresDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Postgres.Host,
		c.Postgres.Port,
		c.Postgres.User,
		c.Postgres.Password,
		c.Postgres.DBName,
		c.Postgres.SSLMode,
	)
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./misc")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("SYNCER_POSTGRES_HOST", "localhost")
	viper.SetDefault("SYNCER_POSTGRES_PORT", 5432)
	viper.SetDefault("SYNCER_POSTGRES_USER", "postgres")
	viper.SetDefault("SYNCER_POSTGRES_PASSWORD", "postgres")
	viper.SetDefault("SYNCER_POSTGRES_DBNAME", "chat")
	viper.SetDefault("SYNCER_POSTGRES_SSLMODE", "disable")
	viper.SetDefault("SYNCER_REDIS_HOST", "localhost")
	viper.SetDefault("SYNCER_REDIS_PORT", 6379)
	viper.SetDefault("SYNCER_REDIS_PASSWORD", "")
	viper.SetDefault("SYNCER_REDIS_DB", 0)
	viper.SetDefault("SYNCER_SERVER_PORT", 50051)
	viper.SetDefault("SYNCER_REPLICATION_SLOT", "syncer_slot")
	viper.SetDefault("SYNCER_REPLICATION_PUBLICATION", "syncer_pub")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	config := &Config{}

	// Load PostgreSQL configuration
	config.Postgres.Host = viper.GetString("SYNCER_POSTGRES_HOST")
	config.Postgres.Port = viper.GetInt("SYNCER_POSTGRES_PORT")
	config.Postgres.User = viper.GetString("SYNCER_POSTGRES_USER")
	config.Postgres.Password = viper.GetString("SYNCER_POSTGRES_PASSWORD")
	config.Postgres.DBName = viper.GetString("SYNCER_POSTGRES_DBNAME")
	config.Postgres.SSLMode = viper.GetString("SYNCER_POSTGRES_SSLMODE")

	// Load Redis configuration
	config.Redis.Host = viper.GetString("SYNCER_REDIS_HOST")
	config.Redis.Port = viper.GetInt("SYNCER_REDIS_PORT")
	config.Redis.Password = viper.GetString("SYNCER_REDIS_PASSWORD")
	config.Redis.DB = viper.GetInt("SYNCER_REDIS_DB")

	// Load server configuration
	config.Server.Port = viper.GetInt("SYNCER_SERVER_PORT")

	// Load replication configuration
	config.Replication.Slot = viper.GetString("SYNCER_REPLICATION_SLOT")
	config.Replication.Publication = viper.GetString("SYNCER_REPLICATION_PUBLICATION")

	return config, nil
}

// GetDSN returns the PostgreSQL connection string
func (c *PostgresConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// GetAddr returns the Redis connection address
func (c *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
} 