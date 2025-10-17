# XDP47 - Edge Orchestrator for Retail/Kiosk Fleets (Go)

## 1) Summary (Business Need & Capabilities)
Retail/HoReCa chains operate hundreds of edge devices (kiosks, POS, info-panels) across locations. Manual updates and reliability issues cause lost sales and high ops costs.  
**Edge Orchestrator** is a centralized control plane + lightweight per-device agent that provides:
- **Zero-touch provisioning** for new devices
- **Version rollouts/rollbacks** of apps/containers
- **Real-time observability** (CPU/IO/network, crashes, kernel signals via eBPF)
- **Self-healing** (health checks, restart/rollback by policy)
- **Security**: mTLS, signed artifacts, token rotation
- **Multi-tenancy**: one control plane for multiple brands/clients

**Stack:** Go 1.22+, REST/gRPC, eBPF (cilium/ebpf), systemd D-Bus, containerd/k3s (optional), OpenTelemetry, PostgreSQL (control plane) + SQLite (agent).

---

## 2) Roles (Actors)
- **Anonymous/Viewer** – public status board (read-only)
- **Operator** – manages devices, rollouts, policies, metrics/alerts
- **Security Analyst** – artifact signatures, key rotations, compliance
- **FinOps** – cost/savings view (downtime cost, energy windows)
- **Tenant Admin** – per-tenant RBAC & user management
- **Platform Admin** – global control plane admin (tenants, quotas, regions)

---

## 3) Main Use Cases / Scenarios
| UC | Name | Description | Roles |
|---|---|---|---|
| 3.1 | **Zero-touch Registration** | Device boots, agent claims with a short-lived token, validates, and gets assigned tenant/location/labels. | Operator, Tenant Admin |
| 3.2 | **Define Policy** | Declarative policies (health, restart/rollback, resource limits, maintenance windows). | Operator |
| 3.3 | **Version Rollout** | Create rollout (waves, channels: dev/canary/prod), monitor, auto-rollback on SLO/health breach. | Operator |
| 3.4 | **Live Monitoring** | Real-time metrics/events (SSE/WebSocket): CPU, PSI, IO latency, TCP retransmits, process exits (eBPF). | Operator |
| 3.5 | **Incident & Recovery** | Auto-restart, pin to previous version, ticket + post-mortem bundle (artifact, logs, kernel events). | Operator, Security Analyst |
| 3.6 | **Device Operations** | Filter by labels/locations; drain/cordon; guarded remote exec; change config channel. | Operator |
| 3.7 | **Secrets & Artifacts** | Manage signed build artifacts, agent attestation, credential rotation. | Security Analyst, Admin |
| 3.8 | **User Management** | CRUD users/roles/tenants; SSO (OIDC). | Tenant Admin, Platform Admin |
| 3.9 | **Reports & Compliance** | Uptime/SLA, MTTR, changes/rollouts, incidents; export to SIEM. | Operator, Security Analyst, FinOps |
| 3.10 | **Financial View** | Downtime cost, projected savings from automation/energy modes. | FinOps |

---

## 4) Main Views (Frontend)
| View | Description | URI |
|---|---|---|
| **Home** | Intro, CTA to add/register a device. | `/` |
| **Devices** | List/filter/labels, status, version, location; actions: drain/rollback/exec. | `/devices` |
| **Rollouts** | Create/manage rollouts; waves, channels, status, failure ratio. | `/rollouts` |
| **Policies** | Define policies for health/auto-heal/resources/maintenance. | `/policies` |
| **Dashboards** | Real-time metrics/events; drill-down by device/location. | `/dashboards` |
| **Alerts** | Alerts/incidents, filters, integrations (email/webhook). | `/alerts` |
| **Tenants/Users** | RBAC, invites, roles, SSO. | `/tenants`, `/users` |
| **Reports** | Uptime/SLA, MTTR, changes, compliance (PDF/CSV). | `/reports` |

---

## 5) API Resources (Backend)
| Resource | Description | Endpoints |
|---|---|---|
| **Auth** | OIDC login → JWT; mTLS agent ↔ control plane. | `POST /api/login`, `POST /api/logout` |
| **Devices** | List/register/details; live status SSE. | `GET /api/devices`, `POST /api/devices/claim`, `GET /api/devices/{id}` |
| **Device Actions** | drain/cordon/exec/rollback. | `POST /api/devices/{id}:drain`, `POST /api/devices/{id}:rollback`, `POST /api/devices/{id}:exec` |
| **Metrics/Events** | Streaming metrics/events. | `GET /api/devices/{id}/metrics/stream` |
| **Rollouts** | CRUD rollouts; dry-run/simulate. | `GET/POST /api/rollouts`, `GET/PUT /api/rollouts/{id}`, `POST /api/rollouts/{id}:simulate` |
| **Policies** | CRUD policies; validation/versioning. | `GET/POST /api/policies`, `GET/PUT /api/policies/{id}` |
| **Artifacts** | Artifact registry, signatures, provenance. | `GET/POST /api/artifacts`, `GET /api/artifacts/{digest}` |
| **Tenants/Users** | RBAC & users. | `GET/POST /api/tenants`, `GET/POST /api/users` |
| **Reports** | Generate/download reports. | `GET /api/reports?from=&to=&type=` |
| **Active Ops** | Stream active ops (rollouts/incidents). | `GET /api/active-ops` |

