package service

import (
	"strings"
	"time"

	"automarket/models"
	"automarket/storage"
)

type Vehicles struct {
	vehicles *storage.Table[models.Vehicle]
	users    *storage.Table[models.User]
}

func NewVehicles(vehicles *storage.Table[models.Vehicle], users *storage.Table[models.User]) *Vehicles {
	return &Vehicles{vehicles: vehicles, users: users}
}

type PublishInput struct {
	Brand   string
	Model   string
	Year    int
	Price   int64
	Mileage int
}

// Catalog devuelve las publicaciones visibles al público (HU-VIS-01).
func (s *Vehicles) Catalog() []models.Listing {
	published := s.vehicles.Filter(func(v models.Vehicle) bool { return v.Status == models.StatusPublished })
	listings := make([]models.Listing, 0, len(published))
	for _, v := range published {
		listings = append(listings, s.toListing(v))
	}
	return listings
}

// Listing devuelve un vehículo publicado con el contacto de su vendedor
// (HU-VIS-01). Los no publicados no existen para el público.
func (s *Vehicles) Listing(id int) (models.Listing, error) {
	v, err := s.vehicles.Get(id)
	if err != nil {
		return models.Listing{}, translate(err)
	}
	if v.Status != models.StatusPublished {
		return models.Listing{}, ErrNotFound
	}
	return s.toListing(v), nil
}

// Publish registra la publicación en estado pendiente y la asocia al
// vendedor (HU-VEN-01).
func (s *Vehicles) Publish(sellerID int, in PublishInput) (models.Vehicle, error) {
	return s.vehicles.Insert(func(id int) models.Vehicle {
		return models.Vehicle{
			ID:        id,
			SellerID:  sellerID,
			Brand:     strings.TrimSpace(in.Brand),
			Model:     strings.TrimSpace(in.Model),
			Year:      in.Year,
			Price:     in.Price,
			Mileage:   in.Mileage,
			Status:    models.StatusPending,
			CreatedAt: time.Now().UTC(),
		}
	}, nil)
}

// MarkSold reporta la venta de un vehículo propio y lo retira del catálogo
// (HU-VEN-01).
func (s *Vehicles) MarkSold(sellerID, vehicleID int) (models.Vehicle, error) {
	v, err := s.vehicles.Update(vehicleID, func(v *models.Vehicle) error {
		if v.SellerID != sellerID {
			return ErrNotOwner
		}
		if v.Status == models.StatusSold {
			return ErrInvalidState
		}
		v.Status = models.StatusSold
		return nil
	})
	return v, translate(err)
}

// Approve pasa una publicación pendiente a publicada (HU-ADM-01).
func (s *Vehicles) Approve(vehicleID int) (models.Vehicle, error) {
	v, err := s.vehicles.Update(vehicleID, func(v *models.Vehicle) error {
		if v.Status != models.StatusPending {
			return ErrInvalidState
		}
		v.Status = models.StatusPublished
		return nil
	})
	return v, translate(err)
}

// Delete elimina una publicación (HU-ADM-01).
func (s *Vehicles) Delete(vehicleID int) error {
	return translate(s.vehicles.Delete(vehicleID))
}

func (s *Vehicles) toListing(v models.Vehicle) models.Listing {
	listing := models.Listing{Vehicle: v}
	if seller, err := s.users.Get(v.SellerID); err == nil {
		listing.Seller = seller.Contact()
	}
	return listing
}
