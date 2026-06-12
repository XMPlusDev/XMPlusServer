package instance

import (
	"context"
	"log"
	"time"

	"github.com/xmplusdev/xmray/api"
	"github.com/xmplusdev/xmray/controller"
	"github.com/xmplusdev/xmray/node"
	"github.com/xmplusdev/xmray/scheduler"
)

const defaultServerNodePollInterval = 60 * time.Second

func pollDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultServerNodePollInterval
	}
	return time.Duration(seconds) * time.Second
}

// startServerNodePoller creates and starts a PeriodicTask that watches the
// panel's node list and dynamically starts/stops controllers. The task is
// stored on the Instance so Close() shuts it down with everything else.
// A separate goroutine listens on serverPollTrigger so a "server_change"
// Reverb event causes an immediate sync without waiting for the next tick.
func (i *Instance) startServerNodePoller(
	rootClient *api.Client,
	controllerConfig *node.Config,
	pusher func(string, any) error,
	initialPollInterval int,
) {
	current := pollDuration(initialPollInterval)

	var task *scheduler.PeriodicTask
	task = scheduler.NewWithDelay("[ServerPoller]", "server_nodes", current, func() error {
		newInterval := i.syncServerNodes(rootClient, controllerConfig, pusher)
		if newInterval != task.Periodic.Interval {
			log.Printf("[ServerPoller] Poll interval changed: %v → %v", task.Periodic.Interval, newInterval)
			task.RestartWithInterval(newInterval)
			i.updateServerStatusInterval(newInterval)
		}
		return nil
	})

	if err := task.Start(); err != nil {
		log.Printf("[ServerPoller] Failed to start: %v", err)
		return
	}

	i.statusLock.Lock()
	i.serverPoller = task
	i.statusLock.Unlock()

	// Trigger listener: fires an immediate sync when a "server_change" event
	// arrives over Reverb, then resets the task so the next scheduled fire
	// is a full interval away.
	ctx, cancel := context.WithCancel(context.Background())
	i.reverbCancels = append(i.reverbCancels, cancel)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-i.serverPollTrigger:
				log.Printf("[ServerPoller] server_change received — syncing immediately")
				newInterval := i.syncServerNodes(rootClient, controllerConfig, pusher)
				task.RestartWithInterval(newInterval)
				i.updateServerStatusInterval(newInterval)
			}
		}
	}()
}

// syncServerNodes fetches the current node list, starts/stops controllers as
// needed, and returns the poll interval from the API response.
func (i *Instance) syncServerNodes(
	rootClient *api.Client,
	controllerConfig *node.Config,
	pusher func(string, any) error,
) time.Duration {
	resp, err := rootClient.GetServerNodes()
	if err != nil {
		log.Printf("[ServerPoller] Failed to fetch node list: %v", err)
		return defaultServerNodePollInterval
	}

	interval := pollDuration(resp.PollInterval)

	i.statusLock.Lock()
	defer i.statusLock.Unlock()

	// Build set of node IDs returned by the panel.
	panelIDs := make(map[int]struct{}, len(resp.Nodes))
	for _, n := range resp.Nodes {
		panelIDs[n.NodeID] = struct{}{}
	}

	// Stop controllers for nodes no longer in the panel list.
	for nodeID, t := range i.controllerMap {
		if _, exists := panelIDs[nodeID]; !exists {
			log.Printf("[ServerPoller] Node %d removed — stopping controller", nodeID)
			if svc, ok := t.(controller.ControllerInterface); ok {
				if err := svc.Close(); err != nil {
					log.Printf("[ServerPoller] Error closing controller for node %d: %v", nodeID, err)
				}
				i.Service = removeService(i.Service, svc)
			}
			delete(i.controllerMap, nodeID)
		}
	}

	// Start controllers for newly added nodes.
	for _, n := range resp.Nodes {
		if _, exists := i.controllerMap[n.NodeID]; exists {
			continue // already running
		}
		log.Printf("[ServerPoller] Node %d added — starting controller", n.NodeID)
		nodeClient := rootClient.ForNode(n.NodeID)
		svc := controller.New(i.Server, nodeClient, controllerConfig, i.Dispatcher, pusher)
		if err := svc.Start(); err != nil {
			log.Printf("[ServerPoller] Failed to start controller for node %d: %v", n.NodeID, err)
			continue
		}
		i.Service = append(i.Service, svc)
		i.controllerMap[n.NodeID] = svc
	}

	return interval
}

func removeService(services []controller.ControllerInterface, target controller.ControllerInterface) []controller.ControllerInterface {
	out := services[:0]
	for _, s := range services {
		if s != target {
			out = append(out, s)
		}
	}
	return out
}
