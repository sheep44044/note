package mq

import (
	"context"
	"fmt"
	"note/config"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
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
		_ = conn.Close() // 如果通道创建失败，记得关闭连接
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	queues := []string{
		"favorite_queue",
		"react_queue",
		"history_queue",
		"feed_queue",
		"ai_queue",
	}

	// 遍历初始化，只要有一个失败，整个启动过程就应该失败
	for _, q := range queues {
		if err := initQueue(ch, q); err != nil {
			_ = ch.Close()   // 尽力清理
			_ = conn.Close() // 尽力清理
			return nil, fmt.Errorf("failed to init queue %s: %w", q, err)
		}
	}

	return &RabbitMQ{
		conn:    conn,
		channel: ch,
	}, nil
}

func initQueue(ch *amqp.Channel, queueName string) error {
	_, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable (持久化)
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	return err
}

// Close 关闭连接
func (r *RabbitMQ) Close() {
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			// 这种错误通常是因为连接已经关闭了，记录一下即可，不影响主流程
			zap.L().Warn("Failed to close rabbitmq channel", zap.Error(err))
		}
	}
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			zap.L().Warn("Failed to close rabbitmq connection", zap.Error(err))
		}
	}
}

// Publish 发送消息的通用方法
func (r *RabbitMQ) Publish(queueName string, body []byte) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := r.channel.PublishWithContext(ctx,
		"",        // exchange
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // 消息持久化
		})

	if err != nil {
		zap.L().Error("Failed to publish message", zap.String("queue", queueName), zap.Error(err))
		return err
	}

	return nil
}

// Consume 消费消息的通用方法 (返回一个只读通道)
func (r *RabbitMQ) Consume(queueName string) (<-chan amqp.Delivery, error) {
	msgs, err := r.channel.Consume(
		queueName, // queue
		"",        // consumer name (empty for auto-generated)
		true,      // auto-ack (自动确认，如果业务逻辑复杂建议改为 false 手动确认)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}
