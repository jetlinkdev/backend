package main

import (
	"flag"
	"fmt"

	"github.com/gorilla/mux"

	hubhandlers "jetlink/handlers"
	"jetlink/database"
	"jetlink/firebase"
	"jetlink/redis"
	"jetlink/routes"
	"jetlink/server"
	"jetlink/utils"
)

var (
	addr      = flag.String("addr", ":8080", "http service address")
	redisAddr = flag.String("redis-addr", "localhost:6379", "Redis server address")
)

func main() {
	flag.Parse()

	// Load environment variables from .env.local or .env
	utils.LoadEnv(".env.local")
	utils.LoadEnv(".env")

	// Initialize logger
	logger := utils.NewLogger()

	// Initialize Firebase Admin SDK
	firebaseProjectID := utils.GetEnv("FIREBASE_PROJECT_ID", "jetlink-47eb8")
	if err := firebase.InitFirebaseWithConfig(firebaseProjectID); err != nil {
		logger.Warn(fmt.Sprintf("Firebase initialization failed: %v", err))
		logger.Info("Firebase token verification will not be available")
	} else {
		logger.Info("Firebase Admin SDK initialized successfully")
	}

	// Get MySQL DSN from environment or use default
	mysqlDSN := utils.GetEnv("MYSQL_DSN", "root:~Densus_88@tcp(localhost:3306)/jetlink?charset=utf8mb4&parseTime=True&loc=Local")

	// Override addr with environment variable if provided
	serverAddr := utils.GetEnv("SERVER_ADDR", *addr)
	*addr = serverAddr

	// Initialize database with MySQL connection
	db, err := database.InitDB(mysqlDSN)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to initialize database: %v", err))
		return
	}
	defer db.Close()

	// Get Redis address from environment or command line
	redisAddrFromEnv := utils.GetEnv("REDIS_ADDR", "localhost:6379")
	if *redisAddr == "localhost:6379" && redisAddrFromEnv != "localhost:6379" {
		*redisAddr = redisAddrFromEnv
	}

	// Initialize Redis
	redisConfig := redis.DefaultConfig()
	redisConfig.Addr = *redisAddr

	// Get Redis password from environment
	redisPassword := utils.GetEnv("REDIS_PASSWORD", "")
	if redisPassword != "" {
		redisConfig.Password = redisPassword
	}

	// Get Redis DB from environment
	redisDB := utils.GetEnv("REDIS_DB", "0")
	if redisDB != "0" {
		// Parse redisDB to int if needed
	}

	redisClient, err := redis.InitRedis(redisConfig)
	if err != nil {
		logger.Warn(fmt.Sprintf("Redis connection failed, running without Redis: %v", err))
		logger.Info("Orders will be stored in memory only (no persistence)")
	} else {
		logger.Info(fmt.Sprintf("Connected to Redis at %s", redisConfig.Addr))
		defer redisClient.Close()
	}

	// Create order repository
	orderRepo := database.NewOrderRepository(db)

	// Create Redis repositories
	var orderRedis *redis.OrderRedis
	var bidRedis *redis.BidRedis

	if redisClient != nil {
		orderRedis = redis.NewOrderRedis(redisClient)
		bidRedis = redis.NewBidRedis(redisClient)
	}

	// Create hub with Redis support
	hub := hubhandlers.NewHubWithRedis(orderRedis, bidRedis)
	go hub.Run()

	// Create router
	router := mux.NewRouter()

	// Setup routes
	routes.SetupRoutes(router, hub, logger, orderRepo)

	// Create and start server
	srv := server.New(*addr, router, logger)
	srv.Start()

	// Wait for shutdown signal
	srv.WaitForShutdownSignal()

	// Shutdown server gracefully
	srv.Shutdown()
}