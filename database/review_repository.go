package database

import (
	"database/sql"
	"fmt"
	"jetlink/models"
	"time"
)

// ReviewRepository handles review database operations
type ReviewRepository struct {
	db *DB
}

// NewReviewRepository creates a new ReviewRepository instance
func NewReviewRepository(db *DB) *ReviewRepository {
	return &ReviewRepository{
		db: db,
	}
}

// CreateReview creates a new review in the database
func (repo *ReviewRepository) CreateReview(review *models.Review) error {
	query := `
		INSERT INTO jetlink_reviews (order_id, user_id, driver_id, rating, review, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := repo.db.Exec(query,
		review.OrderID,
		review.UserID,
		review.DriverID,
		review.Rating,
		review.Review,
		time.Now().Unix(),
		time.Now().Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to create review: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %v", err)
	}

	review.ID = id
	return nil
}

// GetReviewByOrderID retrieves a review by order ID
func (repo *ReviewRepository) GetReviewByOrderID(orderID int64) (*models.Review, error) {
	query := `
		SELECT id, order_id, user_id, driver_id, rating, review, created_at, updated_at
		FROM jetlink_reviews
		WHERE order_id = ?
	`

	review := &models.Review{}
	err := repo.db.QueryRow(query, orderID).Scan(
		&review.ID,
		&review.OrderID,
		&review.UserID,
		&review.DriverID,
		&review.Rating,
		&review.Review,
		&review.CreatedAt,
		&review.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Review not found
		}
		return nil, fmt.Errorf("failed to get review: %v", err)
	}

	return review, nil
}

// GetReviewsByDriverID retrieves all reviews for a driver
func (repo *ReviewRepository) GetReviewsByDriverID(driverID string, limit int) ([]*models.Review, error) {
	query := `
		SELECT id, order_id, user_id, driver_id, rating, review, created_at, updated_at
		FROM jetlink_reviews
		WHERE driver_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := repo.db.Query(query, driverID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviews: %v", err)
	}
	defer rows.Close()

	var reviews []*models.Review
	for rows.Next() {
		review := &models.Review{}
		err := rows.Scan(
			&review.ID,
			&review.OrderID,
			&review.UserID,
			&review.DriverID,
			&review.Rating,
			&review.Review,
			&review.CreatedAt,
			&review.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review: %v", err)
		}
		reviews = append(reviews, review)
	}

	return reviews, nil
}

// GetDriverAverageRating calculates the average rating for a driver
func (repo *ReviewRepository) GetDriverAverageRating(driverID string) (float64, error) {
	query := `
		SELECT AVG(rating) as avg_rating
		FROM jetlink_reviews
		WHERE driver_id = ?
	`

	var avgRating sql.NullFloat64
	err := repo.db.QueryRow(query, driverID).Scan(&avgRating)
	if err != nil {
		return 0, fmt.Errorf("failed to get average rating: %v", err)
	}

	if !avgRating.Valid {
		return 0, nil
	}

	return avgRating.Float64, nil
}

// GetDriverTotalReviews counts the total number of reviews for a driver
func (repo *ReviewRepository) GetDriverTotalReviews(driverID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM jetlink_reviews
		WHERE driver_id = ?
	`

	var count int
	err := repo.db.QueryRow(query, driverID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count reviews: %v", err)
	}

	return count, nil
}

// HasReviewedOrder checks if a user has already reviewed an order
func (repo *ReviewRepository) HasReviewedOrder(orderID int64) (bool, error) {
	query := `
		SELECT EXISTS(SELECT 1 FROM jetlink_reviews WHERE order_id = ?)
	`

	var exists bool
	err := repo.db.QueryRow(query, orderID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check review existence: %v", err)
	}

	return exists, nil
}
