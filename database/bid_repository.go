package database

import (
	"database/sql"
	"fmt"

	"jetlink/models"

	_ "github.com/go-sql-driver/mysql"
)

// BidRepository handles bid-related database operations
type BidRepository struct {
	db *DB
}

// NewBidRepository creates a new bid repository
func NewBidRepository(db *DB) *BidRepository {
	return &BidRepository{
		db: db,
	}
}

// CreateBid creates a new bid in the database
func (r *BidRepository) CreateBid(bid *models.Bid) error {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	INSERT INTO jetlink_bids (
		order_id, driver_id, bid_price, estimated_arrival_time, eta_minutes,
		status, message, created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, FROM_UNIXTIME(?), ?, ?, ?, FROM_UNIXTIME(?), FROM_UNIXTIME(?), NULL)
	`

	stmt, err := r.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		bid.OrderID,
		bid.DriverID,
		bid.BidPrice,
		bid.EstimatedArrivalTime,
		bid.ETAMinutes,
		bid.Status,
		bid.Message,
		bid.CreatedAt,
		bid.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert bid: %v", err)
	}

	// Get the auto-generated ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %v", err)
	}

	// Set the ID in the bid object
	bid.ID = id

	return nil
}

// GetBid retrieves a bid by ID from the database
func (r *BidRepository) GetBid(id int64) (*models.Bid, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	SELECT id, order_id, driver_id, bid_price, UNIX_TIMESTAMP(estimated_arrival_time),
		eta_minutes, status, message, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at),
		UNIX_TIMESTAMP(deleted_at)
	FROM jetlink_bids WHERE id = ? AND deleted_at IS NULL
	`

	row := r.db.QueryRow(query, id)

	var bid models.Bid
	var deletedAt sql.NullInt64
	err := row.Scan(
		&bid.ID,
		&bid.OrderID,
		&bid.DriverID,
		&bid.BidPrice,
		&bid.EstimatedArrivalTime,
		&bid.ETAMinutes,
		&bid.Status,
		&bid.Message,
		&bid.CreatedAt,
		&bid.UpdatedAt,
		&deletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bid with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get bid: %v", err)
	}

	if deletedAt.Valid {
		bid.DeletedAt = &deletedAt.Int64
	}

	return &bid, nil
}

// GetBidsByOrderID retrieves all bids for a specific order
func (r *BidRepository) GetBidsByOrderID(orderID int64) ([]*models.Bid, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	SELECT id, order_id, driver_id, bid_price, UNIX_TIMESTAMP(estimated_arrival_time),
		eta_minutes, status, message, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at),
		UNIX_TIMESTAMP(deleted_at)
	FROM jetlink_bids WHERE order_id = ? AND deleted_at IS NULL ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query bids: %v", err)
	}
	defer rows.Close()

	var bids []*models.Bid
	for rows.Next() {
		var bid models.Bid
		var deletedAt sql.NullInt64
		err := rows.Scan(
			&bid.ID,
			&bid.OrderID,
			&bid.DriverID,
			&bid.BidPrice,
			&bid.EstimatedArrivalTime,
			&bid.ETAMinutes,
			&bid.Status,
			&bid.Message,
			&bid.CreatedAt,
			&bid.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bid: %v", err)
		}
		if deletedAt.Valid {
			bid.DeletedAt = &deletedAt.Int64
		}
		bids = append(bids, &bid)
	}

	return bids, nil
}

// GetBidsByDriverID retrieves all bids for a specific driver
func (r *BidRepository) GetBidsByDriverID(driverID string) ([]*models.Bid, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	SELECT id, order_id, driver_id, bid_price, UNIX_TIMESTAMP(estimated_arrival_time),
		eta_minutes, status, message, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at),
		UNIX_TIMESTAMP(deleted_at)
	FROM jetlink_bids WHERE driver_id = ? AND deleted_at IS NULL ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to query bids: %v", err)
	}
	defer rows.Close()

	var bids []*models.Bid
	for rows.Next() {
		var bid models.Bid
		var deletedAt sql.NullInt64
		err := rows.Scan(
			&bid.ID,
			&bid.OrderID,
			&bid.DriverID,
			&bid.BidPrice,
			&bid.EstimatedArrivalTime,
			&bid.ETAMinutes,
			&bid.Status,
			&bid.Message,
			&bid.CreatedAt,
			&bid.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bid: %v", err)
		}
		if deletedAt.Valid {
			bid.DeletedAt = &deletedAt.Int64
		}
		bids = append(bids, &bid)
	}

	return bids, nil
}

