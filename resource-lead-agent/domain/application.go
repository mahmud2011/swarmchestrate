package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
)

var (
	ErrApplicationNotFound  error = errors.New("application not found")
	ErrApplicationNotSynced error = errors.New("application not synced")
)

type ApplicationState string

const (
	StateProgressing ApplicationState = "progressing"
	StateHealthy     ApplicationState = "healthy"
	StateFailed      ApplicationState = "failed"
	StateNULL        ApplicationState = "NULL"
)

type Application struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Labels            *Labels         `json:"labels"`
	QoS               *QoS            `json:"qos" gorm:"column:qos"`
	CloudSvc          json.RawMessage `json:"cloud_svc"`
	CloudSvcVersion   *string         `json:"cloud_svc_version"`
	CloudSvcCluster   *string         `json:"cloud_svc_cluster"`
	CloudSvcNodes     *string         `json:"cloud_svc_nodes"`
	CloudSvcStatus    *string         `json:"cloud_svc_status"`
	CloudSvcHeartbeat *time.Time      `json:"cloud_svc_heartbeat"`
	FogSvc            json.RawMessage `json:"fog_svc"`
	FogSvcVersion     *string         `json:"fog_svc_version"`
	FogSvcCluster     *string         `json:"fog_svc_cluster"`
	FogSvcNodes       *string         `json:"fog_svc_nodes"`
	FogSvcStatus      *string         `json:"fog_svc_status"`
	FogSvcHeartbeat   *time.Time      `json:"fog_svc_heartbeat"`
	EdgeSvc           json.RawMessage `json:"edge_svc"`
	EdgeSvcVersion    *string         `json:"edge_svc_version"`
	EdgeSvcCluster    *string         `json:"edge_svc_cluster"`
	EdgeSvcNodes      *string         `json:"edge_svc_nodes"`
	EdgeSvcStatus     *string         `json:"edge_svc_status"`
	EdgeSvcHeartbeat  *time.Time      `json:"edge_svc_heartbeat"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type QoS struct {
	Energy      float64 `json:"energy"`
	Pricing     float64 `json:"pricing"`
	Performance float64 `json:"performance"`
}

func (q QoS) Value() (driver.Value, error) {
	return json.Marshal(q)
}

func (q *QoS) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &q)
}

func (qos QoS) Validate() error {
	return validation.ValidateStruct(&qos,
		validation.Field(&qos.Energy, validation.Min(0.0), validation.Max(1.0)),
		validation.Field(&qos.Pricing, validation.Min(0.0), validation.Max(1.0)),
		validation.Field(&qos.Performance, validation.Min(0.0), validation.Max(1.0)),
	)
}

type Labels map[string]string

func (l Labels) Value() (driver.Value, error) {
	return json.Marshal(l)
}

func (l *Labels) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &l)
}

type ApplicationFetchCtr struct {
	ApplicationID   *string
	CloudSvcCluster *string
	FogSvcCluster   *string
	EdgeSvcCluster  *string
}

type ApplicationListCtr struct {
	CloudSvcCluster *string
	FogSvcCluster   *string
	EdgeSvcCluster  *string
	CloudSvcStatus  *ApplicationState
	FogSvcStatus    *ApplicationState
	EdgeSvcStatus   *ApplicationState
}
