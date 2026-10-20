package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	roleSeller = "vendedor"
	roleAdmin  = "administrador"
	minYear    = 1950

	maxImageBytes  = 2 << 20                  // 2 MB por imagen
	maxUploadBytes = 12*maxImageBytes + 1<<20 // doce imágenes y los demás campos
)

type app struct {
	api      *backend
	sessions *sessionStore
}

type pageUser struct {
	Email string
	Home  string
}

// ---------- Utilidades ----------

func (a *app) current(c *gin.Context) (string, session, bool) {
	id, err := c.Cookie(cookieName)
	if err != nil {
		return "", session{}, false
	}
	s, ok := a.sessions.get(id)
	return id, s, ok
}

func (a *app) render(c *gin.Context, status int, name, title string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}
	data["Title"] = title
	if id, s, ok := a.current(c); ok {
		data["User"] = pageUser{Email: s.Email, Home: homeFor(s.Role)}
		if f := a.sessions.popFlash(id); f != nil {
			data["Flash"] = f
		}
	}
	c.HTML(status, name, data)
}

func (a *app) notice(c *gin.Context, kind, text string) {
	if id, _, ok := a.current(c); ok {
		a.sessions.setFlash(id, flash{Kind: kind, Text: text})
	}
}

func homeFor(role string) string {
	if role == roleAdmin {
		return "/admin"
	}
	return "/seller"
}

func setSessionCookie(c *gin.Context, id string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieName, id, maxAge, "/", "", false, true)
}

func statusOf(err error) int {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		return apiErr.Status
	}
	return 0
}

// friendly traduce el fallo del backend a un mensaje para la persona.
func friendly(err error, badRequest, conflict string) string {
	switch statusOf(err) {
	case http.StatusBadRequest:
		return badRequest
	case http.StatusUnauthorized:
		return "Tu sesión expiró. Inicia sesión de nuevo."
	case http.StatusForbidden:
		return "No tienes permiso para esta operación."
	case http.StatusNotFound:
		return "No se encontró la publicación."
	case http.StatusConflict:
		return conflict
	case 0:
		return "No pudimos contactar al servidor. Intenta de nuevo."
	}
	return "Ocurrió un error inesperado."
}

// fail avisa del error y vuelve a la pantalla; si la sesión expiró, la cierra.
func (a *app) fail(c *gin.Context, id string, err error, back, badRequest, conflict string) {
	if statusOf(err) == http.StatusUnauthorized {
		a.sessions.delete(id)
		setSessionCookie(c, "", -1)
		c.Redirect(http.StatusSeeOther, "/login?expired=1")
		return
	}
	a.notice(c, "error", friendly(err, badRequest, conflict))
	c.Redirect(http.StatusSeeOther, back)
}

func (a *app) message(c *gin.Context, status int, heading, text string) {
	a.render(c, status, "message.html", heading, gin.H{"Heading": heading, "Text": text})
}

// require deja pasar solo a quien tiene sesión con el rol indicado.
func (a *app) require(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, s, ok := a.current(c)
		if !ok {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		if s.Role != role {
			a.message(c, http.StatusForbidden, "No tienes permiso", "Esta sección no está disponible para tu rol.")
			c.Abort()
			return
		}
		c.Next()
	}
}

// ---------- Público ----------

func (a *app) catalog(c *gin.Context) {
	listings, err := a.api.catalog("")
	data := gin.H{"Vehicles": listings}
	if err != nil {
		data["Error"] = "No pudimos cargar el catálogo. Intenta de nuevo en unos minutos."
	}
	a.render(c, http.StatusOK, "catalog.html", "Vehículos en venta", data)
}

func (a *app) detail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		a.message(c, http.StatusNotFound, "Publicación no encontrada", "Puede que ya no esté disponible.")
		return
	}
	listing, err := a.api.listing(id)
	switch {
	case statusOf(err) == http.StatusNotFound:
		a.message(c, http.StatusNotFound, "Publicación no encontrada", "Puede que ya no esté disponible.")
	case err != nil:
		a.message(c, http.StatusBadGateway, "No pudimos cargar la publicación", "Intenta de nuevo en unos minutos.")
	default:
		// Los datos de contacto solo se muestran cuando la persona los solicita.
		a.render(c, http.StatusOK, "detail.html", listing.Brand+" "+listing.Model, gin.H{
			"V":           listing,
			"ShowContact": c.Query("contacto") == "1",
		})
	}
}

