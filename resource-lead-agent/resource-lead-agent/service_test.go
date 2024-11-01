package resourceleadagent_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/algo/borda"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain/dto"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb/repository/pg"
	rla "github.com/mahmud2011/swarmchestrate/resource-lead-agent/resource-lead-agent"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/utils"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/utils/testutils"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/utils/testutils/mocks"
)

type KBServiceTestSuite struct {
	suite.Suite
	container  testcontainers.Container
	kbSvc      *kb.Service
	rlaSvc     *rla.Service
	mockRaft   *mocks.MockRaftService
	mockLeader *mocks.MockLeaderService
}

func TestKBServiceTestSuite(t *testing.T) {
	suite.Run(t, new(KBServiceTestSuite))
}

func (suite *KBServiceTestSuite) SetupTest() {
	testDB, container, err := testutils.SetupTestDatabase()
	if err != nil {
		suite.NoError(err)
	}
	suite.container = container

	clusterRepo := pg.NewClusterRepository(testDB)
	nodeRepo := pg.NewNodeRepository(testDB)
	applicationRepo := pg.NewApplicationRepository(testDB)
	suite.kbSvc = kb.NewService(clusterRepo, nodeRepo, applicationRepo)
	suite.mockRaft = new(mocks.MockRaftService)
	suite.mockLeader = new(mocks.MockLeaderService)
	suite.rlaSvc = rla.NewService(suite.kbSvc, suite.mockRaft, suite.mockLeader, &borda.Borda{})
}

func (suite *KBServiceTestSuite) TearDownTest() {
	if suite.container != nil {
		_ = testcontainers.TerminateContainer(suite.container)
	}
}

func (suite *KBServiceTestSuite) TestUpdateApplicationNameChange() {
	ctx := context.Background()

	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeCloud, IP: "192.168.1.1", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(cloudCluster)
	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeFog, IP: "192.168.1.2", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(fogCluster)
	edgeCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeEdge, IP: "192.168.1.3", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(edgeCluster)

	svcVer, err := uuid.NewV7()
	suite.NoError(err)
	svcVerStr := svcVer.String()

	cNodes := "Node-C1,NodeC2"
	fNodes := "Node-F1,NodeF2"
	eNodes := "Node-E1,NodeE2"

	status := "Healthy"
	now := time.Now().UTC()

	app := &domain.Application{
		Name: "App1",
		Labels: &domain.Labels{
			"key-1": "val-1",
		},
		QoS: &domain.QoS{
			Energy:      0.1,
			Pricing:     0.2,
			Performance: 0.3,
		},
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcVersion:   &svcVerStr,
		CloudSvcCluster:   &cloudCluster.ID,
		CloudSvcNodes:     &cNodes,
		CloudSvcStatus:    &status,
		CloudSvcHeartbeat: &now,
		FogSvc:            json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "fog-deployment"}}`),
		FogSvcVersion:     &svcVerStr,
		FogSvcCluster:     &fogCluster.ID,
		FogSvcNodes:       &fNodes,
		FogSvcStatus:      &status,
		FogSvcHeartbeat:   &now,
		EdgeSvc:           json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "edge-deployment"}}`),
		EdgeSvcVersion:    &svcVerStr,
		EdgeSvcCluster:    &edgeCluster.ID,
		EdgeSvcNodes:      &eNodes,
		EdgeSvcStatus:     &status,
		EdgeSvcHeartbeat:  &now,
	}

	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)
	suite.NotNil(createdApp)

	newName := "new-name"
	updateReq := &dto.UpdateApplicationReq{
		ID:   createdApp.ID,
		Name: &newName,
	}

	updatedApp, err := suite.rlaSvc.UpdateApplication(ctx, updateReq)
	suite.NoError(err)
	suite.NotNil(updatedApp)

	suite.Equal(newName, updatedApp.Name)

	suite.Equal(createdApp.Labels, updatedApp.Labels)
	suite.Equal(createdApp.QoS, updatedApp.QoS)
	suite.Equal(createdApp.CloudSvc, updatedApp.CloudSvc)
	suite.NotEqual(createdApp.CloudSvcVersion, updatedApp.CloudSvcVersion)
	suite.Equal(createdApp.CloudSvcCluster, updatedApp.CloudSvcCluster)
	suite.Equal(createdApp.CloudSvcNodes, updatedApp.CloudSvcNodes)
	suite.Nil(updatedApp.CloudSvcStatus)
	suite.Nil(updatedApp.CloudSvcHeartbeat)

	suite.Equal(createdApp.FogSvc, updatedApp.FogSvc)
	suite.NotEqual(createdApp.FogSvcVersion, updatedApp.FogSvcVersion)
	suite.Equal(createdApp.FogSvcCluster, updatedApp.FogSvcCluster)
	suite.Equal(createdApp.FogSvcNodes, updatedApp.FogSvcNodes)
	suite.Nil(updatedApp.FogSvcStatus)
	suite.Nil(updatedApp.FogSvcHeartbeat)

	suite.Equal(createdApp.EdgeSvc, updatedApp.EdgeSvc)
	suite.NotEqual(createdApp.EdgeSvcVersion, updatedApp.EdgeSvcVersion)
	suite.Equal(createdApp.EdgeSvcCluster, updatedApp.EdgeSvcCluster)
	suite.Equal(createdApp.EdgeSvcNodes, updatedApp.EdgeSvcNodes)
	suite.Nil(updatedApp.EdgeSvcStatus)
	suite.Nil(updatedApp.EdgeSvcHeartbeat)

	suite.NotEqual(createdApp.UpdatedAt, updatedApp.UpdatedAt)
}

