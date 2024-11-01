package rest_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/algo/borda"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain/dto"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/external/kb/repository/pg"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/internal/rest"
	rla "github.com/mahmud2011/swarmchestrate/resource-lead-agent/resource-lead-agent"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/utils/testutils"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/utils/testutils/mocks"
)

type RLAHandlerTestSuite struct {
	suite.Suite
	server     *httptest.Server
	container  testcontainers.Container
	mockRaft   *mocks.MockRaftService
	mockLeader *mocks.MockLeaderService
}

func TestRLAHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RLAHandlerTestSuite))
}

func (suite *RLAHandlerTestSuite) SetupTest() {
	// Setup the test database
	testDB, container, err := testutils.SetupTestDatabase()
	suite.NoError(err)
	suite.container = container

	// Setup the repositories and service
	clusterRepo := pg.NewClusterRepository(testDB)
	nodeRepo := pg.NewNodeRepository(testDB)
	applicationRepo := pg.NewApplicationRepository(testDB)
	kbSvc := kb.NewService(clusterRepo, nodeRepo, applicationRepo)
	suite.mockRaft = new(mocks.MockRaftService)
	suite.mockLeader = new(mocks.MockLeaderService)
	rlaSvc := rla.NewService(kbSvc, suite.mockRaft, suite.mockLeader, &borda.Borda{})

	r := chi.NewRouter()
	apiRouter := chi.NewRouter()
	r.Mount("/api", apiRouter)

	handler := rest.NewRLAHandler(apiRouter, rlaSvc)

	r.Post("/v1/clusters", handler.CreateCluster)
	r.Get("/v1/clusters/{id}", handler.GetCluster)
	r.Get("/v1/clusters", handler.ListClusters)
	r.Patch("/v1/clusters/{id}", handler.UpdateCluster)
	r.Delete("/v1/clusters/{id}", handler.DeleteCluster)

	r.Put("/v1/clusters/{id}/nodes", handler.UpsertNodes)

	r.Post("/v1/applications", handler.CreateApplication)
	r.Get("/v1/applications/{id}", handler.GetApplication)
	r.Patch("/v1/applications/{id}", handler.UpdateApplication)
	r.Delete("/v1/applications/{id}", handler.DeleteApplication)

	// Create the test server
	suite.server = httptest.NewServer(r)
}

func (suite *RLAHandlerTestSuite) TearDownTest() {
	if suite.container != nil {
		_ = testcontainers.TerminateContainer(suite.container)
	}
}

func (suite *RLAHandlerTestSuite) TestAddCluster() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create the request payload
	reqPayload := dto.AddClusterReq{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.1",
		Role: domain.ClusterRoleWorker,
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	// Create the request
	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	// Check the response status code
	suite.Equal(http.StatusCreated, resp.StatusCode)

	// Decode the response body
	var cluster domain.Cluster
	err = json.NewDecoder(resp.Body).Decode(&cluster)
	suite.NoError(err)

	// Check the response body
	suite.Equal(reqPayload.Type, cluster.Type)
	suite.Equal(reqPayload.IP, cluster.IP)
	suite.Equal(domain.ClusterRoleWorker, cluster.Role)
}

