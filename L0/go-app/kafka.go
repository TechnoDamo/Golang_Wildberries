package main

import (
	"context"
	"encoding/json"
	"log"
	"time"
	"github.com/segmentio/kafka-go"
)

var (
    kafkaProducer *kafka.Writer
    kafkaReader   *kafka.Reader
)

func initKafka(brokers []string, topic string, groupID string) {
    kafkaProducer = kafka.NewWriter(kafka.WriterConfig{
        Brokers:  brokers,
        Balancer: &kafka.LeastBytes{},
        BatchTimeout: 10 * time.Millisecond,
        WriteTimeout: 10 * time.Second,
        RequiredAcks: 1, 
        Async:        false,
    })

    kafkaReader = kafka.NewReader(kafka.ReaderConfig{
        Brokers:  brokers,
        GroupID:  groupID,
        Topic:    topic,
        MinBytes: 10e3,  
        MaxBytes: 10e6,  
    })

    log.Println("Kafka producer and consumer initialized")
    
    testProducerConnection()
}

func testProducerConnection() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    testMsg := kafka.Message{
        Topic: "orders",
        Value: []byte("test connection"),
    }
    
    err := kafkaProducer.WriteMessages(ctx, testMsg)
    if err != nil {
        log.Printf("Producer connection test failed: %v", err)
    } else {
        log.Println("Producer connection test successful")
    }
}

func consumeOrders() {
	for {
		m, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Println("Kafka read error:", err)
			continue
		}

		var order Order
		if err := json.Unmarshal(m.Value, &order); err != nil {
			log.Println("Invalid order JSON:", err)
			continue
		}

		if err := insertOrderToDB(order); err != nil {
			log.Println("DB insert error:", err)
			continue
		}

		log.Printf("Order %s inserted into DB", order.OrderUID)
	}
}
