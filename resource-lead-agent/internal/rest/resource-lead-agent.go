package rest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/google/uuid"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain/dto"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/utils"
)

type IRLAService interface {
	Isleader() bool
	GetLeaderHost(ctx context.Context) (string, error)

	CreateCluster(ctx context.Context, d *dto.AddClusterReq) (*domain.Cluster, error)
	GetCluster(ctx context.Context, ctr *domain.ClusterFetchCtr) (*domain.Cluster, error)
	ListClusters(ctx context.Context, ctr *domain.ClusterListCtr) ([]*domain.Cluster, error)
	UpdateCluster(ctx context.Context, d *dto.UpdateClusterReq) (*domain.Cluster, error)
	DeleteCluster(ctx context.Context, id string) error

	SyncClusterConfig(ctx context.Context) (map[string]string, error)

	GetClusterScheduledApps(ctx context.Context, d *dto.ClusterScheduledApps) ([]*dto.ScheduledApplication, error)
	GetClusterScheduledApp(ctx context.Context, d *dto.ClusterScheduledApp) (*dto.ScheduledApplication, error)
	ClusterApplicationStatus(ctx context.Context, d *dto.ClusterApplicationStatus) error

	UpsertNodes(ctx context.Context, clusterID string, un *dto.UpsertNode) ([]*domain.Node, error)

	CreateApplication(ctx context.Context, d *dto.AddApplicationReq) (*domain.Application, error)
	GetApplication(ctx context.Context, ctr *domain.ApplicationFetchCtr) (*domain.Application, error)
	ListApplications(ctx context.Context, ctr *domain.ApplicationListCtr) ([]*domain.Application, error)
	UpdateApplication(ctx context.Context, d *dto.UpdateApplicationReq) (*domain.Application, error)
	DeleteApplication(ctx context.Context, id string) error
}

type RLAHandler struct {
	rlaSvc IRLAService
}

type Application struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
}

type Metadata struct {
	Name   string        `yaml:"name"`
	Labels domain.Labels `yaml:"labels"`
	QoS    domain.QoS    `yaml:"qos"`
}

func NewRLAHandler(r *chi.Mux, rlaSvc IRLAService) *RLAHandler {
	handler := &RLAHandler{
		rlaSvc: rlaSvc,
	}

	r.Post("/v1/clusters", handler.CreateCluster)
	r.Get("/v1/clusters/{id}", handler.GetCluster)
	r.Get("/v1/clusters", handler.ListClusters)
	r.Patch("/v1/clusters/{id}", handler.UpdateCluster)
	r.Delete("/v1/clusters/{id}", handler.DeleteCluster)

	r.Put("/v1/clusters/{id}/nodes", handler.UpsertNodes)

	r.Post("/v1/applications", handler.CreateApplication)
	r.Get("/v1/applications/{id}", handler.GetApplication)
	r.Get("/v1/applications", handler.ListApplications)
	r.Patch("/v1/applications/{id}", handler.UpdateApplication)
	r.Delete("/v1/applications/{id}", handler.DeleteApplication)

	// worker clusters will get clusters' configs (currently IPs) through this API
	r.Get("/v1/clusters/config", handler.SyncClusterConfig)
	// worker cluster will get scheduled apps through this API
	r.Get("/v1/clusters/{cluster-id}/applications/scheduled", handler.ListClusterScheduledApps)
	// This API is sending heartbeat and checking if the app exists or the cluster is still responsible for the app or not
	r.Post("/v1/clusters/{cluster-id}/applications/{application-id}/status", handler.ClusterApplicationStatus)

	return handler
}

func (h *RLAHandler) forwardToLeader(w http.ResponseWriter, r *http.Request) {
	leaderHost, err := h.rlaSvc.GetLeaderHost(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	resp, err := utils.ForwardReq(leaderHost, r)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": "Failed to copy response"})
		return
	}
}

// CreateCluster godoc
// @Summary      Create a new cluster
// @Description  Accepts cluster configuration and registers it in the system
// @Tags         clusters
// @Accept       json
// @Produce      json
// @Param        cluster  body      dto.AddClusterReq  true  "Cluster request payload"
// @Success      201      {object}  domain.Cluster
// @Failure      400      {object}  map[string]interface{}
// @Failure      500      {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters [post]
func (h *RLAHandler) CreateCluster(w http.ResponseWriter, r *http.Request) {
	req := new(dto.AddClusterReq)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	cluster, err := h.rlaSvc.CreateCluster(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, cluster)
}

