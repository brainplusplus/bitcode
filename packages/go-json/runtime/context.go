package runtime

import "time"

type Session struct {
	UserID   string   `json:"user_id"`
	Locale   string   `json:"locale"`
	TenantID string   `json:"tenant_id"`
	Groups   []string `json:"groups"`
}

type ExecutionMeta struct {
	ID        string    `json:"id"`
	Program   string    `json:"program"`
	StartedAt time.Time `json:"started_at"`
	Depth     int       `json:"depth"`
	StepCount int       `json:"step_count"`
}

func (s *Session) ToMap() map[string]any {
	return map[string]any{
		"user_id":   s.UserID,
		"locale":    s.Locale,
		"tenant_id": s.TenantID,
		"groups":    s.Groups,
	}
}

func (m *ExecutionMeta) ToMap() map[string]any {
	return map[string]any{
		"id":         m.ID,
		"program":    m.Program,
		"started_at": m.StartedAt.Format(time.RFC3339),
		"depth":      m.Depth,
		"step_count": m.StepCount,
	}
}
