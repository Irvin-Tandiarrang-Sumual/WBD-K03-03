package health

type HealthResponse struct {
	Status    string            `json:"status" example:"ok"`
	Timestamp string            `json:"timestamp,omitempty" example:"2026-10-25T14:45:00Z"`
	Services  map[string]string `json:"services,omitempty"`
}