func (suite *KBServiceTestSuite) TestUpdateApplicationLabelsChange() {
	ctx := context.Background()

	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeCloud, IP: "192.168.1.1", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(cloudCluster)
	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeFog, IP: "192.168.1.2", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(fogCluster)
	edgeCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeEdge, IP: "192.168.1.3", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(edgeCluster)

	svcVer, err := uuid.NewV7()
	suite.NoError(err)
	svcVerStr := svcVer.String()

	cNodes := "Node-C1,NodeC2"
	fNodes := "Node-F1,NodeF2"
	eNodes := "Node-E1,NodeE2"

	status := "Healthy"
	now := time.Now().UTC()

	app := &domain.Application{
		Name: "App1",
		Labels: &domain.Labels{
			"key-1": "val-1",
		},
		QoS: &domain.QoS{
			Energy:      0.1,
			Pricing:     0.2,
			Performance: 0.3,
		},
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcVersion:   &svcVerStr,
		CloudSvcCluster:   &cloudCluster.ID,
		CloudSvcNodes:     &cNodes,
		CloudSvcStatus:    &status,
		CloudSvcHeartbeat: &now,
		FogSvc:            json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "fog-deployment"}}`),
		FogSvcVersion:     &svcVerStr,
		FogSvcCluster:     &fogCluster.ID,
		FogSvcNodes:       &fNodes,
		FogSvcStatus:      &status,
		FogSvcHeartbeat:   &now,
		EdgeSvc:           json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "edge-deployment"}}`),
		EdgeSvcVersion:    &svcVerStr,
		EdgeSvcCluster:    &edgeCluster.ID,
		EdgeSvcNodes:      &eNodes,
		EdgeSvcStatus:     &status,
		EdgeSvcHeartbeat:  &now,
	}

	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)
	suite.NotNil(createdApp)

	newLabels := &domain.Labels{
		"key-2": "val-2",
	}

	updateReq := &dto.UpdateApplicationReq{
		ID:     createdApp.ID,
		Labels: newLabels,
	}

	updatedApp, err := suite.rlaSvc.UpdateApplication(ctx, updateReq)
	suite.NoError(err)
	suite.NotNil(updatedApp)

	suite.Equal(createdApp.Name, updatedApp.Name)

	suite.Equal(newLabels, updatedApp.Labels)

	suite.Equal(createdApp.QoS, updatedApp.QoS)
	suite.Equal(createdApp.CloudSvc, updatedApp.CloudSvc)
	suite.NotEqual(createdApp.CloudSvcVersion, updatedApp.CloudSvcVersion)
	suite.Equal(createdApp.CloudSvcCluster, updatedApp.CloudSvcCluster)
	suite.Equal(createdApp.CloudSvcNodes, updatedApp.CloudSvcNodes)
	suite.Nil(updatedApp.CloudSvcStatus)
	suite.Nil(updatedApp.CloudSvcHeartbeat)

	suite.Equal(createdApp.FogSvc, updatedApp.FogSvc)
	suite.NotEqual(createdApp.FogSvcVersion, updatedApp.FogSvcVersion)
	suite.Equal(createdApp.FogSvcCluster, updatedApp.FogSvcCluster)
	suite.Equal(createdApp.FogSvcNodes, updatedApp.FogSvcNodes)
	suite.Nil(updatedApp.FogSvcStatus)
	suite.Nil(updatedApp.FogSvcHeartbeat)

	suite.Equal(createdApp.EdgeSvc, updatedApp.EdgeSvc)
	suite.NotEqual(createdApp.EdgeSvcVersion, updatedApp.EdgeSvcVersion)
	suite.Equal(createdApp.EdgeSvcCluster, updatedApp.EdgeSvcCluster)
	suite.Equal(createdApp.EdgeSvcNodes, updatedApp.EdgeSvcNodes)
	suite.Nil(updatedApp.EdgeSvcStatus)
	suite.Nil(updatedApp.EdgeSvcHeartbeat)

	suite.NotEqual(createdApp.UpdatedAt, updatedApp.UpdatedAt)
}

