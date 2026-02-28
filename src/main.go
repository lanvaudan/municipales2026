package main

import (
	"database/sql"
	"log"
	"net/http"
	"net/mail"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "../data/subscribers.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS subscribers (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"email" TEXT UNIQUE,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func main() {
	db := initDB()
	defer db.Close()

	// Release mode by default, unless overriden
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Serve static files from the /assets directory
	// In the HTML it is referenced as /assets/logo.png
	r.Static("/assets", "../assets")

	// Load HTML templates
	r.LoadHTMLGlob("../templates/*")

	// Serve the index.html on /
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	// Handle subscribe POST
	type SubscribeRequest struct {
		Email string `json:"email" form:"email"`
	}

	r.POST("/subscribe", func(c *gin.Context) {
		var req SubscribeRequest
		// Allow binding from JSON or URL-encoded form data
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
			return
		}

		if req.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "L'email est requis"})
			return
		}

		// Basic validation
		_, err := mail.ParseAddress(req.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format d'email invalide"})
			return
		}

		// Insert into db
		insertSQL := `INSERT INTO subscribers(email) VALUES (?)`
		_, err = db.Exec(insertSQL, req.Email)
		if err != nil {
			// Check if unique constraint failed
			if err.Error() == "UNIQUE constraint failed: subscribers.email" {
				// We can return a specific conflict error or just a success message if we want to be silent
				c.JSON(http.StatusConflict, gin.H{"error": "Cet email est déjà inscrit"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur, veuillez réessayer"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Inscription réussie !"})
	})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8024"
	}

	log.Printf("Serveur démarré sur http://localhost:%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
