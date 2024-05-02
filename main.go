package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Data represents the sensor data structure
type Data struct {
	ID            string  `gorm:"primaryKey"`
	DeviceID      string  `json:"device_id"`
	IsBackedup    bool    `json:"is_backedup"`
	Temperature   float64 `json:"temperature"`
	Humidity      float64 `json:"humidity"`
	EthyleneLevel float64 `json:"ethylene_level"`
	UploadedBy    string  `json:"uploaded_by"`
	CreatedAt     int64   `json:"created_at"`
}

// QueryParams represents the query parameters for filtering sensor data
type QueryParams struct {
	DeviceID  string `json:"device_id"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

var db *gorm.DB

func main() {
	// Initialize Gin router
	r := gin.Default()

	// Open SQLite database using GORM
	var err error
	db, err = gorm.Open(sqlite.Open("sensor_data.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Auto Migrate the schema
	err = db.AutoMigrate(&Data{})
	if err != nil {
		log.Fatal(err)
	}

	// Define routes
	r.POST("/sensor", handleSensorData)     // POST endpoint to upload sensor data
	r.GET("/sensor_query", handleDataQuery) // GET endpoint to query sensor data

	// Start HTTP server
	log.Println("Server listening on :8080...")
	log.Fatal(r.Run(":8080"))
}

// handleSensorData handles incoming sensor data POST requests
func handleSensorData(c *gin.Context) {
	var data Data
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set the "uploaded_by" field based on request context (e.g., authenticated user)
	if data.UploadedBy == "" {
		data.UploadedBy = "1123341"
	}
	data.ID = time.Now().Format("20060102150405")
	data.CreatedAt = time.Now().Unix()

	// Create new record in the database using GORM
	result := db.Create(&data)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sensor data received and saved successfully"})
}

// handleDataQuery handles GET requests to fetch and process sensor data
func handleDataQuery(c *gin.Context) {
	var params QueryParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var data []Data
	// Fetch data from the database using GORM with filters

	if params.EndTime == 0 {
		params.EndTime = time.Now().Unix()
	}
	startTime := time.Unix(params.StartTime, 0)
	endTime := time.Unix(params.EndTime, 0)

	dd := db.Where("id IS NOT NULL")
	if params.DeviceID != "" {
		dd = dd.Where("device_id = ?", params.DeviceID)
	}
	if params.StartTime != 0 {
		dd = dd.Where("created_at >= ?", startTime)
	}
	if params.EndTime != 0 {
		dd = dd.Where("created_at <= ?", endTime)
	}
	dd.Find(&data)

	// Calculate average values
	var totalTemp, totalHumidity, totalEthyleneLevel float64
	count := len(data)
	for _, d := range data {
		totalTemp += d.Temperature
		totalHumidity += d.Humidity
		totalEthyleneLevel += d.EthyleneLevel
	}

	var avgTemp, avgHumidity, avgEthyleneLevel float64
	if count > 0 {
		avgTemp = totalTemp / float64(count)
		avgHumidity = totalHumidity / float64(count)
		avgEthyleneLevel = totalEthyleneLevel / float64(count)
	}

	c.JSON(http.StatusOK, gin.H{
		"average_temperature":    avgTemp,
		"average_humidity":       avgHumidity,
		"average_ethylene_level": avgEthyleneLevel,
		"data":                   data,
	})
}