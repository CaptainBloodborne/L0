package main

import (
	"L0/handlers"
	"L0/model"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/nats-io/stan.go"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var lock sync.Mutex
var ordersCache = make(map[string]model.Order)

func main() {
	// In-memory cache storage

	// Connection to postgres database
	var user, password, dbname, host string
	var port int
	user = os.Getenv("POSTGRES_USER")
	password = os.Getenv("POSTGRES_PASSWORD")
	dbname = os.Getenv("POSTGRES_DB")
	host = "db"
	port = 5432
	// Connection string
	psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	fmt.Println(psqlconn)
	// open database
	db, err := sql.Open("postgres", psqlconn)
	CheckError(err)
	// close database
	defer db.Close()
	// check db
	err = db.Ping()
	CheckError(err)

	fmt.Println("Connected!")

	// Connect to the NATS Streaming server
	conn, err := stan.Connect("test-cluster", "order-collector", stan.NatsURL("nats://nats:4222"))
	if err != nil {
		fmt.Println("Error connecting to NATS Streaming:", err)
		return
	}
	defer conn.Close()

	// Subscribe to the "task.create" subject
	_, err = conn.Subscribe("order.create", func(m *stan.Msg) {
		lock.Lock()
		var myOrder model.Order
		err = loadFromJSON(m.Data, &myOrder)
		fmt.Println(string(m.Data))

		fmt.Printf("Received order: %s\n", myOrder.OrderUid)

		// Persist the order in cache
		fmt.Printf("Persisting order in cache: %+v\n", myOrder)
		ordersCache[myOrder.OrderUid] = myOrder
		lock.Unlock()

		lock.Lock()
		// Persist the order in database
		result, err := handlers.WriteData(db, myOrder.OrderUid, string(m.Data))
		if err != nil {
			log.Println("Error occured while writing data to postgres!")
			log.Println(err)
		}
		fmt.Println("Data has been written to postgres successfully!")
		fmt.Println(result)
		defer lock.Unlock()
		err = m.Ack()
		if err != nil {
			fmt.Println("Message not acknowledged!")
			fmt.Println()
		}

	}, stan.StartAtTimeDelta(time.Minute), stan.DeliverAllAvailable(), stan.SetManualAckMode(), stan.AckWait(stan.DefaultAckWait))
	if err != nil {
		fmt.Println("Error subscribing to task.create:", err)
		return
	}
	fmt.Println("Waiting for orders!")

	// start http server
	http.HandleFunc("/getOrder", HandleGet)
	log.Fatal(http.ListenAndServe(":8080", nil))
	//select {}

	//__________________________________________________________________________________________________________________

	//arguments := os.Args
	//if len(arguments) == 1 {
	//	fmt.Println("Please provide a filename!")
	//	return
	//}
	//
	//filename := arguments[1]
	//
	//var myOrder model.Order
	//
	//err = loadFromJSON(filename, &myOrder)
	//jsonString, err := json.Marshal(myOrder)
	//if err == nil {
	//	fmt.Println(string(jsonString))
	//	fmt.Println()
	//	fmt.Println(myOrder)
	//} else {
	//	fmt.Println(err)
	//}
	//
	//// Write order info in cache
	//ordersCache[myOrder.OrderUid] = myOrder
	//
	//fileData, err := os.ReadFile(filename)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//var parsedData map[string]interface{}
	//err = json.Unmarshal(fileData, &parsedData)
	//if err != nil {
	//	log.Println(err)
	//}
	//
	//jsonString, err = json.Marshal(parsedData)
	//if err == nil {
	//	fmt.Println(string(jsonString))
	//} else {
	//	log.Println(err)
	//}
	//fmt.Println()
	//
	//// Add data to postgres
	//orderUid := parsedData["order_uid"]
	//result, err := db.Exec("INSERT INTO orders (order_uid, order_info) VALUES ($1, $2)", orderUid, string(jsonString))
	//if err != nil {
	//	log.Println(err)
	//}
	//fmt.Println("Query result is:", result)

}

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

func loadFromJSON(data []byte, key interface{}) error {
	//in, err := os.Open(filename)
	//if err != nil {
	//	return err
	//}
	reader := bytes.NewReader(data)
	decodeJSON := json.NewDecoder(reader)
	err := decodeJSON.Decode(key)
	if err != nil {
		return err
	}
	//in.Close()
	return nil
}

func HandleGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	orderUid := r.URL.Query().Get("order_uid")

	lock.Lock()
	data := ordersCache[orderUid]
	defer lock.Unlock()
	json.NewEncoder(w).Encode(data)
	fmt.Println("Send order: ", data.OrderUid)

}
