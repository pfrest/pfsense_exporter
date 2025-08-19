package utils

// BoolToFloat64 converts a boolean value to a float64 (1.0 for true, 0.0 for false).
func BoolToFloat64(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
