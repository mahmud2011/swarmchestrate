package kb_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb/repository/pg"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/utils/testutils"
)

type KBServiceTestSuite struct {
	suite.Suite
	container testcontainers.Container
	svc       *kb.Service
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
	suite.svc = kb.NewService(clusterRepo, nodeRepo, applicationRepo)
}

func (suite *KBServiceTestSuite) TearDownTest() {
	if suite.container != nil {
		_ = testcontainers.TerminateContainer(suite.container)
	}
}

func (suite *KBServiceTestSuite) TestCreateCluster() {
	ctx := context.Background()
	cluster := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}

	createdCluster, err := suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)
	suite.NotNil(createdCluster)
	suite.NotEmpty(createdCluster.ID)
	suite.Equal(cluster.Type, createdCluster.Type)
	suite.NotEmpty(createdCluster.CreatedAt)
	suite.NotEmpty(createdCluster.UpdatedAt)
}

func (suite *KBServiceTestSuite) TestCreateClusterDuplicateIP() {
	ctx := context.Background()
	cluster := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}

	createdCluster, err := suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)
	suite.NotNil(createdCluster)

	cluster = &domain.Cluster{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}

	_, err = suite.svc.CreateCluster(ctx, cluster)
	suite.Error(err)
}

func (suite *KBServiceTestSuite) TestGetCluster() {
	ctx := context.Background()
	cluster := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}

	createdCluster, err := suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)

	fetchCtr := &domain.ClusterFetchCtr{ClusterID: &createdCluster.ID}
	fetchedCluster, err := suite.svc.GetCluster(ctx, fetchCtr)
	suite.NoError(err)
	suite.NotNil(fetchedCluster)
	suite.Equal(createdCluster.ID, fetchedCluster.ID)
	suite.Equal(createdCluster.Type, fetchedCluster.Type)
	suite.Equal(createdCluster.CreatedAt, fetchedCluster.CreatedAt.UTC())
	suite.Equal(createdCluster.UpdatedAt, fetchedCluster.UpdatedAt.UTC())
}

func (suite *KBServiceTestSuite) TestListClusters() {
	ctx := context.Background()
	cluster1 := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}
	cluster2 := &domain.Cluster{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.0.2",
		Role: domain.ClusterRoleMaster,
	}

	_, err := suite.svc.CreateCluster(ctx, cluster1)
	suite.NoError(err)
	_, err = suite.svc.CreateCluster(ctx, cluster2)
	suite.NoError(err)

	clusters, err := suite.svc.ListClusters(ctx, &domain.ClusterListCtr{})
	suite.NoError(err)
	suite.Len(clusters, 2)
}

func (suite *KBServiceTestSuite) TestListClustersByType() {
	ctx := context.Background()
	cluster1 := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}
	cluster2 := &domain.Cluster{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.0.2",
		Role: domain.ClusterRoleMaster,
	}

	_, err := suite.svc.CreateCluster(ctx, cluster1)
	suite.NoError(err)
	_, err = suite.svc.CreateCluster(ctx, cluster2)
	suite.NoError(err)

	clusterType := domain.ClusterTypeCloud
	clusters, err := suite.svc.ListClusters(ctx, &domain.ClusterListCtr{ClusterType: &clusterType})
	suite.NoError(err)
	suite.Len(clusters, 1)
	suite.Equal(cluster1.Type, clusters[0].Type)
}

func (suite *KBServiceTestSuite) TestUpdateCluster() {
	ctx := context.Background()
	cluster := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}

	createdCluster, err := suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)

	createdCluster.Type = domain.ClusterTypeFog
	updatedCluster, err := suite.svc.UpdateCluster(ctx, createdCluster)
	suite.NoError(err)
	suite.Equal(createdCluster.Type, updatedCluster.Type)
}

func (suite *KBServiceTestSuite) TestDeleteCluster() {
	ctx := context.Background()
	cluster := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}

	createdCluster, err := suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)

	err = suite.svc.DeleteCluster(ctx, createdCluster)
	suite.NoError(err)

	fetchCtr := &domain.ClusterFetchCtr{ClusterID: &createdCluster.ID}
	fetchedCluster, err := suite.svc.GetCluster(ctx, fetchCtr)
	suite.NoError(err)
	suite.Nil(fetchedCluster)
}

