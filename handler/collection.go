package handler

import (
	"net/http"
	"vectordb/db"
	"vectordb/model"

	"github.com/gin-gonic/gin"
)

func GetDBInfo(c *gin.Context) {
	res, err := db.QueryGetDBInfo()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "db info got",
		"data":    res,
	})
}

func CreateCollection(c *gin.Context) {
	col := new(model.ReqCreateCollection)
	if err := c.ShouldBindJSON(col); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := db.QueryCreateCollection(col); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "collection created",
	})
}

func DeleteCollection(c *gin.Context) {
	col := c.Param("collection_name")

	if err := db.QueryDeleteCollection(col); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "collection deleted",
	})
}

func GetCollectionInfo(c *gin.Context) {
	col := c.Param("collection_name")

	res, err := db.QueryGetCollectionInfo(col)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "collection info got",
		"data":    res,
	})
}
