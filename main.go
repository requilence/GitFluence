package main

import (
	"net/http"
	"worker"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/rs", func(c *gin.Context) {
		url, _ := c.GetQuery("url")
		rs, err := worker.GetRepoStat(url)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		c.JSON(200, rs)
	})

	r.Run(":8080")
}
