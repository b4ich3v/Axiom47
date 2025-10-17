# Course_Go_Project_Summary

## Project name
Axiom47

## Project author
Име Фамилия (FN: XXXXX)

## № Names of Participants FN
1. Име 1 (FN: XXXXX)  
2. Име 2 (FN: XXXXX)  
3. Име 3 (FN: XXXXX)

---

## 1. Short project description (Business needs and system features)

В ерата на дигиталните услуги, фирмите се нуждаят от точни прогнози за **вероятност клиент да отпадне (churn)** и **очаквана пожизнена стойност (Customer Lifetime Value, CLV)**, за да планират бюджети за задържане, оферти и капацитет. Платформата **Churn & CLV Intelligence Platform** предоставя край-до-край решение за ingest на данни, моделиране (логистична регресия, survival анализ, BG/NBD + Gamma-Gamma), оценка и реално-времеви скоринг чрез HTTP/gRPC API. Системата включва:

- **Роли и права:** Анонимeн потребител (ограничена демо визуализация), Анализатор, Данни инженeр, ML инженер, Продуктов мениджър, Администратор.  
- **Данни:** вътрешнофирмени събития (логове, транзакции, CRM), както и синтетични/публични данни за разработка и демонстрации.  
- **Модели:**  
  - Churn: Логистична регресия (L1/L2) с калибрация (Platt/Isotonic) и **survival** (Kaplan–Meier, Cox PH) за време-до-отпадане.  
  - CLV: **BG/NBD** (frequency/recency) + **Gamma–Gamma** за монетизиране; прогнозни интервали.  
- **Оценка и интерпретируемост:** AUC/PR-AUC, Brier Score, калибрационни криви, sMAPE/MASE за CLV, SHAP-подобни пермутационни важности, partial dependence.  
- **Инфра/стек (Go):** `gonum/{stat,mat,distuv}`, `chi`/`echo` (HTTP), `grpc`, Postgres/Parquet, Kafka/NATS за стрийм, `asynq` за batch jobs, `ristretto` за кеш.  
- **Фронтенд:** уеб клиент (Go `html/template` или React/Vue) за табла, сегменти, експерименти и мониторинг.  
- **Реално време:** WebSocket/SSE за мониторинг на drift, throughput и експерименти.  

---

## 2. Main Use Cases / Scenarios

| Use case name | Brief Descriptions | Actors Involved |
|---|---|---|
| 2.1. Onboard Data Sources | Данни инженер регистрира източници (DB, S3, Kafka), дефинира схеми и политики за опресняване/анонимизация. | Data Engineer, Administrator |
| 2.2. Feature Engineering | Създаване на feature pipelines (RFM, тендениции, сезонност, лагове), версиониране и валидиране. | Data Engineer, ML Engineer |
| 2.3. Train Churn Model | Обучение на логистична регресия и/или Cox PH; автоматична калибрация; запис на артефакти (двойки: модел + метаданни). | ML Engineer |
| 2.4. Train CLV Model | Обучение на BG/NBD + Gamma–Gamma; оценка с backtesting; избор на хиперпараметри. | ML Engineer |
| 2.5. Evaluate & Compare | Сравнение на експерименти/версии по AUC, Brier, calibration slope, MAPE/Pinball loss (CLV). | Analyst, ML Engineer |
| 2.6. Score in Batch | Планиран batch скоринг на цял клиентски басейн; запис в таблица `scores` за BI/CRM. | Analyst, Product Manager |
| 2.7. Real‑time Scoring API | Онлайн скоринг за конкретен клиент/събитие с <50 ms латентност; кеш и rate‑limit. | External Services, Product Manager |
| 2.8. Segmentation & Targeting | Изграждане на сегменти по p(churn), CLV, риск и очаквана печалба; експорт към кампании. | Analyst, Product Manager |
| 2.9. A/B/n Experiments | Дефиниране на кампании, събиране на събития и Байесов анализ (вероятност за превъзходство, expected loss). | Analyst, Product Manager |
| 2.10. Monitoring & Drift | Наблюдение на входни фийчъри (PSI/JS divergence), калибрация и стабилност на метрики; аларми. | ML Engineer, Administrator |
| 2.11. Access Control | Управление на роли/права, одит логове, ключове и токени за API. | Administrator |

---

## 3. Main Views (Frontend)

