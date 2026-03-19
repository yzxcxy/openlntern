package contracts

// RetrievedMemory is the provider-agnostic retrieval result exposed to upper layers.
type RetrievedMemory struct {
	Content string
	Score   float64
}

// SyncMessage is the provider-agnostic plain-text message unit used during long-term memory sync.
type SyncMessage struct {
	MsgID   string
	Role    string
	Content string
}