func (suite *KBServiceTestSuite) TestUpsertNodes() {
	ctx := context.Background()

	// Create a cluster
	cluster := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}
	createdCluster, err := suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)
	suite.NotNil(createdCluster)

	// Create initial nodes
	nodes := []*domain.Node{
		{
			Name:                   "Node1",
			ClusterID:              createdCluster.ID,
			Location:               "Europe/Berlin",
			CPU:                    4,
			CPUArch:                "amd64",
			Memory:                 1099511627776,
			NetworkBandwidth:       100000000000,
			EphemeralStorage:       1099511627776,
			Energy:                 25,
			Pricing:                10,
			IsReady:                true,
			IsSchedulable:          true,
			IsPIDPressureExists:    false,
			IsMemoryPressureExists: false,
			IsDiskPressureExists:   false,
		},
		{
			Name:                   "Node2",
			ClusterID:              createdCluster.ID,
			Location:               "Europe/Berlin",
			CPU:                    8,
			CPUArch:                "arm64",
			Memory:                 1099511627886,
			NetworkBandwidth:       400000000000,
			EphemeralStorage:       2099511627776,
			Energy:                 12,
			Pricing:                5,
			IsReady:                true,
			IsSchedulable:          true,
			IsPIDPressureExists:    false,
			IsMemoryPressureExists: false,
			IsDiskPressureExists:   false,
		},
	}

	// Upsert nodes
	upsertedNodes, err := suite.svc.UpsertNodes(ctx, nodes)
	suite.NoError(err)
	suite.Len(upsertedNodes, 2)

	// Update one node
	nodes[0].CPU = 8

	// Add a new node
	newNode := &domain.Node{
		Name:                   "Node3",
		ClusterID:              createdCluster.ID,
		Location:               "Europe/Berlin",
		CPU:                    8,
		CPUArch:                "arm64",
		Memory:                 1099511627886,
		NetworkBandwidth:       400000000000,
		EphemeralStorage:       2099511627776,
		Energy:                 12,
		Pricing:                5,
		IsReady:                true,
		IsSchedulable:          true,
		IsPIDPressureExists:    false,
		IsMemoryPressureExists: false,
		IsDiskPressureExists:   false,
	}
	nodes = append(nodes, newNode)

	// Upsert nodes again
	upsertedNodes, err = suite.svc.UpsertNodes(ctx, nodes)
	suite.NoError(err)
	suite.Len(upsertedNodes, 3)

	// Verify the updates
	listCtr := &domain.NodeListCtr{}
	fetchedNodes, err := suite.svc.ListNodes(ctx, listCtr)
	suite.NoError(err)
	suite.Len(fetchedNodes, 3)

	for _, node := range fetchedNodes {
		if node.Name == "Node1" {
			suite.Equal(8, node.CPU)
		}
	}
}

func (suite *KBServiceTestSuite) TestListNodes() {
	ctx := context.Background()

	// Create a cluster
	cluster := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}
	createdCluster, err := suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)
	suite.NotNil(createdCluster)

	nodes := []*domain.Node{
		{
			Name:                   "Node1",
			ClusterID:              createdCluster.ID,
			Location:               "Europe/Berlin",
			CPU:                    4,
			CPUArch:                "amd64",
			Memory:                 1099511627776,
			NetworkBandwidth:       100000000000,
			EphemeralStorage:       1099511627776,
			Energy:                 25,
			Pricing:                10,
			IsReady:                true,
			IsSchedulable:          true,
			IsPIDPressureExists:    false,
			IsMemoryPressureExists: false,
			IsDiskPressureExists:   false,
		},
		{
			Name:                   "Node2",
			ClusterID:              createdCluster.ID,
			Location:               "Europe/Berlin",
			CPU:                    8,
			CPUArch:                "arm64",
			Memory:                 1099511627886,
			NetworkBandwidth:       400000000000,
			EphemeralStorage:       2099511627776,
			Energy:                 12,
			Pricing:                5,
			IsReady:                true,
			IsSchedulable:          true,
			IsPIDPressureExists:    false,
			IsMemoryPressureExists: false,
			IsDiskPressureExists:   false,
		},
	}

	upsertedNodes, err := suite.svc.UpsertNodes(ctx, nodes)
	suite.NoError(err)
	suite.Len(upsertedNodes, 2)

	cluster = &domain.Cluster{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.0.2",
		Role: domain.ClusterRoleMaster,
	}
	createdCluster, err = suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)
	suite.NotNil(createdCluster)

	nodes = []*domain.Node{
		{
			Name:                   "Node1",
			ClusterID:              createdCluster.ID,
			Location:               "Europe/Berlin",
			CPU:                    4,
			CPUArch:                "amd64",
			Memory:                 1099511627776,
			NetworkBandwidth:       100000000000,
			EphemeralStorage:       1099511627776,
			Energy:                 25,
			Pricing:                10,
			IsReady:                true,
			IsSchedulable:          true,
			IsPIDPressureExists:    false,
			IsMemoryPressureExists: false,
			IsDiskPressureExists:   false,
		},
		{
			Name:                   "Node2",
			ClusterID:              createdCluster.ID,
			Location:               "Europe/Berlin",
			CPU:                    8,
			CPUArch:                "arm64",
			Memory:                 1099511627886,
			NetworkBandwidth:       400000000000,
			EphemeralStorage:       2099511627776,
			Energy:                 12,
			Pricing:                5,
			IsReady:                true,
			IsSchedulable:          true,
			IsPIDPressureExists:    false,
			IsMemoryPressureExists: false,
			IsDiskPressureExists:   false,
		},
	}

	upsertedNodes, err = suite.svc.UpsertNodes(ctx, nodes)
	suite.NoError(err)
	suite.Len(upsertedNodes, 2)

	// List nodes
	fetchCtr := &domain.NodeListCtr{}
	nodes, err = suite.svc.ListNodes(ctx, fetchCtr)
	suite.NoError(err)
	suite.Len(nodes, 4)

	fetchCtr = &domain.NodeListCtr{
		ClusterID: &createdCluster.ID,
	}
	nodes, err = suite.svc.ListNodes(ctx, fetchCtr)
	suite.NoError(err)
	suite.Len(nodes, 2)
}

