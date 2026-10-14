# AutoMarket · Interfaz web

Interfaz de demostración para el backend de AutoMarket. Presenta el catálogo de
vehículos y los paneles de vendedor y administrador.

Renderiza las páginas **en el servidor** con [Gin](https://github.com/gin-gonic/gin)
y plantillas `html/template`. No usa JavaScript: la navegación, los formularios y
las ventanas de confirmación funcionan solo con HTML y CSS.

## Requisitos

- Go 1.26.2 o superior (`go version`).
- Conexión a internet la primera vez, para descargar Gin.
- El **backend de AutoMarket encendido** en `http://localhost:8080`.

## Ejecución

Primero encienda el backend (consulte su README) y después:

```bash
go run .
```

Abra `http://localhost:3000`.

| Ajuste | Valor | Dónde se cambia |
|---|---|---|
| Dirección de la interfaz | `localhost:3000` | constante `listenAddr` en `main.go` |
| Dirección del backend | `http://localhost:8080` | constante `backendURL` en `main.go` |

## Pantallas

| Ruta | Quién | Qué muestra |
|---|---|---|
| `/` | Todos | Catálogo: tarjetas con marca, modelo, precio, año y kilometraje |
| `/vehicles/:id` | Todos | Detalle del vehículo. Los datos de contacto del vendedor aparecen al pulsar **Ver datos de contacto** |
| `/login` | Todos | Inicio de sesión de vendedores y administradores |
| `/register` | Todos | Registro como vendedor |
| `/seller` | Vendedor | Formulario de nueva publicación y sus publicaciones, con **Reportar venta** |
| `/admin` | Administrador | Pendientes, publicadas y vendidas, con **Autorizar** y eliminar publicación o vendedor |

En la esquina superior derecha aparecen **Iniciar sesión** y **Registrarse como
vendedor**; con sesión iniciada, **Mi panel** y **Salir**.

Las acciones destructivas o irreversibles (reportar una venta, eliminar una
publicación o un vendedor) piden confirmación en una ventana modal.

## Cómo funciona

```
Navegador ──► Interfaz (Gin + plantillas) ──► API del backend
```

- La interfaz **no guarda datos de negocio**: toda la información sale de la API.
- Al iniciar sesión, el token del backend se guarda **en el servidor de la
  interfaz**; el navegador solo recibe una cookie opaca (`HttpOnly`, `SameSite=Lax`).
- Cada pantalla pide a la API solo lo que el rol puede ver. El catálogo público se
  consulta sin token; los paneles, con el token del usuario.
- Los permisos los decide siempre el backend: si alguien intenta abrir un panel que
  no le corresponde, recibe un aviso de falta de permiso.
- Los textos se muestran escapados por `html/template`, de modo que el contenido
  introducido por los usuarios nunca se interpreta como HTML.

## Estructura

```
main.go        arranque, rutas y funciones de las plantillas
app.go         manejadores de cada pantalla
backend.go     cliente HTTP de la API
store.go       sesiones de la interfaz (en memoria)
templates/     plantillas HTML
static/        hoja de estilos
```

## Limitaciones

- **Las sesiones de la interfaz viven en memoria**: al reiniciarla hay que iniciar
  sesión de nuevo.
- **Las ventanas de confirmación son solo CSS** (selector `:target`): se cierran con
  *Cancelar* o haciendo clic fuera, pero no con la tecla Esc.
- No hay fotos de los vehículos ni ubicación: la API no maneja esos datos.
- Pensada para demostración; no incluye protección contra falsificación de
  solicitudes (CSRF) más allá de `SameSite=Lax`, ni TLS.
