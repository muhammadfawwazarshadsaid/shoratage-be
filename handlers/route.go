package handlers

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"yolo-server/handlers/action"
	"yolo-server/handlers/bom"
	"yolo-server/handlers/detection"
)

func RegisterRoutes(r *gin.RouterGroup, db *sql.DB, pythonAPI string) {
	
	// Group BOM
	bomGroup := r.Group("/boms")
	{
		bomGroup.GET("", func(c *gin.Context) { bom.GetBOMs(c, db) })
		bomGroup.POST("", func(c *gin.Context) { bom.AddBOMEntry(c, db) })
        bomGroup.POST("/upload", func(c *gin.Context) { bom.ImportBOMs(c, db) })
        bomGroup.GET("/export", func(c *gin.Context) { bom.ExportBOMs(c, db) })
		bomGroup.POST("/batch", func(c *gin.Context) { bom.AddBOMBatch(c, db) })
	}

	// Group Detection
	detectionGroup := r.Group("/detect")
	{
		detectionGroup.POST("/:bomCode", func(c *gin.Context) { detection.HandleDetectAndCompare(c, db, pythonAPI) })
		detectionGroup.GET("/:bomCode", func(c *gin.Context) { detection.GetDetectionResult(c, db) })
		detectionGroup.DELETE("/:bomCode", func(c *gin.Context) { detection.ResetDetectionResult(c, db) })
	}

	// Group Action Items
	actionGroup := r.Group("/action-items")
	{
		actionGroup.POST("", func(c *gin.Context) { action.FinalizeActionItems(c, db) })
		actionGroup.GET("", func(c *gin.Context) { action.GetActionItems(c, db) })
		actionGroup.PATCH("/:id/status", func(c *gin.Context) { action.UpdateActionItemStatus(c, db) })
	}
}