func (suite *KBServiceTestSuite) TestDeleteNode() {
	ctx := context.Background()

	// Create a cluster
	cluster := &domain.Cluster{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.0.1",
		Role: domain.ClusterRoleMaster,
	}
	createdCluster, err := suite.svc.CreateCluster(ctx, cluster)
	suite.NoError(err)
	suite.NotNil(createdCluster)

	// Add a node
	node := []*domain.Node{
		{

			Name:                   "Node1",
			ClusterID:              createdCluster.ID,
			CPU:                    4,
			CPUArch:                "amd64",
			Memory:                 1099511627776,
			NetworkBandwidth:       100000000000,
			EphemeralStorage:       1099511627776,
			Energy:                 25,
			Pricing:                10,
			IsReady:                true,
			IsSchedulable:          true,
			IsPIDPressureExists:    false,
			IsMemoryPressureExists: false,
			IsDiskPressureExists:   false,
		},
	}
	_, err = suite.svc.UpsertNodes(ctx, node)
	suite.NoError(err)

	// Delete the node
	err = suite.svc.DeleteNode(ctx, node[0])
	suite.NoError(err)
}

func (suite *KBServiceTestSuite) TestAddApplication() {
	ctx := context.Background()

	// Add an application
	application := &domain.Application{
		Name: "Test Application",
		Labels: &domain.Labels{
			"label-key": "label-value",
		},
		QoS: &domain.QoS{
			Energy:      0.5,
			Pricing:     0.7,
			Performance: 0.9,
		},
		CloudSvc: json.RawMessage(`{
			"apiVersion": "v1",
			"kind": "ConfigMap",
			"metadata": {
				"name": "example-config",
				"namespace": "default"
			},
			"data": {
				"key": "value"
			}
		}`),
	}
	createdApplication, err := suite.svc.CreateApplication(ctx, application)
	suite.NoError(err)
	suite.NotNil(createdApplication)
	suite.Equal(application.Name, createdApplication.Name)
}

func (suite *KBServiceTestSuite) TestGetApplication() {
	ctx := context.Background()

	// Add an application
	application := &domain.Application{
		Name: "Test Application",
		Labels: &domain.Labels{
			"label-key": "label-value",
		},
		QoS: &domain.QoS{
			Energy:      0.5,
			Pricing:     0.7,
			Performance: 0.9,
		},
		CloudSvc: json.RawMessage(`{
			"apiVersion": "v1",
			"kind": "ConfigMap",
			"metadata": {
				"name": "example-config",
				"namespace": "default"
			},
			"data": {
				"key": "value"
			}
		}`),
	}
	createdApplication, err := suite.svc.CreateApplication(ctx, application)
	suite.NoError(err)

	// Get the application
	createdApplicationID := createdApplication.ID
	fetchCtr := &domain.ApplicationFetchCtr{ApplicationID: &createdApplicationID}
	fetchedApplication, err := suite.svc.GetApplication(ctx, fetchCtr)
	suite.NoError(err)
	suite.NotNil(fetchedApplication)
	suite.Equal(createdApplication.ID, fetchedApplication.ID)
}

