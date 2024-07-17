package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"block-scanner/config"
	"block-scanner/internal/models"

	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queueName  string
}

var rabbitMQ *RabbitMQ

func InitRabbitMQ(cfg *config.Config) error {
	fmt.Println("=========", cfg.RabbitMQ.URL)

	conn, err := amqp.DialConfig(cfg.RabbitMQ.URL, amqp.Config{
		Dial: amqp.DefaultDial(10 * time.Second),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	_, err = ch.QueueDeclare(
		cfg.RabbitMQ.QueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	rabbitMQ = &RabbitMQ{
		connection: conn,
		channel:    ch,
		queueName:  cfg.RabbitMQ.QueueName,
	}

	return nil
}

func PublishTransactions(transactions []*models.Transaction) error {
	if rabbitMQ == nil || rabbitMQ.channel == nil {
		return fmt.Errorf("RabbitMQ not initialized")
	}

	for _, tx := range transactions {
		body, err := json.Marshal(tx)
		if err != nil {
			return err
		}

		err = rabbitMQ.channel.Publish(
			"",                 // exchange
			rabbitMQ.queueName, // routing key
			false,              // mandatory
			false,              // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			})
		if err != nil {
			return fmt.Errorf("error publishing to queue %s: %v", rabbitMQ.queueName, err)
		}
	}
	return nil
}

func CloseRabbitMQ() {
	if rabbitMQ.channel != nil {
		rabbitMQ.channel.Close()
	}
	if rabbitMQ.connection != nil {
		rabbitMQ.connection.Close()
	}
}
