package horizon

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const jobPageSize = 50

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func NewClient(baseURL string, token string, skipTLSVerify bool) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLSVerify},
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Client) get(path string, out interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("horizon API returned status %d for %s", resp.StatusCode, path)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (c *Client) GetStats() (*Stats, error) {
	var stats Stats
	if err := c.get("/horizon/api/stats", &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (c *Client) GetWorkload() ([]WorkloadItem, error) {
	var workload []WorkloadItem
	if err := c.get("/horizon/api/workload", &workload); err != nil {
		return nil, err
	}
	return workload, nil
}

func (c *Client) GetMasters() ([]MasterSupervisor, error) {
	var raw map[string]MasterSupervisor
	if err := c.get("/horizon/api/masters", &raw); err != nil {
		return nil, err
	}
	masters := make([]MasterSupervisor, 0, len(raw))
	for _, m := range raw {
		masters = append(masters, m)
	}
	return masters, nil
}

// getJobCounts paginates through all jobs at the given API path.
// Horizon's PHP side treats starting_at="0" as falsy (→ -1), so we break as
// soon as a page is not full (< 50 items) rather than relying on an empty page.
func (c *Client) getJobCounts(apiPath string) (*JobCounts, error) {
	counts := &JobCounts{
		ByQueue: map[string]int64{},
		ByClass: map[string]int64{},
	}

	cursor := int64(-1)
	for {
		path := apiPath + "?starting_at=" + strconv.FormatInt(cursor, 10)
		var page JobListResponse
		if err := c.get(path, &page); err != nil {
			return nil, err
		}

		counts.Total = page.Total

		for _, j := range page.Jobs {
			counts.ByQueue[j.Queue]++
			counts.ByClass[j.Name]++
		}

		// A page with fewer than jobPageSize items is the last page.
		if len(page.Jobs) < jobPageSize {
			break
		}

		cursor = page.Jobs[len(page.Jobs)-1].Index
	}

	return counts, nil
}

func (c *Client) GetPendingJobCounts() (*JobCounts, error) {
	return c.getJobCounts("/horizon/api/jobs/pending")
}

func (c *Client) GetCompletedJobCounts() (*JobCounts, error) {
	return c.getJobCounts("/horizon/api/jobs/completed")
}

func (c *Client) GetSilencedJobCounts() (*JobCounts, error) {
	return c.getJobCounts("/horizon/api/jobs/silenced")
}

func (c *Client) GetFailedJobCounts() (*JobCounts, error) {
	return c.getJobCounts("/horizon/api/jobs/failed")
}

func (c *Client) GetMonitoredTags() ([]MonitoredTag, error) {
	var tags []MonitoredTag
	if err := c.get("/horizon/api/monitoring", &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func (c *Client) GetBatches() ([]Batch, error) {
	var result BatchResponse
	if err := c.get("/horizon/api/batches", &result); err != nil {
		return nil, err
	}
	return result.Batches, nil
}

func (c *Client) GetQueueMetrics() (map[string]MetricSnapshot, error) {
	var names []string
	if err := c.get("/horizon/api/metrics/queues", &names); err != nil {
		return nil, err
	}
	result := make(map[string]MetricSnapshot, len(names))
	for _, name := range names {
		var snapshots []MetricSnapshot
		path := "/horizon/api/metrics/queues/" + url.PathEscape(name)
		if err := c.get(path, &snapshots); err != nil || len(snapshots) == 0 {
			continue
		}
		result[name] = snapshots[len(snapshots)-1]
	}
	return result, nil
}

func (c *Client) GetJobMetrics() (map[string]MetricSnapshot, error) {
	var names []string
	if err := c.get("/horizon/api/metrics/jobs", &names); err != nil {
		return nil, err
	}
	result := make(map[string]MetricSnapshot, len(names))
	for _, name := range names {
		var snapshots []MetricSnapshot
		path := "/horizon/api/metrics/jobs/" + url.PathEscape(name)
		if err := c.get(path, &snapshots); err != nil || len(snapshots) == 0 {
			continue
		}
		result[name] = snapshots[len(snapshots)-1]
	}
	return result, nil
}
