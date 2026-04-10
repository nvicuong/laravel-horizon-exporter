package collector

import (
	"errors"
	"log"

	"github.com/horizon-exporter/horizon"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "horizon"

type Collector struct {
	client *horizon.Client

	// Core stats
	up             *prometheus.Desc
	status         *prometheus.Desc
	jobsPerMinute  *prometheus.Desc
	processes      *prometheus.Desc
	recentJobs     *prometheus.Desc
	recentlyFailed *prometheus.Desc

	// Stats periods (window sizes)
	recentJobsPeriod     *prometheus.Desc
	recentlyFailedPeriod *prometheus.Desc

	// Stats wait & hottest queue info
	statsWaitSeconds        *prometheus.Desc
	statsMaxRuntimeQueue    *prometheus.Desc
	statsMaxThroughputQueue *prometheus.Desc

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

	// Master topology
	masterStatus *prometheus.Desc

	// Supervisor topology
	supervisorStatus       *prometheus.Desc
	supervisorProcesses    *prometheus.Desc
	supervisorMaxProcesses *prometheus.Desc
	supervisorMinProcesses *prometheus.Desc
	supervisorTimeout      *prometheus.Desc
	supervisorMaxTries     *prometheus.Desc
	supervisorMemory       *prometheus.Desc

	// Pending jobs
	pendingTotal   *prometheus.Desc
	pendingByQueue *prometheus.Desc
	pendingByClass *prometheus.Desc

	// Completed jobs
	completedTotal   *prometheus.Desc
	completedByQueue *prometheus.Desc
	completedByClass *prometheus.Desc

	// Silenced jobs
	silencedTotal   *prometheus.Desc
	silencedByQueue *prometheus.Desc
	silencedByClass *prometheus.Desc

	// Failed jobs
	failedTotal   *prometheus.Desc
	failedByQueue *prometheus.Desc
	failedByClass *prometheus.Desc

	// Monitored tags
	monitoredTagJobs *prometheus.Desc

	// Batches
	batchTotal       *prometheus.Desc
	batchTotalJobs   *prometheus.Desc
	batchPendingJobs *prometheus.Desc
	batchFailedJobs  *prometheus.Desc
	batchProgress    *prometheus.Desc
	batchCancelled   *prometheus.Desc
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

		up:             d("", "up", "1 if the Horizon API is reachable, 0 otherwise"),
		status:         d("", "status", "Horizon status: 1=running, 0=paused/inactive"),
		jobsPerMinute:  d("", "jobs_per_minute", "Jobs processed per minute"),
		processes:      d("", "processes", "Total worker processes currently running"),
		recentJobs:     d("", "recent_jobs_total", "Recent jobs within the recentJobs period window"),
		recentlyFailed: d("", "recently_failed_total", "Recently failed jobs within the recentlyFailed period window"),

		recentJobsPeriod:     d("", "recent_jobs_period_minutes", "Time window in minutes for the recentJobs counter"),
		recentlyFailedPeriod: d("", "recently_failed_period_minutes", "Time window in minutes for the recentlyFailed counter"),

		statsWaitSeconds:        d("stats", "wait_seconds", "Estimated wait time in seconds per queue", "queue"),
		statsMaxRuntimeQueue:    d("stats", "max_runtime_queue_info", "Queue with the highest average job runtime (1=current)", "queue"),
		statsMaxThroughputQueue: d("stats", "max_throughput_queue_info", "Queue with the highest throughput (1=current)", "queue"),

		queueLength:    d("queue", "length", "Jobs waiting in the queue", "queue"),
		queueWait:      d("queue", "wait_seconds", "Estimated wait time in seconds", "queue"),
		queueProcesses: d("queue", "processes", "Worker processes for the queue", "queue"),

		queueThroughput: d("queue", "throughput", "Jobs per minute (latest snapshot)", "queue"),
		queueRuntime:    d("queue", "runtime_milliseconds", "Average job runtime ms (latest snapshot)", "queue"),
		queueWaitSnap:   d("queue", "wait_time_seconds", "Average wait time seconds (latest snapshot)", "queue"),

		jobThroughput: d("job", "throughput", "Jobs per minute (latest snapshot)", "job"),
		jobRuntime:    d("job", "runtime_milliseconds", "Average runtime ms (latest snapshot)", "job"),

		masterStatus: d("master", "status", "Master supervisor status: 1=running, 0=other", "master"),

		supervisorStatus:       d("supervisor", "status", "Supervisor status: 1=running, 0=other", "master", "supervisor"),
		supervisorProcesses:    d("supervisor", "processes", "Worker processes in supervisor per queue", "master", "supervisor", "queue"),
		supervisorMaxProcesses: d("supervisor", "max_processes", "Configured max worker processes", "master", "supervisor"),
		supervisorMinProcesses: d("supervisor", "min_processes", "Configured min worker processes", "master", "supervisor"),
		supervisorTimeout:      d("supervisor", "timeout_seconds", "Configured job timeout in seconds", "master", "supervisor"),
		supervisorMaxTries:     d("supervisor", "max_tries", "Configured max job attempts", "master", "supervisor"),
		supervisorMemory:       d("supervisor", "memory_limit_megabytes", "Configured worker memory limit in MB", "master", "supervisor"),

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
		batchTotalJobs:   d("batch", "total_jobs", "Total jobs in batch (summed by name)", "name"),
		batchPendingJobs: d("batch", "pending_jobs", "Pending jobs in batch (summed by name)", "name"),
		batchFailedJobs:  d("batch", "failed_jobs", "Failed jobs in batch (summed by name)", "name"),
		batchProgress:    d("batch", "progress", "Average completion progress of batch 0-100 (by name)", "name"),
		batchCancelled:   d("batch", "cancelled", "Number of cancelled batches with this name", "name"),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.allDescs() {
		ch <- d
	}
}

func (c *Collector) allDescs() []*prometheus.Desc {
	return []*prometheus.Desc{
		c.up, c.status, c.jobsPerMinute, c.processes, c.recentJobs, c.recentlyFailed,
		c.recentJobsPeriod, c.recentlyFailedPeriod,
		c.statsWaitSeconds, c.statsMaxRuntimeQueue, c.statsMaxThroughputQueue,
		c.queueLength, c.queueWait, c.queueProcesses,
		c.queueThroughput, c.queueRuntime, c.queueWaitSnap,
		c.jobThroughput, c.jobRuntime,
		c.masterStatus,
		c.supervisorStatus, c.supervisorProcesses,
		c.supervisorMaxProcesses, c.supervisorMinProcesses,
		c.supervisorTimeout, c.supervisorMaxTries, c.supervisorMemory,
		c.pendingTotal, c.pendingByQueue, c.pendingByClass,
		c.completedTotal, c.completedByQueue, c.completedByClass,
		c.silencedTotal, c.silencedByQueue, c.silencedByClass,
		c.failedTotal, c.failedByQueue, c.failedByClass,
		c.monitoredTagJobs,
		c.batchTotal, c.batchTotalJobs, c.batchPendingJobs, c.batchFailedJobs, c.batchProgress, c.batchCancelled,
	}
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	g := func(desc *prometheus.Desc, v float64, lv ...string) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	// ── Stats ────────────────────────────────────────────────────────────────
	stats, err := c.client.GetStats()
	if err != nil {
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching stats: %v", err)
		}
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
	g(c.processes, float64(stats.Processes))
	g(c.recentJobs, float64(stats.RecentJobs))
	g(c.recentlyFailed, float64(stats.RecentlyFailed))
	if stats.Periods.RecentJobs > 0 {
		g(c.recentJobsPeriod, float64(stats.Periods.RecentJobs))
	}
	if stats.Periods.RecentlyFailed > 0 {
		g(c.recentlyFailedPeriod, float64(stats.Periods.RecentlyFailed))
	}
	for queue, wait := range stats.WaitTime {
		g(c.statsWaitSeconds, float64(wait), queue)
	}
	if stats.QueueWithMaxRuntime != "" {
		g(c.statsMaxRuntimeQueue, 1, stats.QueueWithMaxRuntime)
	}
	if stats.QueueWithMaxThroughput != "" {
		g(c.statsMaxThroughputQueue, 1, stats.QueueWithMaxThroughput)
	}

	// ── Workload ─────────────────────────────────────────────────────────────
	if workload, err := c.client.GetWorkload(); err != nil {
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching workload: %v", err)
		}
	} else {
		for _, w := range workload {
			g(c.queueLength, float64(w.Length), w.Name)
			g(c.queueWait, float64(w.Wait), w.Name)
			g(c.queueProcesses, float64(w.Processes), w.Name)
		}
	}

	// ── Queue metric snapshots ───────────────────────────────────────────────
	if qm, err := c.client.GetQueueMetrics(); err != nil {
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching queue metrics: %v", err)
		}
	} else {
		for queue, snap := range qm {
			g(c.queueThroughput, snap.Throughput, queue)
			g(c.queueRuntime, snap.Runtime, queue)
			g(c.queueWaitSnap, float64(snap.Wait), queue)
		}
	}

	// ── Job class metric snapshots ───────────────────────────────────────────
	if jm, err := c.client.GetJobMetrics(); err != nil {
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching job metrics: %v", err)
		}
	} else {
		for job, snap := range jm {
			g(c.jobThroughput, snap.Throughput, job)
			g(c.jobRuntime, snap.Runtime, job)
		}
	}

	// ── Masters / supervisors ────────────────────────────────────────────────
	if masters, err := c.client.GetMasters(); err != nil {
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching masters: %v", err)
		}
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
				g(c.supervisorMaxProcesses, float64(s.Options.MaxProcesses), m.Name, s.Name)
				g(c.supervisorMinProcesses, float64(s.Options.MinProcesses), m.Name, s.Name)
				if s.Options.Timeout > 0 {
					g(c.supervisorTimeout, float64(s.Options.Timeout), m.Name, s.Name)
				}
				if s.Options.Tries > 0 {
					g(c.supervisorMaxTries, float64(s.Options.Tries), m.Name, s.Name)
				}
				if s.Options.Memory > 0 {
					g(c.supervisorMemory, float64(s.Options.Memory), m.Name, s.Name)
				}
			}
		}
	}

	// ── Pending jobs ─────────────────────────────────────────────────────────
	if counts, err := c.client.GetPendingJobCounts(); err != nil {
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching pending jobs: %v", err)
		}
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
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching completed jobs: %v", err)
		}
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
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching silenced jobs: %v", err)
		}
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
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching failed jobs: %v", err)
		}
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
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching monitored tags: %v", err)
		}
	} else {
		for _, t := range tags {
			g(c.monitoredTagJobs, float64(t.Count), t.Tag)
		}
	}

	// ── Batches ──────────────────────────────────────────────────────────────
	if batches, err := c.client.GetBatches(); err != nil {
		if !errors.Is(err, horizon.ErrEndpointUnavailable) {
			log.Printf("error fetching batches: %v", err)
		}
	} else {
		type agg struct {
			pending   int64
			failed    int64
			total     int64
			progress  float64
			count     int64
			cancelled int64
		}
		byName := make(map[string]*agg, len(batches))
		for _, b := range batches {
			a := byName[b.Name]
			if a == nil {
				a = &agg{}
				byName[b.Name] = a
			}
			a.pending += b.PendingJobs
			a.failed += b.FailedJobs
			a.total += b.TotalJobs
			a.progress += b.Progress
			a.count++
			if b.CancelledAt != nil {
				a.cancelled++
			}
		}
		g(c.batchTotal, float64(len(batches)))
		for name, a := range byName {
			avgProgress := 0.0
			if a.count > 0 {
				avgProgress = a.progress / float64(a.count)
			}
			g(c.batchTotalJobs, float64(a.total), name)
			g(c.batchPendingJobs, float64(a.pending), name)
			g(c.batchFailedJobs, float64(a.failed), name)
			g(c.batchProgress, avgProgress, name)
			g(c.batchCancelled, float64(a.cancelled), name)
		}
	}
}
