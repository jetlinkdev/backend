package database

import (
	"database/sql"
	"fmt"

	"jetlink/models"

	_ "github.com/go-sql-driver/mysql"
)

// OrderRepository handles order-related database operations
type OrderRepository struct {
	db *DB
}

// GetDB returns the database instance
func (r *OrderRepository) GetDB() *DB {
	return r.db
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *DB) *OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

// CreateOrder creates a new order in the database
func (r *OrderRepository) CreateOrder(order *models.Order) error {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	INSERT INTO jetlink_orders (
		user_id, driver_id, pickup, pickup_latitude, pickup_longitude,
		destination, destination_latitude, destination_longitude,
		notes, time, payment, status, fare, bid_price, estimated_arrival_time,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, FROM_UNIXTIME(?), FROM_UNIXTIME(?))
	`

	stmt, err := r.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	var timeValue *int64
	if order.Time != nil {
		timeValue = order.Time
	} else {
		timeValue = nil
	}

	var etaValue *int64
	if order.EstimatedArrivalTime != nil {
		etaValue = order.EstimatedArrivalTime
	} else {
		etaValue = nil
	}

	result, err := stmt.Exec(
		order.UserID,
		order.DriverID,
		order.Pickup,
		order.PickupLatitude,
		order.PickupLongitude,
		order.Destination,
		order.DestinationLatitude,
		order.DestinationLongitude,
		order.Notes,
		timeValue,
		order.Payment,
		order.Status,
		order.Fare,
		order.BidPrice,
		etaValue,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert order: %v", err)
	}

	// Get the auto-generated ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %v", err)
	}

	// Set the ID in the order object
	order.ID = id

	return nil
}

// GetOrder retrieves an order by ID from the database
func (r *OrderRepository) GetOrder(id int64) (*models.Order, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `SELECT id, user_id, driver_id, pickup, pickup_latitude, pickup_longitude, destination, destination_latitude, destination_longitude, notes, time, payment, status, fare, bid_price, UNIX_TIMESTAMP(estimated_arrival_time), UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM jetlink_orders WHERE id = ?`

	row := r.db.QueryRow(query, id)

	var order models.Order
	var timeValue *int64
	var etaValue *int64
	err := row.Scan(
		&order.ID,
		&order.UserID,
		&order.DriverID,
		&order.Pickup,
		&order.PickupLatitude,
		&order.PickupLongitude,
		&order.Destination,
		&order.DestinationLatitude,
		&order.DestinationLongitude,
		&order.Notes,
		&timeValue,
		&order.Payment,
		&order.Status,
		&order.Fare,
		&order.BidPrice,
		&etaValue,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get order: %v", err)
	}

	// Handle nullable time field
	if timeValue != nil {
		order.Time = timeValue
	}

	// Handle nullable ETA field
	if etaValue != nil {
		order.EstimatedArrivalTime = etaValue
	}

	return &order, nil
}

// UpdateOrder updates an existing order in the database
func (r *OrderRepository) UpdateOrder(order *models.Order) error {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	UPDATE jetlink_orders SET
		user_id = ?, driver_id = ?, pickup = ?, pickup_latitude = ?, pickup_longitude = ?,
		destination = ?, destination_latitude = ?, destination_longitude = ?,
		notes = ?, time = ?, payment = ?, status = ?, fare = ?, bid_price = ?, estimated_arrival_time = FROM_UNIXTIME(?),
		updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`

	stmt, err := r.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	var timeValue *int64
	if order.Time != nil {
		timeValue = order.Time
	} else {
		timeValue = nil
	}

	var etaValue *int64
	if order.EstimatedArrivalTime != nil {
		etaValue = order.EstimatedArrivalTime
	} else {
		etaValue = nil
	}

	_, err = stmt.Exec(
		order.UserID,
		order.DriverID,
		order.Pickup,
		order.PickupLatitude,
		order.PickupLongitude,
		order.Destination,
		order.DestinationLatitude,
		order.DestinationLongitude,
		order.Notes,
		timeValue,
		order.Payment,
		order.Status,
		order.Fare,
		order.BidPrice,
		etaValue,
		order.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update order: %v", err)
	}

	return nil
}

// GetOrdersByUserID retrieves all orders for a specific user
func (r *OrderRepository) GetOrdersByUserID(userID string) ([]*models.Order, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `SELECT id, user_id, driver_id, pickup, pickup_latitude, pickup_longitude, destination, destination_latitude, destination_longitude, notes, time, payment, status, fare, bid_price, UNIX_TIMESTAMP(estimated_arrival_time), UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM jetlink_orders WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %v", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		var timeValue *int64
		var etaValue *int64
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.DriverID,
			&order.Pickup,
			&order.PickupLatitude,
			&order.PickupLongitude,
			&order.Destination,
			&order.DestinationLatitude,
			&order.DestinationLongitude,
			&order.Notes,
			&timeValue,
			&order.Payment,
			&order.Status,
			&order.Fare,
			&order.BidPrice,
			&etaValue,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %v", err)
		}

		// Handle nullable time field
		if timeValue != nil {
			order.Time = timeValue
		}

		// Handle nullable ETA field
		if etaValue != nil {
			order.EstimatedArrivalTime = etaValue
		}

		orders = append(orders, &order)
	}

	return orders, nil
}

// GetOrdersByStatus retrieves all orders with a specific status
func (r *OrderRepository) GetOrdersByStatus(status string) ([]*models.Order, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `SELECT id, user_id, driver_id, pickup, pickup_latitude, pickup_longitude, destination, destination_latitude, destination_longitude, notes, time, payment, status, fare, bid_price, UNIX_TIMESTAMP(estimated_arrival_time), UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM jetlink_orders WHERE status = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %v", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		var timeValue *int64
		var etaValue *int64
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.DriverID,
			&order.Pickup,
			&order.PickupLatitude,
			&order.PickupLongitude,
			&order.Destination,
			&order.DestinationLatitude,
			&order.DestinationLongitude,
			&order.Notes,
			&timeValue,
			&order.Payment,
			&order.Status,
			&order.Fare,
			&order.BidPrice,
			&etaValue,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %v", err)
		}

		// Handle nullable time field
		if timeValue != nil {
			order.Time = timeValue
		}

		// Handle nullable ETA field
		if etaValue != nil {
			order.EstimatedArrivalTime = etaValue
		}

		orders = append(orders, &order)
	}

	return orders, nil
}

// GetAllOrders retrieves all orders from the database
func (r *OrderRepository) GetAllOrders() ([]*models.Order, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `SELECT id, user_id, driver_id, pickup, pickup_latitude, pickup_longitude, destination, destination_latitude, destination_longitude, notes, time, payment, status, fare, bid_price, UNIX_TIMESTAMP(estimated_arrival_time), UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM jetlink_orders ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %v", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		var timeValue *int64
		var etaValue *int64
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.DriverID,
			&order.Pickup,
			&order.PickupLatitude,
			&order.PickupLongitude,
			&order.Destination,
			&order.DestinationLatitude,
			&order.DestinationLongitude,
			&order.Notes,
			&timeValue,
			&order.Payment,
			&order.Status,
			&order.Fare,
			&order.BidPrice,
			&etaValue,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %v", err)
		}

		// Handle nullable time field
		if timeValue != nil {
			order.Time = timeValue
		}

		// Handle nullable ETA field
		if etaValue != nil {
			order.EstimatedArrivalTime = etaValue
		}

		orders = append(orders, &order)
	}

	return orders, nil
}