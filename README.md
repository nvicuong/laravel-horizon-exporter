# horizon-exporter

A [Prometheus](https://prometheus.io/) exporter for [Laravel Horizon](https://laravel.com/docs/horizon).

The exporter connects directly to the Horizon HTTP API and exposes metrics via HTTP for Prometheus to scrape.

## Table of Contents

- [Features](#features)
- [Usage](#usage)
  - [Options and defaults](#options-and-defaults)
  - [CLI Examples](#cli-examples)
  - [Docker Examples](#docker-examples)
- [Metrics collected](#metrics-collected)
- [FAQ](#faq)
- [Contributing](#contributing)

## Features

- Exposes core Horizon stats: status, jobs per minute/hour, processes, failed jobs
- Per-queue workload: length, wait time, worker processes
- Per-queue and per-job-class throughput and runtime snapshots (fetched concurrently)
- Master supervisor and per-supervisor topology with process limits
- Paginated job counts (pending, completed, silenced, failed) broken down by queue and job class
- Monitored tag job counts
- Batch aggregation by name (pending, failed, total jobs, progress, cancelled count)
- Bearer token authentication support
- TLS skip-verify option for self-signed certificates

## Usage

`horizon-exporter` is released as a compiled binary. It requires minimal configuration — only `--horizon.url` is mandatory.

### Options and defaults

| Option | Description | Default |
|---|---|---|
| `--web.listen-ip` | IP address to listen on (empty = all interfaces) | `` (all) |
| `--web.listen-port` | Port to listen on for metrics | `9888` |
| `--web.telemetry-path` | Path under which to expose metrics | `/metrics` |
| `--horizon.url` | Base URL of the Laravel application **(required)** | `http://localhost` |
| `--horizon.token` | Bearer token for Horizon API authentication | `` |
| `--horizon.tls-skip-verify` | Skip TLS certificate verification | `false` |
| `--horizon.endpoint.exclude` | Comma-separated endpoints to skip (repeatable) | `` (none) |

### Horizon API endpoints

The exporter calls the following Horizon API endpoints on every scrape. Use `--horizon.endpoint.exclude` to disable any of them.

| Endpoint key | Horizon API path | Metrics produced |
|---|---|---|
| `stats` | `/horizon/api/stats` | `horizon_up`, `horizon_status`, `horizon_jobs_per_minute`, `horizon_processes`, `horizon_recent_jobs_total`, `horizon_recently_failed_total`, `horizon_recent_jobs_period_minutes`, `horizon_recently_failed_period_minutes`, `horizon_stats_wait_seconds`, `horizon_stats_max_runtime_queue_info`, `horizon_stats_max_throughput_queue_info` |
| `workload` | `/horizon/api/workload` | `horizon_queue_length`, `horizon_queue_wait_seconds`, `horizon_queue_processes` |
| `masters` | `/horizon/api/masters` | `horizon_master_status`, `horizon_supervisor_status`, `horizon_supervisor_processes`, `horizon_supervisor_max_processes`, `horizon_supervisor_min_processes`, `horizon_supervisor_timeout_seconds`, `horizon_supervisor_max_tries`, `horizon_supervisor_memory_limit_megabytes` |
| `jobs/pending` | `/horizon/api/jobs/pending` | `horizon_pending_jobs_total`, `horizon_pending_jobs_by_queue`, `horizon_pending_jobs_by_class` |
| `jobs/completed` | `/horizon/api/jobs/completed` | `horizon_completed_jobs_total`, `horizon_completed_jobs_by_queue`, `horizon_completed_jobs_by_class` |
| `jobs/silenced` | `/horizon/api/jobs/silenced` | `horizon_silenced_jobs_total`, `horizon_silenced_jobs_by_queue`, `horizon_silenced_jobs_by_class` |
| `jobs/failed` | `/horizon/api/jobs/failed` | `horizon_failed_jobs_total`, `horizon_failed_jobs_by_queue`, `horizon_failed_jobs_by_class` |
| `metrics/queues` | `/horizon/api/metrics/queues` + per-queue | `horizon_queue_throughput`, `horizon_queue_runtime_milliseconds`, `horizon_queue_wait_time_seconds` |
| `metrics/jobs` | `/horizon/api/metrics/jobs` + per-class | `horizon_job_throughput`, `horizon_job_runtime_milliseconds` |
| `batches` | `/horizon/api/batches` | `horizon_batch_total`, `horizon_batch_total_jobs`, `horizon_batch_pending_jobs`, `horizon_batch_failed_jobs`, `horizon_batch_progress`, `horizon_batch_cancelled` |
| `monitoring` | `/horizon/api/monitoring` | `horizon_monitored_tag_jobs_total` |

> **Note:** `stats` cannot be meaningfully excluded — it is the primary health check endpoint and its availability controls `horizon_up`. Excluding it will suppress all stats-derived metrics but `horizon_up` will still emit `1`.

### CLI Examples

- Build binaries:
  ```
  # Build Linux AMD64 binary
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o horizon-exporter-linux-amd64

  # Build Linux ARM64 binary
  CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o horizon-exporter-linux-arm64
  ```

- Run with defaults against a local Laravel app:
  ```
  ./horizon-exporter-linux-amd64 --horizon.url=http://localhost
  ```

- Run on a specific IP and port:
  ```
  ./horizon-exporter --horizon.url=http://myapp.internal \
    --web.listen-ip=192.168.1.10 \
    --web.listen-port=9888
  ```

- Run with Bearer token authentication:
  ```
  ./horizon-exporter --horizon.url=https://myapp.example.com \
    --horizon.token=my-secret-token
  ```

- Run with TLS skip verify (e.g. self-signed certs):
  ```
  ./horizon-exporter --horizon.url=https://myapp.example.com \
    --horizon.tls-skip-verify
  ```

- Exclude expensive paginated job endpoints (reduces scrape time and Horizon load):
  ```
  ./horizon-exporter --horizon.url=https://myapp.example.com \
    --horizon.endpoint.exclude=jobs/pending,jobs/completed,jobs/silenced
  ```

- Exclude multiple endpoints using repeated flags:
  ```
  ./horizon-exporter --horizon.url=https://myapp.example.com \
    --horizon.endpoint.exclude=batches \
    --horizon.endpoint.exclude=monitoring \
    --horizon.endpoint.exclude=metrics/queues \
    --horizon.endpoint.exclude=metrics/jobs
  ```

- Minimal mode — only core stats, queue workload, and master topology:
  ```
  ./horizon-exporter-linux-amd64 --horizon.url=https://myapp.example.com \
    --horizon.endpoint.exclude=jobs/pending,jobs/completed,jobs/silenced,jobs/failed,metrics/queues,metrics/jobs,batches,monitoring
  ```

### Docker Examples

- Build the image:
  ```
  docker build -t horizon-exporter .
  ```

- Run with a remote Horizon URL:
  ```
  docker run -d --restart=unless-stopped \
    -p 9888:9888 \
    horizon-exporter \
    --horizon.url=https://myapp.example.com
  ```

- Run with authentication and TLS skip:
  ```
  docker run -d --restart=unless-stopped \
    -p 9888:9888 \
    horizon-exporter \
    --horizon.url=https://myapp.example.com \
    --horizon.token=my-secret-token \
    --horizon.tls-skip-verify
  ```

- Run in minimal mode (exclude expensive endpoints):
  ```
  docker run -d --restart=unless-stopped \
    -p 9888:9888 \
    horizon-exporter \
    --horizon.url=https://myapp.example.com \
    --horizon.endpoint.exclude=jobs/pending,jobs/completed,jobs/silenced,jobs/failed,metrics/queues,metrics/jobs,batches,monitoring
  ```

## Metrics collected

### Core

| Metric | Type | Description |
|---|---|---|
| `horizon_up` | Gauge | `1` if the Horizon API is reachable, `0` otherwise |
| `horizon_status` | Gauge | Horizon status: `1` = running, `0` = paused/inactive |
| `horizon_jobs_per_minute` | Gauge | Jobs processed per minute |
| `horizon_jobs_per_hour` | Gauge | Jobs processed per hour |
| `horizon_processes` | Gauge | Total worker processes currently running |
| `horizon_recent_jobs_total` | Gauge | Recent jobs in the monitoring window |
| `horizon_paused_masters` | Gauge | Number of paused master supervisors |

### Stats

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_stats_failed_jobs` | Gauge | | Total failed jobs reported by Horizon stats |
| `horizon_stats_wait_seconds` | Gauge | `queue` | Estimated wait time in seconds per queue (from stats endpoint) |

### Queue Workload

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_queue_length` | Gauge | `queue` | Number of jobs waiting in the queue |
| `horizon_queue_wait_seconds` | Gauge | `queue` | Estimated wait time in seconds |
| `horizon_queue_processes` | Gauge | `queue` | Number of worker processes assigned to the queue |

### Queue Metric Snapshots

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_queue_throughput` | Gauge | `queue` | Jobs per minute (latest snapshot) |
| `horizon_queue_runtime_milliseconds` | Gauge | `queue` | Average job runtime in ms (latest snapshot) |
| `horizon_queue_wait_time_seconds` | Gauge | `queue` | Average wait time in seconds (latest snapshot) |

### Job Class Metric Snapshots

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_job_throughput` | Gauge | `job` | Jobs per minute for the job class (latest snapshot) |
| `horizon_job_runtime_milliseconds` | Gauge | `job` | Average runtime in ms for the job class (latest snapshot) |

### Master & Supervisor Topology

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_master_status` | Gauge | `master`, `environment` | Master supervisor status: `1` = running, `0` = other |
| `horizon_supervisor_status` | Gauge | `master`, `supervisor` | Supervisor status: `1` = running, `0` = other |
| `horizon_supervisor_processes` | Gauge | `master`, `supervisor`, `queue` | Number of worker processes in the supervisor per queue |
| `horizon_supervisor_max_processes` | Gauge | `master`, `supervisor` | Configured maximum number of worker processes |
| `horizon_supervisor_min_processes` | Gauge | `master`, `supervisor` | Configured minimum number of worker processes |

### Pending Jobs

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_pending_jobs_total` | Gauge | | Total number of pending jobs |
| `horizon_pending_jobs_by_queue` | Gauge | `queue` | Pending jobs broken down by queue |
| `horizon_pending_jobs_by_class` | Gauge | `class` | Pending jobs broken down by job class |

### Completed Jobs

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_completed_jobs_total` | Gauge | | Total completed jobs within the retention window |
| `horizon_completed_jobs_by_queue` | Gauge | `queue` | Completed jobs broken down by queue |
| `horizon_completed_jobs_by_class` | Gauge | `class` | Completed jobs broken down by job class |

### Silenced Jobs

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_silenced_jobs_total` | Gauge | | Total number of silenced jobs |
| `horizon_silenced_jobs_by_queue` | Gauge | `queue` | Silenced jobs broken down by queue |
| `horizon_silenced_jobs_by_class` | Gauge | `class` | Silenced jobs broken down by job class |

### Failed Jobs

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_failed_jobs_total` | Gauge | | Total number of failed jobs |
| `horizon_failed_jobs_by_queue` | Gauge | `queue` | Failed jobs broken down by queue |
| `horizon_failed_jobs_by_class` | Gauge | `class` | Failed jobs broken down by job class |

### Monitored Tags

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_monitored_tag_jobs_total` | Gauge | `tag` | Total active + failed jobs for the monitored tag |

### Batches

Batch metrics are aggregated by `name` to avoid unbounded label cardinality from unique batch UUIDs.

| Metric | Type | Labels | Description |
|---|---|---|---|
| `horizon_batch_total` | Gauge | | Total number of job batches |
| `horizon_batch_total_jobs` | Gauge | `name` | Total jobs dispatched in batches with this name |
| `horizon_batch_pending_jobs` | Gauge | `name` | Total pending jobs across batches with this name |
| `horizon_batch_failed_jobs` | Gauge | `name` | Total failed jobs across batches with this name |
| `horizon_batch_progress` | Gauge | `name` | Average completion progress (0–100) across batches with this name |
| `horizon_batch_cancelled` | Gauge | `name` | Number of cancelled batches with this name |

## FAQ

**How do I update the "Metrics collected" section?**

Run the exporter and copy the output from:
```
curl -s http://localhost:9888/metrics | grep "^#"
```

**Why are batch metrics aggregated by name instead of ID?**

Batch IDs are unique UUIDs. Using them as Prometheus labels causes unbounded label cardinality which grows indefinitely and consumes excessive memory in Prometheus. Aggregating by `name` is safe and still provides actionable signal.

**Why are queue/job metric snapshots fetched concurrently?**

Horizon's metrics API requires one HTTP request per queue/job class. With many queues, serial fetching can exceed Prometheus's default scrape timeout. The exporter fans out these requests concurrently (up to 10 in-flight) to keep scrape times low.

## Contributing

Contributions are welcome. Please open an issue before starting any significant work, then submit a pull request against `main`.