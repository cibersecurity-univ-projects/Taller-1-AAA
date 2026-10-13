package models

import "time"

type VehicleStatus string

const (
	StatusPending   VehicleStatus = "pendiente"
	StatusPublished VehicleStatus = "publicada"
	StatusSold      VehicleStatus = "vendida"
)

type Vehicle struct {
	ID        int           `json:"id"`
	SellerID  int           `json:"seller_id"`
	Brand     string        `json:"brand"`
	Model     string        `json:"model"`
	Year      int           `json:"year"`
	Price     int64         `json:"price"`
	Mileage   int           `json:"mileage"`
	Status    VehicleStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

// Listing es la vista pública de un vehículo junto con el contacto de su vendedor.
type Listing struct {
	Vehicle
	Seller Contact `json:"seller"`
}
