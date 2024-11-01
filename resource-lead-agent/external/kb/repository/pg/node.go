package pg

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
)

// NodeRepository handles the CRUD operations for Node entities.
type NodeRepository struct {
	*gorm.DB
}

// NewNodeRepository creates a new instance of NodeRepository.
func NewNodeRepository(conn *gorm.DB) *NodeRepository {
	return &NodeRepository{conn}
}

// Store saves a new Node entity to the database.
func (r *NodeRepository) Store(ctx context.Context, m *domain.Node) (*domain.Node, error) {
	if err := r.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}

	return m, nil
}

// FetchOne retrieves a single Node based on the provided criteria.
func (r *NodeRepository) FetchOne(ctx context.Context, ctr *domain.NodeFetchCtr) (*domain.Node, error) {
	m := domain.Node{}
	qry := r.Model(&domain.Node{})

	if ctr.ClusterID != nil {
		qry = qry.Where("cluster_id", ctr.ClusterID)
	}

	if ctr.NodeName != nil {
		qry = qry.Where("name", ctr.NodeName)
	}

	if err := qry.WithContext(ctx).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch node: %w", err)
	}

	return &m, nil
}

// Fetch retrieves multiple Nodes based on the provided criteria.
func (r *NodeRepository) Fetch(ctx context.Context, ctr *domain.NodeListCtr) ([]*domain.Node, error) {
	var m []*domain.Node
	qry := r.Model(&domain.Node{})

	if ctr.ClusterID != nil {
		qry = qry.Where("cluster_id", ctr.ClusterID)
	}

	if err := qry.WithContext(ctx).Find(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch nodes: %w", err)
	}

	return m, nil
}

func (r *NodeRepository) Update(ctx context.Context, m *domain.Node) (*domain.Node, error) {
	if err := r.WithContext(ctx).Save(m).Error; err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	return m, nil
}

// Delete removes a Node entity based on the provided criteria.
func (r *NodeRepository) Delete(ctx context.Context, m *domain.Node) error {
	if err := r.WithContext(ctx).Delete(&m).Error; err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}
