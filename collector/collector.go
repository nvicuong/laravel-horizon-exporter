package collector

import (
	"log"
	"sync"

	"github.com/horizon-exporter/horizon"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "horizon"

type Collector struct {
	client *horizon.Client
	mu     sync.Mutex

	// Core stats
	up            *prometheus.Desc
	status        *prometheus.Desc
	jobsPerMinute *prometheus.Desc
	jobsPerHour   *prometheus.Desc
	processes     *prometheus.Desc
	recentJobs    *prometheus.Desc
	pausedMasters *prometheus.Desc

	// Workload per queue
	queueLength    *prometheus.Desc
	queueWait      *prometheus.Desc
	queueProcesses *prometheus.Desc

	// Metric snapshots per queue
	queueThroughput *prometheus.Desc
	queueRuntime    *prometheus.Desc
	queueWaitSnap   *prometheus.Desc

	// Metric snapshots per job class
	jobThroughput *prometheus.Desc
	jobRuntime    *prometheus.Desc

	// Master / supervisor topology
	masterStatus        *prometheus.Desc
	supervisorStatus    *prometheus.Desc
	supervisorProcesses *prometheus.Desc

	// Pending jobs
	pendingTotal      *prometheus.Desc
	pendingByQueue    *prometheus.Desc
	pendingByClass    *prometheus.Desc

	// Completed jobs
	completedTotal    *prometheus.Desc
	completedByQueue  *prometheus.Desc
	completedByClass  *prometheus.Desc

	// Silenced jobs
	silencedTotal     *prometheus.Desc
	silencedByQueue   *prometheus.Desc
	silencedByClass   *prometheus.Desc

	// Failed jobs
	failedTotal       *prometheus.Desc
	failedByQueue     *prometheus.Desc
	failedByClass     *prometheus.Desc

	// Monitored tags
	monitoredTagJobs *prometheus.Desc

	// Batches
	batchTotal         *prometheus.Desc
	batchPendingJobs   *prometheus.Desc
	batchFailedJobs    *prometheus.Desc
	batchProgress      *prometheus.Desc
	batchCancelled     *prometheus.Desc
}

func New(client *horizon.Client) *Collector {
	d := func(subsystem, name, help string, labels ...string) *prometheus.Desc {
		return prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, name),
			help, labels, nil,
		)
	}

	return &Collector{
		client: client,

		up:            d("", "up", "1 if the Horizon API is reachable, 0 otherwise"),
		status:        d("", "status", "Horizon status: 1=running, 0=paused/inactive"),
		jobsPerMinute: d("", "jobs_per_minute", "Jobs processed per minute"),
		jobsPerHour:   d("", "jobs_per_hour", "Jobs processed per hour"),
		processes:     d("", "processes", "Total worker processes currently running"),
		recentJobs:    d("", "recent_jobs_total", "Recent jobs in the monitoring window"),
		pausedMasters: d("", "paused_masters", "Number of paused master supervisors"),

		queueLength:    d("queue", "length", "Jobs waiting in the queue", "queue"),
		queueWait:      d("queue", "wait_seconds", "Estimated wait time in seconds", "queue"),
		queueProcesses: d("queue", "processes", "Worker processes for the queue", "queue"),

		queueThroughput: d("queue", "throughput", "Jobs per minute (latest snapshot)", "queue"),
		queueRuntime:    d("queue", "runtime_milliseconds", "Average job runtime ms (latest snapshot)", "queue"),
		queueWaitSnap:   d("queue", "wait_time_seconds", "Average wait time seconds (latest snapshot)", "queue"),

		jobThroughput: d("job", "throughput", "Jobs per minute (latest snapshot)", "job"),
		jobRuntime:    d("job", "runtime_milliseconds", "Average runtime ms (latest snapshot)", "job"),

		masterStatus:        d("master", "status", "Master supervisor status: 1=running, 0=other", "master"),
		supervisorStatus:    d("supervisor", "status", "Supervisor status: 1=running, 0=other", "master", "supervisor"),
		supervisorProcesses: d("supervisor", "processes", "Worker processes in supervisor", "master", "supervisor", "queue"),

		pendingTotal:   d("pending_jobs", "total", "Total pending jobs"),
		pendingByQueue: d("pending_jobs", "by_queue", "Pending jobs by queue", "queue"),
		pendingByClass: d("pending_jobs", "by_class", "Pending jobs by job class", "class"),

		completedTotal:   d("completed_jobs", "total", "Total completed jobs in retention window"),
		completedByQueue: d("completed_jobs", "by_queue", "Completed jobs by queue", "queue"),
		completedByClass: d("completed_jobs", "by_class", "Completed jobs by job class", "class"),

		silencedTotal:   d("silenced_jobs", "total", "Total silenced jobs"),
		silencedByQueue: d("silenced_jobs", "by_queue", "Silenced jobs by queue", "queue"),
		silencedByClass: d("silenced_jobs", "by_class", "Silenced jobs by job class", "class"),

		failedTotal:   d("failed_jobs", "total", "Total failed jobs"),
		failedByQueue: d("failed_jobs", "by_queue", "Failed jobs by queue", "queue"),
		failedByClass: d("failed_jobs", "by_class", "Failed jobs by job class", "class"),

		monitoredTagJobs: d("monitored_tag", "jobs_total", "Jobs (active + failed) for monitored tag", "tag"),

		batchTotal:       d("batch", "total", "Total number of job batches"),
		batchPendingJobs: d("batch", "pending_jobs", "Pending jobs in batch", "id", "name"),
		batchFailedJobs:  d("batch", "failed_jobs", "Failed jobs in batch", "id", "name"),
		batchProgress:    d("batch", "progress", "Completion progress of batch (0-100)", "id", "name"),
		batchCancelled:   d("batch", "cancelled", "1 if batch is cancelled, 0 otherwise", "id", "name"),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.allDescs() {
		ch <- d
	}
}