// image reenvía una imagen del backend. Se pide con la sesión de la persona,
// si la hay: las de una publicación pendiente solo las ven su dueño y el
// administrador, y el backend responde 404 a cualquier otro.
func (a *app) image(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	token := ""
	if _, s, ok := a.current(c); ok {
		token = s.Token
	}
	resp, err := a.api.image(token, id, c.Param("name"))
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private")
	c.DataFromReader(http.StatusOK, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

// ---------- Sesión ----------

func (a *app) loginPage(c *gin.Context) {
	if _, s, ok := a.current(c); ok {
		c.Redirect(http.StatusSeeOther, homeFor(s.Role))
		return
	}
	a.render(c, http.StatusOK, "login.html", "Iniciar sesión", gin.H{
		"Registered": c.Query("registered") == "1",
		"Expired":    c.Query("expired") == "1",
	})
}

func (a *app) login(c *gin.Context) {
	email := strings.ToLower(strings.TrimSpace(c.PostForm("email")))
	token, role, err := a.api.login(email, c.PostForm("password"))
	if err != nil {
		msg := "Correo o contraseña incorrectos."
		status := http.StatusUnauthorized
		if statusOf(err) != http.StatusUnauthorized {
			msg = "No pudimos iniciar sesión. Revisa los datos e intenta de nuevo."
			status = http.StatusBadRequest
		}
		a.render(c, status, "login.html", "Iniciar sesión", gin.H{"Error": msg, "Email": email})
		return
	}
	id := a.sessions.create(token, email, role)
	setSessionCookie(c, id, int(sessionTTL.Seconds()))
	c.Redirect(http.StatusSeeOther, homeFor(role))
}

func (a *app) registerPage(c *gin.Context) {
	a.render(c, http.StatusOK, "register.html", "Registrarse como vendedor", nil)
}

func (a *app) register(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	email := strings.TrimSpace(c.PostForm("email"))
	phone := strings.TrimSpace(c.PostForm("phone"))
	err := a.api.register(name, email, phone, c.PostForm("password"))
	if err != nil {
		msg := friendly(err,
			"Revisa los datos: todos los campos son obligatorios, el correo debe ser válido y la contraseña debe tener al menos 12 caracteres.",
			"Ese correo ya está registrado.")
		a.render(c, http.StatusBadRequest, "register.html", "Registrarse como vendedor", gin.H{
			"Error": msg, "Name": name, "Email": email, "Phone": phone,
		})
		return
	}
	c.Redirect(http.StatusSeeOther, "/login?registered=1")
}

func (a *app) logout(c *gin.Context) {
	if id, _, ok := a.current(c); ok {
		a.sessions.delete(id)
	}
	setSessionCookie(c, "", -1)
	c.Redirect(http.StatusSeeOther, "/")
}

// ---------- Vendedor ----------

func (a *app) sellerPage(c *gin.Context) {
	id, s, _ := a.current(c)
	listings, err := a.api.catalog(s.Token)
	if err != nil {
		a.fail(c, id, err, "/", "Datos inválidos.", "No se pudieron cargar tus publicaciones.")
		return
	}
	var mine []Listing
	for _, l := range slices.Backward(listings) { // más recientes primero
		if l.Seller.Email == s.Email {
			mine = append(mine, l)
		}
	}
	a.render(c, http.StatusOK, "seller.html", "Mis publicaciones", gin.H{
		"Mine":    mine,
		"MinYear": minYear,
		"MaxYear": time.Now().Year() + 1,
	})
}

func (a *app) publish(c *gin.Context) {
	id, s, _ := a.current(c)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)
	year, errYear := strconv.Atoi(c.PostForm("year"))
	price, errPrice := strconv.ParseInt(c.PostForm("price"), 10, 64)
	mileage, errMileage := strconv.ParseInt(c.DefaultPostForm("mileage", "0"), 10, 64)
	badData := fmt.Sprintf("Revisa los datos: el año debe estar entre %d y %d, el precio debe ser mayor que cero y las imágenes, JPEG o PNG de máximo 2 MB (hasta 12).", minYear, time.Now().Year()+1)
	images, errImages := uploadedImages(c)
	if errYear != nil || errPrice != nil || errMileage != nil || errImages != nil {
		a.notice(c, "error", badData)
		c.Redirect(http.StatusSeeOther, "/seller")
		return
	}

	_, err := a.api.publish(s.Token, publishInput{
		Brand:   strings.TrimSpace(c.PostForm("brand")),
		Model:   strings.TrimSpace(c.PostForm("model")),
		Year:    year,
		Price:   price,
		Mileage: mileage,
	}, images)
	if err != nil {
		a.fail(c, id, err, "/seller", badData, "No se pudo publicar.")
		return
	}
	a.notice(c, "ok", "Publicación enviada. Quedará visible cuando un administrador la autorice.")
	c.Redirect(http.StatusSeeOther, "/seller")
}

// uploadedImages lee las imágenes que eligió la persona en el formulario.
// Si no eligió ninguna, el navegador manda un archivo vacío: se ignora. El
// backend es quien valida el tipo, el tamaño y la cantidad.
func uploadedImages(c *gin.Context) ([]uploadFile, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}
	var images []uploadFile
	for _, header := range form.File["images"] {
		if header.Size == 0 {
			continue
		}
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
		file.Close()
		if err != nil {
			return nil, err
		}
		images = append(images, uploadFile{name: header.Filename, data: data})
	}
	return images, nil
}

