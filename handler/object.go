package handler

import (
	"net/http"
	"vectordb/db"
	"vectordb/model"

	"github.com/gin-gonic/gin"
)

func InsertObject(c *gin.Context) {
	col := c.Param("collection_name")
	obj := new(model.ReqInsertObject)
	if err := c.ShouldBindJSON(obj); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	id, err := db.QueryInsertObject(col, obj)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "object inserted",
		"data": gin.H{
			"id": id,
		},
	})
}

func InsertObjects(c *gin.Context) {
	col := c.Param("collection_name")
	objs := new(model.ReqInsertObjects)
	if err := c.ShouldBindJSON(objs); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ids, err := db.QueryInsertObjects(col, objs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "objects inserted",
		"data": gin.H{
			"ids": ids,
		},
	})
}

func DeleteObject(c *gin.Context) {
	col := c.Param("collection_name")
	obj := c.Param("object_id")

	if err := db.QueryDeleteObject(col, obj); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "object deleted",
	})
}

func UpdateObject(c *gin.Context) {
	col := c.Param("collection_name")
	obj := new(model.ReqUpdateObject)
	if err := c.ShouldBindJSON(obj); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := db.QueryUpdateObject(col, obj); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "object updated",
	})
}

func GetObjects(c *gin.Context) {
	col := c.Param("collection_name")
	req := new(model.ReqGetObjects)
	if err := c.ShouldBindJSON(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res, err := db.QueryGetObjects(col, req.Offset, req.Limit)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "selected objects got",
		"data":    res,
	})
}

func GetObjectInfo(c *gin.Context) {
	col := c.Param("collection_name")
	obj := c.Param("object_id")

	res, err := db.QueryGetObjectInfo(col, obj)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "object info got",
		"data":    res,
	})
}

func SearchObject(c *gin.Context) {
	col := c.Param("collection_name")
	obj := new(model.ReqSearchObject)
	if err := c.ShouldBindJSON(obj); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res, err := db.QuerySearchObject(col, obj)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "similar objects retrieved",
		"data":    res,
	})
}
