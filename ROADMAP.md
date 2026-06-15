# ROADMAP — Reescritura `project-management` (Java → Go)

Mapa de hitos de la reescritura. Cada hito es una unidad coherente con un
**criterio de hecho** verificable. Se avanza hito por hito; no se da por cerrado
hasta cumplir su criterio.

Leyenda de estado: ✅ hecho · 🟡 parcial · ⬜ pendiente

---

## M0 — Base / esqueleto del servicio ✅

**Objetivo:** cimientos que no se reescriben después.

| Pieza | Estado | Archivo |
|---|---|---|
| Structs de dominio (Project, Sensor, Equipment) | ✅ | `internal/domain/domain.go` |
| Config externalizada + DSN MySQL + pool + retry | ✅ | `internal/config/config.go` |
| Conexión MySQL con `sqlx` | ✅ | `internal/config/config.go` |
| Graceful shutdown (signal → drain 15s) | ✅ | `cmd/server/main.go` |
| Logging estructurado (service/instance_id/trace_id/event) | ✅ | `internal/observability/logging.go` |
| Registro en Consul (raw HTTP, fail-fast=false) | ✅ | `internal/discovery/consul.go` |
| Health check real (ping MySQL) | ✅ | `cmd/server/main.go` |
| Migración inicial (goose) | ✅ | `migrations/00001_init_schema.sql` |
| Usuario DB de mínimos privilegios | ✅ | `scripts/01_min_privileges.sql` |

**Criterio de hecho:** `go build/vet/test` verde; servicio arranca, registra en
Consul, `/health` responde. ✅ (build/vet OK; arranque real contra MySQL/Consul → M8)

---

## M1 — Projects CRUD ✅

**Objetivo:** endpoint completo de proyectos con paridad de reglas Java.

| Capa | Estado | Archivo |
|---|---|---|
| Repository | ✅ | `internal/repository/project.go` |
| Service (validaciones, soft delete, merge) | ✅ | `internal/service/project.go` |
| Handler + rutas | ✅ | `internal/handler/project.go` |
| Contrato OpenAPI | ✅ | `contracts/openapi/...` |

**Reglas portadas:** id duplicado → 400, name requerido → 400, mismatch id
body/URL → 400, borrar con sensores activos → 400, inexistente → 404, update
merge (campo vacío conserva previo).

**Criterio de hecho:** rutas GET/POST/PUT/DELETE responden códigos correctos. ✅

---

## M2 — Sensors CRUD ✅

| Capa | Estado | Archivo |
|---|---|---|
| Repository (CRUD completo) | ✅ | `internal/repository/sensor.go` |
| Service (allow-set de tipos, dup, project-exists) | ✅ | `internal/service/sensor.go` |
| Handler + rutas anidadas | ✅ | `internal/handler/sensor.go` |
| Contrato OpenAPI | ✅ | `contracts/openapi/...` |

**Criterio de hecho:** rutas anidadas `/projects/{id}/sensors[/{sid}]` despachan
correctamente (verificado por `routing_test.go`). ✅

---

## M3 — Equipment (list + create) ✅

| Capa | Estado | Archivo |
|---|---|---|
| Repository (list + insert) | ✅ | `internal/repository/equipment.go` |
| Service (allow-set, sin dup, sin refetch) | ✅ | `internal/service/equipment.go` |
| Handler + rutas | ✅ | `internal/handler/equipment.go` |
| Contrato OpenAPI | ✅ | `contracts/openapi/...` |

**Nota de paridad:** baseline no expone update/delete ni chequea dup de
equipment_id (descansa en la PK). Misma laguna que Java → colisión PK devuelve
500, no 400. Documentado, no "arreglado" para mantener paridad.

**Criterio de hecho:** GET/POST `/projects/{id}/equipment` responden. ✅

---

## M4 — Pirámide de pruebas (CLAUDE.md §3) ✅

| Capa | Estado | Detalle |
|---|---|---|
| Unit — repository | ✅ | `project_test.go`: ErrNotFound (ErrNoRows + softDelete 0-rows) |
| Unit — service (reglas de negocio) | ✅ | project/sensor/equipment: dup, tipo, soft-delete, merge, mismatch, project-exists |
| Routing (handlers) | ✅ | `routing_test.go` (httptest + fakes) |
| Integration (Testcontainers MySQL) | ✅ | `integration_test.go` (tag `integration`): CRUD + goose + counts, verde contra MySQL 8.0 |
| Contract tests (vs OpenAPI / JSON Schema) | ⬜ | opcional — valor menor |
| CDCT (pact-go) | ⬜ | opcional — sin productores/consumidores AMQP en este servicio |

**Criterio de hecho:** unit cubre todas las reglas ✅; integration contra MySQL
efímero verde ✅. → **M4 cumple criterio.**

**Cómo correr integration:** `go test -tags=integration ./internal/repository/`
(requiere Docker; excluido del `go test ./...` por defecto). Verificado: 2/2 PASS,
goose aplica `00001_init_schema.sql`, round-trip de tipos (float, DATETIME→time).

**Pendiente opcional:** contract tests y CDCT no bloquean — este servicio es
REST/CRUD sin AMQP. Se dejan como mejora futura, no como deuda de M4.

