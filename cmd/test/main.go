package main

import (
	"fmt"
	"time"
	"github.com/horizon-exporter/horizon"
)

func timed(label string, fn func()) {
	t := time.Now()
	fn()
	fmt.Printf("%-35s %v\n", label+":", time.Since(t))
}

func main() {
	c := horizon.NewClient("http://192.168.56.104", "", false)

	timed("GetStats", func() { _, err := c.GetStats(); if err != nil { fmt.Println("err:", err) } })
	timed("GetWorkload", func() { _, err := c.GetWorkload(); if err != nil { fmt.Println("err:", err) } })
	timed("GetMasters", func() { _, err := c.GetMasters(); if err != nil { fmt.Println("err:", err) } })
	timed("GetQueueMetrics", func() { _, err := c.GetQueueMetrics(); if err != nil { fmt.Println("err:", err) } })
	timed("GetJobMetrics", func() { _, err := c.GetJobMetrics(); if err != nil { fmt.Println("err:", err) } })
	timed("GetPendingJobCounts", func() { r, err := c.GetPendingJobCounts(); if err != nil { fmt.Println("err:", err) } else { fmt.Printf("[total=%d queues=%d classes=%d] ", r.Total, len(r.ByQueue), len(r.ByClass)) } })
	timed("GetCompletedJobCounts", func() { r, err := c.GetCompletedJobCounts(); if err != nil { fmt.Println("err:", err) } else { fmt.Printf("[total=%d queues=%d classes=%d] ", r.Total, len(r.ByQueue), len(r.ByClass)) } })
	timed("GetSilencedJobCounts", func() { r, err := c.GetSilencedJobCounts(); if err != nil { fmt.Println("err:", err) } else { fmt.Printf("[total=%d queues=%d classes=%d] ", r.Total, len(r.ByQueue), len(r.ByClass)) } })
	timed("GetFailedJobCounts", func() { r, err := c.GetFailedJobCounts(); if err != nil { fmt.Println("err:", err) } else { fmt.Printf("[total=%d queues=%d classes=%d] ", r.Total, len(r.ByQueue), len(r.ByClass)) } })
	timed("GetMonitoredTags", func() { _, err := c.GetMonitoredTags(); if err != nil { fmt.Println("err:", err) } })
	timed("GetBatches", func() { _, err := c.GetBatches(); if err != nil { fmt.Println("err:", err) } })
}
