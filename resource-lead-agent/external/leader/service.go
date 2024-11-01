package leader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain/dto"
	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/internal/config"
)

const ingressPort = 80

type Service struct {
	cnf *config.Config
}

func NewService(cnf *config.Config) *Service {
	return &Service{
		cnf: cnf,
	}
}

func (s *Service) CreateCluster(_ context.Context, leaderHost string, d *dto.AddClusterReq) (*domain.Cluster, error) {
	url := fmt.Sprintf("http://%s:%d/swarmchestrate/api/v1/clusters", leaderHost, ingressPort)

	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
		}

		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to register cluster: %d", resp.StatusCode)
	}

	respBody := &domain.Cluster{}
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return nil, err
	}

	return respBody, nil
}

func (s *Service) UpdateCluster(_ context.Context, leaderHost string, d *dto.UpdateClusterReq) (*domain.Cluster, error) {
	url := fmt.Sprintf("http://%s:%d/swarmchestrate/api/v1/clusters/%s", leaderHost, ingressPort, d.ID)

	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
		}

		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to upgrade cluster role: %d", resp.StatusCode)
	}

	respBody := &domain.Cluster{}
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return nil, err
	}

	return respBody, nil
}