func (suite *KBServiceTestSuite) TestListApplications() {
	ctx := context.Background()

	// Add applications
	application1 := &domain.Application{
		Name: "Test Application 1",
		Labels: &domain.Labels{
			"label-key-1": "label-value-1",
		},
		QoS: &domain.QoS{
			Energy:      0.5,
			Pricing:     0.7,
			Performance: 0.9,
		},
		CloudSvc: json.RawMessage(`{
			"apiVersion": "v1",
			"kind": "ConfigMap",
			"metadata": {
				"name": "example-config",
				"namespace": "default"
			},
			"data": {
				"key": "value"
			}
		}`),
	}
	application2 := &domain.Application{
		Name: "Test Application 2",
		Labels: &domain.Labels{
			"label-key-2": "label-value-2",
		},
		QoS: &domain.QoS{
			Energy:      0.6,
			Pricing:     0.8,
			Performance: 1.0,
		},
		CloudSvc: json.RawMessage(`{
			"apiVersion": "v1",
			"kind": "ConfigMap",
			"metadata": {
				"name": "example-config",
				"namespace": "default"
			},
			"data": {
				"key": "value"
			}
		}`),
	}
	_, err := suite.svc.CreateApplication(ctx, application1)
	suite.NoError(err)
	_, err = suite.svc.CreateApplication(ctx, application2)
	suite.NoError(err)

	// List applications
	ctr := &domain.ApplicationListCtr{}
	applications, err := suite.svc.ListApplications(ctx, ctr)
	suite.NoError(err)
	suite.Len(applications, 2)
}

func (suite *KBServiceTestSuite) TestListPendingApplications() {
	ctx := context.Background()

	// Add applications
	application1 := &domain.Application{
		Name: "Test Application 1",
		Labels: &domain.Labels{
			"label-key-1": "label-value-1",
		},
		QoS: &domain.QoS{
			Energy:      0.5,
			Pricing:     0.7,
			Performance: 0.9,
		},
		CloudSvc: json.RawMessage(`{
			"apiVersion": "v1",
			"kind": "ConfigMap",
			"metadata": {
				"name": "example-config",
				"namespace": "default"
			},
			"data": {
				"key": "value"
			}
		}`),
	}
	application2 := &domain.Application{
		Name: "Test Application 2",
		Labels: &domain.Labels{
			"label-key-2": "label-value-2",
		},
		QoS: &domain.QoS{
			Energy:      0.6,
			Pricing:     0.8,
			Performance: 1.0,
		},
		CloudSvc: json.RawMessage(`{
			"apiVersion": "v1",
			"kind": "ConfigMap",
			"metadata": {
				"name": "example-config",
				"namespace": "default"
			},
			"data": {
				"key": "value"
			}
		}`),
	}
	_, err := suite.svc.CreateApplication(ctx, application1)
	suite.NoError(err)
	_, err = suite.svc.CreateApplication(ctx, application2)
	suite.NoError(err)

	// List applications
	clusterID := "NULL"
	ctr := &domain.ApplicationListCtr{CloudSvcCluster: &clusterID}
	applications, err := suite.svc.ListApplications(ctx, ctr)
	suite.NoError(err)
	suite.Len(applications, 2)
}

func (suite *KBServiceTestSuite) TestUpdateApplication() {
	ctx := context.Background()

	// Add an application
	application := &domain.Application{
		Name: "Test Application",
		Labels: &domain.Labels{
			"label-key-1": "label-value-1",
		},
		QoS: &domain.QoS{
			Energy:      0.5,
			Pricing:     0.7,
			Performance: 0.9,
		},
		CloudSvc: json.RawMessage(`{
			"apiVersion": "v1",
			"kind": "ConfigMap",
			"metadata": {
				"name": "example-config",
				"namespace": "default"
			},
			"data": {
				"key": "value"
			}
		}`),
	}
	createdApplication, err := suite.svc.CreateApplication(ctx, application)
	suite.NoError(err)

	createdApplication.Name = "Test Application Updated"
	// Update the application
	updatedApplication, err := suite.svc.UpdateApplication(ctx, createdApplication)
	suite.NoError(err)
	suite.Equal(createdApplication.Name, updatedApplication.Name)
	suite.NotNil(updatedApplication.CloudSvc)
}

func (suite *KBServiceTestSuite) TestDeleteApplication() {
	ctx := context.Background()

	// Add an application
	application := &domain.Application{
		Name: "Test Application",
		Labels: &domain.Labels{
			"label-key-1": "label-value-1",
		},
		QoS: &domain.QoS{
			Energy:      0.5,
			Pricing:     0.7,
			Performance: 0.9,
		},
		CloudSvc: json.RawMessage(`{
			"apiVersion": "v1",
			"kind": "ConfigMap",
			"metadata": {
				"name": "example-config",
				"namespace": "default"
			},
			"data": {
				"key": "value"
			}
		}`),
	}
	createdApplication, err := suite.svc.CreateApplication(ctx, application)
	suite.NoError(err)

	// Delete the application
	err = suite.svc.DeleteApplication(ctx, createdApplication)
	suite.NoError(err)

	createdApplicationID := createdApplication.ID
	// Verify the application is deleted
	fetchCtr := &domain.ApplicationFetchCtr{ApplicationID: &createdApplicationID}
	fetchedApplication, err := suite.svc.GetApplication(ctx, fetchCtr)
	suite.NoError(err)
	suite.Nil(fetchedApplication)
}
