package queue

import (
	"fmt"
	"time"

	"block-scanner/config"

	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queueName  string
}

var rabbitMQ *RabbitMQ

func InitRabbitMQ(cfg *config.Config) error {

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
		true,  // no-wait
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

func PublishTransactions(messages [][]byte) error {
	if rabbitMQ == nil || rabbitMQ.channel == nil {
		return fmt.Errorf("RabbitMQ not initialized")
	}

	for _, message := range messages {

		err := rabbitMQ.channel.Publish(
			"",                 // exchange
			rabbitMQ.queueName, // routing key
			false,              // mandatory
			false,              // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        message,
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
