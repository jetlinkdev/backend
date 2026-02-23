package main

import (
	"flag"
	"fmt"

	"github.com/gorilla/mux"

	hubhandlers "jetlink/handlers"
	"jetlink/database"
	"jetlink/redis"
	"jetlink/utils"
	"jetlink/routes"
	"jetlink/server"
)

var (
	addr      = flag.String("addr", ":8080", "http service address")
	redisAddr = flag.String("redis-addr", "localhost:6379", "Redis server address")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := utils.NewLogger()

	// Initialize database with MySQL connection
	// Format: username:password@protocol(address)/dbname?param=value
	mysqlDSN := "root:~Densus_88@tcp(localhost:3306)/jetlink?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := database.InitDB(mysqlDSN)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to initialize database: %v", err))
		return
	}
	defer db.Close()

	// Initialize Redis
	redisConfig := redis.DefaultConfig()
	redisConfig.Addr = *redisAddr
	
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