func (suite *KBServiceTestSuite) TestUpdateApplicationQoSChange() {
	ctx := context.Background()

	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeCloud, IP: "192.168.1.1", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(cloudCluster)
	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeFog, IP: "192.168.1.2", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(fogCluster)
	edgeCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeEdge, IP: "192.168.1.3", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(edgeCluster)

	svcVer, err := uuid.NewV7()
	suite.NoError(err)
	svcVerStr := svcVer.String()

	cNodes := "Node-C1,NodeC2"
	fNodes := "Node-F1,NodeF2"
	eNodes := "Node-E1,NodeE2"

	status := "Healthy"
	now := time.Now().UTC()

	app := &domain.Application{
		Name: "App1",
		Labels: &domain.Labels{
			"key-1": "val-1",
		},
		QoS: &domain.QoS{
			Energy:      0.1,
			Pricing:     0.2,
			Performance: 0.3,
		},
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcVersion:   &svcVerStr,
		CloudSvcCluster:   &cloudCluster.ID,
		CloudSvcNodes:     &cNodes,
		CloudSvcStatus:    &status,
		CloudSvcHeartbeat: &now,
		FogSvc:            json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "fog-deployment"}}`),
		FogSvcVersion:     &svcVerStr,
		FogSvcCluster:     &fogCluster.ID,
		FogSvcNodes:       &fNodes,
		FogSvcStatus:      &status,
		FogSvcHeartbeat:   &now,
		EdgeSvc:           json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "edge-deployment"}}`),
		EdgeSvcVersion:    &svcVerStr,
		EdgeSvcCluster:    &edgeCluster.ID,
		EdgeSvcNodes:      &eNodes,
		EdgeSvcStatus:     &status,
		EdgeSvcHeartbeat:  &now,
	}

	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)
	suite.NotNil(createdApp)

	newQoS := &domain.QoS{
		Energy:      0.2,
		Pricing:     0.3,
		Performance: 0.4,
	}

	updateReq := &dto.UpdateApplicationReq{
		ID:  createdApp.ID,
		QoS: newQoS,
	}

	updatedApp, err := suite.rlaSvc.UpdateApplication(ctx, updateReq)
	suite.NoError(err)
	suite.NotNil(updatedApp)

	suite.Equal(createdApp.Name, updatedApp.Name)
	suite.Equal(createdApp.Labels, updatedApp.Labels)

	suite.Equal(newQoS, updatedApp.QoS)

	suite.Equal(createdApp.CloudSvc, updatedApp.CloudSvc)
	suite.NotEqual(createdApp.CloudSvcVersion, updatedApp.CloudSvcVersion)
	suite.Nil(updatedApp.CloudSvcCluster)
	suite.Nil(updatedApp.CloudSvcNodes)
	suite.Nil(updatedApp.CloudSvcStatus)
	suite.Nil(updatedApp.CloudSvcHeartbeat)

	suite.Equal(createdApp.FogSvc, updatedApp.FogSvc)
	suite.NotEqual(createdApp.FogSvcVersion, updatedApp.FogSvcVersion)
	suite.Nil(updatedApp.FogSvcCluster)
	suite.Nil(updatedApp.FogSvcNodes)
	suite.Nil(updatedApp.FogSvcStatus)
	suite.Nil(updatedApp.FogSvcHeartbeat)

	suite.Equal(createdApp.EdgeSvc, updatedApp.EdgeSvc)
	suite.NotEqual(createdApp.EdgeSvcVersion, updatedApp.EdgeSvcVersion)
	suite.Nil(updatedApp.EdgeSvcCluster)
	suite.Nil(updatedApp.EdgeSvcNodes)
	suite.Nil(updatedApp.EdgeSvcStatus)
	suite.Nil(updatedApp.EdgeSvcHeartbeat)

	suite.NotEqual(createdApp.UpdatedAt, updatedApp.UpdatedAt)
}

