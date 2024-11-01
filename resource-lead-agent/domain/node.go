package domain

import (
	"errors"
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
)

type Node struct {
	ClusterID              string    `json:"cluster_id"                gorm:"primaryKey"`
	Name                   string    `json:"name"                      gorm:"primaryKey"`
	Location               string    `json:"location"`
	CPU                    int       `json:"cpu"`
	CPUArch                string    `json:"cpu_arch"`
	Memory                 float64   `json:"memory"`
	NetworkBandwidth       float64   `json:"network_bandwidth"`
	EphemeralStorage       float64   `json:"ephemeral_storage"`
	Energy                 float64   `json:"energy"`
	Pricing                float64   `json:"pricing"`
	IsReady                bool      `json:"is_ready"`
	IsSchedulable          bool      `json:"is_schedulable"`
	IsPIDPressureExists    bool      `json:"is_pid_pressure_exists"    gorm:"column:is_pid_pressure_exists"`
	IsMemoryPressureExists bool      `json:"is_memory_pressure_exists"`
	IsDiskPressureExists   bool      `json:"is_disk_pressure_exists"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (n Node) Validate() error {
	return validation.ValidateStruct(&n,
		validation.Field(&n.Name, validation.Required),
		validation.Field(&n.Location, validation.Required, validation.By(validateLocation)),
		validation.Field(&n.CPU, validation.Required, validation.Min(1)),
		validation.Field(&n.CPUArch, validation.Required),
		validation.Field(&n.Memory, validation.Required, validation.Min(0.1)),
		validation.Field(&n.NetworkBandwidth, validation.Required, validation.Min(0.1)),
		validation.Field(&n.EphemeralStorage, validation.Required, validation.Min(0.1)),
		validation.Field(&n.Energy, validation.Required, validation.Min(0.1)),
		validation.Field(&n.Pricing, validation.Required, validation.Min(0.1)),
	)
}

type NodeFetchCtr struct {
	ClusterID *string
	NodeName  *string
}

type NodeListCtr struct {
	ClusterID *string
}

func validateLocation(value interface{}) error {
	location, _ := value.(string)
	matched, _ := regexp.MatchString(`^[A-Za-z]+/[A-Za-z_]+$`, location)
	if !matched {
		return errors.New("location must be in Region/Zone format")
	}
	return nil
}
