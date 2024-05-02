package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Data represents the sensor data structure
type Data struct {
	ID            string    `gorm:"primaryKey;column:id" json:"id"`
	DeviceID      string    `json:"device_id" gorm:"index;column:device_id"`
	IsBackedup    bool      `json:"is_backedup" gorm:"column:is_backedup"`
	Temperature   float64   `json:"temperature" gorm:"column:temperature"`
	Humidity      float64   `json:"humidity" gorm:"column:humidity"`
	EthyleneLevel float64   `json:"ethylene_level" gorm:"column:ethylene_level"`
	UploadedBy    string    `json:"uploaded_by" gorm:"column:uploaded_by"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at;type:timestamp with time zone"`
}

func (d Data) TableName() string {
	return "sensor_data"
}

// QueryParams represents the query parameters for filtering sensor data
type QueryParams struct {
	DeviceID  string `json:"device_id"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

var db *gorm.DB

// CorHandler is a middleware function that adds the "Access-Control-Allow-Origin" header
func CorHandler(c *gin.Context) {
	// origin := c.Request.Header.Get("Origin")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
	// c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}
	c.Next()
}

func main() {
	// Initialize Gin router
	r := gin.Default()
	r.Use(CorHandler)

	// Open Postgres database using GORM
	var err error
	db, err = gorm.Open(postgres.Open("postgres://peer_t9lq_user:SSuEyEAo39US5swHgtTLazWBZBZMI8am@dpg-copm7okf7o1s73e2b6m0-a.frankfurt-postgres.render.com/peer_t9lq"), &gorm.Config{})
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
	data.CreatedAt = time.Now()

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
	var params = QueryParams{
		DeviceID:  c.Query("device_id"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
	}

	// Fetch data from the database using GORM with filters
	start, _ := strconv.Atoi(params.StartTime)
	end, _ := strconv.Atoi(params.EndTime)

	dd := db.Where("id IS NOT NULL")
	if params.DeviceID != "" {
		dd = dd.Where("device_id = ?", params.DeviceID)
	}

	if start != 0 {
		startTime := time.Unix(int64(start), 0)
		dd = dd.Where("created_at >= ?", startTime)
	}
	if end != 0 {
		endTime := time.Unix(int64(end), 0)
		dd = dd.Where("created_at <= ?", endTime)
	}

	var data []Data
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