func (suite *KBServiceTestSuite) TestUpdateApplicationSvcChange() {
	ctx := context.Background()

	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeCloud, IP: "192.168.1.1", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(cloudCluster)
	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeFog, IP: "192.168.1.2", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(fogCluster)
	edgeCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeEdge, IP: "192.168.1.3", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	suite.NotNil(edgeCluster)

	svcVer, err := uuid.NewV7()
	suite.NoError(err)
	svcVerStr := svcVer.String()

	cNodes := "Node-C1,NodeC2"
	fNodes := "Node-F1,NodeF2"
	eNodes := "Node-E1,NodeE2"

	status := "Healthy"
	now := time.Now().UTC()

	app := &domain.Application{
		Name: "App1",
		Labels: &domain.Labels{
			"key-1": "val-1",
		},
		QoS: &domain.QoS{
			Energy:      0.1,
			Pricing:     0.2,
			Performance: 0.3,
		},
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcVersion:   &svcVerStr,
		CloudSvcCluster:   &cloudCluster.ID,
		CloudSvcNodes:     &cNodes,
		CloudSvcStatus:    &status,
		CloudSvcHeartbeat: &now,
		FogSvc:            json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "fog-deployment"}}`),
		FogSvcVersion:     &svcVerStr,
		FogSvcCluster:     &fogCluster.ID,
		FogSvcNodes:       &fNodes,
		FogSvcStatus:      &status,
		FogSvcHeartbeat:   &now,
		EdgeSvc:           json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "edge-deployment"}}`),
		EdgeSvcVersion:    &svcVerStr,
		EdgeSvcCluster:    &edgeCluster.ID,
		EdgeSvcNodes:      &eNodes,
		EdgeSvcStatus:     &status,
		EdgeSvcHeartbeat:  &now,
	}

	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)
	suite.NotNil(createdApp)

	newSvc := json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment-1"}}`)

	updateReq := &dto.UpdateApplicationReq{
		ID:       createdApp.ID,
		CloudSvc: newSvc,
	}

	updatedApp, err := suite.rlaSvc.UpdateApplication(ctx, updateReq)
	suite.NoError(err)
	suite.NotNil(updatedApp)

	suite.Equal(createdApp.Name, updatedApp.Name)
	suite.Equal(createdApp.Labels, updatedApp.Labels)
	suite.Equal(createdApp.QoS, updatedApp.QoS)

	suite.Equal(newSvc, updatedApp.CloudSvc)

	suite.NotEqual(createdApp.CloudSvcVersion, updatedApp.CloudSvcVersion)
	suite.Equal(createdApp.CloudSvcCluster, updatedApp.CloudSvcCluster)
	suite.Equal(createdApp.CloudSvcNodes, updatedApp.CloudSvcNodes)
	suite.Nil(updatedApp.CloudSvcStatus)
	suite.Nil(updatedApp.CloudSvcHeartbeat)

	suite.Equal(createdApp.FogSvc, updatedApp.FogSvc)
	suite.Equal(createdApp.FogSvcVersion, updatedApp.FogSvcVersion)
	suite.Equal(createdApp.FogSvcCluster, updatedApp.FogSvcCluster)
	suite.Equal(createdApp.FogSvcNodes, updatedApp.FogSvcNodes)
	suite.NotNil(updatedApp.FogSvcStatus)
	suite.NotNil(updatedApp.FogSvcHeartbeat)

	suite.Equal(createdApp.EdgeSvc, updatedApp.EdgeSvc)
	suite.Equal(createdApp.EdgeSvcVersion, updatedApp.EdgeSvcVersion)
	suite.Equal(createdApp.EdgeSvcCluster, updatedApp.EdgeSvcCluster)
	suite.Equal(createdApp.EdgeSvcNodes, updatedApp.EdgeSvcNodes)
	suite.NotNil(updatedApp.EdgeSvcStatus)
	suite.NotNil(updatedApp.EdgeSvcHeartbeat)

	suite.NotEqual(createdApp.UpdatedAt, updatedApp.UpdatedAt)
}

func (suite *KBServiceTestSuite) TestGetClusterScheduledApps() {
	ctx := context.Background()

	// Create clusters
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud, IP: "192.168.1.1", Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)
	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeFog, IP: "192.168.1.2", Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)
	edgeCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeEdge, IP: "192.168.1.3", Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)
	cluster4, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeEdge, IP: "192.168.1.4", Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	svcID, err := uuid.NewV7()
	suite.NoError(err)
	svcIDStr := svcID.String()
	nodes := "Node-1,Node-2"

	// app1: CloudSvc + FogSvc (both with status assumed "NULL")
	app1 := &domain.Application{
		Name:   "App1",
		Labels: &domain.Labels{},
		QoS: &domain.QoS{
			Energy: 0.1,
		},
		CloudSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcVersion: &svcIDStr,
		CloudSvcCluster: &cloudCluster.ID,
		CloudSvcNodes:   &nodes,

		FogSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "fog-deployment"}}`),
		FogSvcVersion: &svcIDStr,
		FogSvcCluster: &fogCluster.ID,
		FogSvcNodes:   &nodes,
	}

	// app2: CloudSvc only
	app2 := &domain.Application{
		Name:   "App2",
		Labels: &domain.Labels{},
		QoS: &domain.QoS{
			Energy:      0.1,
			Performance: 0.1,
		},
		CloudSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcVersion: &svcIDStr,
		CloudSvcCluster: &cloudCluster.ID,
		CloudSvcNodes:   &nodes,
	}

	// app3: EdgeSvc only.
	app3 := &domain.Application{
		Name:   "App3",
		Labels: &domain.Labels{},
		QoS: &domain.QoS{
			Energy: 0.1,
		},
		EdgeSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "edge-deployment"}}`),
		EdgeSvcVersion: &svcIDStr,
		EdgeSvcCluster: &edgeCluster.ID,
		EdgeSvcNodes:   &nodes,
	}

	status := string(domain.StateProgressing)
	app4 := &domain.Application{
		Name:   "App4",
		Labels: &domain.Labels{},
		QoS: &domain.QoS{
			Energy: 0.1,
		},
		CloudSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment-app4"}}`),
		CloudSvcVersion: &svcIDStr,
		CloudSvcCluster: &cluster4.ID,
		CloudSvcNodes:   &nodes,
		CloudSvcStatus:  &status,
	}

	createdApp1, err := suite.kbSvc.CreateApplication(ctx, app1)
	suite.NoError(err)
	suite.NotNil(createdApp1)

	createdApp2, err := suite.kbSvc.CreateApplication(ctx, app2)
	suite.NoError(err)
	suite.NotNil(createdApp2)

	createdApp3, err := suite.kbSvc.CreateApplication(ctx, app3)
	suite.NoError(err)
	suite.NotNil(createdApp3)

	createdApp4, err := suite.kbSvc.CreateApplication(ctx, app4)
	suite.NoError(err)
	suite.NotNil(createdApp4)

	// When fetching scheduled apps for cloud cluster, only app1 and app2 should be returned.
	apps, err := suite.rlaSvc.GetClusterScheduledApps(ctx, &dto.ClusterScheduledApps{
		ClusterID: cloudCluster.ID,
	})
	suite.NoError(err)
	suite.Len(apps, 2) // app4 is not scheduled because its CloudSvcStatus is not "NULL"
	suite.Equal(cloudCluster.IP, *apps[0].CloudIP)
	suite.Equal(fogCluster.IP, *apps[0].FogIP)

	// When fetching scheduled apps for edge cluster, only app3 should be returned.
	apps, err = suite.rlaSvc.GetClusterScheduledApps(ctx, &dto.ClusterScheduledApps{
		ClusterID: edgeCluster.ID,
	})
	suite.NoError(err)
	suite.Len(apps, 1)

	apps4, err := suite.rlaSvc.GetClusterScheduledApps(ctx, &dto.ClusterScheduledApps{
		ClusterID: cluster4.ID,
	})
	suite.NoError(err)
	suite.Len(apps4, 0)

	expectedLabels := &domain.Labels{
		"app.kubernetes.io/part-of":   "swarmchestrate",
		"app.kubernetes.io/component": "application",
		"app.kubernetes.io/name":      createdApp3.Name,
		"app.kubernetes.io/id":        createdApp3.ID,
		"app.kubernetes.io/version":   *createdApp3.EdgeSvcVersion,
	}
	suite.Equal(expectedLabels, apps[0].Application.Labels)
}

