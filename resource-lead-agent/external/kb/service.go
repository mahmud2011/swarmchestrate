package kb

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
)

// IClusterRepository defines the methods for ClusterRepository.
type IClusterRepository interface {
	Store(ctx context.Context, m *domain.Cluster) (*domain.Cluster, error)
	FetchOne(ctx context.Context, ctr *domain.ClusterFetchCtr) (*domain.Cluster, error)
	Fetch(ctx context.Context, ctr *domain.ClusterListCtr) ([]*domain.Cluster, error)
	Update(ctx context.Context, m *domain.Cluster) (*domain.Cluster, error)
	Delete(ctx context.Context, m *domain.Cluster) error
}

type INodeRepository interface {
	Store(ctx context.Context, m *domain.Node) (*domain.Node, error)
	FetchOne(ctx context.Context, ctr *domain.NodeFetchCtr) (*domain.Node, error)
	Fetch(ctx context.Context, ctr *domain.NodeListCtr) ([]*domain.Node, error)
	Update(ctx context.Context, m *domain.Node) (*domain.Node, error)
	Delete(ctx context.Context, m *domain.Node) error
}

// IApplicationRepository defines the methods for ApplicationRepository.
type IApplicationRepository interface {
	Store(ctx context.Context, m *domain.Application) (*domain.Application, error)
	FetchOne(ctx context.Context, ctr *domain.ApplicationFetchCtr) (*domain.Application, error)
	Fetch(ctx context.Context, ctr *domain.ApplicationListCtr) ([]*domain.Application, error)
	Update(ctx context.Context, m *domain.Application) (*domain.Application, error)
	Delete(ctx context.Context, m *domain.Application) error
}

type Service struct {
	clusterRepo     IClusterRepository
	nodeRepo        INodeRepository
	applicationRepo IApplicationRepository
}

func NewService(cr IClusterRepository, nr INodeRepository, ar IApplicationRepository) *Service {
	return &Service{
		clusterRepo:     cr,
		nodeRepo:        nr,
		applicationRepo: ar,
	}
}

func (s *Service) CreateCluster(ctx context.Context, m *domain.Cluster) (*domain.Cluster, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	m.ID = id.String()
	m.CreatedAt = time.Now().UTC()
	m.UpdatedAt = time.Now().UTC()

	cluster, err := s.clusterRepo.Store(ctx, m)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (s *Service) GetCluster(ctx context.Context, ctr *domain.ClusterFetchCtr) (*domain.Cluster, error) {
	cluster, err := s.clusterRepo.FetchOne(ctx, ctr)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (s *Service) ListClusters(ctx context.Context, ctr *domain.ClusterListCtr) ([]*domain.Cluster, error) {
	clusters, err := s.clusterRepo.Fetch(ctx, ctr)
	if err != nil {
		return nil, err
	}

	return clusters, nil
}

func (s *Service) UpdateCluster(ctx context.Context, m *domain.Cluster) (*domain.Cluster, error) {
	m.UpdatedAt = time.Now().UTC()

	cluster, err := s.clusterRepo.Update(ctx, m)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (s *Service) DeleteCluster(ctx context.Context, m *domain.Cluster) error {
	if err := s.clusterRepo.Delete(ctx, m); err != nil {
		return err
	}

	return nil
}

func (s *Service) UpsertNodes(ctx context.Context, nodes []*domain.Node) ([]*domain.Node, error) {
	var result []*domain.Node

	for _, node := range nodes {
		clusterID := node.ClusterID
		ctr := &domain.NodeFetchCtr{
			ClusterID: &clusterID,
			NodeName:  &node.Name,
		}

		existingNode, err := s.nodeRepo.FetchOne(ctx, ctr)
		if err != nil {
			return nil, err
		}

		if existingNode != nil {
			node.CreatedAt = existingNode.CreatedAt
			node.UpdatedAt = time.Now().UTC()
			updatedNode, err := s.nodeRepo.Update(ctx, node)
			if err != nil {
				return nil, err
			}
			result = append(result, updatedNode)
		} else {
			node.CreatedAt = time.Now().UTC()
			node.UpdatedAt = time.Now().UTC()
			newNode, err := s.nodeRepo.Store(ctx, node)
			if err != nil {
				return nil, err
			}
			result = append(result, newNode)
		}
	}

	return result, nil
}

func (s *Service) ListNodes(ctx context.Context, ctr *domain.NodeListCtr) ([]*domain.Node, error) {
	nodes, err := s.nodeRepo.Fetch(ctx, ctr)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (s *Service) DeleteNode(ctx context.Context, m *domain.Node) error {
	if err := s.nodeRepo.Delete(ctx, m); err != nil {
		return err
	}

	return nil
}

func (s *Service) CreateApplication(ctx context.Context, m *domain.Application) (*domain.Application, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	m.ID = id.String()
	m.CreatedAt = time.Now().UTC()
	m.UpdatedAt = time.Now().UTC()

	application, err := s.applicationRepo.Store(ctx, m)
	if err != nil {
		return nil, err
	}

	return application, nil
}

func (s *Service) GetApplication(ctx context.Context, ctr *domain.ApplicationFetchCtr) (*domain.Application, error) {
	application, err := s.applicationRepo.FetchOne(ctx, ctr)
	if err != nil {
		return nil, err
	}

	return application, nil
}

func (s *Service) ListApplications(ctx context.Context, ctr *domain.ApplicationListCtr) ([]*domain.Application, error) {
	applications, err := s.applicationRepo.Fetch(ctx, ctr)
	if err != nil {
		return nil, err
	}

	return applications, nil
}

func (s *Service) UpdateApplication(ctx context.Context, m *domain.Application) (*domain.Application, error) {
	m.UpdatedAt = time.Now().UTC()

	application, err := s.applicationRepo.Update(ctx, m)
	if err != nil {
		return nil, err
	}

	return application, nil
}

func (s *Service) DeleteApplication(ctx context.Context, m *domain.Application) error {
	if err := s.applicationRepo.Delete(ctx, m); err != nil {
		return err
	}

	return nil
}
