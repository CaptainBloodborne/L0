package main

import (
	"L0/handlers"
	"L0/model"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/nats-io/stan.go"
	"html/template"
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

	// Restoring cache from postgress if exists
	rows, err := handlers.GetData(db)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Can not get data from postgres!")

	}
	defer rows.Close()

	for rows.Next() {
		var order model.Order
		var orderJson string
		var orderUid string
		err := rows.Scan(&orderUid, &orderJson)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Can not restore cache from postgres!")
		}
		err = json.Unmarshal([]byte(orderJson), &order)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(orderUid)
		fmt.Println(order)
		ordersCache[orderUid] = order
	}

	// Connect to the NATS Streaming server
	conn, err := stan.Connect("test-cluster", "order-collector", stan.NatsURL("nats://nats:4222"))
	if err != nil {
		fmt.Println("Error connecting to NATS Streaming:", err)
		return
	}
	defer conn.Close()

	// Subscribe to the "order.create" subject
	_, err = conn.Subscribe("order.create", func(m *stan.Msg) {
		lock.Lock()

		defer lock.Unlock()

		var myOrder model.Order
		err = json.Unmarshal(m.Data, &myOrder)
		//err = loadFromJSON(m.Data, &myOrder)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Can not validate message!")
		}
		fmt.Println(string(m.Data))

		fmt.Printf("Received order: %s\n", myOrder.OrderUid)

		// Persist the order in cache
		fmt.Printf("Persisting order in cache: %+v\n", myOrder)
		ordersCache[myOrder.OrderUid] = myOrder

		// Persist the order in database
		result, err := handlers.WriteData(db, myOrder.OrderUid, string(m.Data))
		if err != nil {
			log.Println("Error occured while writing data to postgres!")
			log.Println(err)
		} else {
			fmt.Println("Data has been written to postgres successfully!")
			fmt.Println(result)
		}

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

	http.HandleFunc("/", handleRequest)
	log.Fatal(http.ListenAndServe(":8080", nil))

}

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Printf("Host: %s Path: %s\n", r.Host, r.URL.Path)
		http.ServeFile(w, r, "templates/index.html")
	} else if r.Method == "POST" {
		fmt.Printf("Host: %s Path: %s\n", r.Host, r.URL.Path)
		uidOrder := r.FormValue("id")
		lock.Lock()

		defer lock.Unlock()
		if _, ok := ordersCache[uidOrder]; ok {

			orderTemplate := template.Must(template.ParseFiles("templates/order.gohtml"))
			err := orderTemplate.Execute(w, ordersCache[uidOrder])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

		} else {
			http.Error(w, fmt.Sprintf("No data found for ID: %s", uidOrder), http.StatusNotFound)
		}
	}
}
