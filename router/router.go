package router

import (
	"time"

	"github.com/gin-contrib/pprof"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"vectordb/handler"
)

func SetupRouter(mode string) *gin.Engine {
	// mode: release / debug
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(ginzap.Ginzap(zap.L(), time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(zap.L(), true))

	api := r.Group("/api")
	{
		// db
		api.GET("/info", handler.GetDBInfo)

		// collection
		api.POST("/collections", handler.CreateCollection)
		api.DELETE("/collections/:collection_name", handler.DeleteCollection)
		api.GET("/collections/:collection_name", handler.GetCollectionInfo)

		// object
		api.POST("/collections/:collection_name/objects", handler.InsertObject)
		api.POST("/collections/:collection_name/objects/batch", handler.InsertObjects)
		api.DELETE("/collections/:collection_name/objects/:object_id", handler.DeleteObject)
		api.PUT("/collections/:collection_name/objects/:object_id", handler.UpdateObject)
		api.GET("/collections/:collection_name/objects", handler.GetObjects)
		api.GET("/collections/:collection_name/objects/:object_id", handler.GetObjectInfo)
		api.POST("/collections/:collection_name/objects/search", handler.SearchObject)
	}

	// host:port/debug/pprof/
	pprof.Register(r)

	return r
}
