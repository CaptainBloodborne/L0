package handlers

import (
	"database/sql"
)

func WriteData(db *sql.DB, orderUid, order string) (result sql.Result, err error) {
	result, err = db.Exec("INSERT INTO orders (order_uid, order_info) VALUES ($1, $2)", orderUid, order)

	return result, err
}

func GetData(db *sql.DB) (rows *sql.Rows, err error) {
	rows, err = db.Query("SELECT order_uid, order_info FROM orders")

	return rows, err
}
