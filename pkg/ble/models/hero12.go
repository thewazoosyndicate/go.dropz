package models

// Hero12Handler handles Hero 12 specific operations
type Hero12Handler struct {
	BaseHandler
}

// Hero12 has improved BLE capabilities
func (h *Hero12Handler) GetCapabilities() ModelCapabilities {
	return ModelCapabilities{
		SupportsEnhancedBLE:     true,
		RequiresExtendedPairing: false,
		MaxConnectionRetries:    4,
		PairingTimeoutMs:        4000,
		SupportedFeatures:       []string{"auto_hibernate", "quick_pair"},
	}
}

// Hero13Handler handles Hero 13 specific operations
type Hero13Handler struct {
	BaseHandler
}

// Hero13 has the latest BLE capabilities
func (h *Hero13Handler) GetCapabilities() ModelCapabilities {
	return ModelCapabilities{
		SupportsEnhancedBLE:     true,
		RequiresExtendedPairing: false,
		MaxConnectionRetries:    5,
		PairingTimeoutMs:        5000,
		SupportedFeatures:       []string{"auto_hibernate", "quick_pair", "enhanced_wifi"},
	}
}

// Hero10Handler handles Hero 10 specific operations
type Hero10Handler struct {
	BaseHandler
}

// Hero10 has basic BLE capabilities
func (h *Hero10Handler) GetCapabilities() ModelCapabilities {
	return ModelCapabilities{
		SupportsEnhancedBLE:     false,
		RequiresExtendedPairing: true,
		MaxConnectionRetries:    3,
		PairingTimeoutMs:        3000,
		SupportedFeatures:       []string{"basic_pairing"},
	}
}

// Hero9Handler handles Hero 9 specific operations
type Hero9Handler struct {
	BaseHandler
}

// Hero9 has basic BLE capabilities
func (h *Hero9Handler) GetCapabilities() ModelCapabilities {
	return ModelCapabilities{
		SupportsEnhancedBLE:     false,
		RequiresExtendedPairing: true,
		MaxConnectionRetries:    3,
		PairingTimeoutMs:        3000,
		SupportedFeatures:       []string{"basic_pairing"},
	}
}