// SyncClusterConfig godoc
// @Summary      Get cluster configuration for worker clusters
// @Description  Used by worker clusters to fetch current configuration (e.g., IPs)
// @Tags         clusters
// @Produce      json
// @Success      200  {object}  interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters/config [get]
func (h *RLAHandler) SyncClusterConfig(w http.ResponseWriter, r *http.Request) {
	resp, err := h.rlaSvc.SyncClusterConfig(r.Context())
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, resp)
}

// GetCluster godoc
// @Summary      Get cluster by ID
// @Description  Returns a cluster by UUID if it exists
// @Tags         clusters
// @Produce      json
// @Param        id   path      string  true  "Cluster ID"
// @Success      200  {object}  domain.Cluster
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters/{id} [get]
func (h *RLAHandler) GetCluster(w http.ResponseWriter, r *http.Request) {
	paramID := chi.URLParam(r, "id")

	if err := validation.Validate(paramID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster ID"})
		return
	}

	cluster, err := h.rlaSvc.GetCluster(r.Context(), &domain.ClusterFetchCtr{ClusterID: &paramID})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if cluster == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]interface{}{"error": "Cluster not found"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, cluster)
}

// ListClusters godoc
// @Summary      List clusters
// @Description  Returns all clusters, optionally filtered by type
// @Tags         clusters
// @Produce      json
// @Param        type  query     string  false  "Cluster Type"  Enums(cloud, fog, edge)
// @Success      200   {array}   domain.Cluster
// @Failure      400   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters [get]
func (h *RLAHandler) ListClusters(w http.ResponseWriter, r *http.Request) {
	paramType := r.URL.Query().Get("type")
	ctr := &domain.ClusterListCtr{}

	if paramType != "" {
		clusterType := domain.ClusterType(paramType)
		ctr.ClusterType = &clusterType

		if err := validation.Validate(clusterType, validation.In(
			domain.ClusterTypeCloud,
			domain.ClusterTypeFog,
			domain.ClusterTypeEdge,
		)); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster type"})
			return
		}
	}

	clusters, err := h.rlaSvc.ListClusters(r.Context(), ctr)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, clusters)
}

// UpdateCluster godoc
// @Summary      Update a cluster
// @Description  Updates an existing cluster's configuration
// @Tags         clusters
// @Accept       json
// @Produce      json
// @Param        id    path      string              true  "Cluster ID"
// @Param        body  body      dto.UpdateClusterReq  true  "Cluster update payload"
// @Success      200   {object}  dto.UpdateClusterResp
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters/{id} [patch]
func (h *RLAHandler) UpdateCluster(w http.ResponseWriter, r *http.Request) {
	paramID := chi.URLParam(r, "id")

	if err := validation.Validate(paramID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster ID"})
		return
	}

	req := new(dto.UpdateClusterReq)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	req.ID = paramID
	updatedCluster, err := h.rlaSvc.UpdateCluster(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if updatedCluster == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]interface{}{"error": "invalid cluster id"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, updatedCluster)
}

// DeleteCluster godoc
// @Summary      Delete a cluster
// @Description  Deletes a cluster by UUID
// @Tags         clusters
// @Produce      json
// @Param        id   path      string  true  "Cluster ID"
// @Success      204  {object}  nil
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters/{id} [delete]
func (h *RLAHandler) DeleteCluster(w http.ResponseWriter, r *http.Request) {
	if !h.rlaSvc.Isleader() {
		h.forwardToLeader(w, r)
		return
	}

	paramID := chi.URLParam(r, "id")

	if err := validation.Validate(paramID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster ID"})
		return
	}

	if err := h.rlaSvc.DeleteCluster(r.Context(), paramID); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusNoContent)
	render.JSON(w, r, nil)
}

