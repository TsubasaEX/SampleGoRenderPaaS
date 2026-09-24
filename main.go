package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var ctx = context.Background()

// Config structure matching your nested yaml layout and redis settings
type Config struct {
	DB struct {
		Postgres struct {
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
			Dbname   string `mapstructure:"dbname"`
			SSLMode  string `mapstructure:"sslmode"`
		} `mapstructure:"postgres"`

		Redis struct {
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
			SSL      bool   `mapstructure:"ssl"`
		} `mapstructure:"redis"`
	} `mapstructure:"db"`
}

// Student ORM Model
type Student struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name      string    `gorm:"type:text;column:name" json:"name"`
	Class     string    `gorm:"type:text;column:class" json:"class"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Student) TableName() string {
	return "student"
}

// LoginRequest payload structure
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthMiddleware validates Bearer token session from Redis before accessing protected routes
func AuthMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			return
		}

		// Expecting format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format. Expected 'Bearer <token>'"})
			return
		}

		token := parts[1]

		sessionKey := fmt.Sprintf("session:%s", token)
		username, err := rdb.Get(ctx, sessionKey).Result()
		if err == redis.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			return
		} else if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.Set("username", username)
		c.Next()
	}
}

// LoadConfig loads .env, binds variables, and parses config.yml
func LoadConfig() (*Config, error) {
	// 1. Load .env file into environment variables
	_ = godotenv.Load()

	// CRITICAL: Tell Viper to allow empty string values from environment variables
	viper.AllowEmptyEnv(true)

	// 2. Bind specific .env keys to Viper config paths (Postgres & Redis)
	viper.BindEnv("db.postgres.host", "db_postgres_host")
	viper.BindEnv("db.postgres.port", "db_postgres_port")
	viper.BindEnv("db.postgres.username", "db_postgres_username")
	viper.BindEnv("db.postgres.password", "db_postgres_password")
	viper.BindEnv("db.postgres.dbname", "db_postgres_dbname")
	viper.BindEnv("db.postgres.sslmode", "db_postgres_sslmode")

	viper.BindEnv("db.redis.host", "db_redis_host")
	viper.BindEnv("db.redis.port", "db_redis_port")
	viper.BindEnv("db.redis.username", "db_redis_username")
	viper.BindEnv("db.redis.password", "db_redis_password")
	viper.BindEnv("db.redis.ssl", "db_redis_ssl")

	// 3. Read from config.yml if it exists
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Build DSN string using loaded config values
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Taipei",
		cfg.DB.Postgres.Host,
		cfg.DB.Postgres.Port,
		cfg.DB.Postgres.Username,
		cfg.DB.Postgres.Password,
		cfg.DB.Postgres.Dbname,
		cfg.DB.Postgres.SSLMode,
	)

	// Connect to database using GORM
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Build Redis options conditionally based on whether a username is provided
	redisOpts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.DB.Redis.Host, cfg.DB.Redis.Port),
		Password: cfg.DB.Redis.Password,
	}
	if cfg.DB.Redis.Username != "" {
		redisOpts.Username = cfg.DB.Redis.Username
	}

	// If connecting to a remote host (like Render), enable TLS
	// You can check if the host contains "render.com" or just enable it for remote IPs
	if cfg.DB.Redis.SSL {
		redisOpts.TLSConfig = &tls.Config{
			InsecureSkipVerify: false,
		}
	}

	rdb := redis.NewClient(redisOpts)
	log.Println("Connected to Redis via structured configuration.")

	// Verify Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	r := gin.Default()

	// Endpoint 1: Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "echo health",
		})
	})

	// Endpoint 2: Login Route (Generates Redis Session Token)
	r.POST("/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
			return
		}

		// Hardcoded check for demo (replace with DB verification if needed)
		if req.Username != "admin" || req.Password != "password123" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		sessionToken := uuid.New().String()
		sessionKey := fmt.Sprintf("session:%s", sessionToken)

		// Save token in Redis for 24 hours
		err := rdb.Set(ctx, sessionKey, req.Username, 24*time.Hour).Err()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Login successful",
			"token":   sessionToken,
		})
	})

	// Protected Routes Group (Protected by Redis Session AuthMiddleware)
	protected := r.Group("/")
	protected.Use(AuthMiddleware(rdb))
	{
		// Endpoint 3: Student list from postgres (Protected)
		protected.GET("/students", func(c *gin.Context) {
			username, _ := c.Get("username")
			var students []Student

			if result := db.Find(&students); result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": result.Error.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"logged_in_user": username,
				"count":          len(students),
				"data":           students,
			})
		})
	}

	log.Println("Server is running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
