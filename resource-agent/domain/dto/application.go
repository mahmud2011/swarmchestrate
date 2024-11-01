package dto

import (
	"time"

	"github.com/mahmud2011/swarmchestrate/resource-agent/domain"
)

type ScheduledApplication struct {
	Application *Application `json:"application"`
	CloudIP     *string      `json:"cloud_ip,omitempty"`
	FogIP       *string      `json:"fog_ip,omitempty"`
	EdgeIP      *string      `json:"edge_ip,omitempty"`
}

type Application struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Labels     *Labels                  `json:"labels"`
	Svc        []map[string]interface{} `json:"svc"`
	SvcVersion *string                  `json:"svc_version"`
	SvcNodes   *string                  `json:"svc_nodes"`
}

type QoS struct {
	Energy      float64 `json:"energy"`
	Pricing     float64 `json:"pricing"`
	Latency     float64 `json:"latency"`
	Performance float64 `json:"performance"`
}

type Labels map[string]string

type ClusterApplicationStatus struct {
	ClusterID     string `json:"-"`
	ApplicationID string `json:"-"`

	ApplicationVersion string                  `json:"application_version"`
	Status             domain.ApplicationState `json:"status"`
	Message            string                  `json:"message"`
}

type ApplicationConfig struct {
	AppID        string    `json:"app_id"`
	K8sResources string    `json:"k8s_resources"`
	AppVersion   string    `json:"app_version"`
	DeployedAt   time.Time `json:"deployed_at"`
}
