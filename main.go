package main

import (
	"context"
	"fmt"
	. "github.com/Shopify/sarama"
	"github.com/yogeshsr/kafka-protobuf-console-consumer/consumer"
	"github.com/yogeshsr/kafka-protobuf-console-consumer/protobuf_decoder"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
	"time"
)

var (
	brokerList               = kingpin.Flag("broker-list", "List of brokers to connect").Short('b').Default("localhost:9092").Strings()
	topic                    = kingpin.Flag("topic", "Topic name").Short('t').String()
	protoImportDirs          = kingpin.Flag("proto-dir", "foo/dir1 bar/dir2").Strings()
	protoFileNameWithMessage = kingpin.Flag("file", "Proto file name").String()
	messageName              = kingpin.Flag("message", "Proto message name").String()

	fromBeginning = kingpin.Flag("from-beginning", "Read from beginning").Bool()
	prettyJson    = kingpin.Flag("pretty", "Format output").Bool()
	withSeparator = kingpin.Flag("with-separator", "Adds separator between messages. Useful with --pretty").Bool()

)

func main() {

	kingpin.Parse()

	if len(*brokerList) == 0 || len(*topic) == 0 || len(*protoImportDirs) == 0 || len(*protoFileNameWithMessage) == 0 ||
		len(*messageName) == 0 {
		// TODO fix --help should work when Flags are marked Required, currently its supported by making Flags optional and checking this way
		fmt.Println("Missing required params; try --help")
		os.Exit(1)
	}

	// Init config, specify appropriate version
	config := NewConfig()
	config.Version = V1_0_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = offset()

	// Start with a client
	client, err := NewClient(*brokerList, config)
	if err != nil {
		panic(err)
	}
	defer func() { _ = client.Close() }()

	// Start a new consumer group
	consumerGroup := consumerGroup()
	fmt.Printf("Starting consumer group; %s\n\n", consumerGroup)

	group, err := NewConsumerGroupFromClient(consumerGroup, client)
	if err != nil {
		panic(err)
	}
	defer func() { _ = group.Close() }()

	// Track errors
	go func() {
		for err := range group.Errors() {
			fmt.Println("group error", err)
		}
	}()

	// Iterate over consumer sessions.
	ctx := context.Background()
	for {
		topics := []string{*topic}
		protobufJSONStringify := protobuf_decoder.NewProtobufJSONStringify(*protoImportDirs, *protoFileNameWithMessage, *messageName)

		handler := consumer.NewSimpleConsumerGroupHandler(
			protobufJSONStringify, *prettyJson, *fromBeginning, *withSeparator, )

		err := group.Consume(ctx, topics, handler)
		if err != nil {
			panic(err)
		}
	}
}

func consumerGroup() string {
	//TODO consumer group can also be read from cmd line
	return fmt.Sprintf("kafka-protobuf-console-consumer-%d", time.Now().UnixNano()/1000000)
}

func offset() int64 {
	if *fromBeginning {
		return OffsetOldest
	}
	return OffsetNewest
}