package horizon

import (
	"encoding/json"
	"strconv"
)

type FlexInt int64

func (f *FlexInt) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "" {
			*f = 0
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*f = FlexInt(n)
		return nil
	}
	var n int64
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*f = FlexInt(n)
	return nil
}

type StatsPeriods struct {
	RecentJobs     int `json:"recentJobs"`
	RecentlyFailed int `json:"recentlyFailed"`
}

type Stats struct {
	Status                 string           `json:"status"`
	JobsPerMinute          float64          `json:"jobsPerMinute"`
	RecentlyFailed         int64            `json:"recentlyFailed"`
	Processes              int              `json:"processes"`
	WaitTime               map[string]int64 `json:"wait"`
	QueueWithMaxRuntime    string           `json:"queueWithMaxRuntime"`
	QueueWithMaxThroughput string           `json:"queueWithMaxThroughput"`
	RecentJobs             int64            `json:"recentJobs"`
	Periods                StatsPeriods     `json:"periods"`
}

type WorkloadItem struct {
	Name      string `json:"name"`
	Length    int64  `json:"length"`
	Wait      int64  `json:"wait"`
	Processes int    `json:"processes"`
}

type MasterSupervisor struct {
	Name        string       `json:"name"`
	PID         string       `json:"pid"`
	Status      string       `json:"status"`
	Supervisors []Supervisor `json:"supervisors"`
}

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
	PID       string            `json:"pid"`
	Status    string            `json:"status"`
	Processes ProcessMap        `json:"processes"`
	Options   SupervisorOptions `json:"options"`
}

type SupervisorOptions struct {
	Queue        string  `json:"queue"`
	Connection   string  `json:"connection"`
	Balance      string  `json:"balance"`
	MaxProcesses FlexInt `json:"maxProcesses"`
	MinProcesses FlexInt `json:"minProcesses"`
	Timeout      FlexInt `json:"timeout"`
	Tries        FlexInt `json:"maxTries"`
	Memory       FlexInt `json:"memory"`
}

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