func (suite *KBServiceTestSuite) TestGetClusterScheduledApp() {
	ctx := context.Background()

	// Create clusters
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeCloud, IP: "192.168.1.1", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeFog, IP: "192.168.1.2", Role: domain.ClusterRoleWorker})
	suite.NoError(err)
	edgeCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{Type: domain.ClusterTypeEdge, IP: "192.168.1.3", Role: domain.ClusterRoleWorker})
	suite.NoError(err)

	svcID, err := uuid.NewV7()
	suite.NoError(err)
	svcIDStr := svcID.String()
	nodes := "Node-1,Node-2"

	app1 := &domain.Application{
		Name:   "App1",
		Labels: &domain.Labels{},
		QoS: &domain.QoS{
			Energy: 0.1,
		},

		CloudSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcVersion: &svcIDStr,
		CloudSvcCluster: &cloudCluster.ID,
		CloudSvcNodes:   &nodes,

		FogSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "fog-deployment"}}`),
		FogSvcVersion: &svcIDStr,
		FogSvcCluster: &fogCluster.ID,
		FogSvcNodes:   &nodes,
	}

	app2 := &domain.Application{
		Name:   "App2",
		Labels: &domain.Labels{},
		QoS: &domain.QoS{
			Energy:      0.1,
			Performance: 0.1,
		},

		CloudSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcVersion: &svcIDStr,
		CloudSvcCluster: &cloudCluster.ID,
		CloudSvcNodes:   &nodes,
	}

	app3 := &domain.Application{
		Name:   "App3",
		Labels: &domain.Labels{},
		QoS: &domain.QoS{
			Energy: 0.1,
		},

		EdgeSvc:        json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "edge-deployment"}}`),
		EdgeSvcVersion: &svcIDStr,
		EdgeSvcCluster: &edgeCluster.ID,
		EdgeSvcNodes:   &nodes,
	}

	createdApp1, err := suite.kbSvc.CreateApplication(ctx, app1)
	suite.NoError(err)
	suite.NotNil(createdApp1)

	createdApp2, err := suite.kbSvc.CreateApplication(ctx, app2)
	suite.NoError(err)
	suite.NotNil(createdApp2)

	createdApp3, err := suite.kbSvc.CreateApplication(ctx, app3)
	suite.NoError(err)
	suite.NotNil(createdApp3)

	app, err := suite.rlaSvc.GetClusterScheduledApp(ctx, &dto.ClusterScheduledApp{
		ClusterID: edgeCluster.ID,
		AppID:     createdApp3.ID,
	})
	suite.NoError(err)
	suite.NotNil(app)

	expectedLabels := &domain.Labels{
		"app.kubernetes.io/part-of":   "swarmchestrate",
		"app.kubernetes.io/component": "application",
		"app.kubernetes.io/name":      createdApp3.Name,
		"app.kubernetes.io/id":        createdApp3.ID,
		"app.kubernetes.io/version":   *createdApp3.EdgeSvcVersion,
	}
	suite.Equal(expectedLabels, app.Application.Labels)
	suite.Equal(app3.EdgeSvc, app.Application.Svc)
}

