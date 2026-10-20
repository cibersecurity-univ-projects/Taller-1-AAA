// Interfaz web de demostración de AutoMarket. Renderiza páginas HTML en el
// servidor con Gin y plantillas de html/template, y consume la API del
// backend. No forma parte del backend entregable.
package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	listenAddr = "localhost:3000"
	backendURL = "http://localhost:8080"
	cookieName = "am_session"
	sessionTTL = time.Hour
)

//go:embed templates static
var assets embed.FS

func main() {
	gin.SetMode(gin.ReleaseMode)

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"money":   formatThousands,
		"status":  statusLabel,
		"confirm": newConfirm,
	}).ParseFS(assets, "templates/*.html"))
	static, err := fs.Sub(assets, "static")
	if err != nil {
		log.Fatal(err)
	}

	a := &app{api: newBackend(backendURL), sessions: newSessionStore()}

	r := gin.New()
	r.Use(gin.Recovery())
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatal(err)
	}
	r.SetHTMLTemplate(tmpl)
	r.StaticFS("/static", http.FS(static))

	r.GET("/", a.catalog)
	r.GET("/vehicles/:id", a.detail)
	r.GET("/images/:id/:name", a.image)
	r.GET("/login", a.loginPage)
	r.POST("/login", a.login)
	r.GET("/register", a.registerPage)
	r.POST("/register", a.register)
	r.POST("/logout", a.logout)

	seller := r.Group("/seller", a.require(roleSeller))
	seller.GET("", a.sellerPage)
	seller.POST("/publish", a.publish)
	seller.POST("/vehicles/:id/sold", a.sold)

	admin := r.Group("/admin", a.require(roleAdmin))
	admin.GET("", a.adminPage)
	admin.POST("/vehicles/:id/approve", a.approve)
	admin.POST("/vehicles/:id/delete", a.deleteVehicle)
	admin.POST("/users/:id/delete", a.deleteUser)

	log.Printf("Interfaz en http://%s (backend: %s)", listenAddr, backendURL)
	log.Fatal(r.Run(listenAddr))
}

// formatThousands escribe 78000000 como 78.000.000.
func formatThousands(n int64) string {
	digits := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 && digits[i-1] != '-' {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	return b.String()
}

// confirmData alimenta la plantilla "confirm" (modal de confirmación).
type confirmData struct {
	ID, Action, Title, Text, Button, Kind string
}

func newConfirm(id, action, title, text, button, kind string) confirmData {
	return confirmData{ID: id, Action: action, Title: title, Text: text, Button: button, Kind: kind}
}

func statusLabel(s string) string {
	switch s {
	case "pendiente":
		return "Pendiente de aprobación"
	case "publicada":
		return "Publicada"
	case "vendida":
		return "Vendida"
	}
	return s
}
