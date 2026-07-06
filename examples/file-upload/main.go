package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mobentum/kern"
)

func main() {
	app := kern.Default()

	// ensure uploads directory exists
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		log.Fatal(err)
	}

	// upload form
	app.GET("/", func(c *kern.Context) {
		c.HTML(200, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>File Upload</title>
				<style>
					body { font-family: sans-serif; max-width: 600px; margin: 50px auto; }
					form { border: 2px dashed #ccc; padding: 30px; border-radius: 10px; }
					input[type="file"] { margin: 20px 0; }
					button { background: #4CAF50; color: white; padding: 10px 20px; border: none; 
							border-radius: 5px; cursor: pointer; font-size: 16px; }
					button:hover { background: #45a049; }
				</style>
			</head>
			<body>
				<h1>File Upload Example</h1>
				<form action="/upload" method="post" enctype="multipart/form-data">
					<input type="file" name="file" required>
					<button type="submit">Upload File</button>
				</form>
			</body>
			</html>
		`)
	})

	// upload handler
	app.POST("/upload", func(c *kern.Context) {
		// get file metadata
		file, err := c.File("file")
		if err != nil {
			c.JSON(400, map[string]string{"error": "No file uploaded"})
			return
		}

		// generate unique filename
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
		dst := filepath.Join("./uploads", filename)

		// save file
		if err := c.SaveFile(file, dst); err != nil {
			c.JSON(500, map[string]string{"error": "Failed to save file"})
			return
		}

		c.JSON(200, map[string]interface{}{
			"message":  "File uploaded successfully",
			"filename": filename,
			"size":     file.Size,
			"url":      "/uploads/" + filename,
		})
	})

	// serve uploaded files
	app.Static("/uploads/", "./uploads")

	log.Println("Upload server at http://localhost:8000")
	log.Fatal(app.Run("localhost:8000"))
}
