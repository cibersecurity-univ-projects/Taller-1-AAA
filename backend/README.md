# AutoMarket · Backend

API HTTP para un sitio de venta de vehículos usados. Expone un catálogo público
y funciones diferenciadas por rol (visitante, vendedor y administrador),
protegidas con el marco **AAA**: Authentication, Authorization y Accounting.

Escrita en Go usando solo la librería estándar y [Gin](https://github.com/gin-gonic/gin)
como única dependencia externa.

## Requisitos

- Go 1.26.2 o superior (`go version`).
- Conexión a internet la primera vez, para descargar Gin.

## Ejecución

El servidor necesita tres variables de entorno:

| Variable | Uso |
|---|---|
| `AUTOMARKET_AUDIT_KEY` | Clave que sella la bitácora de auditoría (mínimo 32 caracteres) |
| `AUTOMARKET_ADMIN_EMAIL` | Correo de la cuenta de administrador, creada al arrancar si no existe |
| `AUTOMARKET_ADMIN_PASSWORD` | Contraseña de esa cuenta |

**PowerShell**

```powershell
$env:AUTOMARKET_AUDIT_KEY = "una-clave-secreta-de-al-menos-32-caracteres"
$env:AUTOMARKET_ADMIN_EMAIL = "admin@automarket.co"
$env:AUTOMARKET_ADMIN_PASSWORD = "AdminSegura123"
go run .
```

**Bash**

```bash
export AUTOMARKET_AUDIT_KEY="una-clave-secreta-de-al-menos-32-caracteres"
export AUTOMARKET_ADMIN_EMAIL="admin@automarket.co"
export AUTOMARKET_ADMIN_PASSWORD="AdminSegura123"
go run .
```

El servicio escucha en `http://localhost:8080` y guarda sus datos en la carpeta
`data/`, que crea al arrancar. Si falta alguna variable, el servidor no inicia y
lo indica.

> Use siempre la **misma clave de auditoría** sobre una carpeta `data/` existente:
> con otra clave la bitácora no verifica y el servidor se niega a arrancar.
> Para empezar de cero, borre `data/`.

## Operaciones

Las rutas protegidas exigen `Authorization: Bearer <token>`, con el token que
devuelve `POST /auth/login`.

| Método | Ruta | Quién | Qué hace |
|---|---|---|---|
| `GET` | `/vehicles` | Todos (según el rol) | Lista de publicaciones |
| `GET` | `/vehicles/:id` | Público | Vehículo y datos de contacto del vendedor |
| `POST` | `/auth/register` | Público | Registro de vendedor |
| `POST` | `/auth/login` | Público | Inicio de sesión; devuelve el token |
| `POST` | `/vehicles` | Vendedor | Publica un vehículo en estado `pendiente` |
| `PATCH` | `/vehicles/:id/sold` | Vendedor dueño | Reporta la venta; sale del catálogo |
| `PATCH` | `/vehicles/:id/approve` | Administrador | Autoriza una publicación pendiente |
| `DELETE` | `/vehicles/:id` | Administrador | Elimina una publicación |
| `DELETE` | `/users/:id` | Administrador | Elimina un vendedor y sus publicaciones |

### Qué devuelve `GET /vehicles` según quién pregunta

| Solicitante | Resultado |
|---|---|
| Visitante (sin token) | Solo publicaciones `publicada` |
| Vendedor | Las publicadas y todas las suyas, de cualquier estado |
| Administrador | Todas |
| Token inválido o vencido | `401` |

### Estados de una publicación

```
pendiente ──(administrador autoriza)──► publicada ──(vendedor reporta)──► vendida
```

Solo las publicadas son visibles al público.

### Datos de entrada

**Registro** (`POST /auth/register`)

```json
{ "name": "Ana Perez", "email": "ana@mail.com", "phone": "3001234567", "password": "una-contraseña-larga" }
```

La contraseña debe tener entre 12 y 128 caracteres. Todo registro crea un
vendedor; el rol no se puede elegir.

**Publicación** (`POST /vehicles`)

```json
{ "brand": "Renault", "model": "Koleos", "year": 2018, "price": 78000000, "mileage": 99000 }
```

El año debe estar entre 1950 y el año siguiente al actual; el precio, mayor que cero.

### Códigos de respuesta

| Código | Significado |
|---|---|
| `200`, `201`, `204` | Operación realizada |
| `400` | Datos inválidos |
| `401` | Falta autenticación, o el token es inválido o expiró |
| `403` | El rol no permite la operación, o el recurso es de otro usuario |
| `404` | El recurso no existe (o no es visible para el solicitante) |
| `409` | Estado incompatible (por ejemplo, autorizar algo que no está pendiente) o correo ya registrado |

## Seguridad (AAA)

### Authentication

- Contraseñas con **PBKDF2-HMAC-SHA256**, 600 000 iteraciones y sal aleatoria por
  usuario; la comparación es en tiempo constante.
- Sesiones por token aleatorio de 128 bits (`crypto/rand`) con expiración de 1 hora.
- Un mismo mensaje para correo inexistente y contraseña incorrecta, para no revelar
  qué cuentas existen.
- Al eliminar un usuario, su token deja de ser válido.

### Authorization

- Control por rol en el router (`middleware/authorization.go`).
- La regla de propiedad (solo el dueño reporta su venta) y la visibilidad del
  catálogo según el rol están en la capa de servicio.
- El rol nunca se recibe del cliente: el registro siempre crea vendedores y el
  administrador se crea solo al arrancar.

### Accounting

- Toda solicitud queda registrada en `data/audit.log`, incluidas las públicas y
  las rechazadas (401, 403, 404...).
- Cada entrada guarda actor, rol, IP, método, ruta, resultado y, en intentos sin
  identidad verificada, el correo usado.
- La bitácora está **encadenada con HMAC-SHA256**: cada entrada incluye el hash de
  la anterior. Al arrancar se verifica toda la cadena y, si fue alterada, el
  servidor no inicia.

## Estructura

```
main.go, router.go   arranque, configuración y rutas
models/              entidades: usuario, vehículo, entrada de auditoría
storage/             tablas genéricas persistidas en archivos JSON
service/             reglas de negocio
handlers/            capa HTTP: validación y respuestas
middleware/          Authentication, Authorization y Accounting
auth/                contraseñas, sesiones e identidad del solicitante
audit/               bitácora encadenada
```

Las dependencias van en un solo sentido: `handlers → service → storage → models`.
La capa de servicio no conoce HTTP.

## Persistencia

Archivos JSON dentro de `data/`:

| Archivo | Contenido |
|---|---|
| `users.json` | Usuarios, con la contraseña solo como hash |
| `vehicles.json` | Publicaciones |
| `audit.log` | Bitácora, una entrada por línea |

La escritura es atómica (archivo temporal y renombrado) y los archivos se crean
con permisos restringidos.

## Pruebas rápidas

```bash
B=http://localhost:8080
J='Content-Type: application/json'

# registro e inicio de sesión
curl -i -X POST $B/auth/register -H "$J" \
  -d '{"name":"Ana Perez","email":"ana@mail.com","phone":"3001234567","password":"UnaClaveLarga123"}'
TOKEN=$(curl -s -X POST $B/auth/login -H "$J" \
  -d '{"email":"ana@mail.com","password":"UnaClaveLarga123"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')

# publicar (queda pendiente) y consultar el catálogo público
curl -i -X POST $B/vehicles -H "$J" -H "Authorization: Bearer $TOKEN" \
  -d '{"brand":"Renault","model":"Koleos","year":2018,"price":78000000,"mileage":99000}'
curl -i $B/vehicles

# una operación protegida sin token responde 401
curl -i -X POST $B/vehicles -H "$J" -d '{}'
```

## Limitaciones conocidas

- **Solo HTTP, sin TLS.** Las credenciales y los tokens viajan sin cifrar; en un
  despliegue real debe ponerse TLS.
- **Las sesiones viven en memoria**: al reiniciar el servidor hay que iniciar sesión
  de nuevo.
- **La bitácora no detecta que le quiten las últimas líneas**: la cadena detecta
  modificar o borrar entradas intermedias, pero no truncar el final.
- **Si no se puede escribir la bitácora**, se registra el aviso en el log del
  servidor y la operación, que ya se ejecutó, no se deshace.
- No hay límite de intentos de inicio de sesión ni cierre de sesión explícito.
- La persistencia en archivos JSON no está pensada para alta concurrencia.
