package borda

import (
	"sort"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/utils"
)

// ScoredNode represents a node along with its computed scores.
type ScoredNode struct {
	*domain.Node
	avgLatency float64

	cpuBorda              int
	memoryBorda           int
	networkBandwidthBorda int
	storageBorda          int

	EnergyBorda   int
	PricingBorda  int
	LatencyBorda  int
	CapacityBorda int

	WeightedScore float64
}

// ScoredCluster holds aggregated scores for a cluster.
type ScoredCluster struct {
	ID                   string
	AllNodes             []string
	TotalWeightedScore   float64
	FilterdWeightedScore float64
	FilteredNodes        []string
}

// ScoreAttr defines a score attribute with getter and setter functions.
type ScoreAttr struct {
	name         string
	getter       func(*ScoredNode) float64
	setter       func(*ScoredNode, int)
	higherBetter bool
}

// Borda is the default implementation of the NodeScorer interface.
type Borda struct{}

// ScoreAndFilterNodes scores and filters nodes based on various criteria.
// It applies a Borda count per attribute (energy, pricing, latency, cpu, memory, network bandwidth, storage)
// and then weights the individual scores using the provided QoS.
// Finally, it aggregates the scores per cluster and returns the best cluster ID along with
// the names of nodes that meet or exceed the average weighted score.
func (s *Borda) ScoreAndFilterNodes(nodes []*domain.Node, qos domain.QoS) (string, []string) {
	eligibleNodes := utils.FilterEligibleNodes(nodes)
	if len(eligibleNodes) == 0 {
		return "", nil
	}

	scoredNodes := make([]ScoredNode, len(eligibleNodes))
	for i, node := range eligibleNodes {
		scoredNodes[i] = ScoredNode{Node: node}
	}

	scoreAttrs := []ScoreAttr{
		{
			name: "energy",
			getter: func(sn *ScoredNode) float64 {
				return sn.Energy
			},
			setter: func(sn *ScoredNode, v int) {
				sn.EnergyBorda = v
			},
			higherBetter: false,
		},
		{
			name: "pricing",
			getter: func(sn *ScoredNode) float64 {
				return sn.Pricing
			},
			setter: func(sn *ScoredNode, v int) {
				sn.PricingBorda = v
			},
			higherBetter: false,
		},
		{
			name: "latency",
			getter: func(sn *ScoredNode) float64 {
				return sn.avgLatency
			},
			setter: func(sn *ScoredNode, v int) {
				sn.LatencyBorda = v
			},
			higherBetter: false,
		},
		{
			name: "cpu",
			getter: func(sn *ScoredNode) float64 {
				return float64(sn.CPU)
			},
			setter: func(sn *ScoredNode, v int) {
				sn.cpuBorda = v
			},
			higherBetter: true,
		},
		{
			name: "memory",
			getter: func(sn *ScoredNode) float64 {
				return sn.Memory
			},
			setter: func(sn *ScoredNode, v int) {
				sn.memoryBorda = v
			},
			higherBetter: true,
		},
		{
			name: "network_bandwidth",
			getter: func(sn *ScoredNode) float64 {
				return sn.NetworkBandwidth
			},
			setter: func(sn *ScoredNode, v int) {
				sn.networkBandwidthBorda = v
			},
			higherBetter: true,
		},
		{
			name: "storage",
			getter: func(sn *ScoredNode) float64 {
				return sn.EphemeralStorage
			},
			setter: func(sn *ScoredNode, v int) {
				sn.storageBorda = v
			},
			higherBetter: true,
		},
	}

	// Compute Borda counts for each attribute.
	for _, attr := range scoreAttrs {
		sort.Slice(scoredNodes, func(i, j int) bool {
			vi := attr.getter(&scoredNodes[i])
			vj := attr.getter(&scoredNodes[j])
			if attr.higherBetter {
				return vi > vj
			}
			return vi < vj
		})
		for i := 0; i < len(scoredNodes); i++ {
			borda := len(scoredNodes) - i
			attr.setter(&scoredNodes[i], borda)
		}
	}

	// Compute capacity as the sum of cpu, memory, network bandwidth, and storage Borda scores.
	for i := range scoredNodes {
		scoredNodes[i].CapacityBorda = scoredNodes[i].cpuBorda +
			scoredNodes[i].memoryBorda +
			scoredNodes[i].networkBandwidthBorda +
			scoredNodes[i].storageBorda
	}

	// Set default QoS weights if all are below threshold.
	if qos.Energy < 0.1 && qos.Pricing < 0.1 && qos.Performance < 0.1 {
		qos.Energy = 1.0
		qos.Pricing = 1.0
		qos.Performance = 1.0
	}

	// Calculate weighted score for each node.
	for i := range scoredNodes {
		scoredNodes[i].WeightedScore = float64(scoredNodes[i].EnergyBorda)*qos.Energy +
			float64(scoredNodes[i].PricingBorda)*qos.Pricing +
			float64(scoredNodes[i].CapacityBorda)*qos.Performance
	}

	// Compute the average weighted score.
	var totalWeightedScore float64
	for _, scoredNode := range scoredNodes {
		totalWeightedScore += scoredNode.WeightedScore
	}
	nodeWeightedAverage := totalWeightedScore / float64(len(scoredNodes))

	// Aggregate scores per cluster.
	scoredClusters := map[string]*ScoredCluster{}
	for _, scoredNode := range scoredNodes {
		clusterID := scoredNode.Node.ClusterID
		if _, exists := scoredClusters[clusterID]; !exists {
			scoredClusters[clusterID] = &ScoredCluster{
				ID: clusterID,
			}
		}
		scoredClusters[clusterID].TotalWeightedScore += scoredNode.WeightedScore
		scoredClusters[clusterID].AllNodes = append(scoredClusters[clusterID].AllNodes, scoredNode.Node.Name)
	}

	// Filter nodes that meet or exceed the average weighted score.
	filteredNodes := []*ScoredNode{}
	for _, scoredNode := range scoredNodes {
		if scoredNode.WeightedScore >= nodeWeightedAverage {
			filteredNodes = append(filteredNodes, &scoredNode)
		}
	}

	// Aggregate filtered node scores by cluster.
	for _, filteredNode := range filteredNodes {
		clusterID := filteredNode.Node.ClusterID
		scoredClusters[clusterID].FilterdWeightedScore += filteredNode.WeightedScore
		scoredClusters[clusterID].FilteredNodes = append(
			scoredClusters[clusterID].FilteredNodes,
			filteredNode.Node.Name,
		)
	}

	// Sort clusters based on filtered weighted score (and total weighted score as tie-breaker).
	clusterScores := make([]ScoredCluster, 0, len(scoredClusters))
	for _, cluster := range scoredClusters {
		clusterScores = append(clusterScores, *cluster)
	}

	sort.Slice(clusterScores, func(i, j int) bool {
		if clusterScores[i].FilterdWeightedScore != clusterScores[j].FilterdWeightedScore {
			return clusterScores[i].FilterdWeightedScore > clusterScores[j].FilterdWeightedScore
		}
		return clusterScores[i].TotalWeightedScore > clusterScores[j].TotalWeightedScore
	})

	return clusterScores[0].ID, clusterScores[0].FilteredNodes
}
