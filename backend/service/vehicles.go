package service

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"automarket/models"
	"automarket/storage"
)

type Vehicles struct {
	vehicles *storage.Table[models.Vehicle]
	users    *storage.Table[models.User]

	// imagesDir es la carpeta donde se guardan las imágenes de las publicaciones.
	imagesDir string
}

func NewVehicles(vehicles *storage.Table[models.Vehicle], users *storage.Table[models.User], imagesDir string) *Vehicles {
	return &Vehicles{vehicles: vehicles, users: users, imagesDir: imagesDir}
}

type PublishInput struct {
	Brand   string
	Model   string
	Year    int
	Price   int64
	Mileage int
}

// List devuelve las publicaciones que el solicitante puede ver. Un visitante
// (viewer nil) ve las publicadas (HU-VIS-01); un vendedor, además, las suyas
// de cualquier estado (HU-VEN-01); un administrador, todas (HU-ADM-01).
func (s *Vehicles) List(viewer *models.User) []models.Listing {
	return s.toListings(s.vehicles.Filter(func(v models.Vehicle) bool { return isVisibleTo(viewer, v) }))
}

// isVisibleTo decide si el solicitante puede ver una publicación. Es la misma
// regla para el catálogo y para sus imágenes.
func isVisibleTo(viewer *models.User, v models.Vehicle) bool {
	switch {
	case viewer == nil:
		return v.Status == models.StatusPublished
	case viewer.Role == models.RoleAdmin:
		return true
	default:
		return v.Status == models.StatusPublished || v.SellerID == viewer.ID
	}
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
// vendedor (HU-VEN-01). Las imágenes se validan y se limpian todas antes de
// guardar nada: si una falla, no se guarda ninguna.
func (s *Vehicles) Publish(sellerID int, in PublishInput, images [][]byte) (models.Vehicle, error) {
	if len(images) > MaxImages {
		return models.Vehicle{}, ErrInvalidImage
	}
	cleaned := make([][]byte, len(images))
	names := make([]string, len(images))
	for i, raw := range images {
		data, ext, err := cleanImage(raw)
		if err != nil {
			return models.Vehicle{}, err
		}
		cleaned[i], names[i] = data, strconv.Itoa(i+1)+ext
	}

	vehicle, err := s.vehicles.Insert(func(id int) models.Vehicle {
		return models.Vehicle{
			ID:        id,
			SellerID:  sellerID,
			Brand:     strings.TrimSpace(in.Brand),
			Model:     strings.TrimSpace(in.Model),
			Year:      in.Year,
			Price:     in.Price,
			Mileage:   in.Mileage,
			Images:    names,
			Status:    models.StatusPending,
			CreatedAt: time.Now().UTC(),
		}
	}, nil)
	if err != nil {
		return models.Vehicle{}, err
	}

	for i, data := range cleaned {
		if err := s.saveImage(vehicle.ID, names[i], data); err != nil {
			s.removeImages(vehicle.ID)
			s.vehicles.Delete(vehicle.ID)
			return models.Vehicle{}, err
		}
	}
	return vehicle, nil
}

// Image devuelve una imagen de una publicación si el solicitante puede ver
// la publicación: cualquiera si está publicada; si no, solo su dueño o un
// administrador. Solo se sirven los nombres que la publicación tiene
// registrados.
func (s *Vehicles) Image(viewer *models.User, vehicleID int, name string) ([]byte, error) {
	v, err := s.vehicles.Get(vehicleID)
	if err != nil {
		return nil, translate(err)
	}
	if !isVisibleTo(viewer, v) || !slices.Contains(v.Images, name) {
		return nil, ErrNotFound
	}
	data, err := s.readImage(vehicleID, name)
	return data, translate(err)
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

// Delete elimina una publicación y sus imágenes (HU-ADM-01).
func (s *Vehicles) Delete(vehicleID int) error {
	if err := s.vehicles.Delete(vehicleID); err != nil {
		return translate(err)
	}
	return s.removeImages(vehicleID)
}

// DeleteBySeller elimina todas las publicaciones de un vendedor, con sus
// imágenes. Lo usa Accounts al eliminar un usuario.
func (s *Vehicles) DeleteBySeller(sellerID int) error {
	owned := func(v models.Vehicle) bool { return v.SellerID == sellerID }
	removed := s.vehicles.Filter(owned)
	if err := s.vehicles.DeleteWhere(owned); err != nil {
		return err
	}
	for _, v := range removed {
		if err := s.removeImages(v.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Vehicles) toListings(vehicles []models.Vehicle) []models.Listing {
	listings := make([]models.Listing, 0, len(vehicles))
	for _, v := range vehicles {
		listings = append(listings, s.toListing(v))
	}
	return listings
}

func (s *Vehicles) toListing(v models.Vehicle) models.Listing {
	listing := models.Listing{Vehicle: v}
	if seller, err := s.users.Get(v.SellerID); err == nil {
		listing.Seller = seller.Contact()
	}
	return listing
}
