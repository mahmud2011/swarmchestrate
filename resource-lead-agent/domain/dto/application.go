package dto

import (
	"encoding/json"
	"errors"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"

	"github.com/mahmud2011/swarmchestrate/resource-lead-agent/domain"
)

type ScheduledApplication struct {
	Application *Application `json:"application"`
	CloudIP     *string      `json:"cloud_ip,omitempty"`
	FogIP       *string      `json:"fog_ip,omitempty"`
	EdgeIP      *string      `json:"edge_ip,omitempty"`
}

type Application struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Labels     *domain.Labels  `json:"labels"`
	Svc        json.RawMessage `json:"svc"`
	SvcVersion *string         `json:"svc_version"`
	SvcNodes   *string         `json:"svc_nodes"`
}

type AddApplicationReq struct {
	Name     string          `json:"name"`
	Labels   *domain.Labels  `json:"labels"`
	QoS      *domain.QoS     `json:"qos"`
	CloudSvc json.RawMessage `json:"cloud_svc"`
	FogSvc   json.RawMessage `json:"fog_svc"`
	EdgeSvc  json.RawMessage `json:"edge_svc"`
}

func (aa AddApplicationReq) Validate() error {
	return validation.ValidateStruct(&aa,
		validation.Field(&aa.Name, validation.Required),
		validation.Field(&aa.CloudSvc, validation.By(validateServices(aa.CloudSvc, aa.FogSvc, aa.EdgeSvc))),
	)
}

type UpdateApplicationReq struct {
	ID       string          `json:"-"`
	Name     *string         `json:"name"`
	Labels   *domain.Labels  `json:"labels"`
	QoS      *domain.QoS     `json:"qos"`
	CloudSvc json.RawMessage `json:"cloud_svc"`
	FogSvc   json.RawMessage `json:"fog_svc"`
	EdgeSvc  json.RawMessage `json:"edge_svc"`
}

func (aa UpdateApplicationReq) Validate() error {
	if aa.QoS != nil {
		return validation.Validate(aa.QoS)
	}

	return nil
}

func validateServices(cloudSvc, fogSvc, edgeSvc json.RawMessage) validation.RuleFunc {
	return func(value interface{}) error {
		if len(cloudSvc) == 0 && len(fogSvc) == 0 && len(edgeSvc) == 0 {
			return errors.New("at least one of cloud_svc, fog_svc, or edge_svc must be provided")
		}
		return nil
	}
}

type ClusterApplicationStatus struct {
	ClusterID     string             `json:"-"`
	ClusterType   domain.ClusterType `json:"-"`
	ApplicationID string             `json:"-"`

	ApplicationVersion string                  `json:"application_version"`
	Status             domain.ApplicationState `json:"status"`
	Message            string                  `json:"message"`
}

func (ua ClusterApplicationStatus) Validate() error {
	return validation.ValidateStruct(&ua,
		validation.Field(&ua.ApplicationVersion, validation.Required, is.UUID),
		validation.Field(&ua.Status, validation.Required, validation.In(
			domain.StateFailed,
			domain.StateHealthy,
			domain.StateProgressing,
		)),
	)
}

type ClusterScheduledApps struct {
	ClusterID string
}

type ClusterScheduledApp struct {
	ClusterID string
	AppID     string
}
