package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"note/config"
	"time"

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

	initQueue(ch, "favorite_queue")
	initQueue(ch, "react_queue")
	initQueue(ch, "history_queue")

	return &RabbitMQ{
		conn:    conn,
		channel: ch,
	}, nil
}

func initQueue(ch *amqp.Channel, queueName string) error {
	_, err := ch.QueueDeclare(
		"favorite_queue", // name
		true,             // durable (持久化)
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		slog.Error("Failed to declare a queue", "error", err)
		return err
	}
	return nil
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

// Publish 发送消息的通用方法
func (r *RabbitMQ) Publish(queueName string, body interface{}) error {
	// 将结构体序列化为 JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal msg: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = r.channel.PublishWithContext(ctx,
		"",        // exchange
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         jsonBody,
			DeliveryMode: amqp.Persistent, // 消息持久化
		})

	if err != nil {
		slog.Error("Failed to publish message", "queue", queueName, "error", err)
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
