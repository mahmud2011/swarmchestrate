package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/mahmud2011/swarmchestrate/resource-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-agent/domain/dto"
	"github.com/mahmud2011/swarmchestrate/resource-agent/internal/config"
)

type Service struct {
	cnf *config.Config
}

func NewService(cnf *config.Config) *Service {
	return &Service{
		cnf: cnf,
	}
}

// RegisterCluster registers a new cluster using the provided request data.
// It sends a POST request to the resource-lead service and returns the created cluster details.
func (s *Service) RegisterCluster(_ context.Context, d *dto.AddClusterReq) (*domain.Cluster, error) {
	url := fmt.Sprintf(
		"http://%s:%d/swarmchestrate/api/v1/clusters",
		config.Get().RegisterSvc.Host,
		config.Get().RegisterSvc.Port,
	)

	b, err := json.Marshal(d)
	if err != nil {
		slog.Error("failed to marshal cluster request", "error", err)
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		slog.Error("failed to post cluster registration", "url", url, "error", err)
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
		}
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		err := fmt.Errorf("failed to register cluster: %d", resp.StatusCode)
		slog.Error("unexpected status code during cluster registration", "status", resp.StatusCode, "error", err)
		return nil, err
	}

	respBody := &domain.Cluster{}
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		slog.Error("failed to decode cluster registration response", "error", err)
		return nil, err
	}

	return respBody, nil
}

// SyncClusterConfig retrieves and synchronizes the cluster configuration from the resource-lead service.
// It sends a GET request to obtain configuration key-value pairs.
func (s *Service) SyncClusterConfig(ctx context.Context, resourceLeadHost string) (map[string]string, error) {
	url := fmt.Sprintf("http://%s:%d/swarmchestrate/api/v1/clusters/config", resourceLeadHost, s.cnf.RegisterSvc.Port)

	resp, err := http.Get(url)
	if err != nil {
		slog.Error("failed to get cluster configuration", "url", url, "error", err)
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
		}
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to sync clusters: %d", resp.StatusCode)
		slog.Error(
			"unexpected status code while syncing cluster configuration",
			"status",
			resp.StatusCode,
			"error",
			err,
		)
		return nil, err
	}

	respBody := map[string]string{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		slog.Error("failed to decode cluster configuration response", "error", err)
		return nil, err
	}

	return respBody, nil
}

// UpsertNodes updates or inserts Node information for a given cluster.
// It sends a PUT request with the Node details to the resource-lead service.
func (s *Service) UpsertNodes(resourceLeadHost string, clusterID string, workerNodes []*dto.Node) error {
	url := fmt.Sprintf(
		"http://%s:%d/swarmchestrate/api/v1/clusters/%s/nodes",
		resourceLeadHost,
		s.cnf.RegisterSvc.Port,
		clusterID,
	)

	reqBody := struct {
		Nodes []*dto.Node `json:"nodes"`
	}{
		Nodes: workerNodes,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("failed to marshal nodes data", "error", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("failed to create new request for upserting nodes", "url", url, "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: s.cnf.RegisterSvc.Timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to perform upsert nodes request", "url", url, "error", err)
		return err
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
		}
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to send node info, status code: %d", resp.StatusCode)
		slog.Error("unexpected status code in upsert nodes", "status", resp.StatusCode, "error", err)
		return err
	}

	return nil
}

// GetScheduledApps retrieves the list of scheduled applications for a given cluster.
// It sends a GET request to the resource-lead service and returns the list of scheduled applications.
func (s *Service) GetScheduledApps(resourceLeadHost string, clusterID string) ([]*dto.ScheduledApplication, error) {
	url := fmt.Sprintf(
		"http://%s:%d/swarmchestrate/api/v1/clusters/%s/applications/scheduled",
		resourceLeadHost,
		s.cnf.RegisterSvc.Port,
		clusterID,
	)

	resp, err := http.Get(url)
	if err != nil {
		slog.Error("failed to get scheduled applications", "url", url, "error", err)
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
		}
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to get applications, status code: %d", resp.StatusCode)
		slog.Error(
			"unexpected status code while retrieving scheduled applications",
			"status",
			resp.StatusCode,
			"error",
			err,
		)
		return nil, err
	}

	respBody := []*dto.ScheduledApplication{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		slog.Error("failed to decode scheduled applications response", "error", err)
		return nil, err
	}

	return respBody, nil
}

// NotifyAppHealth notifies the resource-lead service about the health status of an application within a cluster.
// It sends a POST request with the application health details.
func (s *Service) NotifyAppHealth(resourceLeadHost string, ctr *dto.ClusterApplicationStatus) error {
	url := fmt.Sprintf(
		"http://%s:%d/swarmchestrate/api/v1/clusters/%s/applications/%s/status",
		resourceLeadHost,
		s.cnf.RegisterSvc.Port,
		ctr.ClusterID,
		ctr.ApplicationID,
	)

	payload, err := json.Marshal(ctr)
	if err != nil {
		slog.Error("failed to marshal application status", "error", err)
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		slog.Error("failed to post application status", "url", url, "error", err)
		return err
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
		}
		resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		slog.Error("application not found", "url", url)
		return domain.ErrApplicationNotFound
	}

	if resp.StatusCode == http.StatusNotAcceptable {
		slog.Error("application not synced", "url", url)
		return domain.ErrApplicationNotSynced
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to notify application health, status code: %d", resp.StatusCode)
		slog.Error("unexpected status code when notifying app health", "status", resp.StatusCode, "error", err)
		return err
	}

	return nil
}
