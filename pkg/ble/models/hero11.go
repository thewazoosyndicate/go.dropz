package models

// Hero11Handler handles Hero 11 specific operations
type Hero11Handler struct {
	BaseHandler
}

// Hero11 has specific enhanced BLE requirements
func (h *Hero11Handler) GetCapabilities() ModelCapabilities {
	return ModelCapabilities{
		SupportsEnhancedBLE:     true,
		RequiresExtendedPairing: true,
		MaxConnectionRetries:    4,
		PairingTimeoutMs:        4000,
		SupportedFeatures:       []string{"auto_hibernate"},
	}
}
