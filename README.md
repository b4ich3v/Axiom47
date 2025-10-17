# Course_Go_Project_Summary

## Project name
Axiom47

## Project author
Yoan Desislavov Baychev

---

## 1. Short project description (Business needs and system features)

In the era of digital services, companies need accurate predictions for **customer churn probability** and **Customer Lifetime Value (CLV)** to plan retention budgets, offers, and capacity. The **Axiom47** platform provides an end‑to‑end solution for data ingestion, modeling (logistic regression, survival analysis, BG/NBD + Gamma–Gamma), evaluation, and real‑time scoring via HTTP/gRPC APIs. The system includes:

- **Roles & permissions:** Anonymous user (limited demo view), Analyst, Data Engineer, ML Engineer, Product Manager, Administrator.  
- **Data:** internal company events (logs, transactions, CRM) and synthetic/public datasets for development and demos.  
- **Models:**  
  - Churn: Logistic Regression (L1/L2) with probability calibration (Platt/Isotonic) and **survival** models (Kaplan–Meier, Cox PH) for time‑to‑churn.  
  - CLV: **BG/NBD** (frequency/recency) + **Gamma–Gamma** for monetization; prediction intervals.  
- **Evaluation & interpretability:** AUC/PR‑AUC, Brier Score, calibration curves, sMAPE/MASE for CLV, SHAP‑like permutation importances, partial dependence.  
- **Infra/stack (Go):** `gonum/{stat,mat,distuv}`, `chi`/`echo` (HTTP), `grpc`, Postgres/Parquet, Kafka/NATS for streaming, `asynq` for batch jobs, `ristretto` for caching.  
- **Frontend:** web client (Go `html/template` or React/Vue) for dashboards, segments, experiments, and monitoring.  
- **Real time:** WebSocket/SSE for drift, throughput, and experiment monitoring.  

---

## 2. Main Use Cases / Scenarios

| Use case name | Brief Descriptions | Actors Involved |
|---|---|---|
| 2.1. Onboard Data Sources | Data Engineer registers sources (DB, S3, Kafka), defines schemas and refresh/anonymization policies. | Data Engineer, Administrator |
| 2.2. Feature Engineering | Build feature pipelines (RFM, trends, seasonality, lags), version and validate. | Data Engineer, ML Engineer |
| 2.3. Train Churn Model | Train logistic regression and/or Cox PH; automatic calibration; store artifacts (model + metadata). | ML Engineer |
| 2.4. Train CLV Model | Train BG/NBD + Gamma–Gamma; backtesting; hyperparameter selection. | ML Engineer |
| 2.5. Evaluate & Compare | Compare experiments/versions by AUC, Brier, calibration slope, MAPE/Pinball loss (CLV). | Analyst, ML Engineer |
| 2.6. Score in Batch | Scheduled batch scoring of the full customer base; persist to `scores` table for BI/CRM. | Analyst, Product Manager |
| 2.7. Real‑time Scoring API | Online scoring for a given customer/event with <50 ms latency; caching and rate limiting. | External Services, Product Manager |
| 2.8. Segmentation & Targeting | Build segments by p(churn), CLV, risk and expected profit; export to campaigns. | Analyst, Product Manager |
| 2.9. A/B/n Experiments | Define campaigns, collect events and run Bayesian analysis (probability of superiority, expected loss). | Analyst, Product Manager |
| 2.10. Monitoring & Drift | Monitor input features (PSI/JS divergence), calibration and metric stability; alerts. | ML Engineer, Administrator |
| 2.11. Access Control | Manage roles/permissions, audit logs, keys and API tokens. | Administrator |

---

## 3. Main Views (Frontend)

| View name | Brief Descriptions | URI |
|---|---|---|
| 3.1. Home | Intro, documentation, demo stats (anonymous). | `/` |
| 3.2. Dashboard | KPIs: churn rate, CLV distributions, calibration, pipeline health. | `/dashboard` |
| 3.3. Customers | Search by customer; detail with churn probability, forecast CLV, interpretations. | `/customers` |
| 3.4. Segments | Configure segments and export to campaigns. | `/segments` |
| 3.5. Experiments | Manage A/B/n; posterior comparisons, stopping decision rules. | `/experiments` |
| 3.6. Models | List and detail models (version, metrics, calibration, feature versions). | `/models` |
| 3.7. Data Sources | Register/status of sources, schema drift, last ingest. | `/data-sources` |
| 3.8. Pipelines | Graph of feature/score jobs, history, latency, retries. | `/pipelines` |
| 3.9. Monitoring | Drift, alerts, quantiles, scoring stability, latency/throughput. | `/monitoring` |
| 3.10. Admin | Roles, keys, audit logs, configurations. | `/admin` |

