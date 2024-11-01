package dto

type Node struct {
	Name                   string  `json:"name"`
	Location               string  `json:"location"`
	CPU                    int     `json:"cpu"`
	CPUArch                string  `json:"cpu_arch"`
	Memory                 float64 `json:"memory"`
	NetworkBandwidth       float64 `json:"network_bandwidth"`
	EphemeralStorage       float64 `json:"ephemeral_storage"`
	Energy                 float64 `json:"energy"`
	Pricing                float64 `json:"pricing"`
	IsReady                bool    `json:"is_ready"`
	IsSchedulable          bool    `json:"is_schedulable"`
	IsPIDPressureExists    bool    `json:"is_pid_pressure_exists"`
	IsMemoryPressureExists bool    `json:"is_memory_pressure_exists"`
	IsDiskPressureExists   bool    `json:"is_disk_pressure_exists"`
}