func (suite *KBServiceTestSuite) TestClusterApplicationStatus_ApplicationNotExists() {
	ctx := context.Background()

	// Create a cluster to use for the request (even though no application will be found)
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.100",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	// Create a request using a non-existent application ID.
	req := &dto.ClusterApplicationStatus{
		ApplicationID:      "00000000-0000-0000-0000-000000000000",
		ClusterID:          cloudCluster.ID,
		ApplicationVersion: "dummy-version",
		Status:             "Healthy",
	}
	err = suite.rlaSvc.ClusterApplicationStatus(ctx, req)
	suite.Error(err)
	suite.Equal(domain.ErrApplicationNotFound, err)
}

func (suite *KBServiceTestSuite) TestClusterApplicationStatus_ClusterIdMismatch() {
	ctx := context.Background()

	// Create two clusters.
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.101",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)
	suite.NotNil(cloudCluster)

	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.1.102",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)
	suite.NotNil(fogCluster)

	// Create an application that is deployed to the cloud cluster.
	svcVer, err := uuid.NewV7()
	suite.NoError(err)
	svcVerStr := svcVer.String()

	app := &domain.Application{
		Name:              "TestApp",
		Labels:            &domain.Labels{"test": "true"},
		QoS:               &domain.QoS{Energy: 0.1},
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "test-deployment"}}`),
		CloudSvcVersion:   &svcVerStr,
		CloudSvcCluster:   &cloudCluster.ID,
		CloudSvcNodes:     nil,
		CloudSvcStatus:    nil,
		CloudSvcHeartbeat: nil,
	}
	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)
	suite.NotNil(createdApp)

	// Now call ClusterApplicationStatus with a cluster ID that does not match the one in the application.
	req := &dto.ClusterApplicationStatus{
		ApplicationID:      createdApp.ID,
		ClusterID:          fogCluster.ID, // wrong cluster id
		ApplicationVersion: svcVerStr,
		Status:             "Healthy",
	}
	err = suite.rlaSvc.ClusterApplicationStatus(ctx, req)
	suite.Error(err)
	suite.Equal(domain.ErrApplicationNotFound, err)
}

func (suite *KBServiceTestSuite) TestClusterApplicationStatus_VersionMismatch() {
	ctx := context.Background()

	// Create a cloud cluster.
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.103",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	// Create an application deployed to the cloud cluster.
	svcVer, err := uuid.NewV7()
	suite.NoError(err)
	svcVerStr := svcVer.String()

	app := &domain.Application{
		Name:              "TestAppVersionMismatch",
		Labels:            &domain.Labels{"test": "true"},
		QoS:               &domain.QoS{Energy: 0.1},
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "test-deployment"}}`),
		CloudSvcVersion:   &svcVerStr,
		CloudSvcCluster:   &cloudCluster.ID,
		CloudSvcNodes:     nil,
		CloudSvcStatus:    nil,
		CloudSvcHeartbeat: nil,
	}
	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)

	// Call ClusterApplicationStatus with the correct cluster id but a mismatched version.
	wrongVersion := "wrong-version"
	req := &dto.ClusterApplicationStatus{
		ApplicationID:      createdApp.ID,
		ClusterID:          cloudCluster.ID,
		ApplicationVersion: wrongVersion, // version does not match the stored svc version
		Status:             "Healthy",
	}
	err = suite.rlaSvc.ClusterApplicationStatus(ctx, req)
	suite.Error(err)
	suite.Equal(domain.ErrApplicationNotSynced, err)
}