// UpdateBidStatus updates the status of a bid
func (r *BidRepository) UpdateBidStatus(bidID int64, status string, message string) error {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	UPDATE jetlink_bids SET
		status = ?, message = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ? AND deleted_at IS NULL
	`

	stmt, err := r.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(status, message, bidID)
	if err != nil {
		return fmt.Errorf("failed to update bid status: %v", err)
	}

	return nil
}

// UpdateBid updates an existing bid in the database
func (r *BidRepository) UpdateBid(bid *models.Bid) error {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	UPDATE jetlink_bids SET
		order_id = ?, driver_id = ?, bid_price = ?, estimated_arrival_time = FROM_UNIXTIME(?),
		eta_minutes = ?, status = ?, message = ?, updated_at = CURRENT_TIMESTAMP
	WHERE id = ? AND deleted_at IS NULL
	`

	stmt, err := r.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		bid.OrderID,
		bid.DriverID,
		bid.BidPrice,
		bid.EstimatedArrivalTime,
		bid.ETAMinutes,
		bid.Status,
		bid.Message,
		bid.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update bid: %v", err)
	}

	return nil
}

// GetPendingBidsByOrderID retrieves all pending bids for a specific order
func (r *BidRepository) GetPendingBidsByOrderID(orderID int64) ([]*models.Bid, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `
	SELECT id, order_id, driver_id, bid_price, UNIX_TIMESTAMP(estimated_arrival_time),
		eta_minutes, status, message, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at),
		UNIX_TIMESTAMP(deleted_at)
	FROM jetlink_bids WHERE order_id = ? AND status = 'pending' AND deleted_at IS NULL ORDER BY bid_price ASC
	`

	rows, err := r.db.Query(query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query bids: %v", err)
	}
	defer rows.Close()

	var bids []*models.Bid
	for rows.Next() {
		var bid models.Bid
		var deletedAt sql.NullInt64
		err := rows.Scan(
			&bid.ID,
			&bid.OrderID,
			&bid.DriverID,
			&bid.BidPrice,
			&bid.EstimatedArrivalTime,
			&bid.ETAMinutes,
			&bid.Status,
			&bid.Message,
			&bid.CreatedAt,
			&bid.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bid: %v", err)
		}
		if deletedAt.Valid {
			bid.DeletedAt = &deletedAt.Int64
		}
		bids = append(bids, &bid)
	}

	return bids, nil
}

// HasDriverBidForOrder checks if a driver has already placed a bid on an order
func (r *BidRepository) HasDriverBidForOrder(driverID string, orderID int64) (bool, error) {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `SELECT COUNT(*) FROM jetlink_bids WHERE driver_id = ? AND order_id = ? AND deleted_at IS NULL`

	var count int
	err := r.db.QueryRow(query, driverID, orderID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check existing bid: %v", err)
	}

	return count > 0, nil
}

// SoftDeleteBid soft deletes a bid by setting deleted_at timestamp
func (r *BidRepository) SoftDeleteBid(id int64) error {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `UPDATE jetlink_bids SET deleted_at = NOW(), updated_at = NOW() WHERE id = ?`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete bid: %v", err)
	}

	return nil
}

// RestoreBid restores a soft deleted bid by clearing deleted_at
func (r *BidRepository) RestoreBid(id int64) error {
	r.db.mutex.Lock()
	defer r.db.mutex.Unlock()

	query := `UPDATE jetlink_bids SET deleted_at = NULL, updated_at = NOW() WHERE id = ?`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to restore bid: %v", err)
	}

	return nil
}
