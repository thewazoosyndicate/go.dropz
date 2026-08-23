package tlv

import (
	"log/slog"
	"sync"
	"time"
)

// HeaderType represents the TLV packet header type
type HeaderType int

const (
	// HeaderTypeGeneral is for 5-bit length messages (≤31 bytes)
	HeaderTypeGeneral HeaderType = iota
	// HeaderTypeExtended13 is for 13-bit length messages (≤8191 bytes) - RECOMMENDED for sending
	HeaderTypeExtended13
	// HeaderTypeExtended16 is for 16-bit length messages - RECEIVE ONLY (not for sending to camera)
	HeaderTypeExtended16
)

// PacketHeader represents a parsed TLV packet header
type PacketHeader struct {
	IsStart        bool       // True if this is a start packet
	IsContinuation bool       // True if this is a continuation packet
	PacketType     HeaderType // Type of packet (only valid for start packets)
	MessageLength  int        // Total message length (only valid for start packets)
	PacketCounter  int        // Packet counter (only valid for continuation packets)
	HeaderSize     int        // Number of bytes consumed by the header
}

// TLVMessage represents a complete reassembled TLV message
type TLVMessage struct {
	CommandID byte      // Command/Query/Setting ID
	Status    byte      // Response status
	Payload   []byte    // Message payload (after command ID and status)
	Timestamp time.Time // When the message was completed
}

// MessageFragments tracks fragments for a single message being reassembled
type MessageFragments struct {
	commandID      byte
	expectedLength int            // Total expected message length
	receivedLength int            // Bytes received so far
	packets        map[int][]byte // packetCounter -> packet data
	lastPacketNum  int            // Highest packet number seen
	startTime      time.Time      // When first fragment was received
}

// FragmentCollector manages message fragment collection and reassembly
type FragmentCollector struct {
	fragments map[byte]*MessageFragments // commandID -> fragments
	mu        sync.RWMutex
	timeout   time.Duration
	stopCh    chan struct{}
	log       *slog.Logger // nil = silent; set via SetLogger
}

// ResponseTracker tracks pending command responses
type ResponseTracker struct {
	pendingCommands map[byte]chan *TLVMessage
	pushHandler     func(string, *TLVMessage)
	mu              sync.RWMutex
	log             *slog.Logger // nil = silent; set via SetLogger
}
