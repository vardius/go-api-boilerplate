package executioncontext

// Execution context flags
const (
	LIVE   Flag = 1 << iota // live events handling
	REPLAY                  // replay events handling
)

// Flag type
type Flag uint8

// set flag
func (f Flag) set(flag Flag) Flag { return f | flag }

// clear flag
func (f Flag) clear(flag Flag) Flag { return f &^ flag }

// toggle flag
func (f Flag) toggle(flag Flag) Flag { return f ^ flag }

// has flag
func (f Flag) has(flag Flag) bool { return f&flag != 0 }
