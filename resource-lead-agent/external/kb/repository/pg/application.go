package pg

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
)

// ApplicationRepository handles the CRUD operations for Application entities.
type ApplicationRepository struct {
	*gorm.DB
}

// NewApplicationRepository creates a new instance of ApplicationRepository.
func NewApplicationRepository(conn *gorm.DB) *ApplicationRepository {
	return &ApplicationRepository{conn}
}

// Store saves a new Application entity to the database.
func (r *ApplicationRepository) Store(ctx context.Context, m *domain.Application) (*domain.Application, error) {
	if err := r.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}

	return m, nil
}

// FetchOne retrieves a single Application based on the provided criteria.
func (r *ApplicationRepository) FetchOne(
	ctx context.Context,
	ctr *domain.ApplicationFetchCtr,
) (*domain.Application, error) {
	m := domain.Application{}
	qry := r.Model(&domain.Application{})

	if ctr.ApplicationID != nil {
		qry = qry.Where("id", ctr.ApplicationID)
	}

	if ctr.CloudSvcCluster != nil {
		qry = qry.Where("cloud_svc_cluster", ctr.CloudSvcCluster)
	}

	if ctr.FogSvcCluster != nil {
		qry = qry.Where("fog_svc_cluster", ctr.FogSvcCluster)
	}

	if ctr.EdgeSvcCluster != nil {
		qry = qry.Where("edge_svc_cluster", ctr.EdgeSvcCluster)
	}

	if err := qry.WithContext(ctx).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch application: %w", err)
	}

	return &m, nil
}

// Fetch retrieves multiple Applications based on the provided criteria.
func (r *ApplicationRepository) Fetch(
	ctx context.Context,
	ctr *domain.ApplicationListCtr,
) ([]*domain.Application, error) {
	var m []*domain.Application
	qry := r.Model(&domain.Application{})

	if ctr.CloudSvcCluster != nil {
		if *ctr.CloudSvcCluster == "NULL" {
			qry = qry.Where("cloud_svc_cluster IS NULL")
		} else {
			qry = qry.Where("cloud_svc_cluster", ctr.CloudSvcCluster)
		}
	}

	if ctr.FogSvcCluster != nil {
		if *ctr.FogSvcCluster == "NULL" {
			qry = qry.Where("fog_svc_cluster IS NULL")
		} else {
			qry = qry.Where("fog_svc_cluster", ctr.FogSvcCluster)
		}
	}

	if ctr.EdgeSvcCluster != nil {
		if *ctr.EdgeSvcCluster == "NULL" {
			qry = qry.Where("edge_svc_cluster IS NULL")
		} else {
			qry = qry.Where("edge_svc_cluster", ctr.EdgeSvcCluster)
		}
	}

	if ctr.CloudSvcStatus != nil {
		if *ctr.CloudSvcStatus == "NULL" {
			qry = qry.Where("cloud_svc_status IS NULL")
		} else {
			qry = qry.Where("cloud_svc_status", ctr.CloudSvcStatus)
		}
	}

	if ctr.FogSvcStatus != nil {
		if *ctr.FogSvcStatus == "NULL" {
			qry = qry.Where("fog_svc_status IS NULL")
		} else {
			qry = qry.Where("fog_svc_status", ctr.FogSvcStatus)
		}
	}

	if ctr.EdgeSvcStatus != nil {
		if *ctr.EdgeSvcStatus == "NULL" {
			qry = qry.Where("edge_svc_status IS NULL")
		} else {
			qry = qry.Where("edge_svc_status", ctr.EdgeSvcStatus)
		}
	}

	if err := qry.WithContext(ctx).Find(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to fetch applications: %w", err)
	}

	return m, nil
}

// Update modifies an existing Application entity in the database.
func (r *ApplicationRepository) Update(ctx context.Context, m *domain.Application) (*domain.Application, error) {
	if err := r.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, fmt.Errorf("failed to update application: %w", err)
	}

	return m, nil
}

// Delete removes Application entities based on the provided criteria.
func (r *ApplicationRepository) Delete(ctx context.Context, m *domain.Application) error {
	if err := r.WithContext(ctx).Delete(&m).Error; err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	return nil
}