---

## 6) Data Model (Key Entities)
- **Tenant**(id, name, plan, settings)  
- **User**(id, email, role, tenant_id, sso_id)  
- **Device**(id, tenant_id, labels{key:val}, location, status, version, last_seen, channel)  
- **Policy**(id, tenant_id, spec:JSON, version, created_by)  
- **Artifact**(id, name, digest, signature, created_at, sbom_ref)  
- **Rollout**(id, tenant_id, artifact_id, policy_ref, waves[], status)  
- **Metric**(device_id, ts, cpu, mem, psi, io_latency, net_retx, …)  
- **Event**(id, device_id, ts, type, payload)  
- **Incident**(id, device_id, started_at, resolved_at, cause, actions[])  
- **Report**(id, tenant_id, period, kind, url)

---

## 7) Architecture
- **Agent (Go):** systemd unit; minimal Linux capabilities; secure updater (signed binary); executor (container/app); collectors (eBPF, /proc, PSI); mTLS channels.  
- **Control Plane (Go):** REST/gRPC; rollout scheduler (waves/canaries); policy engine; event bus; OTel metrics/traces; Web UI.  
- **Storage:** PostgreSQL; object store for artifacts; time-series via Postgres hypertables / ClickHouse / or OTel Collector → Prometheus.  
- **Integrations:** Webhooks, email, SIEM export (OTel/CEF).

---

## 8) Non-Functional Requirements
- **Reliability:** SLO 99.9% API uptime; MTTR < 30 min for critical incidents.  
- **Security:** mTLS, OIDC, RBAC; artifact signatures (Cosign); token rotation.  
- **Scalability:** 10k devices/cluster; 1k events/sec; horizontal scale.  
- **Agent Perf:** <1–2% CPU, <64 MB RAM; low IO overhead.  
- **Compatibility:** Linux (systemd), containerd/k3s; fallback for non-container apps.  
- **Observability:** OTel metrics/traces/logs; incident ↔ rollout correlation.

---

## 9) High-Level Sequences
1) **Claim Device:** device → auth (short-lived token) → register → assign labels/channel → start heartbeat/metrics stream.  
2) **Rollout:** operator creates rollout → scheduler triggers waves → agent fetches artifact → verify signature → health checks → success or auto-rollback.  
3) **Incident:** health breach → auto-restart → persistent issue → auto-rollback → alert → RCA report bundle.

---

## 10) Security
- **Supply Chain:** Sigstore/Cosign signatures; SBOM (CycloneDX).  
- **Transport:** mTLS agent↔control plane; CRL/rotation.  
- **Privileges:** agent non-root (capabilities); namespaces/CGroups isolation.  
- **Secrets:** short-lived pull tokens; at rest via KMS.

---

## 11) Observability
- **Key Metrics:** API latency, queue depth, rollout success rate, device heartbeat gap.  
- **Kernel/OS Events:** `process_exit`, `tcp_retransmit`, `oom_kill`, `health_fail`.  
- **Traces:** user→API→scheduler→device RPC.  
- **Logs:** structured JSON; correlation by `request_id`/`device_id`.

---

## 12) Risks & Mitigations
- **Heterogeneous hardware/OS:** capability probing + canary channels.  
- **eBPF compatibility:** graceful degrade to `/proc`/inotify when unavailable.  
- **Intermittent connectivity:** offline cache, exponential backoff retries, offline “ops bundle” via USB.

---

## 13) Roadmap / Sprints (8 Weeks)
**Sprint 1 (MVP Core):** Claim/registration, Device list, Heartbeat/metrics stream, simple single-wave rollout.  
**Sprint 2:** Health/rollback policies, Alerts, basic roles (Operator/Admin).  
**Sprint 3:** Channels (dev/canary/prod), multi-tenant, signed artifacts.  
**Sprint 4:** eBPF collectors, Reports (uptime/MTTR), Webhooks, FinOps view.

---

## 14) Acceptance Criteria (Examples)
- New device registers in **< 60s** and appears in `/devices` with labels.  
- Rollout to **10 devices** succeeds; on **≥30% failures** the system auto-rolls back.  
- Live dashboard shows CPU/PSI and `process_exit` events per device in near real time.  
- Unsigned or invalidly signed artifact **must not** execute.  
- A 7-day uptime report (PDF/CSV) is generated and downloadable.

---

## 15) Test Plan (incl. OS/Orchestration)
- **Unit:** policy evaluation, rollout scheduler, signature verify.  
- **Integration:** agent↔API (TLS), registration, rollout, health/rollback.  
- **OS/eBPF:** simulate process exits, network loss (tc/netem), PSI pressure.  
- **Chaos:** network partition, disk pressure, clock skew.  
- **Load:** 10k virtual devices, 1k ev/s, SSE latency p95 < 300 ms.

---

## 16) Demo Script (10–12 min)
1) Claim two “devices” (VM/containers) → they appear in `/devices`.  
2) Create rollout v1.2 → waves 1/2 → monitor; simulate failure on one device → auto-rollback.  
3) Live dashboard shows eBPF `process_exit` and `tcp_retransmit` during the incident.  
4) Generate and download an uptime report.

---

## 17) Stretch Goals
- **Edge-ML anomaly detection** on metrics.  
- **Energy-aware scheduling** (off-hours maintenance).  
- **GitOps integration** (trigger rollouts from PRs).
