package main

import (
	"L0/model"
	"encoding/json"
	"fmt"
	"github.com/nats-io/stan.go"
	"os"
)

func main() {
	// Connect to the NATS Streaming server
	conn, err := stan.Connect("test-cluster", "order-publisher", stan.NatsURL("nats://localhost:4222"))
	if err != nil {
		fmt.Println("Error connecting to NATS Streaming:", err)
		return
	}
	defer conn.Close()

	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide a filename!")
		return
	}

	filename := arguments[1]

	var myOrder model.Order

	err = loadFromJSON(filename, &myOrder)
	jsonOrder, err := json.Marshal(myOrder)
	if err == nil {
		fmt.Println(string(jsonOrder))
		fmt.Println()
		fmt.Println(myOrder)
	} else {
		fmt.Println(err)
	}

	// Publish a sample task to the "task.create" subject
	//task := Task{
	//	ID:        "1",
	//	Title:     "Example order",
	//	Completed: false,
	//	CreatedAt: time.Now(),
	//}
	//
	//taskData, err := json.Marshal(task)
	//if err != nil {
	//	fmt.Println("Error serializing task:", err)
	//	return
	//}
	err = conn.Publish("order.create", jsonOrder)
	if err != nil {
		fmt.Println("Error publishing task create event:", err)
		return
	}

	fmt.Println("Task successfully published to channel!")

	// Wait for the task to be processed
	//fmt.Scanln()

}

func loadFromJSON(filename string, key interface{}) error {
	in, err := os.Open(filename)
	if err != nil {
		return err
	}

	decodeJSON := json.NewDecoder(in)
	err = decodeJSON.Decode(key)
	if err != nil {
		return err
	}
	in.Close()
	return nil
}