// UpsertNodes godoc
// @Summary      Upsert cluster nodes
// @Description  Adds or updates nodes in the specified cluster
// @Tags         nodes
// @Accept       json
// @Produce      json
// @Param        id    path      string         true  "Cluster ID"
// @Param        body  body      dto.UpsertNode true  "Node upsert payload"
// @Success      200   {array}   domain.Node
// @Failure      400   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters/{id}/nodes [put]
func (h *RLAHandler) UpsertNodes(w http.ResponseWriter, r *http.Request) {
	if !h.rlaSvc.Isleader() {
		h.forwardToLeader(w, r)
		return
	}

	paramID := chi.URLParam(r, "id")

	if err := validation.Validate(paramID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster ID"})
		return
	}

	req := &dto.UpsertNode{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	req.ClusterID = paramID

	nodes, err := h.rlaSvc.UpsertNodes(r.Context(), paramID, req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, nodes)
}

// CreateApplication godoc
// @Summary      Create an application
// @Description  Creates a new application from uploaded YAML service files
// @Tags         applications
// @Accept       multipart/form-data
// @Produce      json
// @Param        application  formData  file  false  "Application YAML"
// @Param        cloud_svc    formData  file  false  "Cloud service YAML"
// @Param        fog_svc      formData  file  false  "Fog service YAML"
// @Param        edge_svc     formData  file  false  "Edge service YAML"
// @Success      201          {object}  domain.Application
// @Failure      400          {object}  map[string]interface{}
// @Failure      500          {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/applications [post]
func (h *RLAHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	if !h.rlaSvc.Isleader() {
		h.forwardToLeader(w, r)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	req := new(dto.AddApplicationReq)

	applicationSvcJSON, err := utils.YAMLtoJSON(r, "application")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid application YAML"})
		return
	}

	cloudSvcJSON, err := utils.YAMLtoJSON(r, "cloud_svc")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cloud_svc YAML"})
		return
	}

	fogSvcJSON, err := utils.YAMLtoJSON(r, "fog_svc")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid fog_svc YAML"})
		return
	}

	edgeSvcJSON, err := utils.YAMLtoJSON(r, "edge_svc")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid edge_svc YAML"})
		return
	}

	if applicationSvcJSON != nil {
		appJsonData := []*Application{}

		if err := json.Unmarshal(applicationSvcJSON, &appJsonData); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]interface{}{"error": "error parsing application"})
			return
		}

		req.Name = appJsonData[0].Metadata.Name
		req.Labels = &appJsonData[0].Metadata.Labels
		req.QoS = &appJsonData[0].Metadata.QoS
	}

	req.CloudSvc = cloudSvcJSON
	req.FogSvc = fogSvcJSON
	req.EdgeSvc = edgeSvcJSON

	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	application, err := h.rlaSvc.CreateApplication(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, application)
}

// GetApplication godoc
// @Summary      Get application by ID
// @Description  Returns application details by UUID
// @Tags         applications
// @Produce      json
// @Param        id   path      string  true  "Application ID"
// @Success      200  {object}  domain.Application
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/applications/{id} [get]
func (h *RLAHandler) GetApplication(w http.ResponseWriter, r *http.Request) {
	paramID := chi.URLParam(r, "id")

	if err := validation.Validate(paramID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid application ID"})
		return
	}

	applicationID, err := uuid.Parse(paramID)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	appID := applicationID.String()
	application, err := h.rlaSvc.GetApplication(r.Context(), &domain.ApplicationFetchCtr{ApplicationID: &appID})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if application == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]interface{}{"error": "application not found"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, application)
}

// ListApplications godoc
// @Summary      List all applications
// @Description  Returns all applications currently registered
// @Tags         applications
// @Produce      json
// @Success      200  {array}   domain.Application
// @Failure      500  {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/applications [get]
func (h *RLAHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	applications, err := h.rlaSvc.ListApplications(r.Context(), &domain.ApplicationListCtr{})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, applications)
}

// UpdateApplication godoc
// @Summary      Update an application
// @Description  Updates an existing application using uploaded YAML service files
// @Tags         applications
// @Accept       multipart/form-data
// @Produce      json
// @Param        id           path      string  true  "Application ID"
// @Param        application  formData  file    false "Application YAML"
// @Param        cloud_svc    formData  file    false "Cloud service YAML"
// @Param        fog_svc      formData  file    false "Fog service YAML"
// @Param        edge_svc     formData  file    false "Edge service YAML"
// @Success      200          {object}  domain.Application
// @Failure      400          {object}  map[string]interface{}
// @Failure      404          {object}  map[string]interface{}
// @Failure      500          {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/applications/{id} [patch]
func (h *RLAHandler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	if !h.rlaSvc.Isleader() {
		h.forwardToLeader(w, r)
		return
	}

	paramID := chi.URLParam(r, "id")

	if err := validation.Validate(paramID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid application ID"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	req := new(dto.UpdateApplicationReq)

	applicationSvcJSON, err := utils.YAMLtoJSON(r, "application")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid application YAML"})
		return
	}

	cloudSvcJSON, err := utils.YAMLtoJSON(r, "cloud_svc")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cloud_svc YAML"})
		return
	}

	fogSvcJSON, err := utils.YAMLtoJSON(r, "fog_svc")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid fog_svc YAML"})
		return
	}

	edgeSvcJSON, err := utils.YAMLtoJSON(r, "edge_svc")
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid edge_svc YAML"})
		return
	}

	if applicationSvcJSON != nil {
		appJsonData := []*Application{}

		if err := json.Unmarshal(applicationSvcJSON, &appJsonData); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]interface{}{"error": "error parsing application"})
			return
		}

		req.Name = &appJsonData[0].Metadata.Name
		req.Labels = &appJsonData[0].Metadata.Labels
		req.QoS = &appJsonData[0].Metadata.QoS
	}

	req.ID = paramID
	req.CloudSvc = cloudSvcJSON
	req.FogSvc = fogSvcJSON
	req.EdgeSvc = edgeSvcJSON

	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	application, err := h.rlaSvc.UpdateApplication(r.Context(), req)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if application == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]interface{}{"error": "invalid application id"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, application)
}

