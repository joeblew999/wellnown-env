package main

import "time"

// ChatMessage represents a message in the NATS chat
type ChatMessage struct {
	From string    `json:"from"`
	Text string    `json:"text"`
	Time time.Time `json:"time"`
}

// MonitorMessage represents a captured NATS message for the monitor
type MonitorMessage struct {
	Subject string    `json:"subject"`
	Data    string    `json:"data"`
	Size    int       `json:"size"`
	Time    time.Time `json:"time"`
}

// MonitorStats tracks statistics about monitored messages
type MonitorStats struct {
	TotalMessages int64
	SubjectsSeen  map[string]int64
	StartTime     time.Time
	LastMessage   time.Time
}

// ProcessState represents process-compose process state
type ProcessState struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	IsRunning bool   `json:"is_running"`
	Pid       int    `json:"pid"`
	Health    string `json:"health"`
	Restarts  int    `json:"restarts"`
	ExitCode  int    `json:"exit_code"`
}

// ProcessStates is the response from process-compose /processes endpoint
type ProcessStates struct {
	States []ProcessState `json:"data"`
}

// ServiceRegistration from NATS KV
type ServiceRegistration struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Time string `json:"time"`
}

// UISettings represents UI configuration stored in NATS KV
type UISettings struct {
	// CSS Variant: "regular" or "classless"
	CSSVariant string `json:"css_variant"`
	// Viewport Mode: "responsive", "centered", or "fluid"
	ViewportMode string `json:"viewport_mode"`
	// RTL Support
	RTLEnabled bool   `json:"rtl_enabled"`
	RTLLang    string `json:"rtl_lang"` // ar, he, fa, etc.
}
