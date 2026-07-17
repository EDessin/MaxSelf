# MaxSelf

Level up your health. Maximize your life.

MaxSelf is a game app for becoming your healthiest self. The core idea is simple:
turn daily healthy actions into experience points, levels, streaks, and visible
progress.

## Game Concept

Players earn Health XP by completing real-life actions such as:

- Moving your body through workouts, walks, stretching, or sports
- Eating nourishing meals and drinking enough water
- Sleeping well and keeping a consistent bedtime
- Practicing recovery, mindfulness, and stress management
- Building small habits that compound over time

As players collect XP, they level up their MaxSelf profile, unlock milestones,
and see their progress across core health stats.

## Core Stats

- **Strength**: exercise, resistance training, mobility
- **Fuel**: nutrition, hydration, meal quality
- **Recovery**: sleep, rest, stretching, relaxation
- **Mindset**: meditation, reflection, mood check-ins
- **Consistency**: streaks, routines, and long-term habit building

## First MVP

The first version should focus on a satisfying daily loop:

1. Add or complete a healthy action.
2. Earn Health XP.
3. Watch your level, streaks, and stats grow.
4. Review daily and weekly progress.

## Product Direction

MaxSelf should feel motivating without becoming punishing. The goal is not
perfect optimization, but steady progress: small wins, visible growth, and a
game layer that makes caring for yourself feel rewarding.

## MVP Architecture

MaxSelf now starts as a web app with a Go microservice backend, an Angular
frontend, and PostgreSQL persistence.

```text
Angular + PrimeNG frontend
        |
        | REST
        v
API facade service
        |
        | REST
        v
Identity service  Activity service  Progress service
        \              |              /
         \             v             /
              PostgreSQL via GORM
```

The Go services follow a hexagonal-style structure:

- `domain`: business concepts and rules
- `application`: use cases and ports
- `adapters/inbound/rest`: HTTP handlers
- `adapters/outbound/postgres`: GORM repositories
- `cmd/<service>`: service entrypoints

## Current MVP Slice

The first playable slice includes:

- email/password registration and login
- Google OAuth login endpoints ready for configuration
- JWT-based API access
- activity logging
- XP rewards per activity type
- level calculation
- stat progress tracking
- streak tracking
- colorful Angular dashboard with PrimeNG and Lucide icons
- PostgreSQL persistence through GORM
- local development with Postgres in Docker and Go/Angular run from source
- Docker Compose for deployment-style container checks

## Services

| Service | Port | Responsibility |
| --- | ---: | --- |
| API facade | `8080` | Frontend-facing REST API and service orchestration |
| Identity | `8081` | Users, login, JWTs, Google OAuth callback |
| Activity | `8082` | Health activity types and activity logs |
| Progress | `8083` | XP, levels, stats, and streaks |
| Web | `4201` | Angular + PrimeNG frontend |
| PostgreSQL | `5432` | Persistent app data |

## Local Development

Copy the environment example if you want to customize secrets. The local
`make dev` and `make services` scripts read this `.env` file automatically:

```bash
cp .env.example .env
```

For the normal local development loop, use the Makefile. It starts only
PostgreSQL in Docker, then runs the Go services and Angular frontend directly
from your local source tree:

```bash
make dev
```

Open the frontend:

```text
http://localhost:4201
```

Useful local targets:

```bash
make db-up       # start only PostgreSQL in Docker
make services    # run identity, activity, progress, and api with go run
make web         # run Angular on http://localhost:4201
make db-down     # stop and remove the local PostgreSQL container
make db-reset    # recreate the local database from scratch
make test        # run backend tests
make build-web   # build Angular
```

The local service ports are:

```text
API facade:  http://localhost:8080
Identity:    http://localhost:8081
Activity:    http://localhost:8082
Progress:    http://localhost:8083
Web:         http://localhost:4201
PostgreSQL:  localhost:5432
```

Docker Compose is still available for container-oriented verification:

```bash
docker compose config --quiet
```

## Google Login

Google login is optional for local development. To enable it:

1. Create a Google Cloud OAuth client.
2. Use basic scopes: `openid`, `email`, `profile`.
3. Add this local redirect URL:

```text
http://localhost:8081/auth/google/callback
```

4. Set:

```bash
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URL=http://localhost:8081/auth/google/callback
```

If you already had `make dev` or `make services` running, restart it after
editing `.env`; running services do not reload environment variables.

MaxSelf uses Google only to prove identity. The app still issues its own JWT
for MaxSelf API access.

## Verification

Backend:

```bash
go test ./...
```

Frontend:

```bash
cd web
npm run build
```
