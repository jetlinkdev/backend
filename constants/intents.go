package constants

// Intent constants for WebSocket messages
const (
	IntentCreateOrder       = "create_order"
	IntentCancelOrder       = "cancel_order"
	IntentOrderCancelled    = "order_cancelled"
	IntentPing              = "ping"
	IntentPong              = "pong"
	IntentOrderCreated      = "order_created"
	IntentNewOrderAvailable = "new_order_available"
	IntentSubmitBid         = "submit_bid"
	IntentSelectBid         = "select_bid"
	IntentDriverArrived     = "driver_arrived"
	IntentCompleteTrip      = "complete_trip"
	IntentTripCompleted     = "trip_completed"
	IntentBidAccepted       = "bid_accepted"
	IntentBidRejected       = "bid_rejected"
	IntentNewBidReceived    = "new_bid_received"
	IntentError             = "error"
	
	// Authentication intents
	IntentAuth               = "auth"
	IntentDriverRegistration = "driver_registration"
	IntentDriverRegistered   = "driver_registered"
	IntentCheckDriverStatus  = "check_driver_status"
	IntentDriverStatus       = "driver_status"
)
