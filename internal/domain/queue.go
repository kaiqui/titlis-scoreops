package domain

import "time"

type QueueSnapshot struct {
	Provider    string
	ExternalID  string
	DisplayName string
	ProjectID   string
	TopicID     string
	IsDLQ       bool
	TenantID    int64

	NumUndeliveredMessages   int64
	OldestUnackedAgeSec      int64
	PullMessageCountRate     float64
	SendMessageCountRate     float64
	AckMessageCountRate      float64
	DeadLetterMessageCount   int64
	MessageRetentionDurationSec int64

	HasDLQConfigured  bool
	HasSnapshotPolicy bool
	Labels            map[string]string

	HasMonitorBacklog bool
	HasMonitorAge     bool
	HasMonitorDLQ     bool

	CollectedAt time.Time
}

type QueueThresholds struct {
	BacklogWarning  int64     `json:"backlogWarning"`
	BacklogCritical int64     `json:"backlogCritical"`
	AgeWarningSec   int64     `json:"ageWarningSec"`
	AgeCriticalSec  int64     `json:"ageCriticalSec"`
	P50Backlog      int64     `json:"p50Backlog"`
	P75Backlog      int64     `json:"p75Backlog"`
	P95Backlog      int64     `json:"p95Backlog"`
	P50AgeSec       int64     `json:"p50AgeSec"`
	P75AgeSec       int64     `json:"p75AgeSec"`
	P95AgeSec       int64     `json:"p95AgeSec"`
	CalculatedAt    time.Time `json:"calculatedAt"`
}

// LabelRegistry maps label key → allowed values for the tenant.
type LabelRegistry map[string][]string

// ContainsValue returns true when the registry has the key and value is in the allowed set.
func (r LabelRegistry) ContainsValue(key, value string) bool {
	vals, ok := r[key]
	if !ok || len(vals) == 0 {
		return false
	}
	for _, v := range vals {
		if v == value {
			return true
		}
	}
	return false
}
