package main

import (
	"benjaminarmijo3/eventdrivenrabbit/internal"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, err := internal.ConnectRabbitMQ("benja", "secret", "localhost:5671", "customers",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/ca_certificate.pem",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/client_benja_certificate.pem",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/client_benja_key.pem",
	)
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	client, err := internal.NewRabbitMQClient(conn)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	consumeConn, err := internal.ConnectRabbitMQ("benja", "secret", "localhost:5671", "customers",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/ca_certificate.pem",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/client_benja_certificate.pem",
		"/Users/benja/clases/go/rabbit/tls-gen/basic/result/client_benja_key.pem",
	)

	if err != nil {
		panic(err)
	}

	defer consumeConn.Close()

	consumeClient, err := internal.NewRabbitMQClient(consumeConn)
	if err != nil {
		panic(err)
	}
	defer consumeClient.Close()

	queue, err := consumeClient.CreateQueue("", true, true)
	if err != nil {
		panic(err)
	}

	if err := consumeClient.CreateBinding(queue.Name, queue.Name, "customer_callbacks"); err != nil {
		panic(err)
	}

	messageBus, err := consumeClient.Consume(queue.Name, "customer-api", true)
	if err != nil {
		panic(err)
	}

	go func() {
		for message := range messageBus {
			log.Printf("Message Callback %s", message.CorrelationId)
		}
	}()

	// if err := client.CreateQueue("customers_created", true, false); err != nil {
	// 		panic(err)
	// 	}

	// 	if err := client.CreateQueue("customers_test", false, true); err != nil {
	// 		panic(err)
	// 	}

	// 	if err := client.CreateBinding("customers_created", "customers.created.*", "customer_events"); err != nil {
	// 		panic(err)
	// 	}
	// 	if err := client.CreateBinding("customers_test", "customers.*", "customer_events"); err != nil {
	// 		panic(err)
	// 	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < 10; i++ {
		if err := client.Send(ctx, "customer_events", "customers.created.us", amqp091.Publishing{
			ContentType:   "text/plain",
			DeliveryMode:  amqp091.Persistent,
			ReplyTo:       queue.Name,
			CorrelationId: fmt.Sprintf("customer_created_%d", i),
			Body:          []byte(`A cool message between services`),
		}); err != nil {
			panic(err)
		}
	}

	// time.Sleep(10 * time.Second)
	log.Println(client)

	var blocking chan struct{}
	<-blocking
}
