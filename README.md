# GateKeeper

<div align="center">
  <img src="./assets/logo.png" alt="GateKeeper Logo" width="200"/>
</div>

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)
![Redis](https://img.shields.io/badge/Redis-7.0+-DC382D?style=for-the-badge&logo=redis)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=for-the-badge&logo=postgresql)
![Next.js](https://img.shields.io/badge/Next.js-15-000000?style=for-the-badge&logo=next.js)
![NATS](https://img.shields.io/badge/NATS-JetStream-27AE60?style=for-the-badge&logo=nats)
![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Tracing-000000?style=for-the-badge&logo=opentelemetry)
![Jaeger](https://img.shields.io/badge/Jaeger-Tracing-60C0A8?style=for-the-badge&logo=jaeger)
![Docker](https://img.shields.io/badge/Docker-Containers-2496ED?style=for-the-badge&logo=docker)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Orchestration-326CE5?style=for-the-badge&logo=kubernetes)
![Terraform](https://img.shields.io/badge/Terraform-IaC-7B42BC?style=for-the-badge&logo=terraform)
![AWS EKS](https://img.shields.io/badge/AWS-EKS-232F3E?style=for-the-badge&logo=amazon-aws)

**GateKeeper** is an API Gateway and intelligent rate limiter purpose-built for LLM traffic. It provides multi-algorithm token-aware rate limiting, multi-provider failover routing, semantic caching, asynchronous cost tracking, and comprehensive observability.

### Admin Dashboard Overview
<div align="center">
  <img src="./assets/dashboard_screenshot_1.png" alt="Dashboard Main View" width="800" style="margin-bottom: 20px;"/>
  <br />
  <img src="./assets/dashboard_screenshot_2.png" alt="Dashboard Charts View" width="800"/>
</div>

---

## 🎯 Problems This Solves

Generic rate limiters count requests, not tokens, leaving LLM providers vulnerable to token-blind bursts that blow through budgets. **GateKeeper** solves this by estimating tokens before forwarding requests, atomically reserving quota in Redis, and reconciling actual usage post-response. It also mitigates provider downtime with automatic failover (e.g., Gemini → Anthropic) and reduces LLM spend via semantic vector caching of identical queries.

---

## 🏗 Architecture

```mermaid
graph TD
    Client[Client Applications] -->|HTTPS| LB[Load Balancer / K8s Ingress]
    LB --> GW1[Gateway Node 1]
    LB --> GW2[Gateway Node 2]
    
    subgraph EKS Cluster
    GW1
    GW2
    end
    
    GW1 <-->|Lua Scripts| Redis[(Redis: Rate Limits)]
    GW1 <-->|Vector Search| Cache[(Redis Stack: Cache)]
    GW1 -->|Async Events| NATS[NATS JetStream]
    GW1 -->|OTLP Traces| Jaeger[Jaeger Distributed Tracing]
    
    GW1 -->|LLM Calls| Adapters[Provider Adapters]
    Adapters --> Gemini[Gemini API]
    Adapters --> OpenAI[OpenAI API]
    
    NATS --> Consumer[Usage Consumer Service]
    Consumer --> Postgres[(PostgreSQL)]
    
    Admin[Next.js Admin Dashboard] --> Postgres
```

---

## 🚀 Features

### 1. Multi-Algorithm Token Limiter
Four distinct rate limiting algorithms, configurable per tenant:
- **Token Bucket:** Smooth bursts, ideal for LLM traffic (Default).
- **Sliding Window Log:** Exact tracking, higher memory cost.
- **Sliding Window Counter:** Approximate tracking, memory-efficient.
- **Fixed Window:** Simple but vulnerable to boundary bursts.

### 2. Multi-Provider Routing & Circuit Breaker
Seamless routing across Google Gemini, OpenAI, and Anthropic. Key pooling multiplies throughput, and built-in circuit breakers instantly failover to fallback providers during outages.

### 3. Semantic Caching
Using Redis Stack's Vector Search (FT.SEARCH), repeated prompts with high cosine similarity bypass the LLM entirely, yielding `<5ms` responses and saving 100% of the token cost.

### 4. Async Usage Pipeline
The hot path is decoupled from analytics. Usage events are non-blocking, published to NATS JetStream, and persisted to PostgreSQL by a dedicated consumer service.

---

## 💻 Tech Stack & Design Decisions

| Component | Choice | Rationale |
| :--- | :--- | :--- |
| **Core Gateway** | Go 1.23 | Unmatched concurrency (Goroutines) and low latency. Standard for infra tooling. |
| **State Store** | Redis + Lua | Atomic `EVAL` scripts prevent race conditions in distributed token decrements. |
| **Message Queue** | NATS JetStream | Ultra-lightweight asynchronous event streaming. Offloads Postgres writes from the hot path. |
| **Semantic Cache** | Redis Stack | Built-in Vector similarity search using Cosine distance. |
| **Dashboard** | Next.js 15 | Modern, React-based admin panel with TailwindCSS and Recharts for live telemetry. |

---

## 📈 Performance Benchmarks

GateKeeper aims to be virtually invisible in your network hop.

* **Target Overhead:** `<10ms` p50, `<30ms` p99 latency overhead.
* **Throughput:** Tested at `>5,000 RPS` locally via `k6`. Adding gateway nodes linearly scales capacity because the system is fully stateless.

To run the load test:
```bash
k6 run load-tests/script.js
```

---

## 🛠 Getting Started

### 1. Prerequisites
- Docker & Docker Compose
- Node.js (for Next.js dashboard)
- Go 1.23+

### 2. Launch Infrastructure
```bash
docker-compose -f deploy/docker-compose.yml up -d
```
This spins up Redis Stack, PostgreSQL, NATS, and a mock LLM server.

### 3. Run the Gateway
```bash
go mod tidy
go run cmd/gateway/main.go
```

### 4. Run the Usage Consumer
```bash
go run cmd/usage-consumer/main.go
```

### 5. Launch the Admin Dashboard
```bash
cd dashboard
npm install
npm run dev
```
Visit `http://localhost:3000` to view live usage telemetry.

---
*Built as a production-grade infrastructure showcase.*
