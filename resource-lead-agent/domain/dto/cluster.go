package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
)

type AddClusterReq struct {
	Type   domain.ClusterType `json:"type"`
	Role   domain.ClusterRole `json:"role"`
	IP     string             `json:"ip"`
	RaftID *int               `json:"raft_id"`
}

func (ac AddClusterReq) Validate() error {
	return validation.ValidateStruct(&ac,
		validation.Field(&ac.IP, validation.Required, is.IP),
		validation.Field(&ac.Type, validation.Required, validation.In(
			domain.ClusterTypeCloud,
			domain.ClusterTypeFog,
			domain.ClusterTypeEdge,
		)),
		validation.Field(&ac.Role, validation.Required, validation.In(
			domain.ClusterRoleMaster,
			domain.ClusterRoleWorker,
		)),
	)
}

type UpdateClusterReq struct {
	ID     string             `json:"-"`
	Type   domain.ClusterType `json:"type"`
	Role   domain.ClusterRole `json:"role"`
	IP     string             `json:"ip"`
	RaftID *int               `json:"raft_id"`
}

type UpdateClusterResp struct {
	ID            string             `json:"id"`
	Type          domain.ClusterType `json:"type"`
	IP            string             `json:"ip"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	ClusterConfig map[string]string  `json:"cluster_config"`
}

func (uc UpdateClusterReq) Validate() error {
	return validation.ValidateStruct(&uc,
		validation.Field(&uc.Type, validation.In(
			domain.ClusterTypeCloud,
			domain.ClusterTypeFog,
			domain.ClusterTypeEdge,
		)),
	)
}
