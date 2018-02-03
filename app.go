package main

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/acme/autocert"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
const savePath = "./public/"
const authKey = "generateSomeKey"
const fileSize = 50 //Max file size in MB

const (
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
	letterIdxMax  = 63 / letterIdxBits
)

var src = rand.NewSource(time.Now().UnixNano())

var supportedTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/gif":  true,
	"image/png":  true,
	"text/plain": true,
	"video/mp4": true,
	"video/webm": true,
	"application/zip": true,
}

// Not mine but seem to be fastest way to generate random string
func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func Upload(c echo.Context) error {
	req := c.Request()
	// Auth
	auth := req.Header.Get("Authorization")
	if auth != authKey {
		return c.String(http.StatusForbidden, "Authorization failed")
	}

	// Set max file size
	err := req.ParseMultipartForm(fileSize << 20)
	if err != nil {
		return c.String(http.StatusForbidden, "File size is too big")
	}

	// Load the file
	file, header, err := req.FormFile("file")
	if err != nil {
		return c.String(http.StatusBadRequest, "Bad Request")
	}
	defer file.Close()

	// To read the file type
	buff := make([]byte, 512)
	_, err = file.Read(buff)
	file.Seek(0, 0)
	fileType := http.DetectContentType(buff)

	// Check supported types
	if _, ok := supportedTypes[fileType]; !ok {
		return c.String(http.StatusForbidden, "Unsupported file type")
	}

	extension := filepath.Ext(header.Filename)
	fileName := RandStringBytesMaskImprSrc(5) + extension


	dst, err := os.Create(savePath + fileName)
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		return c.String(http.StatusBadRequest, "Bad Request")
	}

	return c.String(http.StatusCreated, fmt.Sprintf("https://domain.com/%s", fileName))

}

func showImage(c echo.Context) error {
	path := c.Param("id")
	return c.File(savePath + path)
}

func main() {
	e := echo.New()
	e.Pre(middleware.HTTPSRedirect())

	e.AutoTLSManager.HostPolicy = autocert.HostWhitelist("domain.com", "www.domain.com")
	e.AutoTLSManager.Cache = autocert.DirCache("/var/www/.cache")
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	e.File("/", "index.html")
	e.File("/favicon.ico", "favicon.ico")
	e.GET("/:id", showImage)
	e.POST("/upload", Upload)

	go func(c *echo.Echo) {
		e.Logger.Fatal(e.Start(":80"))
	}(e)
	e.Logger.Fatal(e.StartAutoTLS(":443"))

}
