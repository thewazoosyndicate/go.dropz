package models

import (
	"fmt"
	"time"
)

// ModelHandler defines the interface for model-specific behaviors
type ModelHandler interface {
	GetCapabilities() ModelCapabilities
	ShouldUseEnhancedPairing() bool
	GetPairingTimeout() time.Duration
	GetMaxRetries() int
	IsSupported() bool
	GetName() string
}

// BaseHandler provides common functionality for all models
type BaseHandler struct {
	ModelID int
}

func (h *BaseHandler) GetCapabilities() ModelCapabilities {
	return GetCapabilities(h.ModelID)
}

func (h *BaseHandler) ShouldUseEnhancedPairing() bool {
	return ShouldUseEnhancedPairing(h.ModelID)
}

func (h *BaseHandler) GetPairingTimeout() time.Duration {
	return GetPairingTimeout(h.ModelID)
}

func (h *BaseHandler) GetMaxRetries() int {
	return GetMaxRetries(h.ModelID)
}

func (h *BaseHandler) IsSupported() bool {
	return IsSupported(h.ModelID)
}

func (h *BaseHandler) GetName() string {
	return GetModelName(h.ModelID)
}

// CreateHandler creates a model-specific handler for the given model ID with validation
func CreateHandler(modelID int) (ModelHandler, error) {
	// Validate model ID
	if modelID <= 0 {
		return nil, fmt.Errorf("invalid model ID: %d (must be positive)", modelID)
	}

	switch modelID {
	case GoProModelHERO11:
		return &Hero11Handler{BaseHandler: BaseHandler{ModelID: modelID}}, nil
	case GoProModelHERO12:
		return &Hero12Handler{BaseHandler: BaseHandler{ModelID: modelID}}, nil
	case GoProModelHERO13:
		return &Hero13Handler{BaseHandler: BaseHandler{ModelID: modelID}}, nil
	case GoProModelHERO10:
		return &Hero10Handler{BaseHandler: BaseHandler{ModelID: modelID}}, nil
	case GoProModelHERO9:
		return &Hero9Handler{BaseHandler: BaseHandler{ModelID: modelID}}, nil
	default:
		// Log warning but still return a handler for unknown models
		return &BaseHandler{ModelID: modelID}, fmt.Errorf("unsupported GoPro model ID: %d (using base handler)", modelID)
	}
}

// CreateHandlerSafe creates a model-specific handler with fallback to base handler
// This version never returns an error but logs warnings for unknown models
func CreateHandlerSafe(modelID int) ModelHandler {
	handler, err := CreateHandler(modelID)
	if err != nil {
		// For unknown models, just return the base handler without error
		// The caller can check IsSupported() if needed
		if modelID > 0 {
			return &BaseHandler{ModelID: modelID}
		}
		// For invalid model IDs, return a default base handler
		return &BaseHandler{ModelID: 0}
	}
	return handler
}
