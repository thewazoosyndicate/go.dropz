package models

import (
	"fmt"
	"time"
)

// GoPro model constants
const (
	GoProModelHERO9  = 50
	GoProModelHERO10 = 55
	GoProModelHERO11 = 62
	GoProModelHERO12 = 63
	GoProModelHERO13 = 64
)

// Enhanced pairing constants for more robust handling
const (
	MaxPairingRetries     = 3
	PairingRetryDelay     = 2 * time.Second
	ModelDetectionRetries = 3
	ReadinessCheckDelay   = 500 * time.Millisecond
)

// ModelCapabilities represents the capabilities for a specific GoPro model
type ModelCapabilities struct {
	SupportsEnhancedBLE     bool
	RequiresExtendedPairing bool
	MaxConnectionRetries    int
	PairingTimeoutMs        int
	SupportedFeatures       []string
}

// GetModelName returns the human-readable name for a GoPro model ID
func GetModelName(modelID int) string {
	switch modelID {
	case GoProModelHERO9:
		return "HERO9"
	case GoProModelHERO10:
		return "HERO10"
	case GoProModelHERO11:
		return "HERO11"
	case GoProModelHERO12:
		return "HERO12"
	case GoProModelHERO13:
		return "HERO13"
	default:
		return fmt.Sprintf("Unknown (%d)", modelID)
	}
}

// GetCapabilities returns the capabilities for a specific GoPro model
func GetCapabilities(modelID int) ModelCapabilities {
	switch modelID {
	case GoProModelHERO13:
		return ModelCapabilities{
			SupportsEnhancedBLE:     true,
			RequiresExtendedPairing: false,
			MaxConnectionRetries:    5,
			PairingTimeoutMs:        5000,
			SupportedFeatures:       []string{"auto_hibernate", "quick_pair", "enhanced_wifi"},
		}
	case GoProModelHERO12:
		return ModelCapabilities{
			SupportsEnhancedBLE:     true,
			RequiresExtendedPairing: false,
			MaxConnectionRetries:    4,
			PairingTimeoutMs:        4000,
			SupportedFeatures:       []string{"auto_hibernate", "quick_pair"},
		}
	case GoProModelHERO11:
		return ModelCapabilities{
			SupportsEnhancedBLE:     true,
			RequiresExtendedPairing: true,
			MaxConnectionRetries:    4,
			PairingTimeoutMs:        4000,
			SupportedFeatures:       []string{"auto_hibernate"},
		}
	case GoProModelHERO10:
		return ModelCapabilities{
			SupportsEnhancedBLE:     false,
			RequiresExtendedPairing: true,
			MaxConnectionRetries:    3,
			PairingTimeoutMs:        3000,
			SupportedFeatures:       []string{"basic_pairing"},
		}
	case GoProModelHERO9:
		return ModelCapabilities{
			SupportsEnhancedBLE:     false,
			RequiresExtendedPairing: true,
			MaxConnectionRetries:    3,
			PairingTimeoutMs:        3000,
			SupportedFeatures:       []string{"basic_pairing"},
		}
	default:
		return ModelCapabilities{
			SupportsEnhancedBLE:     false,
			RequiresExtendedPairing: true,
			MaxConnectionRetries:    2,
			PairingTimeoutMs:        2000,
			SupportedFeatures:       []string{"basic_pairing"},
		}
	}
}

// IsSupported checks if a given model ID is supported
func IsSupported(modelID int) bool {
	supportedModels := []int{
		GoProModelHERO9,
		GoProModelHERO10,
		GoProModelHERO11,
		GoProModelHERO12,
		GoProModelHERO13,
	}

	for _, supportedModel := range supportedModels {
		if modelID == supportedModel {
			return true
		}
	}
	return false
}

// IsValidModelID checks if the model ID is a known valid GoPro model
func IsValidModelID(modelID int) bool {
	switch modelID {
	case GoProModelHERO9, GoProModelHERO10, GoProModelHERO11, GoProModelHERO12, GoProModelHERO13:
		return true
	default:
		return false
	}
}

// ShouldUseEnhancedPairing determines if enhanced pairing should be used for a model
func ShouldUseEnhancedPairing(modelID int) bool {
	capabilities := GetCapabilities(modelID)
	return capabilities.SupportsEnhancedBLE
}

// GetPairingTimeout returns the recommended pairing timeout for a model
func GetPairingTimeout(modelID int) time.Duration {
	capabilities := GetCapabilities(modelID)
	return time.Duration(capabilities.PairingTimeoutMs) * time.Millisecond
}

// GetMaxRetries returns the maximum retry count for a model
func GetMaxRetries(modelID int) int {
	capabilities := GetCapabilities(modelID)
	return capabilities.MaxConnectionRetries
}

// ValidateModelCapabilities validates that model capabilities are reasonable
func ValidateModelCapabilities(caps ModelCapabilities) error {
	if caps.MaxConnectionRetries < 1 {
		return fmt.Errorf("invalid MaxConnectionRetries: %d (must be at least 1)", caps.MaxConnectionRetries)
	}

	if caps.MaxConnectionRetries > 10 {
		return fmt.Errorf("invalid MaxConnectionRetries: %d (should not exceed 10)", caps.MaxConnectionRetries)
	}

	if caps.PairingTimeoutMs < 1000 {
		return fmt.Errorf("invalid PairingTimeoutMs: %d (must be at least 1000ms)", caps.PairingTimeoutMs)
	}

	if caps.PairingTimeoutMs > 30000 {
		return fmt.Errorf("invalid PairingTimeoutMs: %d (should not exceed 30000ms)", caps.PairingTimeoutMs)
	}

	return nil
}

// GetCapabilitiesSafe returns validated capabilities for a model ID
func GetCapabilitiesSafe(modelID int) (ModelCapabilities, error) {
	caps := GetCapabilities(modelID)

	if err := ValidateModelCapabilities(caps); err != nil {
		return ModelCapabilities{}, fmt.Errorf("invalid capabilities for model %d: %v", modelID, err)
	}

	if !IsSupported(modelID) {
		return caps, fmt.Errorf("model %d is not officially supported", modelID)
	}

	return caps, nil
}
