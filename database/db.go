package database

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

// DB holds the database connection
type DB struct {
	*sql.DB
	mutex sync.Mutex
}

// GlobalDB is the global database instance
var GlobalDB *DB

// InitDB initializes the database connection
func InitDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	database := &DB{DB: db}

	// Create tables
	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	GlobalDB = database
	return database, nil
}

// createTables creates the necessary tables
func (db *DB) createTables() error {
	// Create the users table
	query := `
	CREATE TABLE IF NOT EXISTS jetlink_users (
		id VARCHAR(255) PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		display_name VARCHAR(255),
		photo_url TEXT,
		phone_number VARCHAR(50),
		role ENUM('customer', 'driver') NOT NULL DEFAULT 'customer',
		vehicle_type VARCHAR(100),
		vehicle_plate VARCHAR(20),
		driver_rating DECIMAL(3,2) DEFAULT 0.00,
		total_trips INT DEFAULT 0,
		is_verified BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP NULL,
		last_login_at TIMESTAMP NULL,
		INDEX idx_jetlink_users_email (email),
		INDEX idx_jetlink_users_role (role),
		INDEX idx_jetlink_users_created_at (created_at)
	);
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	// Create the orders table
	query = `
	CREATE TABLE IF NOT EXISTS jetlink_orders (
		id INT AUTO_INCREMENT PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		driver_id VARCHAR(255),
		pickup TEXT NOT NULL,
		pickup_latitude DECIMAL(10, 8),
		pickup_longitude DECIMAL(11, 8),
		destination TEXT NOT NULL,
		destination_latitude DECIMAL(10, 8),
		destination_longitude DECIMAL(11, 8),
		notes TEXT,
		time TIMESTAMP NULL,
		payment VARCHAR(50),
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		fare DECIMAL(10, 2),
		bid_price DECIMAL(10, 2),
		estimated_arrival_time TIMESTAMP NULL,
		route_coordinates TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP NULL,
		FOREIGN KEY (user_id) REFERENCES jetlink_users(id) ON DELETE RESTRICT,
		FOREIGN KEY (driver_id) REFERENCES jetlink_users(id) ON DELETE SET NULL,
		INDEX idx_jetlink_orders_status (status),
		INDEX idx_jetlink_orders_user_id (user_id)
	);
	`

	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create orders table: %v", err)
	}

	// Create the bids table
	query = `
	CREATE TABLE IF NOT EXISTS jetlink_bids (
		id INT AUTO_INCREMENT PRIMARY KEY,
		order_id INT NOT NULL,
		driver_id VARCHAR(255) NOT NULL,
		bid_price DECIMAL(10, 2) NOT NULL,
		estimated_arrival_time TIMESTAMP NOT NULL,
		eta_minutes INT NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		message TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP NULL,
		FOREIGN KEY (order_id) REFERENCES jetlink_orders(id) ON DELETE RESTRICT,
		FOREIGN KEY (driver_id) REFERENCES jetlink_users(id) ON DELETE RESTRICT,
		INDEX idx_jetlink_bids_order_id (order_id),
		INDEX idx_jetlink_bids_driver_id (driver_id),
		INDEX idx_jetlink_bids_status (status)
	);
	`

	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create bids table: %v", err)
	}

	// Create the reviews table
	query = `
	CREATE TABLE IF NOT EXISTS jetlink_reviews (
		id INT AUTO_INCREMENT PRIMARY KEY,
		order_id INT NOT NULL,
		user_id VARCHAR(255) NOT NULL,
		driver_id VARCHAR(255) NOT NULL,
		rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
		review TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP NULL,
		FOREIGN KEY (order_id) REFERENCES jetlink_orders(id) ON DELETE RESTRICT,
		FOREIGN KEY (user_id) REFERENCES jetlink_users(id) ON DELETE RESTRICT,
		FOREIGN KEY (driver_id) REFERENCES jetlink_users(id) ON DELETE RESTRICT,
		UNIQUE KEY unique_order_review (order_id),
		INDEX idx_jetlink_reviews_driver_id (driver_id),
		INDEX idx_jetlink_reviews_user_id (user_id),
		INDEX idx_jetlink_reviews_rating (rating)
	);
	`

	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create reviews table: %v", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}