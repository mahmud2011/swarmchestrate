package dto

import (
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
)

type UpsertNode struct {
	ClusterID string         `json:"-"`
	Nodes     []*domain.Node `json:"nodes"`
}

func (un UpsertNode) Validate() error {
	if err := validation.ValidateStruct(&un,
		validation.Field(&un.Nodes, validation.Required),
	); err != nil {
		return err
	}

	return validation.Validate(un.Nodes)
}
