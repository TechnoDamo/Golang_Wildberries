package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	InitDB()
	r := mux.NewRouter()
	r.HandleFunc("/order", postOrder).Methods("POST")
	//r.HandleFunc("/order/{id}", handleOrder).Methods("GET")
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func postOrder(w http.ResponseWriter, r *http.Request) {
	log.Println("POST /order hit")
	
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		log.Fatal(err)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO task.customers (id) VAlUES ($1) ON CONFLICT DO NOTHING", order.CustomerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_, err = tx.Exec(`INSERT INTO task.orders (id, customer_id, track_number, entry) VALUES ($1, $2, $3, $4)`,
		order.OrderUID, order.CustomerID, order.TrackNumber, order.Entry)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_, err = tx.Exec(`INSERT INTO task.deliveries (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_, err = tx.Exec(`INSERT INTO task.payments (
		order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	for _, item := range order.Items {
		_, err = tx.Exec(`INSERT INTO task.items (
			order_uid, chrt_id, track_number, price, name, sale, size, total_price, nm_id, brand, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Name, item.Sale,
			item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Transaction failed", 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "ok"}`))
}

/*
func getOrder( w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	var order Order

	err := db.QueryRow(`SELECT id, customer_id, track_number, entry FROM task.orders WHERE id=$1`, orderID).Scan(
		&order.OrderUID, &order.CustomerID, &order.TrackNumber, &order.Entry)
	if err != nil {
		http.Error(w, "Order not found", 404)
		return
	}

	err = db.QueryRow(`SELECT name, phone, zip, city, address, region, email FROM task.deliveries WHERE order_uid=$1`, orderID).
		Scan(&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
			&order.Delivery.City, &order.Delivery.Address,
			&order.Delivery.Region, &order.Delivery.Email)
	if err != nil {
		http.Error(w, "Delivery not found", 404)
		return
	}
}
*/