func (suite *RLAHandlerTestSuite) TestAddClusterBadReq() {
	// Create the invalid request payload
	reqPayload := map[string]interface{}{
		"description": "Test Cluster",
		"type":        "invalid_type",
		"ip":          "invalid_ip",
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	// Create the request
	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	// Check the response status code
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	// Decode the response body
	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	// Check the error message
	suite.Contains(responseBody, "error")
}

func (suite *RLAHandlerTestSuite) TestGetCluster() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create a cluster first
	reqPayload := dto.AddClusterReq{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.1",
		Role: domain.ClusterRoleWorker,
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	var createdCluster domain.Cluster
	err = json.NewDecoder(resp.Body).Decode(&createdCluster)
	suite.NoError(err)

	// Test case: Valid UUID
	req, err = http.NewRequest(http.MethodGet, suite.server.URL+"/api/v1/clusters/"+createdCluster.ID, nil)
	suite.NoError(err)

	resp, err = http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var cluster domain.Cluster
	err = json.NewDecoder(resp.Body).Decode(&cluster)
	suite.NoError(err)

	suite.Equal(createdCluster.ID, cluster.ID)
	suite.Equal(createdCluster.Type, cluster.Type)
	suite.Equal(createdCluster.IP, cluster.IP)
}

func (suite *RLAHandlerTestSuite) TestGetClusterInvalidUUID() {
	// Test case: Invalid UUID
	req, err := http.NewRequest(http.MethodGet, suite.server.URL+"/api/v1/clusters/invalid-uuid", nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("Invalid cluster ID", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestGetClusterNonExistentCluster() {
	// Test case: Non-existent Cluster
	nonExistentUUID := uuid.New().String()
	req, err := http.NewRequest(http.MethodGet, suite.server.URL+"/api/v1/clusters/"+nonExistentUUID, nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("Cluster not found", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestListClusters() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create a few clusters first
	clusters := []dto.AddClusterReq{
		{
			Type: domain.ClusterTypeCloud,
			IP:   "192.168.1.1",
			Role: domain.ClusterRoleWorker,
		},
		{
			Type: domain.ClusterTypeFog,
			IP:   "192.168.1.2",
			Role: domain.ClusterRoleWorker,
		},
		{
			Type: domain.ClusterTypeEdge,
			IP:   "192.168.1.3",
			Role: domain.ClusterRoleWorker,
		},
	}

	for _, cluster := range clusters {
		reqBody, err := json.Marshal(cluster)
		suite.NoError(err)

		req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
		suite.NoError(err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		suite.NoError(err)
		_ = resp.Body.Close()

		suite.Equal(http.StatusCreated, resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, suite.server.URL+"/api/v1/clusters", nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var listedClusters []domain.Cluster
	err = json.NewDecoder(resp.Body).Decode(&listedClusters)
	suite.NoError(err)

	suite.Len(listedClusters, 3)
}

func (suite *RLAHandlerTestSuite) TestListClustersInvalidType() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create a few clusters first
	clusters := []dto.AddClusterReq{
		{
			Type: domain.ClusterTypeCloud,
			IP:   "192.168.1.1",
			Role: domain.ClusterRoleWorker,
		},
		{
			Type: domain.ClusterTypeFog,
			IP:   "192.168.1.2",
			Role: domain.ClusterRoleWorker,
		},
		{
			Type: domain.ClusterTypeEdge,
			IP:   "192.168.1.3",
			Role: domain.ClusterRoleWorker,
		},
	}

	for _, cluster := range clusters {
		reqBody, err := json.Marshal(cluster)
		suite.NoError(err)

		req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
		suite.NoError(err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		suite.NoError(err)
		_ = resp.Body.Close()

		suite.Equal(http.StatusCreated, resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, suite.server.URL+"/api/v1/clusters?type=invalid", nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("Invalid cluster type", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestListClustersByType() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create a few clusters first
	clusters := []dto.AddClusterReq{
		{
			Type: domain.ClusterTypeCloud,
			IP:   "192.168.1.1",
			Role: domain.ClusterRoleWorker,
		},
		{
			Type: domain.ClusterTypeFog,
			IP:   "192.168.1.2",
			Role: domain.ClusterRoleWorker,
		},
		{
			Type: domain.ClusterTypeEdge,
			IP:   "192.168.1.3",
			Role: domain.ClusterRoleWorker,
		},
	}

	for _, cluster := range clusters {
		reqBody, err := json.Marshal(cluster)
		suite.NoError(err)

		req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
		suite.NoError(err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		suite.NoError(err)
		_ = resp.Body.Close()

		suite.Equal(http.StatusCreated, resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, suite.server.URL+"/api/v1/clusters?type=fog", nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var filteredClusters []domain.Cluster
	err = json.NewDecoder(resp.Body).Decode(&filteredClusters)
	suite.NoError(err)

	suite.Len(filteredClusters, 1)
	suite.Equal(domain.ClusterTypeFog, filteredClusters[0].Type)
}

func (suite *RLAHandlerTestSuite) TestUpdateCluster() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create a cluster first
	reqPayload := dto.AddClusterReq{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.1",
		Role: domain.ClusterRoleWorker,
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	createdResp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer createdResp.Body.Close()

	var createdCluster domain.Cluster
	err = json.NewDecoder(createdResp.Body).Decode(&createdCluster)
	suite.NoError(err)

	// Test case: Update cluster with valid data
	updatePayload := dto.UpdateClusterReq{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.1.2",
	}
	updateBody, err := json.Marshal(updatePayload)
	suite.NoError(err)

	req, err = http.NewRequest(
		http.MethodPatch,
		suite.server.URL+"/api/v1/clusters/"+createdCluster.ID,
		bytes.NewReader(updateBody),
	)
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var updatedCluster domain.Cluster
	err = json.NewDecoder(resp.Body).Decode(&updatedCluster)
	suite.NoError(err)

	suite.Equal(createdCluster.ID, updatedCluster.ID)
	suite.Equal(updatePayload.Type, updatedCluster.Type)
	suite.Equal(updatePayload.IP, updatedCluster.IP)
}

func (suite *RLAHandlerTestSuite) TestUpdateClusterInvalidUUID() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create a cluster first
	reqPayload := dto.AddClusterReq{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.1",
		Role: domain.ClusterRoleWorker,
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	createdResp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer createdResp.Body.Close()

	var createdCluster domain.Cluster
	err = json.NewDecoder(createdResp.Body).Decode(&createdCluster)
	suite.NoError(err)

	// Test case: Update cluster with invalid UUID
	updatePayload := dto.UpdateClusterReq{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.1.2",
	}
	updateBody, err := json.Marshal(updatePayload)
	suite.NoError(err)

	req, err = http.NewRequest(
		http.MethodPatch,
		suite.server.URL+"/api/v1/clusters/invalid-uuid",
		bytes.NewReader(updateBody),
	)
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("Invalid cluster ID", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestUpdateClusterNonExistentCluster() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create a cluster first
	reqPayload := dto.AddClusterReq{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.1",
		Role: domain.ClusterRoleWorker,
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	createdResp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer createdResp.Body.Close()

	var createdCluster domain.Cluster
	err = json.NewDecoder(createdResp.Body).Decode(&createdCluster)
	suite.NoError(err)

	// Test case: Update non-existent cluster
	updatePayload := dto.UpdateClusterReq{
		Type: domain.ClusterTypeFog,
		IP:   "192.168.1.2",
	}
	updateBody, err := json.Marshal(updatePayload)
	suite.NoError(err)

	nonExistentUUID := uuid.New().String()
	req, err = http.NewRequest(
		http.MethodPatch,
		suite.server.URL+"/api/v1/clusters/"+nonExistentUUID,
		bytes.NewReader(updateBody),
	)
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("invalid cluster id", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestDeleteCluster() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create a cluster first
	reqPayload := dto.AddClusterReq{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.1",
		Role: domain.ClusterRoleWorker,
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/clusters", bytes.NewReader(reqBody))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	createdResp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer createdResp.Body.Close()

	var createdCluster domain.Cluster
	err = json.NewDecoder(createdResp.Body).Decode(&createdCluster)
	suite.NoError(err)

	// Test case: Delete cluster with valid UUID
	req, err = http.NewRequest(http.MethodDelete, suite.server.URL+"/api/v1/clusters/"+createdCluster.ID, nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNoContent, resp.StatusCode)

	// Verify the cluster is deleted
	req, err = http.NewRequest(http.MethodGet, suite.server.URL+"/api/v1/clusters/"+createdCluster.ID, nil)
	suite.NoError(err)

	resp, err = http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNotFound, resp.StatusCode)
}

func (suite *RLAHandlerTestSuite) TestDeleteClusterInvalidUUID() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Delete cluster with invalid UUID
	req, err := http.NewRequest(http.MethodDelete, suite.server.URL+"/api/v1/clusters/invalid-uuid", nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("Invalid cluster ID", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestDeleteClusterNonExistentCluster() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Delete non-existent cluster
	nonExistentUUID := uuid.New().String()
	req, err := http.NewRequest(http.MethodDelete, suite.server.URL+"/api/v1/clusters/"+nonExistentUUID, nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNoContent, resp.StatusCode)
}

func (suite *RLAHandlerTestSuite) TestUpsertNodesCreate() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create the cluster first
	clusterPayload := dto.AddClusterReq{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.1",
		Role: domain.ClusterRoleMaster,
	}
	clusterBody, err := json.Marshal(clusterPayload)
	suite.NoError(err)

	clusterReq, err := http.NewRequest(
		http.MethodPost,
		suite.server.URL+"/api/v1/clusters",
		bytes.NewReader(clusterBody),
	)
	suite.NoError(err)
	clusterReq.Header.Set("Content-Type", "application/json")

	clusterResp, err := http.DefaultClient.Do(clusterReq)
	suite.NoError(err)
	defer clusterResp.Body.Close()

	suite.Equal(http.StatusCreated, clusterResp.StatusCode)

	var cluster domain.Cluster
	err = json.NewDecoder(clusterResp.Body).Decode(&cluster)
	suite.NoError(err)

	// Create the request payload for upserting nodes
	reqPayload := dto.UpsertNode{
		ClusterID: cluster.ID,
		Nodes: []*domain.Node{
			{
				Name:                   "Node1",
				Location:               "Europe/Berlin",
				CPU:                    4,
				CPUArch:                "x86_64",
				Memory:                 16.0,
				NetworkBandwidth:       100.0,
				EphemeralStorage:       500.0,
				Energy:                 200.0,
				Pricing:                0.5,
				IsReady:                true,
				IsSchedulable:          true,
				IsPIDPressureExists:    false,
				IsMemoryPressureExists: false,
				IsDiskPressureExists:   false,
			},
		},
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	// Create the request for upserting nodes
	req, err := http.NewRequest(
		http.MethodPut,
		suite.server.URL+"/api/v1/clusters/"+cluster.ID+"/nodes",
		bytes.NewReader(reqBody),
	)
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	// Check the response status code
	suite.Equal(http.StatusOK, resp.StatusCode)

	// Decode the response body
	var nodes []domain.Node
	err = json.NewDecoder(resp.Body).Decode(&nodes)
	suite.NoError(err)

	// Check the response body
	suite.Len(nodes, 1)
	suite.Equal(reqPayload.Nodes[0].Name, nodes[0].Name)
	suite.Equal(reqPayload.Nodes[0].Location, nodes[0].Location)
	suite.Equal(reqPayload.Nodes[0].CPU, nodes[0].CPU)
	suite.Equal(reqPayload.Nodes[0].CPUArch, nodes[0].CPUArch)
	suite.Equal(reqPayload.Nodes[0].Memory, nodes[0].Memory)
	suite.Equal(reqPayload.Nodes[0].NetworkBandwidth, nodes[0].NetworkBandwidth)
	suite.Equal(reqPayload.Nodes[0].EphemeralStorage, nodes[0].EphemeralStorage)
	suite.Equal(reqPayload.Nodes[0].Energy, nodes[0].Energy)
	suite.Equal(reqPayload.Nodes[0].Pricing, nodes[0].Pricing)
	suite.Equal(reqPayload.Nodes[0].IsReady, nodes[0].IsReady)
	suite.Equal(reqPayload.Nodes[0].IsSchedulable, nodes[0].IsSchedulable)
	suite.Equal(reqPayload.Nodes[0].IsPIDPressureExists, nodes[0].IsPIDPressureExists)
	suite.Equal(reqPayload.Nodes[0].IsMemoryPressureExists, nodes[0].IsMemoryPressureExists)
	suite.Equal(reqPayload.Nodes[0].IsDiskPressureExists, nodes[0].IsDiskPressureExists)
}

func (suite *RLAHandlerTestSuite) TestUpsertNodesUpdateAndCreate() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create the cluster first
	clusterPayload := dto.AddClusterReq{
		Type: domain.ClusterTypeCloud,
		IP:   "192.168.1.1",
		Role: domain.ClusterRoleWorker,
	}
	clusterBody, err := json.Marshal(clusterPayload)
	suite.NoError(err)

	clusterReq, err := http.NewRequest(
		http.MethodPost,
		suite.server.URL+"/api/v1/clusters",
		bytes.NewReader(clusterBody),
	)
	suite.NoError(err)
	clusterReq.Header.Set("Content-Type", "application/json")

	clusterResp, err := http.DefaultClient.Do(clusterReq)
	suite.NoError(err)
	defer clusterResp.Body.Close()

	suite.Equal(http.StatusCreated, clusterResp.StatusCode)

	var cluster domain.Cluster
	err = json.NewDecoder(clusterResp.Body).Decode(&cluster)
	suite.NoError(err)

	// Create the initial node
	initialNodePayload := dto.UpsertNode{

		Nodes: []*domain.Node{
			{
				Name:                   "Node1",
				Location:               "Europe/Berlin",
				CPU:                    4,
				CPUArch:                "x86_64",
				Memory:                 16.0,
				NetworkBandwidth:       100.0,
				EphemeralStorage:       500.0,
				Energy:                 200.0,
				Pricing:                0.5,
				IsReady:                true,
				IsSchedulable:          true,
				IsPIDPressureExists:    false,
				IsMemoryPressureExists: false,
				IsDiskPressureExists:   false,
			},
		},
	}
	initialNodeBody, err := json.Marshal(initialNodePayload)
	suite.NoError(err)

	initialNodeReq, err := http.NewRequest(
		http.MethodPut,
		suite.server.URL+"/api/v1/clusters/"+cluster.ID+"/nodes",
		bytes.NewReader(initialNodeBody),
	)
	suite.NoError(err)
	initialNodeReq.Header.Set("Content-Type", "application/json")

	initialNodeResp, err := http.DefaultClient.Do(initialNodeReq)
	suite.NoError(err)
	defer initialNodeResp.Body.Close()

	suite.Equal(http.StatusOK, initialNodeResp.StatusCode)

	// Create the request payload for upserting nodes (one update, one create)
	reqPayload := dto.UpsertNode{
		ClusterID: cluster.ID,
		Nodes: []*domain.Node{
			{
				Name:                   "Node1", // This node will be updated
				Location:               "Europe/Berlin",
				CPU:                    8, // Updated CPU
				CPUArch:                "x86_64",
				Memory:                 32.0,   // Updated Memory
				NetworkBandwidth:       200.0,  // Updated Network Bandwidth
				EphemeralStorage:       1000.0, // Updated Ephemeral Storage
				Energy:                 400.0,  // Updated Energy
				Pricing:                1.0,    // Updated Pricing
				IsReady:                true,
				IsSchedulable:          true,
				IsPIDPressureExists:    false,
				IsMemoryPressureExists: false,
				IsDiskPressureExists:   false,
			},
			{
				Name:                   "Node2", // This node will be created
				Location:               "Europe/Berlin",
				CPU:                    4,
				CPUArch:                "x86_64",
				Memory:                 16.0,
				NetworkBandwidth:       100.0,
				EphemeralStorage:       500.0,
				Energy:                 200.0,
				Pricing:                0.5,
				IsReady:                true,
				IsSchedulable:          true,
				IsPIDPressureExists:    false,
				IsMemoryPressureExists: false,
				IsDiskPressureExists:   false,
			},
		},
	}
	reqBody, err := json.Marshal(reqPayload)
	suite.NoError(err)

	// Create the request for upserting nodes
	req, err := http.NewRequest(
		http.MethodPut,
		suite.server.URL+"/api/v1/clusters/"+cluster.ID+"/nodes",
		bytes.NewReader(reqBody),
	)
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	// Check the response status code
	suite.Equal(http.StatusOK, resp.StatusCode)

	// Decode the response body
	var nodes []domain.Node
	err = json.NewDecoder(resp.Body).Decode(&nodes)
	suite.NoError(err)

	// Check the response body
	suite.Len(nodes, 2)

	// Check the updated node
	suite.Equal(reqPayload.Nodes[0].Name, nodes[0].Name)
	suite.Equal(reqPayload.Nodes[0].Location, nodes[0].Location)
	suite.Equal(reqPayload.Nodes[0].CPU, nodes[0].CPU)
	suite.Equal(reqPayload.Nodes[0].CPUArch, nodes[0].CPUArch)
	suite.Equal(reqPayload.Nodes[0].Memory, nodes[0].Memory)
	suite.Equal(reqPayload.Nodes[0].NetworkBandwidth, nodes[0].NetworkBandwidth)
	suite.Equal(reqPayload.Nodes[0].EphemeralStorage, nodes[0].EphemeralStorage)
	suite.Equal(reqPayload.Nodes[0].Energy, nodes[0].Energy)
	suite.Equal(reqPayload.Nodes[0].Pricing, nodes[0].Pricing)
	suite.Equal(reqPayload.Nodes[0].IsReady, nodes[0].IsReady)
	suite.Equal(reqPayload.Nodes[0].IsSchedulable, nodes[0].IsSchedulable)
	suite.Equal(reqPayload.Nodes[0].IsPIDPressureExists, nodes[0].IsPIDPressureExists)
	suite.Equal(reqPayload.Nodes[0].IsMemoryPressureExists, nodes[0].IsMemoryPressureExists)
	suite.Equal(reqPayload.Nodes[0].IsDiskPressureExists, nodes[0].IsDiskPressureExists)

	// Check the created node
	suite.Equal(reqPayload.Nodes[1].Name, nodes[1].Name)
	suite.Equal(reqPayload.Nodes[1].Location, nodes[1].Location)
	suite.Equal(reqPayload.Nodes[1].CPU, nodes[1].CPU)
	suite.Equal(reqPayload.Nodes[1].CPUArch, nodes[1].CPUArch)
	suite.Equal(reqPayload.Nodes[1].Memory, nodes[1].Memory)
	suite.Equal(reqPayload.Nodes[1].NetworkBandwidth, nodes[1].NetworkBandwidth)
	suite.Equal(reqPayload.Nodes[1].EphemeralStorage, nodes[1].EphemeralStorage)
	suite.Equal(reqPayload.Nodes[1].Energy, nodes[1].Energy)
	suite.Equal(reqPayload.Nodes[1].Pricing, nodes[1].Pricing)
	suite.Equal(reqPayload.Nodes[1].IsReady, nodes[1].IsReady)
	suite.Equal(reqPayload.Nodes[1].IsSchedulable, nodes[1].IsSchedulable)
	suite.Equal(reqPayload.Nodes[1].IsPIDPressureExists, nodes[1].IsPIDPressureExists)
	suite.Equal(reqPayload.Nodes[1].IsMemoryPressureExists, nodes[1].IsMemoryPressureExists)
	suite.Equal(reqPayload.Nodes[1].IsDiskPressureExists, nodes[1].IsDiskPressureExists)
}

func (suite *RLAHandlerTestSuite) TestAddApplication() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Add application with valid data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add application YAML
	applicationYAML := `
metadata:
  name: test-application
  labels:
    app: test
  qos:
    energy: 0.5
    pricing: 0.3
    latency: 0.2
    performance: 0.8
`
	part, err := writer.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(applicationYAML))
	suite.NoError(err)

	// Add cloud service YAML
	cloudSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: cloud-service
`
	part, err = writer.CreateFormFile("cloud_svc", "cloud_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(cloudSvcYAML))
	suite.NoError(err)

	// Add fog service YAML
	fogSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: fog-service
`
	part, err = writer.CreateFormFile("fog_svc", "fog_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(fogSvcYAML))
	suite.NoError(err)

	// Add edge service YAML
	edgeSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: edge-service
`
	part, err = writer.CreateFormFile("edge_svc", "edge_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(edgeSvcYAML))
	suite.NoError(err)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/applications", body)
	suite.NoError(err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusCreated, resp.StatusCode)

	var application domain.Application
	err = json.NewDecoder(resp.Body).Decode(&application)
	suite.NoError(err)

	suite.Equal("test-application", application.Name)

	labels := *application.Labels
	suite.NoError(err)
	suite.Equal("test", labels["app"])

	expectedQoS := domain.QoS{
		Energy:      0.5,
		Pricing:     0.3,
		Performance: 0.8,
	}
	suite.Equal(expectedQoS, *application.QoS)

	suite.NotNil(application.CloudSvcVersion)
	suite.NotNil(application.FogSvcVersion)
	suite.NotNil(application.EdgeSvcVersion)
}

func (suite *RLAHandlerTestSuite) TestAddApplicationInvalidYAML() {
	suite.mockRaft.On("IsLeader").Return(true)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add application YAML
	invalidYAML := `
metadata:
  labels:
    app: test
  qos:
    energy: 0.5
    pricing: 0.3
    latency: 0.2
    performance: 0.8
`
	part, err := writer.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(invalidYAML))
	suite.NoError(err)

	// Add cloud service YAML
	cloudSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: cloud-service
`
	part, err = writer.CreateFormFile("cloud_svc", "cloud_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(cloudSvcYAML))
	suite.NoError(err)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/applications", body)
	suite.NoError(err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *RLAHandlerTestSuite) TestAddApplicationMissingFile() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Add application with missing file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/applications", body)
	suite.NoError(err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *RLAHandlerTestSuite) TestGetApplication() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Add application with valid data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add application YAML
	applicationYAML := `
metadata:
  name: test-application
  labels:
    app: test
  qos:
    energy: 0.5
    pricing: 0.3
    latency: 0.2
    performance: 0.8
`
	part, err := writer.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(applicationYAML))
	suite.NoError(err)

	// Add cloud service YAML
	cloudSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: cloud-service
`
	part, err = writer.CreateFormFile("cloud_svc", "cloud_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(cloudSvcYAML))
	suite.NoError(err)

	// Add fog service YAML
	fogSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: fog-service
`
	part, err = writer.CreateFormFile("fog_svc", "fog_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(fogSvcYAML))
	suite.NoError(err)

	// Add edge service YAML
	edgeSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: edge-service
`
	part, err = writer.CreateFormFile("edge_svc", "edge_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(edgeSvcYAML))
	suite.NoError(err)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/api/v1/applications", body)
	suite.NoError(err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusCreated, resp.StatusCode)

	var createdApplication domain.Application
	err = json.NewDecoder(resp.Body).Decode(&createdApplication)
	suite.NoError(err)

	// Test case: Get application with valid UUID
	req, err = http.NewRequest(http.MethodGet, suite.server.URL+"/v1/applications/"+createdApplication.ID, nil)
	suite.NoError(err)

	resp, err = http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var fetchedApplication domain.Application
	err = json.NewDecoder(resp.Body).Decode(&fetchedApplication)
	suite.NoError(err)

	suite.Equal(createdApplication.ID, fetchedApplication.ID)
	suite.Equal(createdApplication.Name, fetchedApplication.Name)

	labels := *fetchedApplication.Labels
	suite.NoError(err)
	suite.Equal("test", labels["app"])

	expectedQoS := domain.QoS{
		Energy:      0.5,
		Pricing:     0.3,
		Performance: 0.8,
	}
	suite.Equal(expectedQoS, *fetchedApplication.QoS)
}

func (suite *RLAHandlerTestSuite) TestGetApplicationInvalidUUID() {
	// Test case: Get application with invalid UUID
	req, err := http.NewRequest(http.MethodGet, suite.server.URL+"/v1/applications/invalid-uuid", nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("Invalid application ID", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestGetApplicationNonExistentApplication() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Get non-existent application
	nonExistentUUID := uuid.New().String()
	req, err := http.NewRequest(http.MethodGet, suite.server.URL+"/v1/applications/"+nonExistentUUID, nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("application not found", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestUpdateApplication() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create an application first
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add application YAML
	applicationYAML := `
metadata:
  name: test-application
  labels:
    app: test
  qos:
    energy: 0.5
    pricing: 0.3
    latency: 0.2
    performance: 0.8
`
	part, err := writer.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(applicationYAML))
	suite.NoError(err)

	// Add cloud service YAML
	cloudSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: cloud-service
`
	part, err = writer.CreateFormFile("cloud_svc", "cloud_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(cloudSvcYAML))
	suite.NoError(err)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/v1/applications", body)
	suite.NoError(err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	var createdApplication domain.Application
	err = json.NewDecoder(resp.Body).Decode(&createdApplication)
	suite.NoError(err)

	// Test case: Update application with valid data
	updateBody := &bytes.Buffer{}
	updateWriter := multipart.NewWriter(updateBody)

	// Update application YAML
	updateApplicationYAML := `
metadata:
  name: updated-application
  labels:
    app: updated
  qos:
    energy: 0.6
    pricing: 0.4
    latency: 0.3
    performance: 0.9
`
	part, err = updateWriter.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(updateApplicationYAML))
	suite.NoError(err)

	updateWriter.Close()

	req, err = http.NewRequest(http.MethodPatch, suite.server.URL+"/v1/applications/"+createdApplication.ID, updateBody)
	suite.NoError(err)
	req.Header.Set("Content-Type", updateWriter.FormDataContentType())

	resp, err = http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var updatedApplication domain.Application
	err = json.NewDecoder(resp.Body).Decode(&updatedApplication)
	suite.NoError(err)

	suite.Equal(createdApplication.ID, updatedApplication.ID)
	suite.Equal("updated-application", updatedApplication.Name)

	labels := *updatedApplication.Labels
	suite.NoError(err)
	suite.Equal("updated", labels["app"])

	expectedQoS := domain.QoS{
		Energy:      0.6,
		Pricing:     0.4,
		Performance: 0.9,
	}
	suite.Equal(expectedQoS, *updatedApplication.QoS)
}

func (suite *RLAHandlerTestSuite) TestUpdateApplicationInvalidUUID() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Update application with invalid UUID
	updateBody := &bytes.Buffer{}
	updateWriter := multipart.NewWriter(updateBody)

	// Update application YAML
	updateApplicationYAML := `
metadata:
  name: updated-application
  labels:
    app: updated
  qos:
    energy: 0.6
    pricing: 0.4
    latency: 0.3
    performance: 0.9
`
	part, err := updateWriter.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(updateApplicationYAML))
	suite.NoError(err)

	updateWriter.Close()

	req, err := http.NewRequest(http.MethodPatch, suite.server.URL+"/v1/applications/invalid-uuid", updateBody)
	suite.NoError(err)
	req.Header.Set("Content-Type", updateWriter.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *RLAHandlerTestSuite) TestUpdateApplicationNonExistentApplication() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Update non-existent application
	updateBody := &bytes.Buffer{}
	updateWriter := multipart.NewWriter(updateBody)

	// Update application YAML
	updateApplicationYAML := `
metadata:
  name: updated-application
  labels:
    app: updated
  qos:
    energy: 0.6
    pricing: 0.4
    latency: 0.3
    performance: 0.9
`
	part, err := updateWriter.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(updateApplicationYAML))
	suite.NoError(err)

	updateWriter.Close()

	nonExistentUUID := uuid.New().String()
	req, err := http.NewRequest(http.MethodPatch, suite.server.URL+"/v1/applications/"+nonExistentUUID, updateBody)
	suite.NoError(err)
	req.Header.Set("Content-Type", updateWriter.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNotFound, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	suite.NoError(err)

	suite.Contains(responseBody, "error")
	suite.Equal("invalid application id", responseBody["error"])
}

func (suite *RLAHandlerTestSuite) TestUpdateApplicationChangeCloudSvc() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create an application first
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add application YAML
	applicationYAML := `
metadata:
  name: test-update-cloud-svc-application
  labels:
    app: test
  qos:
    energy: 0.5
    pricing: 0.3
    latency: 0.2
    performance: 0.8
`
	part, err := writer.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(applicationYAML))
	suite.NoError(err)

	// Add cloud service YAML
	cloudSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: cloud-service
`
	part, err = writer.CreateFormFile("cloud_svc", "cloud_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(cloudSvcYAML))
	suite.NoError(err)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/v1/applications", body)
	suite.NoError(err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	createResp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer createResp.Body.Close()

	var createdApplication domain.Application
	err = json.NewDecoder(createResp.Body).Decode(&createdApplication)
	suite.NoError(err)

	// Test case: Update application with valid data
	updateBody := &bytes.Buffer{}
	updateWriter := multipart.NewWriter(updateBody)

	// Update application YAML
	updateCloudSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: cloud-service-updated
`
	part, err = updateWriter.CreateFormFile("cloud_svc", "cloud_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(updateCloudSvcYAML))
	suite.NoError(err)

	updateWriter.Close()

	req, err = http.NewRequest(http.MethodPatch, suite.server.URL+"/v1/applications/"+createdApplication.ID, updateBody)
	suite.NoError(err)
	req.Header.Set("Content-Type", updateWriter.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var updatedApplication domain.Application
	err = json.NewDecoder(resp.Body).Decode(&updatedApplication)
	suite.NoError(err)

	suite.Equal(createdApplication.ID, updatedApplication.ID)
	suite.Equal(createdApplication.Name, updatedApplication.Name)

	labels := *updatedApplication.Labels
	suite.NoError(err)
	suite.Equal("test", labels["app"])

	expectedQoS := domain.QoS{
		Energy:      0.5,
		Pricing:     0.3,
		Performance: 0.8,
	}
	suite.Equal(expectedQoS, *updatedApplication.QoS)
}

func (suite *RLAHandlerTestSuite) TestDeleteApplication() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Create an application first
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add application YAML
	applicationYAML := `
metadata:
  name: test-application
  labels:
    app: test
  qos:
    energy: 0.5
    pricing: 0.3
    latency: 0.2
    performance: 0.8
`
	part, err := writer.CreateFormFile("application", "application.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(applicationYAML))
	suite.NoError(err)

	// Add cloud service YAML
	cloudSvcYAML := `
apiVersion: v1
kind: Service
metadata:
  name: cloud-service
`
	part, err = writer.CreateFormFile("cloud_svc", "cloud_svc.yaml")
	suite.NoError(err)
	_, err = part.Write([]byte(cloudSvcYAML))
	suite.NoError(err)

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/v1/applications", body)
	suite.NoError(err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	var createdApplication domain.Application
	err = json.NewDecoder(resp.Body).Decode(&createdApplication)
	suite.NoError(err)

	// Test case: Delete application with valid UUID
	req, err = http.NewRequest(http.MethodDelete, suite.server.URL+"/v1/applications/"+createdApplication.ID, nil)
	suite.NoError(err)

	resp, err = http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNoContent, resp.StatusCode)

	// Verify the application is deleted
	req, err = http.NewRequest(http.MethodGet, suite.server.URL+"/v1/applications/"+createdApplication.ID, nil)
	suite.NoError(err)

	resp, err = http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNotFound, resp.StatusCode)
}

func (suite *RLAHandlerTestSuite) TestDeleteApplicationInvalidUUID() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Delete application with invalid UUID
	req, err := http.NewRequest(http.MethodDelete, suite.server.URL+"/v1/applications/invalid-uuid", nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *RLAHandlerTestSuite) TestDeleteApplicationNonExistentApplication() {
	suite.mockRaft.On("IsLeader").Return(true)
	// Test case: Delete non-existent application
	nonExistentUUID := uuid.New().String()
	req, err := http.NewRequest(http.MethodDelete, suite.server.URL+"/v1/applications/"+nonExistentUUID, nil)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNoContent, resp.StatusCode)
}
