package main

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// --- Content types ---

type ContentItem struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type Engagement struct {
	Num    string        `json:"num"`
	Title  string        `json:"title"`
	Breton string        `json:"breton"`
	Items  []ContentItem `json:"items"`
}

type Candidate struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Info string `json:"info"`
}

type PageContent struct {
	PageTitle    string       `json:"page_title"`
	HeroTitle    string       `json:"hero_title"`
	Subtitle     string       `json:"subtitle"`
	IntroText    string       `json:"intro_text"` // raw HTML
	Engagements  []Engagement `json:"engagements"`
	TeamIntro    string       `json:"team_intro"`
	Candidates   []Candidate  `json:"candidates"`
	FooterTitle  string       `json:"footer_title"`
	FooterText   string       `json:"footer_text"`
	HighlightBox string       `json:"highlight_box"`
}

var (
	contentMu   sync.RWMutex
	siteContent PageContent
	sessions    sync.Map
)

// --- Helpers ---

func loadEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}

func loadContent(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	contentMu.Lock()
	defer contentMu.Unlock()
	json.Unmarshal(data, &siteContent)
}

func saveContent(path string, c PageContent) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func newSession() string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)
	sessions.Store(token, true)
	return token
}

func isAuthenticated(c *gin.Context) bool {
	cookie, err := c.Cookie("bureau_session")
	if err != nil {
		return false
	}
	_, ok := sessions.Load(cookie)
	return ok
}

func authMiddleware(c *gin.Context) {
	if !isAuthenticated(c) {
		c.Redirect(http.StatusFound, "/bureau/login")
		c.Abort()
		return
	}
	c.Next()
}

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
	loadEnv("../data/.env")
	loadContent("../data/content.json")

	db := initDB()
	defer db.Close()

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.Static("/assets", "../assets")
	r.LoadHTMLGlob("../templates/*")

	// --- Main site ---
	r.GET("/", func(c *gin.Context) {
		contentMu.RLock()
		content := siteContent
		contentMu.RUnlock()
		c.HTML(http.StatusOK, "index.html", gin.H{
			"PageTitle":    content.PageTitle,
			"HeroTitle":    content.HeroTitle,
			"Subtitle":     content.Subtitle,
			"IntroText":    template.HTML(content.IntroText),
			"Engagements":  content.Engagements,
			"TeamIntro":    content.TeamIntro,
			"Candidates":   content.Candidates,
			"FooterTitle":  content.FooterTitle,
			"FooterText":   content.FooterText,
			"HighlightBox": content.HighlightBox,
		})
	})

	// --- Subscribe ---
	type SubscribeRequest struct {
		Email string `json:"email" form:"email"`
	}

	r.POST("/subscribe", func(c *gin.Context) {
		var req SubscribeRequest
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides"})
			return
		}

		if req.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "L'email est requis"})
			return
		}

		_, err := mail.ParseAddress(req.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format d'email invalide"})
			return
		}

		insertSQL := `INSERT INTO subscribers(email) VALUES (?)`
		_, err = db.Exec(insertSQL, req.Email)
		if err != nil {
			if err.Error() == "UNIQUE constraint failed: subscribers.email" {
				c.JSON(http.StatusConflict, gin.H{"error": "Cet email est déjà inscrit"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur, veuillez réessayer"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Inscription réussie !"})
	})

	// --- Bureau ---
	r.GET("/bureau", func(c *gin.Context) {
		if isAuthenticated(c) {
			c.Redirect(http.StatusFound, "/bureau/edit")
		} else {
			c.Redirect(http.StatusFound, "/bureau/login")
		}
	})

	r.GET("/bureau/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "bureau_login.html", nil)
	})

	r.POST("/bureau/login", func(c *gin.Context) {
		password := c.PostForm("password")
		expected := os.Getenv("BUREAU_PASSWORD")
		if expected == "" {
			expected = "changeme"
		}
		if password != expected {
			c.HTML(http.StatusUnauthorized, "bureau_login.html", gin.H{"Error": "Mot de passe incorrect."})
			return
		}
		token := newSession()
		c.SetCookie("bureau_session", token, 86400, "/", "", false, true)
		c.Redirect(http.StatusFound, "/bureau/edit")
	})

	bureau := r.Group("/bureau")
	bureau.Use(authMiddleware)
	{
		bureau.GET("/edit", func(c *gin.Context) {
			contentMu.RLock()
			content := siteContent
			contentMu.RUnlock()
			data, _ := json.Marshal(content)
			c.HTML(http.StatusOK, "bureau_editor.html", gin.H{
				"ContentJSON": template.JS(data),
			})
		})

		bureau.POST("/save", func(c *gin.Context) {
			var incoming PageContent
			if err := c.ShouldBindJSON(&incoming); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
				return
			}
			// Strip template injection attempts
			incoming.IntroText = strings.ReplaceAll(incoming.IntroText, "{{", "")
			incoming.IntroText = strings.ReplaceAll(incoming.IntroText, "}}", "")
			contentMu.Lock()
			siteContent = incoming
			contentMu.Unlock()
			if err := saveContent("../data/content.json", incoming); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur sauvegarde"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Contenu sauvegardé."})
		})

		bureau.GET("/logout", func(c *gin.Context) {
			cookie, err := c.Cookie("bureau_session")
			if err == nil {
				sessions.Delete(cookie)
			}
			c.SetCookie("bureau_session", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/bureau/login")
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8024"
	}

	log.Printf("Serveur démarré sur http://localhost:%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
