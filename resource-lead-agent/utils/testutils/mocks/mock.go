package mocks

import (
	"context"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain/dto"
	"github.com/stretchr/testify/mock"
)

type MockRaftService struct {
	mock.Mock
}

func (m *MockRaftService) IsLeader() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockRaftService) GetLeaderID() uint64 {
	args := m.Called()
	return args.Get(0).(uint64)
}

type MockLeaderService struct {
	mock.Mock
}

func (m *MockLeaderService) CreateCluster(ctx context.Context, leaderHost string, d *dto.AddClusterReq) (*domain.Cluster, error) {
	args := m.Called(ctx, leaderHost, d)
	return args.Get(0).(*domain.Cluster), args.Error(1)
}

func (m *MockLeaderService) UpdateCluster(ctx context.Context, leaderHost string, d *dto.UpdateClusterReq) (*domain.Cluster, error) {
	args := m.Called(ctx, leaderHost, d)
	return args.Get(0).(*domain.Cluster), args.Error(1)
}
