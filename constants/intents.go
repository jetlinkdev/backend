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
	IntentGetMyBids         = "get_my_bids"
	IntentMyBids            = "my_bids"
	IntentSubmitReview      = "submit_review"
	IntentReviewSubmitted   = "review_submitted"

	// Authentication intents
	IntentAuth               = "auth"
	IntentAuthSuccess        = "auth_success"
	IntentAuthProfileNeeded  = "auth_profile_needed"
	IntentCompleteProfile    = "complete_profile"
	IntentDriverRegistration = "driver_registration"
	IntentDriverRegistered   = "driver_registered"
	IntentCheckDriverStatus  = "check_driver_status"
	IntentDriverStatus       = "driver_status"
	IntentUpdateDriverStatus = "update_driver_status"
	IntentSyncActiveOrders   = "sync_active_orders"
)
