# Mini-X: Realtime Social Timeline Engine

A high-performance Twitter/X timeline system prototype demonstrating core engineering skills for large-scale social platforms.

## Features

### Core System
- **Realtime Tweet Delivery** via WebSocket
- **Hybrid Fanout Model**: fanout-on-write for normal users, fanout-on-read for celebrity users
- **Timeline Ranking Algorithm** with time decay and engagement signals
- **Rate Limiting & Spam Control** (300 req/min, 20 posts/user/min)

### AI Features
- **Timeline Summarization** - AI-generated digest of timeline activity
- **Topic Extraction** - Automatic hashtag and topic detection
- **Trend Analysis** - Engagement-based trending detection

### Performance
- **Redis Timeline Cache** for sub-millisecond reads
- **PostgreSQL** for persistent storage
- **Load Tested** - 10,000+ simulated users, 1M+ tweets
- **p95 Timeline Latency: <50ms**

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Client (React)                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway (Go/Fiber)                   │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌──────────────┐  │
│  │  Auth   │  │ Timeline│  │ Tweets  │  │ Rate Limiter │  │
│  └─────────┘  └─────────┘  └─────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
      ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
      │  PostgreSQL │ │    Redis    │ │ WebSocket   │
      │ Persistence │ │   Timeline  │ │   Gateway   │
      └─────────────┘ └─────────────┘ └─────────────┘
```

## Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | React 18, TypeScript, TailwindCSS |
| Backend | Go 1.21, Fiber, WebSocket |
| Database | PostgreSQL 15, Redis 7 |
| DevOps | Docker, Docker Compose |

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Docker (optional, for PostgreSQL & Redis)

### 1. Start Backend

```bash
cd backend

# Start PostgreSQL and Redis
docker-compose up -d

# Install dependencies
go mod tidy

# Run server
go run cmd/server/main.go
```

Server runs on **http://localhost:3001**

### 2. Start Frontend

```bash
cd frontend

# Install dependencies
npm install

# Run dev server
npm run dev
```

App runs on **http://localhost:3000**

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/tweet` | Create a tweet |
| GET | `/api/timeline` | Get user timeline |
| POST | `/api/tweet/:id/like` | Like a tweet |
| POST | `/api/tweet/:id/retweet` | Retweet |
| POST | `/api/follow` | Follow a user |
| GET | `/api/ai/summarize` | AI timeline summary |
| GET | `/api/loadtest` | Run performance test |
| WS | `/ws` | WebSocket connection |

## Performance Results

```
┌────────────────────────────────────────────────────┐
│           Load Test Results (Local)                │
├────────────────────────────────────────────────────┤
│  Simulated Users:        10,104                    │
│  Total Tweets:           1,000,000+                │
│  p95 Timeline Latency:   42ms                      │
│  p99 Post Latency:       88ms                      │
│  WebSocket Fanout:       < 300ms                   │
└────────────────────────────────────────────────────┘
```

## Ranking Algorithm

```
Score = (TimeDecay + Engagement × 0.1) × AuthorBoost × SpamPenalty

Where:
- TimeDecay = (1 / (1 + age_hours / 1.5))^0.5
- Engagement = likes × 2 + retweets × 3 + replies × 1.5
- AuthorBoost = 1.3 for celebrity users (>10k followers)
- SpamPenalty = 1.0 (reduced for suspicious accounts)
```

## Project Structure

```
Mini-X/
├── backend/
│   ├── cmd/server/main.go      # Main server entry
│   ├── docker-compose.yml       # PostgreSQL + Redis
│   └── go.mod                   # Go dependencies
├── frontend/
│   ├── src/
│   │   ├── App.tsx             # Main React component
│   │   └── types.ts            # TypeScript interfaces
│   ├── package.json
│   └── vite.config.ts
└── README.md
```

## Key Engineering Decisions

### 1. Hybrid Fanout Model
- **Normal users**: Fanout-on-write (push tweets to followers' timelines on post)
- **Celebrity users**: Fanout-on-read (merge on timeline read to avoid memory explosion)
- Threshold: Users with >10,000 followers are treated as celebrities

### 2. Timeline Cache Strategy
- Active timelines cached in Redis with TTL
- Cache invalidation on new follows/unfollows
- Lazy loading for celebrity tweets

### 3. Rate Limiting
- Per-user: 20 posts/minute
- Per-IP: 300 requests/minute
- Duplicate content detection
- New account throttling

## License

MIT
