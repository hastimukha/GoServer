package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Product struct {
	Barcode      string  `gorm:"primaryKey" json:"barcode"`
	ProductName  string  `json:"product_name"`
	WeightKg     float64 `json:"weight_kg"`
	WeightGram   float64 `json:"weight_gram"`
	MRP          float64 `json:"mrp"`
	Stock        int     `json:"stock"`
	ProductImage string  `json:"product_image"`
}

var db *gorm.DB

func randomBarcode() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%013d", r.Int63n(9000000000000)+1000000000000)
}

func createProduct(c *gin.Context) {
	barcode := c.PostForm("barcode")
	if barcode == "" {
		barcode = randomBarcode()
	}

	stock := parseInt(c.PostForm("stock"))

	var existing Product
	if err := db.Where("barcode = ?", barcode).First(&existing).Error; err == nil {
		db.Model(&existing).Update("stock", gorm.Expr("stock + ?", stock))
		db.Where("barcode = ?", barcode).First(&existing)
		c.JSON(http.StatusOK, existing)
		return
	}

	p := Product{
		ProductName: c.PostForm("product_name"),
		WeightKg:    parseFloat(c.PostForm("weight_kg")),
		WeightGram:  parseFloat(c.PostForm("weight_gram")),
		MRP:         parseFloat(c.PostForm("mrp")),
		Stock:       stock,
		Barcode:     barcode,
	}

	file, err := c.FormFile("product_image")
	if err == nil {
		os.MkdirAll("uploads", os.ModePerm)
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("uploads/%d%s", time.Now().UnixNano(), ext)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "image save failed"})
			return
		}
		p.ProductImage = filename
	}

	if err := db.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func getProductByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")
	var p Product
	if err := db.Where("barcode = ?", barcode).First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func listProducts(c *gin.Context) {
	var products []Product
	db.Find(&products)
	c.JSON(http.StatusOK, products)
}

func parseFloat(s string) float64 {
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}

func parseInt(s string) int {
	var v int
	fmt.Sscanf(s, "%d", &v)
	return v
}

func main() {
	var err error
	db, err = gorm.Open(sqlite.Open("erp.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&Product{})

	r := gin.Default()
	r.Static("/uploads", "./uploads")
	r.POST("/products", createProduct)
	r.GET("/products", listProducts)
	r.GET("/products/:barcode", getProductByBarcode)
	r.Run(":8080")
}
