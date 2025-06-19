package global

import (
	"context"
	"standardizer/models"

	"github.com/go-redis/redis"
	"github.com/streadway/amqp"
	"github.com/tmc/langchaingo/llms/ollama"
	"gorm.io/gorm"
)

var (
	Db           *gorm.DB
	RedisDB      *redis.Client
	LLM          *ollama.LLM
	CodeAnalyzer *models.CodeAnalyzer
	Ctx          context.Context
	RabbitMQConn *amqp.Connection
)
