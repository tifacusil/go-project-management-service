package domain

import "time"

// Status values shared by projects, sensors, equipment.
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

// Project = monitoring project. Mirrors `projects` table.
type Project struct {
	ProjectID   string    `db:"project_id" json:"project_id" validate:"required,max=64"`
	Name        string    `db:"name" json:"name" validate:"required,max=255"`
	LocationLat *float64  `db:"location_lat" json:"location_lat,omitempty"`
	LocationLng *float64  `db:"location_lng" json:"location_lng,omitempty"`
	Status      string    `db:"status" json:"status" validate:"omitempty,oneof=active inactive"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`

	Sensors []Sensor `db:"-" json:"sensors,omitempty"`
}

// Sensor = device installed on a project. Mirrors `sensors` table.
type Sensor struct {
	SensorID    string     `db:"sensor_id" json:"sensor_id" validate:"required,max=64"`
	ProjectID   string     `db:"project_id" json:"project_id" validate:"required,max=64"`
	SensorType  string     `db:"sensor_type" json:"sensor_type" validate:"omitempty,oneof=gnss rainfall strain soil_pressure deep_displacement camera"`
	Model       *string    `db:"model" json:"model,omitempty"`
	InstallLat  *float64   `db:"install_lat" json:"install_lat,omitempty"`
	InstallLng  *float64   `db:"install_lng" json:"install_lng,omitempty"`
	InstallDepth *float64  `db:"install_depth" json:"install_depth,omitempty"`
	Status      string     `db:"status" json:"status" validate:"omitempty,oneof=active inactive"`
	InstalledAt *time.Time `db:"installed_at" json:"installed_at,omitempty"`
}

// Equipment = non-sensor hardware on a project. Mirrors `equipment` table.
type Equipment struct {
	EquipmentID string  `db:"equipment_id" json:"equipment_id" validate:"required,max=64"`
	ProjectID   string  `db:"project_id" json:"project_id" validate:"required,max=64"`
	Type        *string `db:"type" json:"type,omitempty" validate:"omitempty,oneof=DTU solar_panel battery chassis camera"`
	Model       *string `db:"model" json:"model,omitempty"`
	Serial      *string `db:"serial" json:"serial,omitempty"`
	Status      string  `db:"status" json:"status" validate:"omitempty,oneof=active inactive"`
}
