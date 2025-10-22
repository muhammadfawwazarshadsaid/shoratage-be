package main

import (
	"fmt"
	"log"
	"os"

	"yolo-server/db"
	"yolo-server/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Tidak menemukan file .env, lanjut pakai environment bawaan")
	}

	host := getEnv("POSTGRES_HOST", "localhost")
	user := getEnv("POSTGRES_USER", "user")
	password := getEnv("POSTGRES_PASSWORD", "password")
	dbname := getEnv("POSTGRES_DB", "yolo_db")
	pythonApiUrl := getEnv("PYTHON_API_URL", "http://localhost:5001/predict")
	port := getEnv("PORT", "8081")

	database := db.ConnectDB(host, user, password, dbname)
	defer database.Close()
	fmt.Println("✅ Connected to PostgreSQL")

	router := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}
	router.Use(cors.New(config))

	router.Static("/uploads", "./uploads")

	api := router.Group("/api")
	handlers.RegisterRoutes(api, database, pythonApiUrl)

	fmt.Printf("🚀 Go API running at http://localhost:%s\n", port)
	router.Run(":" + port)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
