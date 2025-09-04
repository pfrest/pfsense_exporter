package utils

import (
	"testing"
)

func TestBoolToFloat64(t *testing.T) {
	// Test true case
	result := BoolToFloat64(true)
	if result != 1.0 {
		t.Errorf("Expected 1.0 for true, got %f", result)
	}

	// Test false case
	result = BoolToFloat64(false)
	if result != 0.0 {
		t.Errorf("Expected 0.0 for false, got %f", result)
	}
}
