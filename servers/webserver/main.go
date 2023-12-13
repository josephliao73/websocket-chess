package main

import (
	"database/sql"
	_ "encoding/gob"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

const dbLoc = "./servers/db/data.db"

var store = sessions.NewCookieStore([]byte("secret"))

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("./dist/*.html")
	router.Static("./assets", "./dist/assets")
	router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", gin.H{})
	})
	router.GET("/:path", func(ctx *gin.Context) {
		path := ctx.Param("path")
		if matched, _ := regexp.MatchString("^[0-9]+$", path); matched {
			ctx.HTML(http.StatusOK, "game.html", gin.H{"id": path})
		} else {
			_, err := os.Stat("dist/" + path + ".html")
			session, _ := store.Get(ctx.Request, "login-session")
			if err == nil {
				if path == "login" {
					if session.Values["username"] == nil {
						ctx.HTML(http.StatusOK, path+".html", gin.H{})
					} else {
						ctx.Redirect(http.StatusFound, "/")
					}
					return
				}
				if path == "" || path == "game" {
					ctx.Redirect(http.StatusFound, "/")
					return
				}
				ctx.HTML(http.StatusOK, path+".html", gin.H{})
			} else {
				ctx.Redirect(http.StatusNotFound, "/")
			}
		}
	})
	router.POST("/login", handleLogin)
	router.POST("/register", handleRegistry)
	router.Run(":8081")
}

// TODO: Put users in database

func handleRegistry(ctx *gin.Context) {
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")
	confirm := ctx.PostForm("confirm")
	email := ctx.PostForm("email")

	if dbHasUser(username) {
		ctx.JSON(400, gin.H{"status": "Username was taken."})
		return
	}
	if email != "" && dbHasEmail(email) {
		ctx.JSON(400, gin.H{"status": "Email already registered."})
		return
	}
	if password != confirm {
		ctx.JSON(400, gin.H{"status": "Passwords do not match."})
		return
	}
	registerUser(username, password, email)
	createSession(ctx, username)
	ctx.JSON(200, gin.H{"status": "Registered!"})
	return
}

func dbHasUser(username string) bool {
	db, err := sql.Open("sqlite3", dbLoc)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var username_exists int
	stripped_username := strings.ToLower(strings.TrimSpace(username))
	err = db.QueryRow("SELECT COUNT(TRIM(LOWER(username))) FROM user WHERE username=(?);", stripped_username).Scan(&username_exists)
	if err != nil {
		panic(err)
	}
	return username_exists > 0
}

func dbHasEmail(email string) bool {
	db, err := sql.Open("sqlite3", dbLoc)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var email_exists int
	stripped_email := strings.ToLower(strings.TrimSpace(email))
	err = db.QueryRow("SELECT COUNT(TRIM(LOWER(email))) FROM user WHERE email=(?);", stripped_email).Scan(&email_exists)
	if err != nil {
		panic(err)
	}
	return email_exists > 0
}

func registerUser(username string, password string, email string) error {
	db, err := sql.Open("sqlite3", dbLoc)
	if err != nil {
		return err
	}
	defer db.Close()

	statement, err := db.Prepare("INSERT INTO user (username, password, email) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer statement.Close()

	password = hashPassword(password)

	if email == "" {
		_, err = statement.Exec(username, password, nil)
	} else {
		_, err = statement.Exec(username, password, email)
	}

	if err != nil {
		return err
	}
	return nil
}

func hashPassword(password string) string {
	bytes := []byte(password)

	hashedPassword, err := bcrypt.GenerateFromPassword(bytes, bcrypt.MinCost)
	if err != nil {
		panic(err)
	}

	return string(hashedPassword)
}

func checkPassword(username string, password string) bool {
	db, err := sql.Open("sqlite3", dbLoc)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	var hash string
	username = strings.ToLower(strings.TrimSpace(username))
	err = db.QueryRow("SELECT password FROM user WHERE TRIM(LOWER(username)) = ?", username).Scan(&hash)
	if err != nil {
		panic(err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func handleLogin(ctx *gin.Context) {
	username := ctx.PostForm("username")
	password := ctx.PostForm("password")
	if checkPassword(username, password) {
		createSession(ctx, username)
		ctx.JSON(200, gin.H{"status": "Logged in!"})
		return
	}
}

func createSession(ctx *gin.Context, username string) {
	session, _ := store.Get(ctx.Request, "login-session")
	db, err := sql.Open("sqlite3", dbLoc)
	var id int
	if err != nil {
		panic(err)
	}
	err = db.QueryRow("SELECT id FROM user WHERE username = ?", username).Scan(&id)
	if err != nil {
		panic(err)
	}
	session.Values["username"] = username
	session.Values["id"] = id
	err = session.Save(ctx.Request, ctx.Writer)
	if err != nil {
		panic(err)
	}
}
