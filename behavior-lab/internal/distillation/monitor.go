package distillation

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

type TriggerReason string

const (
	TriggerRepeatedTask   TriggerReason = "REPEATED_TASK"
	TriggerAccuracyDrop   TriggerReason = "ACCURACY_DROP"
	TriggerExplicitDemand TriggerReason = "EXPLICIT_DEMAND"
	TriggerCostBudget     TriggerReason = "COST_BUDGET"
)

type TrainingRequest struct {
	Task   TaskSpec      `json:"task"`
	Reason TriggerReason `json:"reason"`
}

type TrainingScheduler interface {
	Schedule(context.Context, TrainingRequest) error
}

type MonitorConfig struct {
	RepeatThreshold   int
	AccuracyThreshold float64
	CostBudget        float64
}

// Monitor converts runtime evidence into explicit training requests. It does
// not launch goroutines or train models; schedulers own execution policy.
type Monitor struct {
	Config    MonitorConfig
	Scheduler TrainingScheduler
	mu        sync.Mutex
	counts    map[string]int
	costs     map[string]float64
}

func (monitor *Monitor) ObserveExecution(ctx context.Context, metric tlaloque.ExecutionMetric) {
	if !metric.Succeeded || monitor.Config.RepeatThreshold <= 0 || monitor.Scheduler == nil {
		return
	}
	capability := strings.ToUpper(strings.TrimSpace(metric.Capability))
	monitor.mu.Lock()
	if monitor.counts == nil {
		monitor.counts = map[string]int{}
	}
	monitor.counts[capability]++
	trigger := monitor.counts[capability] >= monitor.Config.RepeatThreshold
	if trigger {
		monitor.counts[capability] = 0
	}
	monitor.mu.Unlock()
	if trigger {
		_ = monitor.Scheduler.Schedule(ctx, TrainingRequest{Task: TaskSpec{Capability: capability}, Reason: TriggerRepeatedTask})
	}
}

func (monitor *Monitor) RecordAccuracy(ctx context.Context, task TaskSpec, accuracy float64) error {
	if accuracy < 0 || accuracy > 1 {
		return fmt.Errorf("accuracy must be between 0 and 1")
	}
	if monitor.Config.AccuracyThreshold > 0 && accuracy < monitor.Config.AccuracyThreshold {
		return monitor.schedule(ctx, task, TriggerAccuracyDrop)
	}
	return nil
}

func (monitor *Monitor) RecordCost(ctx context.Context, task TaskSpec, cost float64) error {
	if cost < 0 {
		return fmt.Errorf("cost cannot be negative")
	}
	task, err := task.normalize()
	if err != nil {
		return err
	}
	monitor.mu.Lock()
	if monitor.costs == nil {
		monitor.costs = map[string]float64{}
	}
	monitor.costs[task.Capability] += cost
	trigger := monitor.Config.CostBudget > 0 && monitor.costs[task.Capability] >= monitor.Config.CostBudget
	if trigger {
		monitor.costs[task.Capability] = 0
	}
	monitor.mu.Unlock()
	if trigger {
		return monitor.schedule(ctx, task, TriggerCostBudget)
	}
	return nil
}

func (monitor *Monitor) Demand(ctx context.Context, task TaskSpec) error {
	return monitor.schedule(ctx, task, TriggerExplicitDemand)
}

func (monitor *Monitor) schedule(ctx context.Context, task TaskSpec, reason TriggerReason) error {
	if monitor.Scheduler == nil {
		return fmt.Errorf("training scheduler is required")
	}
	task, err := task.normalize()
	if err != nil {
		return err
	}
	return monitor.Scheduler.Schedule(ctx, TrainingRequest{Task: task, Reason: reason})
}
