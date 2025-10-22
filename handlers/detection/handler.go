package detection

import (
    "bytes"
    "database/sql"
    "encoding/base64" 
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil" 
    "log"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "time"

    "yolo-server/models"

    "github.com/gin-gonic/gin"
)

func saveUploadedFile(fileHeader *multipart.FileHeader) (string, error) {
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	uniqueFileName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	dstPath := filepath.Join(uploadDir, uniqueFileName)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	return "/uploads/" + uniqueFileName, nil
}

func convertFileHeaderToDataURL(fileHeader *multipart.FileHeader) (string, error) {
    file, err := fileHeader.Open()
    if err != nil {
        return "", err
    }
    defer file.Close()

    bytes, err := ioutil.ReadAll(file)
    if err != nil {
        return "", err
    }

    base64Str := base64.StdEncoding.EncodeToString(bytes)
    
    contentType := http.DetectContentType(bytes)
    
    return fmt.Sprintf("data:%s;base64,%s", contentType, base64Str), nil
}

func HandleDetectAndCompare(c *gin.Context, db *sql.DB, pythonApiUrl string) {
	bomCode := c.Param("bomCode")
	imageFileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File gambar tidak ditemukan"})
		return
	}

	originalImageBase64, err := convertFileHeaderToDataURL(imageFileHeader)
    if err != nil {
        log.Printf("Gagal konversi gambar asli ke base64: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses gambar asli"})
        return
    }

	pythonResp, err := callPythonAPI(c, imageFileHeader, pythonApiUrl)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Gagal mendapatkan prediksi dari Python API", "details": err.Error()})
		return
	}

	bomItems, err := getBOMItemsByCode(db, bomCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data BOM"})
		return
	}

	comparisonResult := compareBOMAndDetections(bomItems, pythonResp.Summary)
	comparisonResult.AnnotatedImage = pythonResp.AnnotatedImage
	comparisonResult.OriginalImage = originalImageBase64
	comparisonJSON, _ := json.Marshal(comparisonResult)
	upsert := `
		INSERT INTO detection_results (bom_code, original_image, annotated_image, comparison_result_json)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (bom_code) DO UPDATE
		SET original_image = EXCLUDED.original_image,
			annotated_image = EXCLUDED.annotated_image,
			comparison_result_json = EXCLUDED.comparison_result_json,
			updated_at = NOW();
	`
	db.Exec(upsert, bomCode, comparisonResult.OriginalImage, comparisonResult.AnnotatedImage, comparisonJSON)
	c.JSON(http.StatusOK, comparisonResult)
}

func callPythonAPI(c *gin.Context, file *multipart.FileHeader, pythonApiUrl string) (*models.PythonResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	image, _ := file.Open()
	part, _ := writer.CreateFormFile("file", filepath.Base(file.Filename))
	io.Copy(part, image)
	image.Close()

	writer.Close()

	req, _ := http.NewRequest("POST", pythonApiUrl, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("python API tidak merespon: %w", err)
	}
	defer resp.Body.Close()

	var result models.PythonResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("gagal decode response dari Python: %w", err)
	}
	return &result, nil
}

func getBOMItemsByCode(db *sql.DB, bomCode string) (map[string]int, error) {
	rows, err := db.Query("SELECT part_name, quantity FROM boms WHERE bom_code = $1", bomCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make(map[string]int)
	for rows.Next() {
		var name string
		var qty int
		rows.Scan(&name, &qty)
		items[name] = qty
	}
	return items, nil
}

func compareBOMAndDetections(bomItems map[string]int, detected []models.DetectionSummary) models.ComparisonResult {
    detectedMap := make(map[string]int)
    for _, s := range detected {
        detectedMap[s.ClassName] = s.Quantity
    }

    var shortage []models.ShortageItem
    var surplus []models.SurplusItem 

    for partName, requiredQty := range bomItems {
        detectedQty, found := detectedMap[partName]

        if !found {
            shortage = append(shortage, models.ShortageItem{
                PartName: partName,
                Required: requiredQty,
                Detected: 0,
                Shortage: requiredQty,
            })
        } else if detectedQty < requiredQty {
            shortage = append(shortage, models.ShortageItem{
                PartName: partName,
                Required: requiredQty,
                Detected: detectedQty,
                Shortage: requiredQty - detectedQty,
            })
        } else if detectedQty > requiredQty {
            surplus = append(surplus, models.SurplusItem{
                PartName: partName,
                Detected: detectedQty,
                Required: requiredQty,
                Surplus:  detectedQty - requiredQty,
            })
        }
    }

    for partName, detectedQty := range detectedMap {
        _, foundInBom := bomItems[partName]
        if !foundInBom {
            surplus = append(surplus, models.SurplusItem{
                PartName: partName,
                Detected: detectedQty,
                Required: 0, 
                Surplus:  detectedQty,
            })
        }
    }

    return models.ComparisonResult{
        ShortageItems: shortage,
        SurplusItems:  surplus,
    }
}

func GetDetectionResult(c *gin.Context, db *sql.DB) {
	bomCode := c.Param("bomCode")
	var resultJSON string
	var isFinalized sql.NullBool
    
	query := "SELECT comparison_result_json, is_finalized FROM detection_results WHERE bom_code=$1"
	err := db.QueryRow(query, bomCode).Scan(&resultJSON, &isFinalized)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No detection result found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch detection result: " + err.Error()})
		return
	}

	var result models.ComparisonResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse comparison result JSON: " + err.Error()})
		return
	}

	result.IsFinalized = isFinalized.Valid && isFinalized.Bool

	c.JSON(http.StatusOK, result)
}

func ResetDetectionResult(c *gin.Context, db *sql.DB) {
	bomCode := c.Param("bomCode")

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	tx.Exec("DELETE FROM actionable_items WHERE bom_code=$1", bomCode)
	tx.Exec("DELETE FROM detection_results WHERE bom_code=$1", bomCode)
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Detection result reset successfully"})
}