---

## M5 — Seed / DataInitializer ✅

**Objetivo:** portar `DataInitializer.java` (carga inicial condicionada por flag).

| Pieza | Estado | Detalle |
|---|---|---|
| Seeder idempotente | ✅ | `internal/seed/seed.go`: siembra solo si tabla vacía |
| Datos | ✅ | 2 proyectos (shangtan/lingzhan) + 17 sensores (gnss/rainfall/camera), paridad Java |
| Flag de activación | ✅ | `SEED_ENABLED` (default true, paridad `app.seed.enabled`) |
| Falla suave | ✅ | error de seed → warn, no impide arranque |
| Unit tests | ✅ | vacío→siembra, no-vacío→no-op, projects-existen→solo-sensores (3/3) |
| Verificado e2e | ✅ | en contenedor real: 2 projects / 17 sensors; re-arranque = no-op |

**Criterio de hecho:** seed idempotente ✅, activable por env (`SEED_ENABLED`) ✅.
Para no sembrar en prod: `SEED_ENABLED=false`.

**Nota:** el seed inserta vía repos (no service) para saltar validaciones de
negocio, igual que el baseline usa mappers directamente.

---

## M6 — Observabilidad / resiliencia completa ✅

| Pieza | Estado | Detalle |
|---|---|---|
| Métricas Prometheus `/metrics` (CLAUDE.md §5) | ✅ | `http_requests_total` + `http_request_duration_seconds` (method/route/status, route=patrón chi); CPU/mem vía `process_*`/`go_*`. Verificado e2e en contenedor. p50..p99 vía `histogram_quantile()`. |
| Auditoría de campos de log estructurado | ✅ | service/instance_id/trace_id/event presentes y verificados en logs e2e |
| Verificación real de registro en Consul | ✅ | hecho en M8: registrado + check passing |

**Nota:** circuit breaker / retry / DLQ (CLAUDE.md §4) **no aplican** a este
servicio — es REST/CRUD, sin llamadas HTTP salientes entre servicios ni consumo
de RabbitMQ. Se documenta como N/A.

---

## M7 — Database-per-service / Strangler Fig ⬜ (futuro)

**Objetivo:** cutover del MySQL único compartido a instancia dedicada.

**Estado actual:** mitigado parcialmente — schema propio + usuario de mínimos
privilegios. `TODO(SE-practice)` en `config.go`. Cutover real pendiente y
coordinado a nivel monorepo.

---

## M8 — Run / verify end-to-end 🟡 (casi ✅)

**Objetivo:** levantar el servicio contra MySQL + Consul reales y verificar el
flujo CRUD completo (no solo build/test).

| Verificación | Estado |
|---|---|
| goose migra al startup (schema canónico, Opción B) | ✅ |
| re-arranque idempotente ("no migrations to run") | ✅ |
| conexión runtime como usuario min-priv | ✅ |
| endpoints CRUD con códigos HTTP correctos | ✅ (201/200/400/404/204) |
| reglas de negocio (dup, tipo inválido, sensor activo) | ✅ |
| persistencia real (soft-deletes en MySQL) | ✅ |
| Consul: registrado + visible + check **passing** | ✅ |
| graceful deregister al apagar | ⚠️ código correcto; no ejercitado por señal (Win/Git Bash no entrega SIGINT a exe nativo) |

**Aprendizajes operativos:**
- **Schema:** Opción B confirmada — goose dueño del schema; el infra solo crea
  DB vacía + usuario. Migración como root (`DB_ROOT_*`), runtime como svc min-priv.
- **`ADVERTISE_HOST=host.docker.internal`** obligatorio cuando Consul corre en
  contenedor y el servicio en el host; con `127.0.0.1` el check falla y Consul
  desregistra tras `DeregisterCriticalServiceAfter`.

**Pendiente para ✅ pleno — RECORDATORIO:** verificar el deregister con una señal
real. El código (`defer reg.Deregister()` en `ctx.Done()`) es estándar y correcto,
pero no se ejercitó por señal (Win/Git Bash no entrega SIGINT a exe nativo).
Trazado en código con `TODO(M8-verify)` en [cmd/server/main.go](cmd/server/main.go).

  Cómo verificar:
  1. `go build -o pmsvc.exe ./cmd/server` y correrlo en una terminal Windows real.
  2. `Ctrl+C`. Debe loguear `consul deregistered`.
  3. `curl http://localhost:8500/v1/agent/services` → ya no aparece el servicio.
  (Alternativa: correr el binario en Linux/contenedor y `kill -INT`.)

---

## Resumen

```
M0 Base                ✅
M1 Projects            ✅
M2 Sensors             ✅
M3 Equipment           ✅
M4 Testing             ✅  (unit + integration; contract/CDCT opcionales)
M5 Seed                ✅
M6 Observabilidad      ✅
M7 Strangler           ⬜ (futuro)
M8 Run/verify e2e      🟡  (CRUD+Consul verde vs infra real; falta deregister por señal)
```

Hitos núcleo (M0–M6) cerrados. Restan: **M7** (Strangler, futuro/coordinado a nivel
monorepo) y el caveat de **M8** (deregister por señal — recordatorio en código).
