package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/abauruel/devfullcycle-imersao13/internal/infra/kafka"
	"github.com/abauruel/devfullcycle-imersao13/internal/market/dto"
	"github.com/abauruel/devfullcycle-imersao13/internal/market/entity"
	"github.com/abauruel/devfullcycle-imersao13/internal/market/transformer"
	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	ordersIn := make(chan *entity.Order)
	ordersOut := make(chan *entity.Order)

	wg := &sync.WaitGroup{}

	defer wg.Wait()
	kafkaMsgChan := make(chan *ckafka.Message)
	configMap := &ckafka.ConfigMap{
		"bootstrap.servers": "host.docker.internal:9094",
		"group.id":          "myGroup",
		"auto.offset.reset": "latest",
	}
	producer := kafka.NewKafkaProducer(configMap)
	kafka := kafka.NewConsumer(configMap, []string{"input"})

	go kafka.Consume(kafkaMsgChan) //T2

	// recebe do canal do kafka, joga no input, processa joga no output e depois publica
	book := entity.NewBook(ordersIn, ordersOut, wg)

	go book.Trade() //T3

	go func() {
		for msg := range kafkaMsgChan {
			wg.Add(1)
			fmt.Println(msg.Value)
			tradeInput := dto.TradeInput{}
			err := json.Unmarshal(msg.Value, &tradeInput)
			if err != nil {
				panic(err)
			}
			order := transformer.TransformInput(tradeInput)
			ordersIn <- order

		}
	}()

	for res := range ordersOut {
		output := transformer.TransformOutput(res)
		outputJson, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Println(err)
		}
		producer.Publish(outputJson, []byte("orders"), "output")
	}
}
