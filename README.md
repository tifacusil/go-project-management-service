# Project Management Service

Microservicio REST en Go para la gestión de **proyectos**, **sensores** y **equipamiento**. Forma parte de un sistema de microservicios más amplio y reemplaza la funcionalidad equivalente del monolito baseline (estrategia Strangler Fig).

> La infraestructura (compose, redes, MySQL, Consul, etc.) se encuentra en: [tifacusil/go-tesis-infra](https://github.com/tifacusil/go-tesis-infra).

## Características

- **API REST** versionada bajo `/api/v1` construida con [chi](https://github.com/go-chi/chi).
- **Persistencia MySQL** vía [sqlx](https://github.com/jmoiron/sqlx) con el driver puro-Go `go-sql-driver/mysql` (sin CGO).
- **Migraciones automáticas** con goose al arranque (usa credenciales root sólo para el DDL).
- **Service discovery** con Consul (registro/desregistro automático; *fail-fast=false*: la app arranca aunque Consul esté caído).
- **Observabilidad**: logging estructurado JSON (`slog`) con `request_id` por petición.
- **Health check** real (`/health`) que verifica conectividad con la base de datos.
- **Graceful shutdown** con drenado de 15 s ante `SIGINT`/`SIGTERM`.

## Stack

| Componente | Tecnología |
|------------|------------|
| Lenguaje | Go 1.25 |
| Router HTTP | chi v5 |
| Base de datos | MySQL |
| Acceso a datos | sqlx |
| Migraciones | goose |
| Discovery | Consul |
| Validación | go-playground/validator |

## Endpoints

Base: `/api/v1`

| Método | Ruta | Descripción |
|--------|------|-------------|
| `GET` | `/projects` | Lista proyectos |
| `POST` | `/projects` | Crea un proyecto |
| `GET` | `/projects/{project_id}` | Obtiene un proyecto |
| `PUT` | `/projects/{project_id}` | Actualiza un proyecto |
| `DELETE` | `/projects/{project_id}` | Elimina un proyecto |
| `GET` | `/projects/{project_id}/sensors` | Lista sensores del proyecto |
| `POST` | `/projects/{project_id}/sensors` | Crea un sensor |
| `GET` | `/projects/{project_id}/sensors/{sensor_id}` | Obtiene un sensor |
| `PUT` | `/projects/{project_id}/sensors/{sensor_id}` | Actualiza un sensor |
| `DELETE` | `/projects/{project_id}/sensors/{sensor_id}` | Elimina un sensor |
| `GET` | `/projects/{project_id}/equipment` | Lista equipamiento del proyecto |
| `POST` | `/projects/{project_id}/equipment` | Crea equipamiento |
| `GET` | `/health` | Health check (verifica la BD) |

## Configuración

La configuración se obtiene de variables de entorno (o de un fichero `.env` opcional; las variables de entorno tienen prioridad).

| Variable | Default | Descripción |
|----------|---------|-------------|
| `SERVICE_NAME` | `project-management-service` | Nombre del servicio |
| `SERVICE_VERSION` | `0.1.0` | Versión |
| `ENV` | `dev` | Entorno |
| `INSTANCE_ID` | — | Id de instancia (lo inyecta el orquestador; vacío => se genera) |
| `HTTP_PORT` | `8080` | Puerto HTTP (se usa para construir la dirección de escucha) |
| `HTTP_ADDR` | — | Dirección de escucha completa; si se setea, tiene prioridad sobre `HTTP_PORT` |
| `ADVERTISE_HOST` | `127.0.0.1` | Host anunciado a Consul (debe ser alcanzable por Consul) |
| `HEALTH_PATH` | `/health` | Ruta del health check |
| `DB_HOST` | `127.0.0.1` | Host de MySQL |
| `DB_PORT` | `3306` | Puerto de MySQL |
| `DB_USER` | `proj-manager` | Usuario de la app (mínimos privilegios) |
| `DB_PASS` | — | Contraseña del usuario de la app |
| `DB_NAME` | `project_management_db` | Nombre de la base de datos |
| `DB_MAX_OPEN_CONNS` | `25` | Conexiones abiertas máximas del pool |
| `DB_MAX_IDLE_CONNS` | `25` | Conexiones idle máximas del pool |
| `DB_ROOT_USER` | — | Usuario root para migraciones (DDL al arranque) |
| `DB_ROOT_PASS` | — | Contraseña root para migraciones |
| `CONSUL_ENABLED` | `true` | Habilita el registro en Consul |
| `CONSUL_HOST` | `localhost` | Host de Consul |
| `CONSUL_PORT` | `8500` | Puerto de Consul |
| `CONSUL_TOKEN` | — | Token ACL de Consul (opcional) |
| `CONSUL_HEALTH_INTERVAL` | `10s` | Intervalo del health check en Consul |

## Build de la imagen Docker

La imagen es multi-stage (binario estático sobre base Alpine, usuario no-root). Se usa Alpine en vez de distroless porque el healthcheck del compose de infra invoca `wget`, que distroless no incluye.

El `build` es **local**: la imagen queda en tu Docker local con el mismo tag que espera el compose de infra. No se sube a ningún registro hasta hacer `push` explícitamente.

```bash
docker build -t ghcr.io/tifacusil/project-management:latest .
```

## Levantar el contenedor (detached) localmente

El contenedor escucha en el puerto **8080** internamente. El compose de infra lo mapea a `8090` en el host, corre en la red `landslide-net` y resuelve MySQL y Consul por nombre de servicio (`mysql-project-management:3306`, `consul-1:8500`).

### Alineado al compose de infra (red `landslide-net`)

```bash
docker run -d --name project-management-1 --hostname project-management-1 \
  --network landslide-net \
  -p 8090:8080 \
  --env-file .env \
  -e INSTANCE_ID=project-management-01 \
  -e HTTP_PORT=8080 \
  -e DB_HOST=mysql-project-management \
  -e DB_PORT=3308 \
  -e CONSUL_HOST=consul-1 \
  -e CONSUL_PORT=8500 \
  -e ADVERTISE_HOST=project-management-1 \
  --init \
  ghcr.io/tifacusil/project-management:latest
```

- `--network landslide-net` une el contenedor a la red de la infra (debe existir; el compose la declara `external`).
- `-p 8090:8080` mapea el puerto del host (8090) al del contenedor (8080).
- `DB_HOST` / `CONSUL_HOST` apuntan a los nombres de servicio en la red, no a `127.0.0.1`.
- `ADVERTISE_HOST=project-management-1` es el hostname con que Consul alcanza el `/health`.
- `--init` reapea procesos zombie (equivale a `init: true` del compose).

### Prueba rápida con MySQL/Consul en el host (Docker Desktop)

> Dentro del contenedor, `127.0.0.1` apunta al **propio contenedor**, no a tu máquina. Si MySQL/Consul corren en el host, usa `host.docker.internal`.

```bash
docker run -d --name pms -p 8090:8080 \
  --env-file .env \
  -e HTTP_PORT=8080 \
  -e DB_HOST=host.docker.internal \
  -e CONSUL_ENABLED=false \
  ghcr.io/tifacusil/project-management:latest
```

### Verificar, ver logs y detener

```bash
# Health check (puerto del host)
curl http://localhost:8090/health

# Logs en streaming
docker logs -f project-management-1

# Detener y eliminar
docker stop project-management-1 && docker rm project-management-1
```

## Ejecución local (sin Docker)

```bash
go run ./cmd/server
```

Requiere un fichero `.env` con la configuración apuntando a una instancia de MySQL accesible.