// DeleteApplication godoc
// @Summary      Delete an application
// @Description  Deletes an application by UUID
// @Tags         applications
// @Produce      json
// @Param        id   path      string  true  "Application ID"
// @Success      204  {object}  nil
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/applications/{id} [delete]
func (h *RLAHandler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	if !h.rlaSvc.Isleader() {
		h.forwardToLeader(w, r)
		return
	}

	paramID := chi.URLParam(r, "id")

	if err := validation.Validate(paramID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster ID"})
		return
	}

	if err := h.rlaSvc.DeleteApplication(r.Context(), paramID); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusNoContent)
	render.JSON(w, r, nil)
}

// ListClusterScheduledApps godoc
// @Summary      List scheduled applications for a cluster
// @Description  Returns all apps scheduled for the given cluster
// @Tags         applications
// @Produce      json
// @Param        cluster-id  path      string  true  "Cluster ID"
// @Success      200         {array}   dto.ScheduledApplication
// @Failure      400         {object}  map[string]interface{}
// @Failure      500         {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters/{cluster-id}/applications/scheduled [get]
func (h *RLAHandler) ListClusterScheduledApps(w http.ResponseWriter, r *http.Request) {
	paramClusterID := chi.URLParam(r, "cluster-id")

	if err := validation.Validate(paramClusterID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster ID"})
		return
	}

	resp, err := h.rlaSvc.GetClusterScheduledApps(r.Context(), &dto.ClusterScheduledApps{
		ClusterID: paramClusterID,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if len(resp) == 0 {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, []dto.ScheduledApplication{})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, resp)
}

// GetClusterApp godoc
// @Summary      Get a specific scheduled application for a cluster
// @Description  Returns a single scheduled app by cluster ID and app ID
// @Tags         applications
// @Produce      json
// @Param        cluster-id      path      string  true  "Cluster ID"
// @Param        application-id  path      string  true  "Application ID"
// @Success      200             {object}  dto.ScheduledApplication
// @Failure      400             {object}  map[string]interface{}
// @Failure      404             {object}  nil
// @Failure      500             {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters/{cluster-id}/applications/{application-id} [get]
func (h *RLAHandler) GetClusterApp(w http.ResponseWriter, r *http.Request) {
	paramClusterID := chi.URLParam(r, "cluster-id")

	if err := validation.Validate(paramClusterID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster ID"})
		return
	}

	paramAppID := chi.URLParam(r, "application-id")
	if err := validation.Validate(paramAppID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid app ID"})
		return
	}

	resp, err := h.rlaSvc.GetClusterScheduledApp(r.Context(), &dto.ClusterScheduledApp{
		ClusterID: paramClusterID,
		AppID:     paramAppID,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if resp == nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, nil)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, resp)
}

// ClusterApplicationStatus godoc
// @Summary      Report application status from a worker cluster
// @Description  Used by a cluster to report the current state of a scheduled app
// @Tags         applications
// @Accept       json
// @Produce      json
// @Param        cluster-id      path      string                       true  "Cluster ID"
// @Param        application-id  path      string                       true  "Application ID"
// @Param        body            body      dto.ClusterApplicationStatus  true  "Status payload"
// @Success      200             {object}  nil
// @Failure      400             {object}  map[string]interface{}
// @Failure      406             {object}  nil
// @Failure      404             {object}  nil
// @Failure      500             {object}  map[string]interface{}
// @Router       /swarmchestrate/api/v1/clusters/{cluster-id}/applications/{application-id}/status [post]
func (h *RLAHandler) ClusterApplicationStatus(w http.ResponseWriter, r *http.Request) {
	paramClusterID := chi.URLParam(r, "cluster-id")

	if err := validation.Validate(paramClusterID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid cluster ID"})
		return
	}

	paramApplicationID := chi.URLParam(r, "application-id")

	if err := validation.Validate(paramApplicationID, is.UUID); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": "Invalid application ID"})
		return
	}

	req := new(dto.ClusterApplicationStatus)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	if err := req.Validate(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	req.ClusterID = paramClusterID
	req.ApplicationID = paramApplicationID
	if err := h.rlaSvc.ClusterApplicationStatus(r.Context(), req); err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, nil)
			return
		}

		if errors.Is(err, domain.ErrApplicationNotSynced) {
			render.Status(r, http.StatusNotAcceptable)
			render.JSON(w, r, nil)
			return
		}

		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]interface{}{"error": err})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, nil)
}
