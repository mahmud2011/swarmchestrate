package domain

import (
	"time"
)

type ClusterType string

const (
	ClusterTypeCloud ClusterType = "cloud"
	ClusterTypeFog   ClusterType = "fog"
	ClusterTypeEdge  ClusterType = "edge"
)

type ClusterRole string

const (
	ClusterRoleMaster ClusterRole = "master"
	ClusterRoleWorker ClusterRole = "worker"
)

type Cluster struct {
	ID        string      `json:"id"`
	Type      ClusterType `json:"type"`
	Role      ClusterRole `json:"role"`
	RaftID    *int        `json:"raft_id"`
	IP        string      `json:"ip"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type ClusterFetchCtr struct {
	ClusterID *string
	RaftID    *int
}

type ClusterListCtr struct {
	ClusterType *ClusterType
	ClusterRole *ClusterRole
}
