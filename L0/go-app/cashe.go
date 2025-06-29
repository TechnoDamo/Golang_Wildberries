package main

import (
	"sync"
	"log"
)

const CacheSize = 100 // max number of items in cache

type OrderCache struct {
	data map[string]Order // the actual cache storage
	keys []string         // keeps track of insertion order
	lock sync.RWMutex     // to make it safe for concurrent use
}

// initialize the global cache
var cache = OrderCache{
	data: make(map[string]Order),
	keys: make([]string, 0, CacheSize),
}

func (c *OrderCache) Get(id string) (Order, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	order, found := c.data[id]
	return order, found
}

func (c *OrderCache) Add(id string, order Order) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// If already exists, no need to add again
	if _, exists := c.data[id]; exists {
		return
	}

	// If cache is full, evict oldest
	if len(c.keys) >= CacheSize {
		oldest := c.keys[0]
		delete(c.data, oldest)
		c.keys = c.keys[1:]
	}

	// Add new entry
	c.data[id] = order
	c.keys = append(c.keys, id)
}


func PreloadCache() {
	const preloadLimit = 100

	log.Println("Preloading cache with last", preloadLimit, "orders...")

	// First get N recent order IDs
	orderIDs := []string{}
	rows, err := db.Query(`SELECT id FROM task.orders ORDER BY created_at DESC LIMIT $1`, preloadLimit)
	if err != nil {
		log.Println("Failed to query order IDs:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			orderIDs = append(orderIDs, id)
		}
	}

	// Reuse your same logic for each order
	for _, orderID := range orderIDs {
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
			log.Println("Error loading order", orderID, ":", err)
			continue
		}

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
				log.Println("Error scanning order row for", orderID, ":", err)
				continue
			}
			items = append(items, item)
		}
		rows.Close()

		if found {
			order.Items = items
			cache.Add(orderID, order)
		}
	}

	log.Printf("Cache warm-up complete: %d orders loaded\n", len(cache.data))
}