func (a *app) sold(c *gin.Context) {
	id, s, _ := a.current(c)
	vehicleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/seller")
		return
	}
	if _, err := a.api.markSold(s.Token, vehicleID); err != nil {
		a.fail(c, id, err, "/seller", "Datos inválidos.", "Esa publicación ya estaba marcada como vendida.")
		return
	}
	a.notice(c, "ok", "Venta reportada. La publicación salió del catálogo.")
	c.Redirect(http.StatusSeeOther, "/seller")
}

// ---------- Administrador ----------

func (a *app) adminPage(c *gin.Context) {
	id, s, _ := a.current(c)
	listings, err := a.api.catalog(s.Token)
	if err != nil {
		a.fail(c, id, err, "/", "Datos inválidos.", "No se pudo cargar el listado.")
		return
	}
	byStatus := map[string][]Listing{}
	for _, l := range slices.Backward(listings) { // más recientes primero
		byStatus[l.Status] = append(byStatus[l.Status], l)
	}
	a.render(c, http.StatusOK, "admin.html", "Moderación", gin.H{
		"Pending":   byStatus["pendiente"],
		"Published": byStatus["publicada"],
		"Sold":      byStatus["vendida"],
	})
}

func (a *app) approve(c *gin.Context) {
	id, s, _ := a.current(c)
	vehicleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}
	if _, err := a.api.approve(s.Token, vehicleID); err != nil {
		a.fail(c, id, err, "/admin", "Datos inválidos.", "Esa publicación ya no está pendiente.")
		return
	}
	a.notice(c, "ok", "Publicación autorizada: ya es visible en el catálogo.")
	c.Redirect(http.StatusSeeOther, "/admin")
}

func (a *app) deleteVehicle(c *gin.Context) {
	id, s, _ := a.current(c)
	vehicleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}
	if err := a.api.deleteVehicle(s.Token, vehicleID); err != nil {
		a.fail(c, id, err, "/admin", "Datos inválidos.", "No se pudo eliminar la publicación.")
		return
	}
	a.notice(c, "ok", "Publicación eliminada.")
	c.Redirect(http.StatusSeeOther, "/admin")
}

func (a *app) deleteUser(c *gin.Context) {
	id, s, _ := a.current(c)
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}
	if err := a.api.deleteUser(s.Token, userID); err != nil {
		a.fail(c, id, err, "/admin", "Datos inválidos.", "No se puede eliminar a ese usuario.")
		return
	}
	a.notice(c, "ok", "Vendedor eliminado junto con todas sus publicaciones.")
	c.Redirect(http.StatusSeeOther, "/admin")
}
