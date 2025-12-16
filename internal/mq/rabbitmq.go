package mq

import (
	"fmt"
	"note/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// New 初始化 RabbitMQ 连接
func New(cfg *config.Config) (*RabbitMQ, error) {
	// 构造连接字符串: amqp://user:password@host:port/
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		cfg.MQUser,
		cfg.MQPassword,
		cfg.MQHost,
		cfg.MQPort,
	)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close() // 如果通道创建失败，记得关闭连接
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	return &RabbitMQ{
		conn:    conn,
		channel: ch,
	}, nil
}

// Close 关闭连接
func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
