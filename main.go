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

func (d Data) Pressent() DataPresented {
	return DataPresented{
		ID:            d.ID,
		DeviceID:      d.DeviceID,
		IsBackedup:    d.IsBackedup,
		Temperature:   d.Temperature,
		Humidity:      d.Humidity,
		EthyleneLevel: d.EthyleneLevel,
		UploadedBy:    d.UploadedBy,
		CreatedAt:     int(d.CreatedAt.Unix()),
	}
}

type DataPresented struct {
	ID            string  `gorm:"primaryKey;column:id" json:"id"`
	DeviceID      string  `json:"device_id" gorm:"index;column:device_id"`
	IsBackedup    bool    `json:"is_backedup" gorm:"column:is_backedup"`
	Temperature   float64 `json:"temperature" gorm:"column:temperature"`
	Humidity      float64 `json:"humidity" gorm:"column:humidity"`
	EthyleneLevel float64 `json:"ethylene_level" gorm:"column:ethylene_level"`
	UploadedBy    string  `json:"uploaded_by" gorm:"column:uploaded_by"`
	CreatedAt     int     `json:"created_at" gorm:"column:created_at;type:timestamp with time zone"`
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

	r.GET("/health", func(c *gin.Context) {
		c.Header("Connection", "close")
		c.JSONP(http.StatusOK, gin.H{"message": "Peer backend server is running..."})
	})

	// Define routes
	r.POST("/sensors", handleSensorData)    // POST endpoint to upload sensor data
	r.GET("/devices", handleDeviceList)     // GET endpoint to list devices
	r.GET("/sensor_query", handleDataQuery) // GET endpoint to query sensor data

	// Start HTTP server
	log.Println("Server listening on :8080...")

	log.Fatal(http.ListenAndServe(":8080", r))
}

type DataRequest struct {
	DeviceID      string  `json:"device_id" gorm:"index;column:device_id"`
	IsBackedup    bool    `json:"is_backedup" gorm:"column:is_backedup"`
	Temperature   float64 `json:"temperature" gorm:"column:temperature"`
	Humidity      float64 `json:"humidity" gorm:"column:humidity"`
	EthyleneLevel float64 `json:"ethylene_level" gorm:"column:ethylene_level"`
	UploadedBy    string  `json:"uploaded_by" gorm:"column:uploaded_by"`
	CreatedAt     int64   `json:"created_at" gorm:"column:created_at;type:timestamp with time zone"`
}

// handleSensorData handles incoming sensor data POST requests
func handleSensorData(c *gin.Context) {
	var req DataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Connection", "close")

	if ua := c.Request.Header.Get("User-Agent"); ua != "" {
		req.UploadedBy = ua
	}

	var data = Data{
		ID:            time.Now().Format("20060102150405"),
		DeviceID:      req.DeviceID,
		CreatedAt:     time.Unix(req.CreatedAt, 0),
		IsBackedup:    req.IsBackedup,
		Temperature:   req.Temperature,
		Humidity:      req.Humidity,
		EthyleneLevel: req.EthyleneLevel,
		UploadedBy:    req.UploadedBy,
	}

	// Set the "uploaded_by" field based on request context (e.g., authenticated user)
	if data.UploadedBy == "" {
		data.UploadedBy = "43542345435435"
	}

	// Create new record in the database using GORM
	result := db.Create(&data)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sensor data received and saved successfully"})
}

// handleDeviceList handles GET requests to list devices
func handleDeviceList(c *gin.Context) {
	var devices []string
	if err := db.Model(&Data{}).Distinct("device_id").Pluck("device_id", &devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices})
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
	dd.
		Order("created_at DESC").
		Find(&data)

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

	var presentedData []DataPresented
	for _, d := range data {
		presentedData = append(presentedData, d.Pressent())
	}

	c.JSON(http.StatusOK, gin.H{
		"average_temperature":    avgTemp,
		"average_humidity":       avgHumidity,
		"average_ethylene_level": avgEthyleneLevel,
		"data":                   presentedData,
	})
}