func (c *Collector) allDescs() []*prometheus.Desc {
	return []*prometheus.Desc{
		c.up, c.status, c.jobsPerMinute, c.jobsPerHour, c.processes, c.recentJobs, c.pausedMasters,
		c.queueLength, c.queueWait, c.queueProcesses,
		c.queueThroughput, c.queueRuntime, c.queueWaitSnap,
		c.jobThroughput, c.jobRuntime,
		c.masterStatus, c.supervisorStatus, c.supervisorProcesses,
		c.pendingTotal, c.pendingByQueue, c.pendingByClass,
		c.completedTotal, c.completedByQueue, c.completedByClass,
		c.silencedTotal, c.silencedByQueue, c.silencedByClass,
		c.failedTotal, c.failedByQueue, c.failedByClass,
		c.monitoredTagJobs,
		c.batchTotal, c.batchPendingJobs, c.batchFailedJobs, c.batchProgress, c.batchCancelled,
	}
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	g := func(desc *prometheus.Desc, v float64, lv ...string) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	// ── Stats ────────────────────────────────────────────────────────────────
	stats, err := c.client.GetStats()
	if err != nil {
		log.Printf("error fetching stats: %v", err)
		g(c.up, 0)
		return
	}
	g(c.up, 1)
	running := 0.0
	if stats.Status == "running" {
		running = 1.0
	}
	g(c.status, running)
	g(c.jobsPerMinute, stats.JobsPerMinute)
	g(c.jobsPerHour, stats.JobsPerHour)
	g(c.processes, float64(stats.Processes))
	g(c.recentJobs, float64(stats.RecentJobs))
	g(c.pausedMasters, float64(stats.PausedMasters))

	// ── Workload ─────────────────────────────────────────────────────────────
	if workload, err := c.client.GetWorkload(); err != nil {
		log.Printf("error fetching workload: %v", err)
	} else {
		for _, w := range workload {
			g(c.queueLength, float64(w.Length), w.Name)
			g(c.queueWait, float64(w.Wait), w.Name)
			g(c.queueProcesses, float64(w.Processes), w.Name)
		}
	}

	// ── Queue metric snapshots ───────────────────────────────────────────────
	if qm, err := c.client.GetQueueMetrics(); err != nil {
		log.Printf("error fetching queue metrics: %v", err)
	} else {
		for queue, snap := range qm {
			g(c.queueThroughput, snap.Throughput, queue)
			g(c.queueRuntime, snap.Runtime, queue)
			g(c.queueWaitSnap, float64(snap.Wait), queue)
		}
	}

	// ── Job class metric snapshots ───────────────────────────────────────────
	if jm, err := c.client.GetJobMetrics(); err != nil {
		log.Printf("error fetching job metrics: %v", err)
	} else {
		for job, snap := range jm {
			g(c.jobThroughput, snap.Throughput, job)
			g(c.jobRuntime, snap.Runtime, job)
		}
	}

	// ── Masters / supervisors ────────────────────────────────────────────────
	if masters, err := c.client.GetMasters(); err != nil {
		log.Printf("error fetching masters: %v", err)
	} else {
		for _, m := range masters {
			isRunning := 0.0
			if m.Status == "running" {
				isRunning = 1.0
			}
			g(c.masterStatus, isRunning, m.Name)
			for _, s := range m.Supervisors {
				svRunning := 0.0
				if s.Status == "running" {
					svRunning = 1.0
				}
				g(c.supervisorStatus, svRunning, m.Name, s.Name)
				for queue, count := range s.Processes {
					g(c.supervisorProcesses, float64(count), m.Name, s.Name, queue)
				}
			}
		}
	}

	// ── Pending jobs ─────────────────────────────────────────────────────────
	if counts, err := c.client.GetPendingJobCounts(); err != nil {
		log.Printf("error fetching pending jobs: %v", err)
	} else {
		g(c.pendingTotal, float64(counts.Total))
		for q, n := range counts.ByQueue {
			g(c.pendingByQueue, float64(n), q)
		}
		for cls, n := range counts.ByClass {
			g(c.pendingByClass, float64(n), cls)
		}
	}

	// ── Completed jobs ───────────────────────────────────────────────────────
	if counts, err := c.client.GetCompletedJobCounts(); err != nil {
		log.Printf("error fetching completed jobs: %v", err)
	} else {
		g(c.completedTotal, float64(counts.Total))
		for q, n := range counts.ByQueue {
			g(c.completedByQueue, float64(n), q)
		}
		for cls, n := range counts.ByClass {
			g(c.completedByClass, float64(n), cls)
		}
	}

	// ── Silenced jobs ────────────────────────────────────────────────────────
	if counts, err := c.client.GetSilencedJobCounts(); err != nil {
		log.Printf("error fetching silenced jobs: %v", err)
	} else {
		g(c.silencedTotal, float64(counts.Total))
		for q, n := range counts.ByQueue {
			g(c.silencedByQueue, float64(n), q)
		}
		for cls, n := range counts.ByClass {
			g(c.silencedByClass, float64(n), cls)
		}
	}

	// ── Failed jobs ──────────────────────────────────────────────────────────
	if counts, err := c.client.GetFailedJobCounts(); err != nil {
		log.Printf("error fetching failed jobs: %v", err)
	} else {
		g(c.failedTotal, float64(counts.Total))
		for q, n := range counts.ByQueue {
			g(c.failedByQueue, float64(n), q)
		}
		for cls, n := range counts.ByClass {
			g(c.failedByClass, float64(n), cls)
		}
	}

	// ── Monitored tags ───────────────────────────────────────────────────────
	if tags, err := c.client.GetMonitoredTags(); err != nil {
		log.Printf("error fetching monitored tags: %v", err)
	} else {
		for _, t := range tags {
			g(c.monitoredTagJobs, float64(t.Count), t.Tag)
		}
	}

	// ── Batches ──────────────────────────────────────────────────────────────
	if batches, err := c.client.GetBatches(); err != nil {
		log.Printf("error fetching batches: %v", err)
	} else {
		g(c.batchTotal, float64(len(batches)))
		for _, b := range batches {
			cancelled := 0.0
			if b.CancelledAt != nil {
				cancelled = 1.0
			}
			g(c.batchPendingJobs, float64(b.PendingJobs), b.ID, b.Name)
			g(c.batchFailedJobs, float64(b.FailedJobs), b.ID, b.Name)
			g(c.batchProgress, b.Progress, b.ID, b.Name)
			g(c.batchCancelled, cancelled, b.ID, b.Name)
		}
	}
}
