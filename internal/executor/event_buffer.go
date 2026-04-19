package executor

// EventBuffer stores a fixed-size sliding window of LogEntry values.
type EventBuffer struct {
	max int
	buf []LogEntry
}

// NewEventBuffer creates a bounded in-memory event buffer.
func NewEventBuffer(max int) *EventBuffer {
	if max < 1 {
		max = 1
	}
	return &EventBuffer{max: max, buf: make([]LogEntry, 0, max)}
}

// Append adds an entry and evicts oldest entries beyond configured size.
func (b *EventBuffer) Append(entry LogEntry) {
	b.buf = append(b.buf, entry)
	if len(b.buf) > b.max {
		start := len(b.buf) - b.max
		out := make([]LogEntry, b.max)
		copy(out, b.buf[start:])
		b.buf = out
	}
}

// Snapshot returns a copy of the currently buffered entries.
func (b *EventBuffer) Snapshot() []LogEntry {
	out := make([]LogEntry, len(b.buf))
	copy(out, b.buf)
	return out
}
