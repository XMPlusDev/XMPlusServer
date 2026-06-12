package instance

import (
	"log"
	"time"

	"github.com/xmplusdev/xmray/api"
	"github.com/xmplusdev/xmray/monitor"
	"github.com/xmplusdev/xmray/scheduler"
)

// serverStatusReportInterval is how often live server stats (cpu, mem, load,
// network speed, etc.) are reported to the panel. This is intentionally
// decoupled from the node poll interval — the panel caches each report in
// Redis and broadcasts it to admins in real time, batching the actual DB
// writes separately, so reporting frequently here is cheap for the backend
// while keeping the admin dashboard live.
const serverStatusReportInterval = 5 * time.Second

// startServerStatusTask starts a single server-status reporting task for the
// whole machine. Used in server ID mode so status is reported once regardless
// of how many nodes run on this server.
func (i *Instance) startServerStatusTask(
	client *api.Client,
	pusher func(string, any) error,
	_ int,
) {
	interval := serverStatusReportInterval

	task := scheduler.NewWithDelay("[ServerStatus]", "server_status", interval, func() error {
		i.reportServerStatus(client, pusher)
		return nil
	})

	if err := task.Start(); err != nil {
		log.Printf("[ServerStatus] Failed to start: %v", err)
		return
	}

	i.statusLock.Lock()
	i.serverStatusTask = task
	i.statusLock.Unlock()
}

func (i *Instance) reportServerStatus(client *api.Client, pusher func(string, any) error) {
	s := monitor.Collect()
	status := &api.ServerStatus{
		CPU:         s.CPU,
		MemUsed:     s.MemUsed,
		MemTotal:    s.MemTotal,
		SwapUsed:    s.SwapUsed,
		SwapTotal:   s.SwapTotal,
		DiskUsed:    s.DiskUsed,
		DiskTotal:   s.DiskTotal,
		Load1:       s.Load1,
		Load5:       s.Load5,
		Load15:      s.Load15,
		NetInSpeed:  s.NetInSpeed,
		NetOutSpeed: s.NetOutSpeed,
		Uptime:      s.Uptime,
	}

	if pusher != nil {
		payload := &api.ServerStatusPayload{
			ServerID: i.instanceConfig.ApiConfig.ServerID,
			Data:     status,
		}
		if err := pusher("server_status", payload); err != nil {
			log.Printf("[ServerStatus] Push failed: %v", err)
		} else {
			log.Printf("[ServerStatus] Pushed server status via Reverb")
		}
		return
	}

	if err := client.ReportServerStatus(status); err != nil {
		log.Printf("[ServerStatus] Report failed: %v", err)
	}
}

// updateServerStatusInterval previously kept the status task's interval in
// sync with the node poll interval. Server status reporting is now a fixed
// serverStatusReportInterval (10s), independent of the (often much longer)
// node poll interval, so this is intentionally a no-op — kept so callers in
// server_poller.go don't need to change.
func (i *Instance) updateServerStatusInterval(_ time.Duration) {}
