package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"context"
	_ "github.com/lib/pq" 
)

func main() {
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	brokers := strings.Split(kafkaBrokers, ",")
	
	topic := getEnv("KAFKA_TOPIC", "orders")
	groupID := getEnv("KAFKA_GROUP_ID", "order_consumers")

	log.Printf("Connecting to Kafka brokers: %v", brokers)
	initKafka(brokers, topic, groupID)

	go consumeOrders()
	
	InitDB()
	PreloadCache()

	r := mux.NewRouter()

	r.HandleFunc("/order", postOrder).Methods("POST")
	r.HandleFunc("/order/{id}", getOrder).Methods("GET")

	staticFileDir := http.Dir("./static/")
	staticFileHandler := http.StripPrefix("/", http.FileServer(staticFileDir))

	r.PathPrefix("/").Handler(staticFileHandler)

	port := getEnv("PORT", "8080")
	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func postOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	orderBytes, err := json.Marshal(order)
	if err != nil {
		http.Error(w, "Failed to serialize order", http.StatusInternalServerError)
		return
	}

	msg := kafka.Message{
		Topic: "orders",
		Value: orderBytes,
	}

	if err := kafkaProducer.WriteMessages(context.Background(), msg); err != nil {
		log.Printf("Kafka producer error: %v", err)
		http.Error(w, "Failed to send message to Kafka", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "queued"}`))
}

func insertOrderToDB(order Order) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO task.customers (id) VALUES ($1) ON CONFLICT DO NOTHING", order.CustomerID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO task.orders (id, customer_id, track_number, entry, created_at) VALUES ($1, $2, $3, $4, $5)`,
		order.OrderUID, order.CustomerID, order.TrackNumber, order.Entry, order.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO task.deliveries (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO task.payments (
		order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		_, err = tx.Exec(`INSERT INTO task.items (
			order_uid, chrt_id, track_number, price, name, sale, size, total_price, nm_id, brand, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Name, item.Sale,
			item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	cache.Add(order.OrderUID, order)
	return nil
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("GET /order/{id} hit")

	vars := mux.Vars(r)
	orderID := vars["id"]

	// Check cache
	if orderID, found := cache.Get(orderID); found {
		log.Println("Cache hit:", orderID)
		json.NewEncoder(w).Encode(orderID)
		return
	}
	log.Println("Cache miss:", orderID)

	query := `
SELECT 
    o.id AS order_uid, o.customer_id, o.track_number, o.entry, o.created_at,
    d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
    p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt, p.bank,
    p.delivery_cost, p.goods_total, p.custom_fee,
    i.chrt_id, i.track_number, i.price, i.name, i.sale, i.size, i.total_price,
    i.nm_id, i.brand, i.status
FROM task.orders o
LEFT JOIN task.deliveries d ON o.id = d.order_uid
LEFT JOIN task.payments p ON o.id = p.order_uid
LEFT JOIN task.items i ON o.id = i.order_uid
WHERE o.id = $1
`

	rows, err := db.Query(query, orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var order Order
	var items []Item
	found := false

	for rows.Next() {
		var item Item
		found = true
		err := rows.Scan(
			&order.OrderUID, &order.CustomerID, &order.TrackNumber, &order.Entry, &order.CreatedAt,
			&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
			&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
			&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
			&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDT,
			&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee,
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Name, &item.Sale,
			&item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	if !found {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	order.Items = items
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func postOrder1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("POST /order hit")

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

	_, err = tx.Exec(`INSERT INTO task.orders (id, customer_id, track_number, entry, created_at) VALUES ($1, $2, $3, $4, $5)`,
		order.OrderUID, order.CustomerID, order.TrackNumber, order.Entry, order.CreatedAt)
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

	cache.Add(order.OrderUID, order)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "ok"}`))
}
