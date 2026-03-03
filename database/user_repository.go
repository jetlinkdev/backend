package database

import (
	"database/sql"
	"fmt"
	"time"

	"jetlink/models"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user in the database
func (r *UserRepository) CreateUser(user *models.User) error {
	query := `
	INSERT INTO jetlink_users (
		id, email, display_name, photo_url, phone_number, role,
		vehicle_type, vehicle_plate, driver_rating, total_trips, is_verified,
		created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, FROM_UNIXTIME(?), FROM_UNIXTIME(?), NULL)
	`

	_, err := r.db.DB.Exec(query,
		user.ID,
		user.Email,
		user.DisplayName,
		user.PhotoURL,
		user.PhoneNumber,
		user.Role,
		user.VehicleType,
		user.VehiclePlate,
		user.DriverRating,
		user.TotalTrips,
		user.IsVerified,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}

	return nil
}

// GetUserByID retrieves a user by their Firebase UID
func (r *UserRepository) GetUserByID(id string) (*models.User, error) {
	query := `
	SELECT id, email, display_name, photo_url, phone_number, role,
		   vehicle_type, vehicle_plate, driver_rating, total_trips, is_verified,
		   UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at),
		   UNIX_TIMESTAMP(last_login_at), UNIX_TIMESTAMP(deleted_at)
	FROM jetlink_users WHERE id = ? AND deleted_at IS NULL
	`

	user := &models.User{}
	var lastLoginAt sql.NullInt64
	var deletedAt sql.NullInt64

	err := r.db.DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.PhotoURL,
		&user.PhoneNumber,
		&user.Role,
		&user.VehicleType,
		&user.VehiclePlate,
		&user.DriverRating,
		&user.TotalTrips,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
		&deletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // User not found
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Int64
	}

	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Int64
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	query := `
	SELECT id, email, display_name, photo_url, phone_number, role,
		   vehicle_type, vehicle_plate, driver_rating, total_trips, is_verified,
		   UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at),
		   UNIX_TIMESTAMP(last_login_at), UNIX_TIMESTAMP(deleted_at)
	FROM jetlink_users WHERE email = ? AND deleted_at IS NULL
	`

	user := &models.User{}
	var lastLoginAt sql.NullInt64
	var deletedAt sql.NullInt64

	err := r.db.DB.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.PhotoURL,
		&user.PhoneNumber,
		&user.Role,
		&user.VehicleType,
		&user.VehiclePlate,
		&user.DriverRating,
		&user.TotalTrips,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
		&deletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // User not found
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Int64
	}

	if deletedAt.Valid {
		user.DeletedAt = &deletedAt.Int64
	}

	return user, nil
}

// UpdateUser updates an existing user
func (r *UserRepository) UpdateUser(user *models.User) error {
	query := `
	UPDATE jetlink_users
	SET email = ?, display_name = ?, photo_url = ?, phone_number = ?,
		vehicle_type = ?, vehicle_plate = ?, driver_rating = ?, total_trips = ?,
		is_verified = ?, updated_at = FROM_UNIXTIME(?)
	WHERE id = ?
	`

	_, err := r.db.DB.Exec(query,
		user.Email,
		user.DisplayName,
		user.PhotoURL,
		user.PhoneNumber,
		user.VehicleType,
		user.VehiclePlate,
		user.DriverRating,
		user.TotalTrips,
		user.IsVerified,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %v", err)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *UserRepository) UpdateLastLogin(id string) error {
	query := `UPDATE jetlink_users SET last_login_at = NOW(), updated_at = NOW() WHERE id = ?`

	_, err := r.db.DB.Exec(query, id)

	if err != nil {
		return fmt.Errorf("failed to update last login: %v", err)
	}

	return nil
}

// RegisterDriver registers a new driver with vehicle information
func (r *UserRepository) RegisterDriver(user *models.User) error {
	query := `
	UPDATE jetlink_users
	SET role = 'driver',
		vehicle_type = ?,
		vehicle_plate = ?,
		is_verified = TRUE,
		updated_at = FROM_UNIXTIME(?)
	WHERE id = ?
	`

	_, err := r.db.DB.Exec(query,
		user.VehicleType,
		user.VehiclePlate,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to register driver: %v", err)
	}

	return nil
}

// IsDriverRegistered checks if a user is registered as a driver
func (r *UserRepository) IsDriverRegistered(id string) (bool, error) {
	query := `SELECT COUNT(*) FROM jetlink_users WHERE id = ? AND role = 'driver' AND deleted_at IS NULL`

	var count int
	err := r.db.QueryRow(query, id).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check driver registration: %v", err)
	}

	return count > 0, nil
}

// SoftDeleteUser soft deletes a user by setting deleted_at timestamp
func (r *UserRepository) SoftDeleteUser(id string) error {
	query := `UPDATE jetlink_users SET deleted_at = NOW(), updated_at = NOW() WHERE id = ?`

	_, err := r.db.DB.Exec(query, id)

	if err != nil {
		return fmt.Errorf("failed to soft delete user: %v", err)
	}

	return nil
}

// RestoreUser restores a soft deleted user by clearing deleted_at
func (r *UserRepository) RestoreUser(id string) error {
	query := `UPDATE jetlink_users SET deleted_at = NULL, updated_at = NOW() WHERE id = ?`

	_, err := r.db.DB.Exec(query, id)

	if err != nil {
		return fmt.Errorf("failed to restore user: %v", err)
	}

	return nil
}

// DeleteUser deletes a user from the database (hard delete - use with caution)
func (r *UserRepository) DeleteUser(id string) error {
	query := `DELETE FROM jetlink_users WHERE id = ?`

	_, err := r.db.DB.Exec(query, id)

	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return nil
}

// UpdateDriverRating updates a driver's average rating and total trips
func (r *UserRepository) UpdateDriverRating(driverID string, rating float64, totalTrips int) error {
	query := `
	UPDATE jetlink_users
	SET driver_rating = ?,
		total_trips = ?,
		updated_at = FROM_UNIXTIME(?)
	WHERE id = ?
	`

	_, err := r.db.Exec(query, rating, totalTrips, time.Now().Unix(), driverID)
	if err != nil {
		return fmt.Errorf("failed to update driver rating: %v", err)
	}

	return nil
}

// UpdateDriverStatus updates the driver's online status
// Note: This is a simple implementation. For production, consider:
// 1. Adding a 'driver_status' column to jetlink_users table
// 2. Using Redis for real-time status tracking
// 3. Adding status history tracking
func (r *UserRepository) UpdateDriverStatus(driverID string, status string) error {
	// For now, we'll just log the status update
	// In production, you should store this in database or Redis
	// This is a placeholder implementation
	_ = driverID
	_ = status
	return nil
}
