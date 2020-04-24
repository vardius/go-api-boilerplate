package executioncontext

// Flag type
type Flag uint8

// Set flag
func (f Flag) Set(flag Flag) Flag { return f | flag }

// Clear flag
func (f Flag) Clear(flag Flag) Flag { return f &^ flag }

// Toggle flag
func (f Flag) Toggle(flag Flag) Flag { return f ^ flag }

// Has flag
func (f Flag) Has(flag Flag) bool { return f&flag != 0 }
