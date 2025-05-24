package models

import "time"

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

// CreateHandler creates a model-specific handler for the given model ID
func CreateHandler(modelID int) ModelHandler {
	switch modelID {
	case GoProModelHERO11:
		return &Hero11Handler{BaseHandler: BaseHandler{ModelID: modelID}}
	case GoProModelHERO12:
		return &Hero12Handler{BaseHandler: BaseHandler{ModelID: modelID}}
	case GoProModelHERO13:
		return &Hero13Handler{BaseHandler: BaseHandler{ModelID: modelID}}
	case GoProModelHERO10:
		return &Hero10Handler{BaseHandler: BaseHandler{ModelID: modelID}}
	case GoProModelHERO9:
		return &Hero9Handler{BaseHandler: BaseHandler{ModelID: modelID}}
	default:
		return &BaseHandler{ModelID: modelID}
	}
}
