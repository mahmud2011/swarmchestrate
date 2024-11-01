package pg

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
)

// ClusterRepository handles the CRUD operations for Cluster entities.
type ClusterRepository struct {
	*gorm.DB
}

// NewClusterRepository creates a new instance of ClusterRepository.
func NewClusterRepository(conn *gorm.DB) *ClusterRepository {
	return &ClusterRepository{conn}
}

// Store saves a new Cluster entity to the database.
func (r *ClusterRepository) Store(ctx context.Context, m *domain.Cluster) (*domain.Cluster, error) {
	if err := r.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}

	return m, nil
}

// FetchOne retrieves a single Cluster based on the provided criteria.
func (r *ClusterRepository) FetchOne(ctx context.Context, ctr *domain.ClusterFetchCtr) (*domain.Cluster, error) {
	m := domain.Cluster{}
	qry := r.Model(&domain.Cluster{})

	if ctr.ClusterID != nil {
		qry = qry.Where("id", ctr.ClusterID)
	}

	if ctr.RaftID != nil {
		qry = qry.Where("raft_id", ctr.RaftID)
	}

	if err := qry.WithContext(ctx).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to fetch cluster: %w", err)
	}

	return &m, nil
}

// Fetch retrieves multiple Clusters based on the provided criteria.
func (r *ClusterRepository) Fetch(ctx context.Context, ctr *domain.ClusterListCtr) ([]*domain.Cluster, error) {
	var m []*domain.Cluster
	qry := r.Model(&domain.Cluster{})

	if ctr.ClusterType != nil {
		qry = qry.Where("type", ctr.ClusterType)
	}

	if ctr.ClusterRole != nil {
		qry = qry.Where("role", ctr.ClusterRole)
	}

	if err := qry.WithContext(ctx).Find(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to fetch clusters: %w", err)
	}

	return m, nil
}

// Update modifies an existing Cluster entity in the database.
func (r *ClusterRepository) Update(ctx context.Context, m *domain.Cluster) (*domain.Cluster, error) {
	if err := r.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}

	return m, nil
}

// Delete removes Cluster entities based on the provided criteria.
func (r *ClusterRepository) Delete(ctx context.Context, m *domain.Cluster) error {
	if err := r.WithContext(ctx).Delete(&m).Error; err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	return nil
}
