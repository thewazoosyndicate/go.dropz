package tlv

const (
	// Packet size limits
	MaxPacketSize    = 20    // BLE packet maximum size
	MaxMessageLength = 8191  // Maximum message length we can send (13-bit)
	MaxReceiveLength = 65535 // Maximum message length we can receive (16-bit)

	// Header bit masks
	ContinuationBit = 0x80 // Bit 7: Continuation packet indicator
	HeaderTypeMask  = 0x60 // Bits 5-6: Header type (for start packets)
	GeneralLenMask  = 0x1F // Bits 0-4: 5-bit length (for general packets)
	Extended13Mask  = 0x1F // Bits 0-4: Upper 5 bits of 13-bit length
	PacketCountMask = 0x0F // Bits 0-3: Packet counter (for continuation)

	// Header types (bits 5-6 of first byte)
	HeaderGeneral    = 0x00 // 00: General (5-bit length)
	HeaderExtended13 = 0x20 // 01: Extended (13-bit length) - RECOMMENDED
	HeaderExtended16 = 0x40 // 10: Extended (16-bit length) - RECEIVE ONLY

	// Fragment collection
	DefaultTimeout  = 10 * 1000 // 10 seconds in milliseconds
	CleanupInterval = 30 * 1000 // 30 seconds cleanup interval
)
