package swarmbench

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"tlaloc.local/behaviorlab/internal/tlaloque"
)

const ReplicationSchemaR0 = "tlaloc.swarm-bench-replication.r0"

// ReplicationRun is the control condition for the decomposition experiment:
// N replicas of the SAME five-Tlaloque swarm process disjoint shards of the
// dataset concurrently. Nothing about the DAG shape changes — Topology is
// identical to a single Execute() run of the same plan — only how many
// dataset items are in flight at once. This isolates "more parallel
// hardware" from "better decomposition": Score must not move with N (same
// logic answers every item either way), while WallClockMS should fall.
type ReplicationRun struct {
	Schema         string         `json:"schema"`
	RunID          string         `json:"run_id"`
	ReplicaCount   int            `json:"replica_count"`
	ItemCount      int            `json:"item_count"`
	WallClockMS    int64          `json:"wall_clock_ms"`
	ItemsPerSecond float64        `json:"items_per_second"`
	Topology       Topology       `json:"topology"`
	Score          Score          `json:"score"`
	NodeErrors     map[string]int `json:"node_errors,omitempty"`
}

// ExecuteReplicated partitions dataset.Items into replicaCount shards and
// runs one goroutine per shard, each driving its own sequential stream of
// SwarmRunner.Run calls against the shared registry. The in-process workers
// are stateless pure functions, so sharing one registry across replicas is
// exactly what a real resident HTTP_JSON service would do under N concurrent
// callers — this is not simulating N separate processes, it is measuring
// what N-way concurrent load against the same resident individuals costs.
func ExecuteReplicated(ctx context.Context, registry *tlaloque.Registry, plan tlaloque.SwarmPlan, dataset Dataset, terminalNodeID string, replicaCount int) (ReplicationRun, error) {
	plan, err := plan.Normalize()
	if err != nil {
		return ReplicationRun{}, err
	}
	if replicaCount <= 0 {
		replicaCount = 1
	}
	runner := tlaloque.SwarmRunner{Registry: registry}

	type outcome struct {
		itemID string
		fields Fields
	}
	results := make([]outcome, len(dataset.Items))
	var mu sync.Mutex
	nodeErrors := map[string]int{}

	shards := make([][]int, replicaCount)
	for index := range dataset.Items {
		shard := index % replicaCount
		shards[shard] = append(shards[shard], index)
	}

	start := time.Now()
	var wg sync.WaitGroup
	for _, shard := range shards {
		shard := shard
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, itemIndex := range shard {
				item := dataset.Items[itemIndex]
				input, marshalErr := marshalTaskInput(item)
				if marshalErr != nil {
					continue
				}
				report, runErr := runner.Run(ctx, plan, item.ID, input)
				if runErr != nil {
					mu.Lock()
					for _, node := range report.Nodes {
						if node.Error != "" {
							nodeErrors[node.NodeID]++
						}
					}
					mu.Unlock()
					results[itemIndex] = outcome{itemID: item.ID}
					continue
				}
				output, ok := report.TerminalOutputs[terminalNodeID]
				if !ok {
					mu.Lock()
					nodeErrors[terminalNodeID+"/missing_terminal"]++
					mu.Unlock()
					results[itemIndex] = outcome{itemID: item.ID}
					continue
				}
				results[itemIndex] = outcome{itemID: item.ID, fields: swarmFields(output)}
			}
		}()
	}
	wg.Wait()
	wallClock := time.Since(start)

	byItem := make(map[string]Fields, len(results))
	for _, result := range results {
		byItem[result.itemID] = result.fields
	}
	score := ScoreDataset(dataset, func(item Item) Fields { return byItem[item.ID] })

	itemsPerSecond := 0.0
	if wallClock > 0 {
		itemsPerSecond = float64(len(dataset.Items)) / wallClock.Seconds()
	}

	return ReplicationRun{
		Schema:         ReplicationSchemaR0,
		RunID:          plan.ID + "/" + dataset.ID + "/x" + strconv.Itoa(replicaCount),
		ReplicaCount:   replicaCount,
		ItemCount:      len(dataset.Items),
		WallClockMS:    wallClock.Milliseconds(),
		ItemsPerSecond: itemsPerSecond,
		Topology:       AnalyzeTopology(plan),
		Score:          score,
		NodeErrors:     nodeErrors,
	}, nil
}

func marshalTaskInput(item Item) ([]byte, error) {
	return json.Marshal(taskInput{Text: item.Text, ReferenceDate: item.ReferenceDate})
}
