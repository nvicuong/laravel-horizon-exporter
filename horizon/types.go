package horizon

import "encoding/json"

type Stats struct {
	Status                 string           `json:"status"`
	JobsPerMinute          float64          `json:"jobsPerMinute"`
	JobsPerHour            float64          `json:"jobsPerHour"`
	FailedJobs             int64            `json:"failedJobs"`
	Processes              int              `json:"processes"`
	WaitTime               map[string]int64 `json:"wait"`
	QueueWithMaxRuntime    string           `json:"queueWithMaxRuntime"`
	QueueWithMaxThroughput string           `json:"queueWithMaxThroughput"`
	RecentJobs             int64            `json:"recentJobs"`
	PausedMasters          int              `json:"pausedMasters"`
}

type WorkloadItem struct {
	Name      string `json:"name"`
	Length    int64  `json:"length"`
	Wait      int64  `json:"wait"`
	Processes int    `json:"processes"`
}

type MasterSupervisor struct {
	Name        string       `json:"name"`
	Environment string       `json:"environment"`
	PID         string       `json:"pid"`
	Status      string       `json:"status"`
	Supervisors []Supervisor `json:"supervisors"`
}

// ProcessMap handles both {"redis:queue": 1} and [] (inactive supervisors).
type ProcessMap map[string]int

func (p *ProcessMap) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		*p = ProcessMap{}
		return nil
	}
	var m map[string]int
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*p = ProcessMap(m)
	return nil
}

type Supervisor struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Processes ProcessMap        `json:"processes"`
	Options   SupervisorOptions `json:"options"`
}

type SupervisorOptions struct {
	Queue        string `json:"queue"`
	Connection   string `json:"connection"`
	Balance      string `json:"balance"`
	MaxProcesses string `json:"maxProcesses"`
	MinProcesses string `json:"minProcesses"`
	Timeout      string `json:"timeout"`
	Tries        string `json:"maxTries"`
}

// JobEntry is a single job from the pending/completed/silenced/failed list endpoints.
type JobEntry struct {
	ID         string `json:"id"`
	Connection string `json:"connection"`
	Queue      string `json:"queue"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Index      int64  `json:"index"`
}

type JobListResponse struct {
	Jobs  []JobEntry `json:"jobs"`
	Total int64      `json:"total"`
}

// JobCounts holds per-queue and per-class counts aggregated from a job list.
type JobCounts struct {
	Total   int64
	ByQueue map[string]int64
	ByClass map[string]int64
}

type MonitoredTag struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type Batch struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	TotalJobs   int64   `json:"totalJobs"`
	PendingJobs int64   `json:"pendingJobs"`
	FailedJobs  int64   `json:"failedJobs"`
	Progress    float64 `json:"progress"`
	CreatedAt   int64   `json:"createdAt"`
	CancelledAt *int64  `json:"cancelledAt"`
	FinishedAt  *int64  `json:"finishedAt"`
}

type BatchResponse struct {
	Batches []Batch `json:"batches"`
}

type MetricSnapshot struct {
	Throughput float64 `json:"throughput"`
	Runtime    float64 `json:"runtime"`
	Wait       int64   `json:"wait"`
	Time       int64   `json:"time"`
}