func (suite *KBServiceTestSuite) TestClusterApplicationStatus_Success() {
	ctx := context.Background()

	// Create a cloud cluster.
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.104",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	// Create an application deployed to the cloud cluster.
	svcVer, err := uuid.NewV7()
	suite.NoError(err)
	svcVerStr := svcVer.String()

	app := &domain.Application{
		Name:              "TestAppSuccess",
		Labels:            &domain.Labels{"test": "true"},
		QoS:               &domain.QoS{Energy: 0.1},
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "test-deployment"}}`),
		CloudSvcVersion:   &svcVerStr,
		CloudSvcCluster:   &cloudCluster.ID,
		CloudSvcNodes:     nil,
		CloudSvcStatus:    nil,
		CloudSvcHeartbeat: nil,
	}
	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)

	// Prepare a request that matches both the cluster ID and the application version.
	req := &dto.ClusterApplicationStatus{
		ApplicationID:      createdApp.ID,
		ClusterID:          cloudCluster.ID,
		ApplicationVersion: svcVerStr,
		Status:             "Healthy",
	}
	err = suite.rlaSvc.ClusterApplicationStatus(ctx, req)
	suite.NoError(err)

	// Retrieve the application and verify that the status and heartbeat have been updated.
	fetchedApp, err := suite.kbSvc.GetApplication(ctx, &domain.ApplicationFetchCtr{
		ApplicationID: &createdApp.ID,
	})
	suite.NoError(err)
	suite.NotNil(fetchedApp.CloudSvcStatus)
	suite.Equal("Healthy", *fetchedApp.CloudSvcStatus)
	suite.NotNil(fetchedApp.CloudSvcHeartbeat)
	// Optionally, check that the heartbeat is recent.
	suite.True(time.Since(*fetchedApp.CloudSvcHeartbeat) < time.Minute)
}

func (suite *KBServiceTestSuite) TestHandleStalledApps_AllStalled() {
	suite.mockRaft.On("IsLeader").Return(true)

	ctx := context.Background()
	// Set heartbeat to one minute ago (older than 30 seconds) so that it is considered stalled.
	pastTime := time.Now().UTC().Add(-1 * time.Minute)

	// Create clusters first since the application references them as foreign keys.
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.101",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.1.102",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	edgeCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeEdge,
		IP:   "192.168.1.103",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	app := &domain.Application{
		Name: "AllStalledApp",
		// Cloud service fields.
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcStatus:    utils.StringToPointer("Healthy"),
		CloudSvcHeartbeat: &pastTime,
		CloudSvcCluster:   utils.StringToPointer(cloudCluster.ID),
		CloudSvcNodes:     utils.StringToPointer("node1,node2"),
		// Fog service fields.
		FogSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "fog-deployment"}}`),
		FogSvcStatus:    utils.StringToPointer("Healthy"),
		FogSvcHeartbeat: &pastTime,
		FogSvcCluster:   utils.StringToPointer(fogCluster.ID),
		FogSvcNodes:     utils.StringToPointer("node3,node4"),
		// Edge service fields.
		EdgeSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "edge-deployment"}}`),
		EdgeSvcStatus:    utils.StringToPointer("Healthy"),
		EdgeSvcHeartbeat: &pastTime,
		EdgeSvcCluster:   utils.StringToPointer(edgeCluster.ID),
		EdgeSvcNodes:     utils.StringToPointer("node5,node6"),
	}

	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)

	// Call the method under test.
	err = suite.rlaSvc.HandleStalledApps()
	suite.NoError(err)

	// Re-fetch the application from the knowledge base.
	updatedApp, err := suite.kbSvc.GetApplication(ctx, &domain.ApplicationFetchCtr{
		ApplicationID: &createdApp.ID,
	})
	suite.NoError(err)

	// All service fields should have been reset to nil.
	suite.Nil(updatedApp.CloudSvcStatus)
	suite.Nil(updatedApp.CloudSvcHeartbeat)
	suite.Nil(updatedApp.CloudSvcCluster)
	suite.Nil(updatedApp.CloudSvcNodes)

	suite.Nil(updatedApp.FogSvcStatus)
	suite.Nil(updatedApp.FogSvcHeartbeat)
	suite.Nil(updatedApp.FogSvcCluster)
	suite.Nil(updatedApp.FogSvcNodes)

	suite.Nil(updatedApp.EdgeSvcStatus)
	suite.Nil(updatedApp.EdgeSvcHeartbeat)
	suite.Nil(updatedApp.EdgeSvcCluster)
	suite.Nil(updatedApp.EdgeSvcNodes)
}

func (suite *KBServiceTestSuite) TestHandleStalledApps_OnlyFogStalled() {
	suite.mockRaft.On("IsLeader").Return(true)
	ctx := context.Background()
	pastTime := time.Now().UTC().Add(-1 * time.Minute) // Stalled heartbeat.
	recentTime := time.Now().UTC()                     // Recent heartbeat.

	// Create valid clusters for cloud and fog.
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.104",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	fogCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.1.105",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	app := &domain.Application{
		Name: "OnlyFogStalledApp",
		// Cloud service: healthy heartbeat.
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcStatus:    utils.StringToPointer("Healthy"),
		CloudSvcHeartbeat: &recentTime,
		CloudSvcCluster:   utils.StringToPointer(cloudCluster.ID),
		CloudSvcNodes:     utils.StringToPointer("node1,node2"),
		// Fog service: stalled heartbeat.
		FogSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "fog-deployment"}}`),
		FogSvcStatus:    utils.StringToPointer("Healthy"),
		FogSvcHeartbeat: &pastTime,
		FogSvcCluster:   utils.StringToPointer(fogCluster.ID),
		FogSvcNodes:     utils.StringToPointer("node3,node4"),
		// No Edge service.
	}

	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)

	err = suite.rlaSvc.HandleStalledApps()
	suite.NoError(err)

	updatedApp, err := suite.kbSvc.GetApplication(ctx, &domain.ApplicationFetchCtr{
		ApplicationID: &createdApp.ID,
	})
	suite.NoError(err)

	// Cloud service should remain intact.
	suite.NotNil(updatedApp.CloudSvcStatus)
	suite.NotNil(updatedApp.CloudSvcHeartbeat)
	suite.NotNil(updatedApp.CloudSvcCluster)
	suite.NotNil(updatedApp.CloudSvcNodes)

	// Fog service should be reset to nil.
	suite.Nil(updatedApp.FogSvcStatus)
	suite.Nil(updatedApp.FogSvcHeartbeat)
	suite.Nil(updatedApp.FogSvcCluster)
	suite.Nil(updatedApp.FogSvcNodes)

	// Edge service remains nil.
	suite.Nil(updatedApp.EdgeSvcStatus)
	suite.Nil(updatedApp.EdgeSvcHeartbeat)
	suite.Nil(updatedApp.EdgeSvcCluster)
	suite.Nil(updatedApp.EdgeSvcNodes)
}