| View name | Brief Descriptions | URI |
|---|---|---|
| 3.1. Home | Интро, документация, демо статистики (анонимно). | `/` |
| 3.2. Dashboard | KPI табло: churn rate, CLV дистрибуции, калибрация, здраве на pipeline-и. | `/dashboard` |
| 3.3. Customers | Търсене по клиент; детайл с вероятност за отпадане, прогнозен CLV, интерпретации. | `/customers` |
| 3.4. Segments | Конфигуриране на сегменти и износ към кампании. | `/segments` |
| 3.5. Experiments | Управление на A/B/n; постериорни сравнения, decision rules за стопиране. | `/experiments` |
| 3.6. Models | Списък и детайл на модели (версия, метрики, калибрация, фийчър версии). | `/models` |
| 3.7. Data Sources | Регистрация/състояние на източници, schema drift, последен ingest. | `/data-sources` |
| 3.8. Pipelines | Граф на feature/score jobs, история, латентност, retries. | `/pipelines` |
| 3.9. Monitoring | Drift, аларми, квантили, стабилност на скоринг, latency/throughput. | `/monitoring` |
| 3.10. Admin | Роли, ключове, одит логове, конфигурации. | `/admin` |

---

## 4. API Resources (Backend)

| Resource | Brief Descriptions | URI |
|---|---|---|
| 4.1. Auth: Login/Logout | Получаване/инвалидация на токени (JWT/opaque). | `/api/login`, `/api/logout` |
| 4.2. Predict Churn | Вход: `customer_id` или фийчъри; изход: `p_churn`, калибрирана вероятност, хоризонт. | `/api/predict/churn` |
| 4.3. Predict CLV | Вход: транзакционна история/фийчъри; изход: очакван CLV + доверителни интервали. | `/api/predict/clv` |
| 4.4. Batch Scoring Jobs | Създаване/ингест на batch задачи, статус, резултати (S3/Parquet). | `/api/jobs/score` |
| 4.5. Training Jobs | Стартиране на обучение (churn/CLV), избиране на фийчър версии; статус. | `/api/jobs/train` |
| 4.6. Experiments | CRUD на експерименти, запис на наблюдения, извличане на постериори и decision rules. | `/api/experiments` |
| 4.7. Data Sources | CRUD на източници, тест на свързаност, schema introspection. | `/api/data-sources` |
| 4.8. Models & Versions | Списък/детайл, метрики, калибрация, артефакти за сваляне/деплой. | `/api/models` |
| 4.9. Monitoring | Метрики, drift (PSI/JS), аларми, конфигурация на прагове. | `/api/monitoring` |
| 4.10. Users & Roles | CRUD на потребители/ключове; роли (Analyst, DataEngineer, MLEngineer, Product, Admin). | `/api/users` |

---

## 5. Technology Stack & Architecture

- **Backend (Go):** `go1.22+`, `chi`/`echo`, `grpc`, `protobuf`, `gonum`, `asynq`, `ristretto`, `viper`, `zap`.  
- **Data:** Postgres (OLTP), Parquet/Arrow (data lake), Kafka/NATS (stream), Redis (кеш/кю).  
- **MLOps:** версиониране на модели и фийчъри, артефакти в S3; canary/shadow deploy; автоматични backtests.  
- **Security:** RBAC, JWT, rate limiting, audit логове, PII маскиране/k-анонимизация.  
- **Deploy:** Docker, CI/CD (GitHub Actions), Helm/ArgoCD (опционално), observability (Prometheus, OpenTelemetry).

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

- **Latency:** p95 < 50 ms за `/predict/*` при кеширан фийчър сет.  
- **Throughput:** 1k RPS със скалиране по хоризонтала.  
- **Reliability:** ≥ 99.9% за core API; идемпотентни job endpoints.  
- **Privacy & Compliance:** минимизиране на PII, audit trails, конфигурируемо задържане на данни.  
- **Observability:** метрики, лога, трасиране; SLO/SLA табла.

---

## 8. Risks & Mitigations

- **Data drift / covariate shift:** автоматични PSI/JS контроли и аларми; бързо прetrain-ване.  
- **Калибрация на вероятности:** периодични reliability диаграми и recalibration.  
- **Leakage/overfitting:** temporal CV, строги cut‑offs, изолирани train/test периоди.  
- **PII & сигурност:** минимизиране/маскиране, токени, least privilege, одит.

---

## 9. Roadmap (MVP → v1.0)

1) MVP: ingest + логистична регресия (churn) + BG/NBD (CLV) + базов Dashboard и `/predict/*`.  
2) v0.9: Cox PH, калибрация, batch scoring jobs, drift мониторинг.  
3) v1.0: A/B/n модул, policy simulation за кампании, export конектори (CRM/ESP).

---

## 10. References (математически ядра)

- Kaplan–Meier, Cox Proportional Hazards; BG/NBD и Gamma–Gamma CLV модели; Brier Score, calibration, PR‑AUC; PSI/JS за drift. (Описанието следва общата структура на „Course Go Project Summary“ шаблон.)

