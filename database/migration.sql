-- Migration script to update jetlink_orders table
-- ===========================================
-- IMPORTANT: Back up your data before running this!
-- ===========================================

-- Option 1: Drop and recreate tables (DATA WILL BE LOST)
-- Uncomment the lines below, run the script, then restart the server

DROP TABLE IF EXISTS jetlink_bids;
DROP TABLE IF EXISTS jetlink_orders;

-- After running this, restart the backend server
-- The tables will be automatically recreated with the correct schema

-- ===========================================

-- Option 2: Manual migration (keep data)
-- If you want to keep existing data, run these ALTER commands:

-- ALTER TABLE jetlink_orders ADD COLUMN bid_price DECIMAL(10, 2);
-- ALTER TABLE jetlink_orders ADD COLUMN estimated_arrival_time TIMESTAMP NULL;

-- CREATE TABLE IF NOT EXISTS jetlink_bids (
--     id INT AUTO_INCREMENT PRIMARY KEY,
--     order_id INT NOT NULL,
--     driver_id VARCHAR(255) NOT NULL,
--     bid_price DECIMAL(10, 2) NOT NULL,
--     estimated_arrival_time TIMESTAMP NOT NULL,
--     eta_minutes INT NOT NULL,
--     status VARCHAR(50) NOT NULL DEFAULT 'pending',
--     message TEXT,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
--     FOREIGN KEY (order_id) REFERENCES jetlink_orders(id) ON DELETE CASCADE,
--     INDEX idx_jetlink_bids_order_id (order_id),
--     INDEX idx_jetlink_bids_driver_id (driver_id),
--     INDEX idx_jetlink_bids_status (status)
-- );
