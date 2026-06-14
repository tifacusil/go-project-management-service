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

## M4 — Pirámide de pruebas (CLAUDE.md §3) 🟡

| Capa | Estado | Detalle |
|---|---|---|
| Unit — repository | ✅ | `project_test.go`: ErrNotFound (ErrNoRows + softDelete 0-rows) |
| Unit — service (reglas de negocio) | ✅ | project/sensor/equipment: dup, tipo, soft-delete, merge, mismatch, project-exists |
| Routing (handlers) | ✅ | `routing_test.go` (httptest + fakes) |
| Contract tests (vs OpenAPI / JSON Schema) | ⬜ | — |
| CDCT (pact-go) | ⬜ | — |
| Integration (Testcontainers MySQL) | ⬜ | flujo CRUD crítico |

**Criterio de hecho:** unit de service cubre todas las reglas ✅; existe al menos
un integration contra MySQL efímero ⬜.

**Sub-estado:** capa unit cerrada. Faltan contract tests, CDCT e integration para
dar M4 por ✅ completo.

---

## M5 — Seed / DataInitializer ⬜

**Objetivo:** portar `DataInitializer.java` (carga inicial condicionada por flag).

**Criterio de hecho:** seed idempotente, activable por env, no corre en prod por defecto.

---

## M6 — Observabilidad / resiliencia completa ⬜

| Pieza | Estado | Detalle |
|---|---|---|
| Métricas Prometheus `/metrics` (CLAUDE.md §5) | ⬜ | CPU, memoria, throughput, latencia p50/p90/p95/p99 |
| Auditoría de campos de log estructurado | 🟡 | base puesta; falta revisar cobertura por evento |
| Verificación real de registro en Consul | ⬜ | depende de M8 |

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

## M8 — Run / verify end-to-end ⬜

**Objetivo:** levantar el servicio contra MySQL + Consul reales y verificar el
flujo CRUD completo (no solo build/test).

**Criterio de hecho:** `go run` arranca, migración aplica, requests reales
devuelven datos, servicio aparece en Consul, deregistra al apagar.

---

## Resumen

```
M0 Base                ✅
M1 Projects            ✅
M2 Sensors             ✅
M3 Equipment           ✅
M4 Testing             🟡  ← siguiente foco sugerido
M5 Seed                ⬜
M6 Observabilidad      ⬜
M7 Strangler           ⬜ (futuro)
M8 Run/verify e2e      ⬜
```

Próximo hito sugerido: **M4** (cerrar unit tests de service) o **M8** (verificar
que lo construido corre de verdad). A decidir.
```
