package model

import (
	"encoding/json"
	"testing"
)

func TestDMABoundaryResponseIncludesDMAName(t *testing.T) {
	dmaName := "DMA Zone One"
	encoded, err := json.Marshal(SuccessResponse(DMABoundary{DmaName: &dmaName}))
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	var response struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if got := response.Data["dma_name"]; got != dmaName {
		t.Fatalf("expected dma_name %q, got %#v", dmaName, got)
	}
}

func TestDMABoundaryResponseIncludesNullDMAName(t *testing.T) {
	encoded, err := json.Marshal(SuccessResponse(DMABoundary{}))
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	var response struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	dmaName, exists := response.Data["dma_name"]
	if !exists {
		t.Fatal("expected dma_name key to be present")
	}
	if dmaName != nil {
		t.Fatalf("expected dma_name to be null, got %#v", dmaName)
	}
}