// Test Case 3: An application has only a cloud service and it is stalled.
func (suite *KBServiceTestSuite) TestHandleStalledApps_OnlyCloudStalled() {
	suite.mockRaft.On("IsLeader").Return(true)
	ctx := context.Background()
	pastTime := time.Now().UTC().Add(-1 * time.Minute)

	// Create valid cluster for cloud.
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.106",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	app := &domain.Application{
		Name:              "OnlyCloudStalledApp",
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcStatus:    utils.StringToPointer("Healthy"),
		CloudSvcHeartbeat: &pastTime,
		CloudSvcCluster:   utils.StringToPointer(cloudCluster.ID),
		CloudSvcNodes:     utils.StringToPointer("node1,node2"),
		// No Fog or Edge services.
	}

	createdApp, err := suite.kbSvc.CreateApplication(ctx, app)
	suite.NoError(err)

	err = suite.rlaSvc.HandleStalledApps()
	suite.NoError(err)

	updatedApp, err := suite.kbSvc.GetApplication(ctx, &domain.ApplicationFetchCtr{
		ApplicationID: &createdApp.ID,
	})
	suite.NoError(err)

	// Cloud service fields should be reset to nil.
	suite.Nil(updatedApp.CloudSvcStatus)
	suite.Nil(updatedApp.CloudSvcHeartbeat)
	suite.Nil(updatedApp.CloudSvcCluster)
	suite.Nil(updatedApp.CloudSvcNodes)

	// Fog and Edge remain nil.
	suite.Nil(updatedApp.FogSvcStatus)
	suite.Nil(updatedApp.FogSvcHeartbeat)
	suite.Nil(updatedApp.FogSvcCluster)
	suite.Nil(updatedApp.FogSvcNodes)

	suite.Nil(updatedApp.EdgeSvcStatus)
	suite.Nil(updatedApp.EdgeSvcHeartbeat)
	suite.Nil(updatedApp.EdgeSvcCluster)
	suite.Nil(updatedApp.EdgeSvcNodes)
}

func (suite *KBServiceTestSuite) TestHandleStalledApps_MixedApps() {
	suite.mockRaft.On("IsLeader").Return(true)
	ctx := context.Background()
	// pastTime for stalled heartbeat; recentTime for healthy heartbeat.
	pastTime := time.Now().UTC().Add(-1 * time.Minute)
	recentTime := time.Now().UTC()

	// Create a valid cluster for cloud services.
	cloudCluster, err := suite.kbSvc.CreateCluster(ctx, &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.107",
		Role: domain.ClusterRoleWorker,
	})
	suite.NoError(err)

	// Application with a stalled cloud service.
	appStalled := &domain.Application{
		Name:              "StalledApp",
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcStatus:    utils.StringToPointer("Healthy"),
		CloudSvcHeartbeat: &pastTime,
		CloudSvcCluster:   utils.StringToPointer(cloudCluster.ID),
		CloudSvcNodes:     utils.StringToPointer("node1,node2"),
	}

	// Application with a healthy cloud service.
	appHealthy := &domain.Application{
		Name:              "HealthyApp",
		CloudSvc:          json.RawMessage(`{"kind": "Deployment", "metadata": {"name": "cloud-deployment"}}`),
		CloudSvcStatus:    utils.StringToPointer("Healthy"),
		CloudSvcHeartbeat: &recentTime,
		CloudSvcCluster:   utils.StringToPointer(cloudCluster.ID),
		CloudSvcNodes:     utils.StringToPointer("node3,node4"),
	}

	createdStalled, err := suite.kbSvc.CreateApplication(ctx, appStalled)
	suite.NoError(err)
	createdHealthy, err := suite.kbSvc.CreateApplication(ctx, appHealthy)
	suite.NoError(err)

	// Call HandleStalledApps to update stalled applications.
	err = suite.rlaSvc.HandleStalledApps()
	suite.NoError(err)

	// Re-fetch both applications.
	updatedStalled, err := suite.kbSvc.GetApplication(ctx, &domain.ApplicationFetchCtr{
		ApplicationID: &createdStalled.ID,
	})
	suite.NoError(err)
	updatedHealthy, err := suite.kbSvc.GetApplication(ctx, &domain.ApplicationFetchCtr{
		ApplicationID: &createdHealthy.ID,
	})
	suite.NoError(err)

	// For the stalled app, cloud service fields should be nil.
	suite.Nil(updatedStalled.CloudSvcStatus)
	suite.Nil(updatedStalled.CloudSvcHeartbeat)
	suite.Nil(updatedStalled.CloudSvcCluster)
	suite.Nil(updatedStalled.CloudSvcNodes)

	// For the healthy app, cloud service fields should remain intact.
	suite.NotNil(updatedHealthy.CloudSvcStatus)
	suite.NotNil(updatedHealthy.CloudSvcHeartbeat)
	suite.NotNil(updatedHealthy.CloudSvcCluster)
	suite.NotNil(updatedHealthy.CloudSvcNodes)
}
