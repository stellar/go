package health

// Status indicates whether the service is health or not.
type Status string

const (
	// StatusPass indicates that the service is health.
	StatusPass Status = "pass"
)