---

## 4. API Resources (Backend)

| Resource | Brief Descriptions | URI |
|---|---|---|
| 4.1. Auth: Login/Logout | Obtain/invalidate tokens (JWT/opaque). | `/api/login`, `/api/logout` |
| 4.2. Predict Churn | Input: `customer_id` or features; output: `p_churn`, calibrated probability, horizon. | `/api/predict/churn` |
| 4.3. Predict CLV | Input: transaction history/features; output: expected CLV + confidence intervals. | `/api/predict/clv` |
| 4.4. Batch Scoring Jobs | Create/ingest batch jobs, status, results (S3/Parquet). | `/api/jobs/score` |
| 4.5. Training Jobs | Start training (churn/CLV), select feature versions; status. | `/api/jobs/train` |
| 4.6. Experiments | CRUD experiments, record observations, retrieve posteriors and decision rules. | `/api/experiments` |
| 4.7. Data Sources | CRUD sources, connectivity test, schema introspection. | `/api/data-sources` |
| 4.8. Models & Versions | List/detail, metrics, calibration, artifacts for download/deploy. | `/api/models` |
| 4.9. Monitoring | Metrics, drift (PSI/JS), alerts, threshold configuration. | `/api/monitoring` |
| 4.10. Users & Roles | CRUD users/keys; roles (Analyst, DataEngineer, MLEngineer, Product, Admin). | `/api/users` |

---

## 5. Technology Stack & Architecture

- **Backend (Go):** `go1.22+`, `chi`/`echo`, `grpc`, `protobuf`, `gonum`, `asynq`, `ristretto`, `viper`, `zap`.  
- **Data:** Postgres (OLTP), Parquet/Arrow (data lake), Kafka/NATS (stream), Redis (cache/queue).  
- **MLOps:** versioning for models and features, artifacts in S3; canary/shadow deploy; automated backtests.  
- **Security:** RBAC, JWT, rate limiting, audit logs, PII masking/k‑anonymization.  
- **Deploy:** Docker, CI/CD (GitHub Actions), Helm/ArgoCD (optional), observability (Prometheus, OpenTelemetry).

---

## 6. Data Model (high level)

- `customers(customer_id, segment, created_at, ...)`  
- `events(customer_id, event_ts, type, props JSONB)`  
- `transactions(customer_id, txn_ts, amount, channel, ...)`  
- `features(customer_id, version, vector, ts)`  
- `scores(customer_id, model_id, p_churn, clv_mean, clv_p05, clv_p95, ts)`  
- `models(model_id, type, version, metrics JSONB, created_at)`  
- `experiments(exp_id, name, variants, status, created_at)`  
- `observations(exp_id, variant, ts, y, amount)`

---

## 7. Non‑functional Requirements

- **Latency:** p95 < 50 ms for `/predict/*` with cached feature set.  
- **Throughput:** 1k RPS with horizontal scaling.  
- **Reliability:** ≥ 99.9% for core APIs; idempotent job endpoints.  
- **Privacy & Compliance:** minimize PII, audit trails, configurable data retention.  
- **Observability:** metrics, logs, tracing; SLO/SLA dashboards.

---

## 8. Risks & Mitigations

- **Data drift / covariate shift:** automated PSI/JS checks and alerts; fast retraining.  
- **Probability calibration:** periodic reliability diagrams and recalibration.  
- **Leakage/overfitting:** temporal CV, strict cut‑offs, isolated train/test periods.  
- **PII & security:** minimization/masking, tokenization, least privilege, audit.

---

## 9. Roadmap (MVP → v1.0)

1) MVP: ingest + logistic regression (churn) + BG/NBD (CLV) + basic Dashboard and `/predict/*`.  
2) v0.9: Cox PH, calibration, batch scoring jobs, drift monitoring.  
3) v1.0: A/B/n module, policy simulation for campaigns, export connectors (CRM/ESP).

---

## 10. References (mathematical cores)

- Kaplan–Meier, Cox Proportional Hazards; BG/NBD and Gamma–Gamma CLV models; Brier Score, calibration, PR‑AUC; PSI/JS for drift. (The description follows the structure of the “Course Go Project Summary” template.)
