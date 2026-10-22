package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Vehicle y Listing reflejan las respuestas JSON del backend.
type Vehicle struct {
	ID       int      `json:"id"`
	SellerID int      `json:"seller_id"`
	Brand    string   `json:"brand"`
	Model    string   `json:"model"`
	Year     int      `json:"year"`
	Price    int64    `json:"price"`
	Mileage  int64    `json:"mileage"`
	Images   []string `json:"images"`
	Status   string   `json:"status"`
}

type Contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type Listing struct {
	Vehicle
	Seller Contact `json:"seller"`
}

type publishInput struct {
	Brand   string `json:"brand"`
	Model   string `json:"model"`
	Year    int    `json:"year"`
	Price   int64  `json:"price"`
	Mileage int64  `json:"mileage"`
}

// apiError conserva el código HTTP que respondió el backend (0 si no hubo
// respuesta) para que la interfaz decida qué mensaje mostrar.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("backend respondió %d: %s", e.Status, e.Message)
}

// backend es el cliente HTTP de la API de AutoMarket.
type backend struct {
	base   string
	client *http.Client
}

func newBackend(base string) *backend {
	return &backend{base: base, client: &http.Client{Timeout: 10 * time.Second}}
}

func (b *backend) do(method, path, token string, in, out any) error {
	var body io.Reader
	contentType := ""
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body, contentType = bytes.NewReader(raw), "application/json"
	}
	return b.send(method, path, token, body, contentType, out)
}

// send hace la petición y lee la respuesta; do y publish solo cambian el cuerpo.
func (b *backend) send(method, path, token string, body io.Reader, contentType string, out any) error {
	req, err := http.NewRequest(method, b.base+path, body)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return &apiError{Message: "no se pudo contactar al backend"}
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &failure)
		return &apiError{Status: resp.StatusCode, Message: failure.Error}
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

// catalog lista las publicaciones que el solicitante puede ver: con token
// vacío, las públicas; con el de un vendedor, además las suyas; con el de un
// administrador, todas.
func (b *backend) catalog(token string) ([]Listing, error) {
	var listings []Listing
	err := b.do(http.MethodGet, "/vehicles", token, nil, &listings)
	return listings, err
}

func (b *backend) listing(id int) (Listing, error) {
	var l Listing
	err := b.do(http.MethodGet, fmt.Sprintf("/vehicles/%d", id), "", nil, &l)
	return l, err
}

func (b *backend) login(email, password string) (token, role string, err error) {
	var out struct {
		Token string `json:"token"`
		Role  string `json:"role"`
	}
	err = b.do(http.MethodPost, "/auth/login", "", map[string]string{"email": email, "password": password}, &out)
	return out.Token, out.Role, err
}

func (b *backend) register(name, email, phone, password string) error {
	return b.do(http.MethodPost, "/auth/register", "", map[string]string{
		"name": name, "email": email, "phone": phone, "password": password,
	}, nil)
}

// uploadFile es una imagen que eligió la persona al publicar.
type uploadFile struct {
	name string
	data []byte
}

// publish envía la publicación como multipart/form-data, que es como el
// backend recibe los datos junto con las imágenes. Escribir en un
// bytes.Buffer no falla, por eso no se revisan esos errores.
func (b *backend) publish(token string, in publishInput, images []uploadFile) (Vehicle, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	form.WriteField("brand", in.Brand)
	form.WriteField("model", in.Model)
	form.WriteField("year", strconv.Itoa(in.Year))
	form.WriteField("price", strconv.FormatInt(in.Price, 10))
	form.WriteField("mileage", strconv.FormatInt(in.Mileage, 10))
	for _, image := range images {
		part, err := form.CreateFormFile("images", image.name)
		if err != nil {
			return Vehicle{}, err
		}
		part.Write(image.data)
	}
	form.Close()

	var v Vehicle
	err := b.send(http.MethodPost, "/vehicles", token, &body, form.FormDataContentType(), &v)
	return v, err
}

// image pide una imagen al backend con el token de la persona (si lo hay) y
// devuelve la respuesta sin leerla, para que quien llama la reenvíe.
func (b *backend) image(token string, id int, name string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/vehicles/%d/images/%s", b.base, id, url.PathEscape(name)), nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return b.client.Do(req)
}

func (b *backend) markSold(token string, id int) (Vehicle, error) {
	var v Vehicle
	err := b.do(http.MethodPatch, fmt.Sprintf("/vehicles/%d/sold", id), token, nil, &v)
	return v, err
}

func (b *backend) approve(token string, id int) (Vehicle, error) {
	var v Vehicle
	err := b.do(http.MethodPatch, fmt.Sprintf("/vehicles/%d/approve", id), token, nil, &v)
	return v, err
}

func (b *backend) deleteVehicle(token string, id int) error {
	return b.do(http.MethodDelete, fmt.Sprintf("/vehicles/%d", id), token, nil, nil)
}

func (b *backend) deleteUser(token string, id int) error {
	return b.do(http.MethodDelete, fmt.Sprintf("/users/%d", id), token, nil, nil)
}
