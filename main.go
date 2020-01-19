package main

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"

	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/gin-gonic/gin"
)

const (
	indexTemplate = "index.html"
)

var recipes []Recipe
var db *gorm.DB

func indexGet(c *gin.Context) {
	db = connectToDatabase()
	defer db.Close()
	db.Select("title, url, image_url, description, created_at, updated_at").Find(&recipes)
	c.HTML(http.StatusOK, indexTemplate, gin.H{
		"recipes": recipes,
	})
}

// Recipe is metadata captured from a recipe url
type Recipe struct {
	gorm.Model
	Title       string `db:"title"`
	URL         string `db:"url"`
	ImageURL    string `db:"image_url"`
	Description string `db:"description"`
}

func indexPost(c *gin.Context) {
	url := c.PostForm("recipeUrl")

	if url == "" {
		c.HTML(http.StatusInternalServerError, indexTemplate, nil)
		return
	}

	log.Printf("Getting URL: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		c.HTML(http.StatusInternalServerError, indexTemplate, nil)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.HTML(http.StatusInternalServerError, indexTemplate, nil)
		return
	}

	recipe := recipeFromResponse(resp.Body)

	log.Printf("Recipe: %+v", recipe)

	db = connectToDatabase()
	defer db.Close()

	db.Create(recipe)

	db.Select("title, url, image_url, description, created_at, updated_at").Find(&recipes)
	c.HTML(http.StatusOK, indexTemplate, gin.H{
		"recipes": recipes,
	})
}

func connectToDatabase() *gorm.DB {
	db, err := gorm.Open("mysql", os.Getenv("DATABASE_URL")+"?charset=utf8&parseTime=True")

	if err != nil {
		panic(err.Error())
	}

	// Migrate the schema
	db.AutoMigrate(&Recipe{})

	log.Printf("Database connection succeeded.")

	return db
}

func recipeFromResponse(responseBody io.ReadCloser) *Recipe {
	body, _ := ioutil.ReadAll(responseBody)

	og := opengraph.NewOpenGraph()
	err := og.ProcessHTML(strings.NewReader(string(body)))

	if err != nil {
		panic(err.Error())
	}

	return &Recipe{
		Title:       og.Title,
		URL:         og.URL,
		ImageURL:    og.Images[0].URL,
		Description: og.Description,
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.html")
	router.Static("/assets", "assets")

	router.GET("/", indexGet)
	router.POST("/", indexPost)

	log.Printf("Starting application on port %s", port)

	router.Run(":" + port)
}
