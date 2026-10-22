# AutoMarket

Sitio de venta de vehículos usados, hecho para el taller de seguridad AAA
(Authentication, Authorization y Accounting). Tiene un backend (la API) y un
frontend (la interfaz web).

Está escrito en Go y usa solo la librería estándar y [Gin](https://github.com/gin-gonic/gin).
No usa JWT, bcrypt, ORM ni drivers de base de datos.

## Qué hace cada parte

**Backend (`backend/`)**: la API en `http://localhost:8080`. Maneja el catálogo,
los usuarios y las publicaciones, con tres roles:

- Visitante: consulta el catálogo y los datos de contacto del vendedor.
- Vendedor: se registra, publica vehículos con hasta 12 fotos y reporta sus ventas.
- Administrador: autoriza publicaciones y elimina publicaciones o vendedores.

**Frontend (`frontend/`)**: la interfaz web en `http://localhost:3000`. Muestra el
catálogo y los paneles de vendedor y administrador. Usa plantillas HTML de Gin,
sin JavaScript, y le pide todo al backend.

## Dónde está el marco AAA

| Parte | Cómo se hizo | Dónde |
|---|---|---|
| Authentication | Contraseñas con PBKDF2-HMAC-SHA256 y sal propia. Sesiones con tokens aleatorios de 128 bits que viven en memoria | `backend/auth/password.go`, `backend/auth/sessions.go`, `backend/middleware/authentication.go`, `backend/handlers/accounts.go` |
| Authorization | Control por rol (visitante, vendedor, administrador) y solo el dueño reporta la venta de su publicación | `backend/middleware/authorization.go`, `backend/service/vehicles.go`, `backend/router.go` |
| Accounting | Cada solicitud queda en una bitácora. Los registros están encadenados y firmados con HMAC-SHA256, y se verifican al arrancar | `backend/middleware/accounting.go`, `backend/audit/logger.go`, `backend/data/audit.log` |

La persistencia usa archivos JSON, con `encoding/json` y `os`: `backend/storage/table.go`
y la carpeta `backend/data/`.

## Cómo ejecutarlo

Necesita Go 1.26.2 o superior (`go version`) e internet la primera vez, para
descargar Gin. Se usan dos terminales, y el backend va primero.

**Terminal 1: backend**

```bash
cd backend
export AUTOMARKET_AUDIT_KEY="clave-secreta-de-auditoria-de-32-chars"
export AUTOMARKET_ADMIN_EMAIL="admin@automarket.co"
export AUTOMARKET_ADMIN_PASSWORD="AdminSegura123"
go run .
```

En PowerShell, las variables se ponen así:

```powershell
$env:AUTOMARKET_AUDIT_KEY = "clave-secreta-de-auditoria-de-32-chars"
$env:AUTOMARKET_ADMIN_EMAIL = "admin@automarket.co"
$env:AUTOMARKET_ADMIN_PASSWORD = "AdminSegura123"
```

**Terminal 2: frontend**

```bash
cd frontend
go run .
```

Abre `http://localhost:3000`.

La carpeta `backend/data/` ya trae datos de ejemplo: 12 vehículos con fotos y 4
usuarios. Hay que usar la clave de auditoría de arriba, porque la bitácora de esa
carpeta está firmada con ella. Con otra clave el backend no arranca.

## Usuarios de ejemplo

| Rol | Correo | Contraseña |
|---|---|---|
| Administrador | `admin@automarket.co` | `AdminSegura123` |
| Vendedor | `carlos.ramirez@automarket.co` | `CarlosRamirez2026` |
| Vendedora | `maria.lopez@automarket.co` | `MariaLopez2026x` |
| Vendedor | `vendedor@automarket.co` | `Vendedor12345` |

Más detalles en `backend/README.md` y `frontend/README.md`.
