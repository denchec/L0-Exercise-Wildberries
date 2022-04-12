package main

import (
	"database/sql"
	"encoding/json"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"html/template"
	"net/http"
	"runtime"
)

type MainView struct {
	DataCache map[string]string
	dbDriver  *sql.DB
}

type ResultPost struct {
	order_uid        string
	rest_information string
}

type OrderData struct {
	OrderUid string `json:"order_uid"`
}

type JsonSupplier struct {
	ResponseJson string
}

func getView() MainView {
	return MainView{DataCache: loadFromPostgres(), dbDriver: getDB()}
}

func getDB() (db *sql.DB) {
	userLog := "user=postgres password=1234 dbname=natsDataBase sslmode=disable"
	db, err := sql.Open("postgres", userLog)
	checkErr(err)
	return
}

func loadFromPostgres() map[string]string {
	userLog := "user=postgres password=1234 dbname=natsDataBase sslmode=disable"
	db, err := sql.Open("postgres", userLog)
	checkErr(err)

	result, err := db.Query("select * from information_table")
	checkErr(err)

	end := map[string]string{}

	for result.Next() {
		var e ResultPost
		err := result.Scan(&e.order_uid, &e.rest_information)
		checkErr(err)

		end[e.order_uid] = e.rest_information
	}

	return end
}

func (view MainView) NatsSubscriptionHandler(msg *nats.Msg) { // Получение новых данных из Nats
	var data OrderData
	err := json.Unmarshal(msg.Data, &data)
	checkErr(err)
	view.DataCache[data.OrderUid] = string(msg.Data)

	_, dbErr := view.dbDriver.Exec("INSERT INTO information_table (order_uid, rest_information) VALUES ($1, $2)", data.OrderUid, string(msg.Data))
	checkErr(dbErr)
}

func (view MainView) queryHandleFunc(w http.ResponseWriter, r *http.Request) {

	tmpl := template.Must(template.ParseFiles("templates/RequestForm.html"))

	request := r.FormValue("request")

	a := JsonSupplier{}
	a.ResponseJson = view.DataCache[request]
	err := tmpl.Execute(w, a)
	checkErr(err)
}

func main() {
	mainView := getView()
	nc, err := nats.Connect(nats.DefaultURL, nats.Name("NatsConn")) // Подключение к серверу Nats
	checkErr(err)

	_, subscribeErr := nc.Subscribe("login", mainView.NatsSubscriptionHandler)
	checkErr(subscribeErr)

	http.HandleFunc("/", mainView.queryHandleFunc)

	listenErr := http.ListenAndServe(":5001", nil) // Подключение к сайту 5001
	checkErr(listenErr)

	runtime.Goexit()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
