package handlers

import (
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"automarket/auth"
	"automarket/models"
	"automarket/service"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type VehicleHandler struct {
	vehicles *service.Vehicles
}

func NewVehicleHandler(vehicles *service.Vehicles) *VehicleHandler {
	return &VehicleHandler{vehicles: vehicles}
}

const (
	// minVehicleYear es el modelo más antiguo que se acepta publicar.
	minVehicleYear = 1950

	// maxPublishBytes limita el cuerpo de una publicación con imágenes: las
	// doce de 2 MB más un margen para los demás campos.
	maxPublishBytes = service.MaxImages*service.MaxImageBytes + 1<<20
)

type publishRequest struct {
	Brand   string `json:"brand" form:"brand" binding:"required,max=50"`
	Model   string `json:"model" form:"model" binding:"required,max=50"`
	Year    int    `json:"year" form:"year" binding:"required,min=1950"`
	Price   int64  `json:"price" form:"price" binding:"required,gt=0"`
	Mileage int    `json:"mileage" form:"mileage" binding:"min=0"`
}

// validYear acepta desde minVehicleYear hasta el año siguiente al actual
// (los modelos del próximo año se venden antes de que empiece).
func validYear(year int) bool {
	return year >= minVehicleYear && year <= time.Now().Year()+1
}

// Catalog: GET /vehicles (público; el contenido depende del rol)
func (h *VehicleHandler) Catalog(c *gin.Context) {
	c.JSON(http.StatusOK, h.vehicles.List(viewer(c)))
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

// Image: GET /vehicles/:id/images/:name (según el rol: las de una publicación
// pendiente solo las ven su dueño y el administrador)
func (h *VehicleHandler) Image(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	name := c.Param("name")
	data, err := h.vehicles.Image(viewer(c), id, name)
	if err != nil {
		respondError(c, err)
		return
	}
	contentType := "image/jpeg"
	if strings.HasSuffix(name, ".png") {
		contentType = "image/png"
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private")
	c.Data(http.StatusOK, contentType, data)
}

// Publish: POST /vehicles (vendedor)
func (h *VehicleHandler) Publish(c *gin.Context) {
	req, images, ok := readPublish(c)
	if !ok {
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
	}, images)
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

// viewer devuelve quién hace la solicitud, o nil si es un visitante.
func viewer(c *gin.Context) *models.User {
	if user, ok := auth.Actor(c); ok {
		return &user
	}
	return nil
}

// readPublish lee la solicitud de publicar. Acepta JSON (sin imágenes) o
// multipart/form-data (con hasta 12 imágenes en el campo "images"). Responde
// 400 y devuelve false si algo no es válido.
func readPublish(c *gin.Context) (publishRequest, [][]byte, bool) {
	var req publishRequest
	if !strings.HasPrefix(c.ContentType(), "multipart/") {
		return req, nil, bindJSON(c, &req)
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPublishBytes)
	if err := c.ShouldBindWith(&req, binding.FormMultipart); err != nil {
		badRequest(c, err.Error())
		return req, nil, false
	}
	headers := c.Request.MultipartForm.File["images"]
	if len(headers) > service.MaxImages {
		respondError(c, service.ErrInvalidImage)
		return req, nil, false
	}
	images := make([][]byte, 0, len(headers))
	for _, header := range headers {
		raw, err := readUpload(header)
		if err != nil {
			respondError(c, service.ErrInvalidImage)
			return req, nil, false
		}
		images = append(images, raw)
	}
	return req, images, true
}

// readUpload lee un archivo subido sin pasarse del tamaño máximo.
func readUpload(header *multipart.FileHeader) ([]byte, error) {
	if header.Size > service.MaxImageBytes {
		return nil, service.ErrInvalidImage
	}
	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, service.MaxImageBytes+1))
}
