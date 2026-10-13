package handlers

import (
	"net/http"

	"automarket/auth"
	"automarket/models"
	"automarket/service"

	"github.com/gin-gonic/gin"
)

type VehicleHandler struct {
	vehicles *service.Vehicles
}

func NewVehicleHandler(vehicles *service.Vehicles) *VehicleHandler {
	return &VehicleHandler{vehicles: vehicles}
}

type publishRequest struct {
	Brand   string `json:"brand" binding:"required,max=50"`
	Model   string `json:"model" binding:"required,max=50"`
	Year    int    `json:"year" binding:"required,min=1900,max=2100"`
	Price   int64  `json:"price" binding:"required,gt=0"`
	Mileage int    `json:"mileage" binding:"min=0"`
}

// Catalog: GET /vehicles (público)
func (h *VehicleHandler) Catalog(c *gin.Context) {
	c.JSON(http.StatusOK, h.vehicles.Catalog())
}

// Detail: GET /vehicles/:id (público)
func (h *VehicleHandler) Detail(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	listing, err := h.vehicles.Listing(id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, listing)
}

// Publish: POST /vehicles (vendedor)
func (h *VehicleHandler) Publish(c *gin.Context) {
	var req publishRequest
	if !bindJSON(c, &req) {
		return
	}
	seller, _ := auth.Actor(c)
	vehicle, err := h.vehicles.Publish(seller.ID, service.PublishInput{
		Brand:   req.Brand,
		Model:   req.Model,
		Year:    req.Year,
		Price:   req.Price,
		Mileage: req.Mileage,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, vehicle)
}

// MarkSold: PATCH /vehicles/:id/sold (vendedor dueño)
func (h *VehicleHandler) MarkSold(c *gin.Context) {
	h.transition(c, func(id int) (models.Vehicle, error) {
		seller, _ := auth.Actor(c)
		return h.vehicles.MarkSold(seller.ID, id)
	})
}

// Approve: PATCH /vehicles/:id/approve (administrador)
func (h *VehicleHandler) Approve(c *gin.Context) {
	h.transition(c, h.vehicles.Approve)
}

// Delete: DELETE /vehicles/:id (administrador)
func (h *VehicleHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.vehicles.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// transition aplica un cambio de estado sobre el vehículo del parámetro :id.
func (h *VehicleHandler) transition(c *gin.Context, change func(id int) (models.Vehicle, error)) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	vehicle, err := change(id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, vehicle)
}
