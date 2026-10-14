package handlers

import (
	"net/http"
	"time"

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

// minVehicleYear es el modelo más antiguo que se acepta publicar.
const minVehicleYear = 1950

type publishRequest struct {
	Brand   string `json:"brand" binding:"required,max=50"`
	Model   string `json:"model" binding:"required,max=50"`
	Year    int    `json:"year" binding:"required,min=1950"`
	Price   int64  `json:"price" binding:"required,gt=0"`
	Mileage int    `json:"mileage" binding:"min=0"`
}

// validYear acepta desde minVehicleYear hasta el año siguiente al actual
// (los modelos del próximo año se venden antes de que empiece).
func validYear(year int) bool {
	return year >= minVehicleYear && year <= time.Now().Year()+1
}

// Catalog: GET /vehicles (público; el contenido depende del rol)
func (h *VehicleHandler) Catalog(c *gin.Context) {
	var viewer *models.User
	if user, ok := auth.Actor(c); ok {
		viewer = &user
	}
	c.JSON(http.StatusOK, h.vehicles.List(viewer))
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
	if !validYear(req.Year) {
		badRequest(c, "year fuera del rango permitido")
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
