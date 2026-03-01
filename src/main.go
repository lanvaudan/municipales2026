package main

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

func newSession(db *sql.DB) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)
	db.Exec(`INSERT INTO sessions(token) VALUES (?)`, token)
	return token
}

func isAuthenticated(c *gin.Context, db *sql.DB) bool {
	cookie, err := c.Cookie("bureau_session")
	if err != nil {
		return false
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ?`, cookie).Scan(&count)
	return count > 0
}

// authMiddleware redirects unauthenticated requests to /login (bureau subdomain root).
func authMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isAuthenticated(c, db) {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// extractSubdomain returns the first DNS label of the Host header (port stripped).
func extractSubdomain(host string) string {
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	parts := strings.SplitN(host, ".", 2)
	if len(parts) >= 2 {
		return parts[0]
	}
	return host
}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "../data/subscribers.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS subscribers (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"email" TEXT UNIQUE,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		"token" TEXT PRIMARY KEY,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

// buildPublicRouter returns the router for the main campaign site.
func buildPublicRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()
	r.Static("/assets", "../assets")
	r.LoadHTMLGlob("../templates/*")

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

	return r
}

// buildBureauRouter returns the router for the admin interface (bureau subdomain).
// All routes are at the root level (no /bureau/ prefix).
func buildBureauRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()
	r.LoadHTMLGlob("../templates/*")

	r.GET("/", func(c *gin.Context) {
		if isAuthenticated(c, db) {
			c.Redirect(http.StatusFound, "/edit")
		} else {
			c.Redirect(http.StatusFound, "/login")
		}
	})

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "bureau_login.html", nil)
	})

	r.POST("/login", func(c *gin.Context) {
		password := c.PostForm("password")
		expected := os.Getenv("BUREAU_PASSWORD")
		if expected == "" {
			expected = "changeme"
		}
		if password != expected {
			c.HTML(http.StatusUnauthorized, "bureau_login.html", gin.H{"Error": "Mot de passe incorrect."})
			return
		}
		token := newSession(db)
		c.SetCookie("bureau_session", token, 86400, "/", "", false, true)
		c.Redirect(http.StatusFound, "/edit")
	})

	protected := r.Group("/")
	protected.Use(authMiddleware(db))
	{
		protected.GET("/edit", func(c *gin.Context) {
			contentMu.RLock()
			content := siteContent
			contentMu.RUnlock()
			data, _ := json.Marshal(content)
			c.HTML(http.StatusOK, "bureau_editor.html", gin.H{
				"ContentJSON": template.JS(data),
			})
		})

		protected.POST("/save", func(c *gin.Context) {
			var incoming PageContent
			if err := c.ShouldBindJSON(&incoming); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalide"})
				return
			}
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

		protected.GET("/emails.csv", func(c *gin.Context) {
			rows, err := db.Query("SELECT email FROM subscribers ORDER BY created_at")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur base de données"})
				return
			}
			defer rows.Close()
			c.Header("Content-Type", "text/csv; charset=utf-8")
			c.Header("Content-Disposition", `attachment; filename="emails_lanvaudan.csv"`)
			for rows.Next() {
				var email string
				if err := rows.Scan(&email); err != nil {
					continue
				}
				fmt.Fprintln(c.Writer, email)
			}
		})

		protected.GET("/logout", func(c *gin.Context) {
			cookie, err := c.Cookie("bureau_session")
			if err == nil {
				db.Exec(`DELETE FROM sessions WHERE token = ?`, cookie)
			}
			c.SetCookie("bureau_session", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
		})
	}

	return r
}

func main() {
	loadEnv("../data/.env")
	loadContent("../data/content.json")

	db := initDB()
	defer db.Close()

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	publicRouter := buildPublicRouter(db)
	bureauRouter := buildBureauRouter(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8024"
	}

	log.Printf("Serveur démarré sur http://localhost:%s\n", port)
	err := http.ListenAndServe(":"+port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if extractSubdomain(r.Host) == "bureau" {
			bureauRouter.ServeHTTP(w, r)
		} else {
			publicRouter.ServeHTTP(w, r)
		}
	}))
	if err != nil {
		log.Fatal(err)
	}
}
