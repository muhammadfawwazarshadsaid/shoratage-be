package action

import (
	"database/sql"
	"fmt" 
	"net/http"

	"yolo-server/models"

	"github.com/gin-gonic/gin"
)

func FinalizeActionItems(c *gin.Context, db *sql.DB) {
	var req models.FinalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No items provided to finalize"})
		return
	}

	bomCode := req.Items[0].BomCode
	if bomCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BomCode cannot be empty"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	stmtInsert, err := tx.Prepare("INSERT INTO actionable_items (bom_code, part_name, item_type, quantity_diff) VALUES ($1, $2, $3, $4)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare insert statement"})
		return
	}
	defer stmtInsert.Close() 
	for _, item := range req.Items {
		if item.BomCode != bomCode {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All items must belong to the same BomCode"})
			return 
		}
		_, err := stmtInsert.Exec(item.BomCode, item.PartName, item.ItemType, item.QuantityDiff)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert item: " + err.Error()})
			return 
		}
	}

	updateQuery := "UPDATE detection_results SET is_finalized = TRUE WHERE bom_code = $1"
	result, err := tx.Exec(updateQuery, bomCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update detection result status: " + err.Error()})
		return 
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		fmt.Printf("Warning: No detection_results found for bom_code %s to mark as finalized.\n", bomCode)
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Actionable items saved and detection marked as finalized successfully"})
}

func GetActionItems(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT id, bom_code, part_name, item_type, quantity_diff, status, created_at, updated_at FROM actionable_items ORDER BY status ASC, created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch actionable items"})
		return
	}
	defer rows.Close()

	var items []models.ActionableItem
	for rows.Next() {
		var item models.ActionableItem
		if err := rows.Scan(&item.ID, &item.BomCode, &item.PartName, &item.ItemType, &item.QuantityDiff, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan item"})
			return
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, items)
}

func UpdateActionItemStatus(c *gin.Context, db *sql.DB) {
	id := c.Param("id")
	var req models.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	validStatuses := map[string]bool{"BARU_MASUK": true, "DITINDAKLANJUTI": true, "SELESAI": true}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
		return
	}

	result, err := db.Exec("UPDATE actionable_items SET status = $1, updated_at = NOW() WHERE id = $2", req.Status, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check affected rows"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}
