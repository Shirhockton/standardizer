package config

import (
	"log/slog"
	"standardizer/global"

	"github.com/streadway/amqp"
)

func InitRabbitMQ(url string) {
	var err error
	global.RabbitMQConn, err = amqp.Dial(url)
	if err != nil {
		slog.Error("连接 RabbitMQ 失败", "error", err)
		return
	}
	slog.Info("成功连接 RabbitMQ")
}
