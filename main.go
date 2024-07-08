package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Message struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	FileName  string    `json:"file_name"`
	FileData  []byte    `json:"-"`
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./messages.db")
	if err != nil {
		log.Fatal(err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		file_name TEXT,
		file_data BLOB
	);
	`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	defer db.Close()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, content, timestamp, file_name FROM messages ORDER BY timestamp DESC")
		if err != nil {
			c.String(http.StatusInternalServerError, "Database query error")
			return
		}
		defer rows.Close()

		messages := []Message{}
		for rows.Next() {
			var msg Message
			if err := rows.Scan(&msg.ID, &msg.Content, &msg.Timestamp, &msg.FileName); err != nil {
				c.String(http.StatusInternalServerError, "Database scan error")
				return
			}
			messages = append(messages, msg)
		}

		c.HTML(http.StatusOK, "index.html", gin.H{"messages": messages})
	})

	r.POST("/message", func(c *gin.Context) {
		content := c.PostForm("content")
		file, err := c.FormFile("file")
		var fileName string
		var fileData []byte

		if err == nil {
			src, err := file.Open()
			if err != nil {
				c.String(http.StatusInternalServerError, "File open error")
				return
			}
			defer src.Close()

			fileData, err = ioutil.ReadAll(src)
			if err != nil {
				c.String(http.StatusInternalServerError, "File read error")
				return
			}

			fileName = file.Filename
		}

		_, err = db.Exec("INSERT INTO messages (content, file_name, file_data) VALUES (?, ?, ?)", content, fileName, fileData)
		if err != nil {
			c.String(http.StatusInternalServerError, "Database insert error")
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	})

	r.GET("/download/:id", func(c *gin.Context) {
		id := c.Param("id")
		var fileName string
		var fileData []byte

		err := db.QueryRow("SELECT file_name, file_data FROM messages WHERE id = ?", id).Scan(&fileName, &fileData)
		if err != nil {
			c.String(http.StatusInternalServerError, "Database query error")
			return
		}

		c.Header("Content-Disposition", "attachment; filename="+fileName)
		c.Data(http.StatusOK, "application/octet-stream", fileData)
	})

	r.POST("/delete/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.Exec("DELETE FROM messages WHERE id = ?", id)
		if err != nil {
			c.String(http.StatusInternalServerError, "Database delete error")
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	})

	r.Run(":8080")
}
