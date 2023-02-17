package main

import (
	"L0/model"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHandleRequest(t *testing.T) {
	// Place an example order in cache
	var myOrder model.Order

	err := loadFromJSON("model.json", &myOrder)
	jsonOrder, err := json.Marshal(myOrder)
	if err == nil {
		fmt.Println(string(jsonOrder))
	} else {
		fmt.Println(err)
	}

	ordersCache[myOrder.OrderUid] = myOrder
	// Set the POST form data
	body := strings.NewReader("id=b563feb7b2b84b6test")

	// Create a new request to the / endpoint
	req, err := http.NewRequest("POST", "/", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Create a new response recorder
	rr := httptest.NewRecorder()

	// Call the handleRequest function with the request and response recorder
	handleRequest(rr, req)

	// Check the status code is what we expect

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handleRequest returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !strings.Contains(rr.Body.String(), "OrderUid: b563feb7b2b84b6test") {

		t.Errorf("handleRequest returned unexpected response: can not find %v on page", myOrder.OrderUid)
	}
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
