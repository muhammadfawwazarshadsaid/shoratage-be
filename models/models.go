package models

import "time"

type BOMEntry struct {
	ID              int    `json:"id"`
	BomCode         string `json:"bomCode"`
	PartReference   string `json:"partReference"`
	PartName        string `json:"partName"`
	PartDescription string `json:"partDescription"`
	Quantity        int    `json:"quantity"`
}

type BOMEntryWithStatus struct {
	BOMEntry          
	HasDetectionResult bool `json:"hasDetectionResult"`
	IsFinalized        bool `json:"isFinalized"`
}

type DetectionSummary struct {
	ClassName     string  `json:"class_name"`
	Quantity      int     `json:"quantity"`
	AvgConfidence float64 `json:"avg_confidence"`
}

type PythonResponse struct {
	Summary        []DetectionSummary `json:"summary"`
	AnnotatedImage string             `json:"annotated_image"`
}

type ShortageItem struct {
	PartName string `json:"partName"`
	Required int    `json:"required"`
	Detected int    `json:"detected"`
	Shortage int    `json:"shortage"`
}

type SurplusItem struct {
	PartName string `json:"partName"`
	Detected int    `json:"detected"`
	Required int    `json:"required"`
	Surplus  int    `json:"surplus"`
}

type ComparisonResult struct {
	ShortageItems  []ShortageItem `json:"shortageItems"`
	SurplusItems   []SurplusItem  `json:"surplusItems"`
	OriginalImage  string         `json:"originalImage"`
	AnnotatedImage string         `json:"annotatedImage"`
	IsFinalized    bool           `json:"isFinalized"`
}

type ActionableItem struct {
	ID           int       `json:"id"`
	BomCode      string    `json:"bomCode"`
	PartName     string    `json:"partName"`
	ItemType     string    `json:"itemType"`
	QuantityDiff int       `json:"quantityDiff"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type FinalizeRequest struct {
	Items []ActionableItem `json:"items"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}
