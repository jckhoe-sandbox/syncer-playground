package main

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	LOG_LEVEL_FATAL   = "FATAL"
	LOG_LEVEL_ERROR   = "ERROR"
	LOG_LEVEL_WARNING = "WARNING"
	LOG_LEVEL_INFO    = "INFO"
	LOG_LEVEL_DEBUG   = "DEBUG"
	LOG_LEVEL_TRACE   = "TRACE"
)

type Config struct {
	Loglevel string
	Http     HttpConfig
	Db       DbConfig
	Redis    RedisConfig
}

type HttpConfig struct {
	Address string
	Port    uint
}

type DbConfig struct {
	Hostname string
	Username string
	Password string
	Port     int
	DbName   string
}

type RedisConfig struct {
	Hostname string
	Password string
	Port     int
}

var config Config

func InitConfig() {
	var err error

	viper.SetConfigName("application")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Panic(err)
	}

	err = viper.Unmarshal(&config)
	if err != nil {
		log.Panic(err)
	}

	var logLevel log.Level
	switch config.Loglevel {
	case LOG_LEVEL_FATAL:
		logLevel = log.InfoLevel
	case LOG_LEVEL_ERROR:
		logLevel = log.ErrorLevel
	case LOG_LEVEL_WARNING:
		logLevel = log.WarnLevel
	case LOG_LEVEL_INFO:
		logLevel = log.InfoLevel
	case LOG_LEVEL_DEBUG:
		logLevel = log.DebugLevel
	case LOG_LEVEL_TRACE:
		logLevel = log.TraceLevel
	default:
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	log.Infof("Config is %+v\n", config)
}
