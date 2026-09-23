package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config structure matching your nested yaml layout
type Config struct {
	DB struct {
		Postgres struct {
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			User     string `mapstructure:"user"`
			Password string `mapstructure:"password"`
			Dbname   string `mapstructure:"dbname"`
			SSLMode  string `mapstructure:"sslmode"`
		} `mapstructure:"postgres"`
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

// LoadConfig loads .env, binds variables, and parses config.yml
func LoadConfig() (*Config, error) {
	// 1. Load .env file into environment variables (ignored if running on Render where env vars are set directly)
	_ = godotenv.Load()

	// 2. Bind specific .env keys to Viper config paths
	viper.BindEnv("db.postgres.host", "db_postgres_host")
	viper.BindEnv("db.postgres.port", "db_postgres_port")
	viper.BindEnv("db.postgres.user", "db_postgres_username")
	viper.BindEnv("db.postgres.password", "db_postgres_password")
	viper.BindEnv("db.postgres.dbname", "db_postgres_dbname")
	viper.BindEnv("db.postgres.sslmode", "db_postgres_sslmode")

	// 3. Alternatively read from config.yml if it exists
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig() // Will fallback gracefully to bound env vars if config.yml is missing

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
		cfg.DB.Postgres.User,
		cfg.DB.Postgres.Password,
		cfg.DB.Postgres.Dbname,
		cfg.DB.Postgres.SSLMode,
	)

	// Connect to database using GORM
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	r := gin.Default()

	// Endpoint 1: Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "echo health",
		})
	})

	// Endpoint 2: Student list from postgres
	r.GET("/students", func(c *gin.Context) {
		var students []Student

		if result := db.Find(&students); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": result.Error.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"count": len(students),
			"data":  students,
		})
	})

	log.Println("Server is running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
