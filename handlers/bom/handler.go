package bom

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"      
	"log"   
	"net/http"
	"strconv"
	"time"

	"yolo-server/models"

	"github.com/gin-gonic/gin"
)

func GetBOMs(c *gin.Context, db *sql.DB) {
	query := `
		SELECT
			b.id, b.bom_code, b.part_reference, b.part_name, b.part_description, b.quantity,
			CASE WHEN dr.bom_code IS NOT NULL THEN TRUE ELSE FALSE END AS has_detection_result,
			COALESCE(ai.is_finalized, FALSE) AS is_finalized
		FROM boms b
		LEFT JOIN detection_results dr ON b.bom_code = dr.bom_code
		LEFT JOIN (
			-- Check if there are ANY 'SELESAI' items for a bom_code to consider it finalized (adjust logic if needed)
			-- Or maybe check if ALL items are 'SELESAI'? This depends on your definition.
			-- Simpler check: If ANY action item exists, maybe it's finalized? Let's check detection_results instead for simplicity now.
			-- Let's assume finalized status is stored or inferred differently.
			-- For now, let's just use a placeholder for is_finalized or derive it simply.
			-- A common approach is to add an is_finalized column to detection_results table itself.
			-- Let's query if ANY related action item exists as a proxy for finalized (can be refined).
			SELECT DISTINCT bom_code, TRUE as is_finalized FROM actionable_items
			-- Or a better approach: add is_finalized BOOLEAN to detection_results table
			-- SELECT bom_code, is_finalized FROM detection_results
		) ai ON b.bom_code = ai.bom_code
		ORDER BY b.bom_code, b.part_name
	`
	rows, err := db.Query(query)

	if err != nil {
		log.Printf("Error querying BOMs with status: %v", err) 
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data BOM"})
		return
	}
	defer rows.Close()

	bomMap := make(map[string][]models.BOMEntryWithStatus)
	bomStatusMap := make(map[string]struct{ HasResult bool; IsFinalized bool })

	for rows.Next() {
		var bom models.BOMEntryWithStatus
		var hasResult, isFinalized sql.NullBool 
		if err := rows.Scan(
			&bom.ID, &bom.BomCode, &bom.PartReference, &bom.PartName,
			&bom.PartDescription, &bom.Quantity,
			&hasResult, &isFinalized, 
		); err != nil {
			log.Printf("Error scanning BOM row: %v", err) 
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memindai baris data BOM"})
			return
		}
		
		bom.HasDetectionResult = hasResult.Valid && hasResult.Bool
		bom.IsFinalized = isFinalized.Valid && isFinalized.Bool

		if _, exists := bomStatusMap[bom.BomCode]; !exists {
			bomStatusMap[bom.BomCode] = struct{ HasResult bool; IsFinalized bool }{
				HasResult: bom.HasDetectionResult,
				IsFinalized: bom.IsFinalized,
			}
		}

		bomMap[bom.BomCode] = append(bomMap[bom.BomCode], bom)
	}

	var finalBoms []models.BOMEntryWithStatus
	bomCodes := []string{}
	for code := range bomMap {
		bomCodes = append(bomCodes, code)
	}

	for _, code := range bomCodes { 
		entries := bomMap[code]
		status := bomStatusMap[code]
		for i := range entries {
			entries[i].HasDetectionResult = status.HasResult
			entries[i].IsFinalized = status.IsFinalized
			finalBoms = append(finalBoms, entries[i])
		}
	}


	c.JSON(http.StatusOK, finalBoms) 
}

