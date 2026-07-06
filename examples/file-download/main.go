package main

import (
	"fmt"
	"log"

	"github.com/mobentum/kern"
)

func main() {
	app := kern.Default()

	app.GET("/", func(c *kern.Context) {
		_ = c.HTML(200, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>File Download</title>
				<style>
					body { font-family: sans-serif; max-width: 600px; margin: 50px auto; }
					.btn { display: inline-block; padding: 10px 20px; margin: 5px;
						   background: #2196F3; color: white; text-decoration: none;
						   border-radius: 5px; }
					.btn:hover { background: #1976D2; }
				</style>
			</head>
			<body>
				<h1>File Download Examples</h1>
				<h2>Download Files</h2>
				<a href="/download/test.txt" class="btn">Download test.txt</a>
				<a href="/download/sample.pdf" class="btn">Download sample.pdf</a>
				
				<h2>Stream Video</h2>
				<video controls width="400">
					<source src="/stream/video.mp4" type="video/mp4">
					Your browser does not support video.
				</video>
				
				<h2>Static Files</h2>
				<a href="/files/readme.md" class="btn">View readme.md</a>
			</body>
			</html>
		`)
	})

	app.GET("/download/{filename}", func(c *kern.Context) {
		filename := c.Param("filename")
		err := c.DownloadFile("./files/"+filename, filename)
		if err != nil {
			_ = c.Text(404, "File not found: %s", err.Error())
		}
	})

	app.GET("/stream/{filename}", func(c *kern.Context) {
		filename := c.Param("filename")
		err := c.StreamFile("./files/" + filename)
		if err != nil {
			_ = c.Text(404, "File not found: %s", err.Error())
		}
	})

	app.GET("/files/{path...}", func(c *kern.Context) {
		err := c.ServeStatic("./files")
		if err != nil {
			_ = c.Text(404, "File not found")
		}
	})

	fmt.Println("Download server at http://localhost:8000")
	log.Fatal(app.Run("localhost:8000"))
}
