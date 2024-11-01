package dto

import (
	"github.com/mahmud2011/swarmchestrate/resource-agent/domain"
)

type AddClusterReq struct {
	Type domain.ClusterType `json:"type"`
	Role domain.ClusterRole `json:"role"`
	IP   string             `json:"ip"`
}
