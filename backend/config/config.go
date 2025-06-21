package config

import (
	"context"
	"log"
	"standardizer/consumer"
	"standardizer/global"

	"github.com/spf13/viper"
)

type Config struct {
	App struct {
		Name string
		Port string
	}
	Database struct {
		Dsn          string
		MaxIdleConns int
		MaxOpenConns int
	}
}

var AppConfig *Config

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Err reading config", err)
	}

	AppConfig = &Config{}

	if err := viper.Unmarshal(AppConfig); err != nil {
		log.Fatalf("Err unmarshalling config", err)
	}
	global.Ctx = context.Background()
	initDB()
	InitRedis()
	InitLLM()
	InitRabbitMQ("amqp://guest:guest@localhost:5672/")
	consumer.StartConsumer()
}