func AddBOMEntry(c *gin.Context, db *sql.DB) {
	var entry models.BOMEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Input tidak valid: " + err.Error()})
		return
	}

	if entry.BomCode == "" || entry.PartName == "" || entry.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field BomCode, PartName, dan Quantity (harus > 0) wajib diisi"})
		return
	}

	query := `
        INSERT INTO boms (bom_code, part_reference, part_name, part_description, quantity)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `
	err := db.QueryRow(
		query,
		entry.BomCode,
		entry.PartReference,
		entry.PartName,
		entry.PartDescription,
		entry.Quantity,
	).Scan(&entry.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan ke database: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

func ImportBOMs(c *gin.Context, db *sql.DB) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	_, err = reader.Read()
	if err == io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File CSV kosong"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca header CSV: " + err.Error()})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}
	defer tx.Rollback() 

	stmt, err := tx.Prepare(`
        INSERT INTO boms (bom_code, part_reference, part_name, part_description, quantity)
        VALUES ($1, $2, $3, $4, $5)
    `)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyiapkan statement database"})
		return
	}
	defer stmt.Close()

	rowCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break 
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca baris CSV: " + err.Error()})
			return
		}

		if len(record) < 5 {
			log.Printf("Baris tidak valid, dilewati: %v", record)
			continue 
		}

		quantity, err := strconv.Atoi(record[4])
		if err != nil || quantity <= 0 {
			log.Printf("Quantity tidak valid, dilewati: %s", record[4])
			continue 
		}
		
		bomCode := record[0]
		partRef := record[1]
		partName := record[2]
		partDesc := record[3]

		if bomCode == "" || partName == "" {
			log.Printf("Data tidak valid, dilewati: %s, %s", bomCode, partName)
			continue 
		}

		_, err = stmt.Exec(bomCode, partRef, partName, partDesc, quantity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan baris: " + err.Error()})
			return
		}
		rowCount++
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal commit transaksi"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": fmt.Sprintf("Import sukses! %d baris berhasil ditambahkan.", rowCount),
	})
}

func ExportBOMs(c *gin.Context, db *sql.DB) {
	rows, err := db.Query("SELECT bom_code, part_reference, part_name, part_description, quantity FROM boms ORDER BY bom_code, part_name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data BOM"})
		return
	}
	defer rows.Close()

	fileName := fmt.Sprintf("boms_export_%s.csv", time.Now().Format("20060102_150405"))

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename="+fileName)

	writer := csv.NewWriter(c.Writer)

	headers := []string{"bom_code", "part_reference", "part_name", "part_description", "quantity"}
	if err := writer.Write(headers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menulis header CSV"})
		return
	}

	for rows.Next() {
		var bom models.BOMEntry
		var record []string

		if err := rows.Scan(&bom.BomCode, &bom.PartReference, &bom.PartName, &bom.PartDescription, &bom.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memindai baris data BOM"})
			return
		}

		record = append(record, bom.BomCode)
		record = append(record, bom.PartReference)
		record = append(record, bom.PartName)
		record = append(record, bom.PartDescription)
		record = append(record, fmt.Sprintf("%d", bom.Quantity))

		if err := writer.Write(record); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menulis baris CSV"})
			return
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan penulisan CSV"})
	}
}

func AddBOMBatch(c *gin.Context, db *sql.DB) {
	var entries []models.BOMEntry
	if err := c.ShouldBindJSON(&entries); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Input tidak valid: " + err.Error()})
		return
	}

	if len(entries) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada entri untuk ditambahkan"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT INTO boms (bom_code, part_reference, part_name, part_description, quantity)
        VALUES ($1, $2, $3, $4, $5)
    `)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyiapkan statement database"})
		return
	}
	defer stmt.Close()

	for _, entry := range entries {
		if entry.BomCode == "" || entry.PartName == "" || entry.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Entri tidak valid: Part '%s' harus memiliki BomCode, PartName, dan Quantity > 0", entry.PartName),
			})
			return
		}

		_, err := stmt.Exec(
			entry.BomCode,
			entry.PartReference,
			entry.PartName,
			entry.PartDescription,
			entry.Quantity,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan entri: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan (commit) transaksi"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("Berhasil menambahkan %d entri BOM", len(entries)),
	})
}