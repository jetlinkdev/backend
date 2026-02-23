# Testing Documentation for Jetlink Backend

## Overview
This document describes the testing strategy and implementation for the Jetlink backend system, focusing on order processing functionality.

## Test Structure
The tests are organized into two main categories:

### 1. Unit Tests (`tests/unit/`)
- Test individual units of code in isolation
- Focus on models and repository layer
- Use mock implementations to isolate units

#### Test Files:
- `model_test.go`: Tests for order models and their serialization
- `repository_test.go`: Tests for order repository operations using mock database

### 2. Integration Tests (`tests/integration/`)
- Test interactions between multiple components
- Validate end-to-end flows
- Test API endpoints and WebSocket connections

#### Test Files:
- `order_integration_test.go`: Basic integration tests for API endpoints
- `websocket_integration_test.go`: Tests for WebSocket-based order processing

## Key Features Tested

### Order Model Testing
- Creation of orders with valid data
- Handling of nullable time fields
- JSON serialization/deserialization
- Proper field validation

### Order Repository Testing
- CRUD operations (Create, Read, Update, Delete)
- Query methods (by user ID, by status)
- Error handling for non-existent records
- Proper handling of nullable time fields

### Order Processing Flow
- Complete order lifecycle from creation to completion
- Handling of immediate vs scheduled pickups
- Status transitions (pending → accepted → completed)

## Special Considerations

### Nullable Time Fields
The `time` field in orders is now nullable, allowing for immediate pickups when no specific time is requested. Tests verify:
- Proper storage of null values
- Correct handling during serialization/deserialization
- Business logic for displaying "immediately" when time is null

### Geographic Coordinates
Added latitude and longitude fields for pickup and destination locations. Tests verify:
- Proper storage of coordinate data
- Validation of coordinate ranges
- Integration with the rest of the order data

## Running Tests

To run all tests:
```bash
cd backend
go test ./tests/... -v
```

To run specific test suites:
```bash
# Unit tests only
go test ./tests/unit -v

# Integration tests only
go test ./tests/integration -v
```

## Test Coverage
The testing strategy aims to cover:
- 100% of business logic paths
- Edge cases (null values, boundary conditions)
- Error handling scenarios
- Integration points between components

## Future Improvements
- Add database integration tests with a test MySQL instance
- Expand WebSocket testing with actual connection simulation
- Add load testing scenarios
- Implement property-based testing for data validation