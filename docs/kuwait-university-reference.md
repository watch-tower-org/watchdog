# Kuwait University Backend — Reference Patterns

This document captures the core architectural patterns from the Kuwait University
backend (the `ku.edu/shared` module + the `services/general` service) that we
mirror in WatchTower. Each section describes the pattern and shows the canonical
example so WatchTower features are built 1:1 the same way.

Stack used throughout: **Go**, **gin**, **uptrace/bun**, **zerolog**,
**golang-jwt/v5**, **golang.org/x/crypto/bcrypt**, **go-playground/validator**,
**joho/godotenv**, **gopkg.in/mail.v2**.

---

## 1. Bootstrap flow (`main.go`)

Order of operations at process start, never deviated from:

1. `config.LoadConfig()` — load env vars.
2. `logger.InitLogger(cfg.LoggerConfig)` — init zerolog.
3. `database.New(cfg.Database)` — connect to Postgres (auto-creates the DB if
   missing, auto-migrates models), `defer db.Close()`.
4. `pkg.NewApplication(cfg, db)` — build the app (wire all controllers, seed
   defaults). Fatal on error.
5. `router.AppRouter(app)` — build the gin engine and register all routes.
6. Start `http.Server` in a goroutine.
7. Block on `SIGINT` / `SIGTERM`, then graceful shutdown with a 30s timeout,
   calling `app.Shutdown()` and `logger.Close()`.

```go
func main() {
    cfg := config.LoadConfig()
    logger.InitLogger(cfg.LoggerConfig)

    db, err := database.New(cfg.Database)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to connect to the database")
    }
    defer db.Close()

    app, err := pkg.NewApplication(cfg, db)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to initialize the application")
    }

    r, err := router.AppRouter(app)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to initialize the router")
    }

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    srv := &http.Server{Addr: ":" + cfg.Server.Port, Handler: r}
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal().Err(err).Msg("Failed to start the server")
        }
    }()

    <-quit
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    done := make(chan struct{})
    go func() {
        srv.Shutdown(shutdownCtx)
        app.Shutdown()
        close(done)
    }()

    select {
    case <-done:
        logger.Close()
    case <-shutdownCtx.Done():
        logger.Close()
    }
}
```

---

## 2. Application wiring (`pkg/app.go`)

- `Application` is a struct holding `Config` plus every feature **Controller**
  (and any cross-cutting service).
- `NewApplication(cfg, db)` creates the app, runs `init...` steps in order, then
  `initDefaults()`.
- `initControllers(db)` constructs each controller, passing its dependencies
  (db, other controllers, config pointers).
- `initDefaults()` seeds required rows (default settings, default roles, default
  admin) — idempotent (checks existence first).
- `Shutdown()` flushes/waits for background services and closes connections.

```go
type Application struct {
    Config *config.Config

    EmailC *email.Controller
    RoleC  *role.Controller
    UserC  *backoffice_user.Controller
    AuthC  *auth.Controller
}

func NewApplication(cfg *config.Config, db *database.Database) (*Application, error) {
    app := &Application{Config: cfg}
    app.initControllers(db)
    if err := app.initDefaults(); err != nil {
        return nil, fmt.Errorf("init defaults: %w", err)
    }
    return app, nil
}

func (app *Application) initControllers(db *database.Database) {
    app.EmailC = email.NewController(db.DB)
    app.RoleC = role.NewController(db.DB, ...)
    app.UserC = backoffice_user.NewController(db.DB, app.RoleC, ...)
    app.AuthC = auth.NewController(db.DB, &app.Config.JWT, app.UserC, ...)
}

func (app *Application) initDefaults() error {
    if err := app.EmailC.CreateDefaultEmailSettings(); err != nil {
        return err
    }
    if err := app.RoleC.CreateDefaultRoles(); err != nil {
        return err
    }
    if err := app.UserC.CreateDefaultSuperAdmin(); err != nil {
        return err
    }
    return nil
}
```

---

## 3. Feature package anatomy (`pkg/<feature>/`)

Every feature is a package with three files: `controller.go`, `handler.go`,
`router.go`. (Helpers like `methods.go` may be added for pure functions.)

### `controller.go`

- `Controller` struct holds dependencies: `db *bun.DB`, other controllers,
  config pointers, caches.
- `NewController(...) *Controller` constructor.
- Business logic + DB access. Returns plain `error`.
- **Sentinel errors** for user-facing failures; everything else is logged and
  mapped to a generic message.

```go
type Controller struct {
    db *bun.DB
}

func NewController(db *bun.DB) *Controller {
    return &Controller{db: db}
}

func (c *Controller) Get(ctx context.Context, id int64) (*model.X, error) {
    var x model.X
    err := c.db.NewSelect().Model(&x).Where("id = ?", id).Scan(ctx)
    if err != nil {
        logger.Ctx(ctx).Error().Msgf("get X id=%d: %v", id, err)
        return nil, errors.New("Failed to retrieve.")
    }
    return &x, nil
}
```

### `handler.go`

- `Handler` struct wraps the `*Controller`.
- `NewHandler(controller *Controller) *Handler`.
- Thin: bind body → validate → sanitize → call controller → respond with `res`.
- Every controller error is mapped through a `res.*` helper.

```go
type Handler struct {
    controller *Controller
}

func NewHandler(controller *Controller) *Handler {
    return &Handler{controller: controller}
}

func (h *Handler) Update(c *gin.Context) {
    var req model.UpdateXRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        res.BadRequest(c, "Invalid request body")
        return
    }
    if err := validator.Validate(&req); err != nil {
        res.BadRequestValidation(c, err)
        return
    }

    x, err := h.controller.Update(c.Request.Context(), &req)
    if err != nil {
        res.BadRequest(c, err.Error())
        return
    }
    res.Ok(c, "X updated successfully", x)
}
```

### `router.go`

- `Router(r *gin.RouterGroup, controller *Controller, ...)` — signature takes
  the group and whatever dependencies the routes need (auth config, etc.).
- Creates the handler, defines the group + middleware, registers routes.

```go
func Router(r *gin.RouterGroup, controller *Controller, cfg *config.JWTConfig) {
    h := NewHandler(controller)

    grp := r.Group("/x")
    grp.Use(middleware.AuthMiddleware(cfg))
    grp.GET("", h.List)
    grp.POST("", h.Create)
}
```

---

## 4. Router wiring (`router/app-router.go`)

- Single place that builds the gin engine and registers every feature router.
- Global middleware stack: `Logger`, `Recovery`, `CORS`, `RateLimit`, `Gzip`,
  `SecurityHeaders`.
- `NoRoute` fallback using `res.NotFound`.
- Versioned group, e.g. `/api/<service>/v1`; each feature registers on it.

```go
func AppRouter(app *pkg.Application) (*gin.Engine, error) {
    gin.SetMode(app.Config.Server.Mode)
    router := gin.New()
    router.Use(middleware.Logger())
    router.Use(middleware.Recovery())
    router.Use(middleware.CORS(&app.Config.CORS))
    router.Use(middleware.RateLimit(&app.Config.RateLimit))
    router.Use(middleware.Gzip())

    router.NoRoute(func(c *gin.Context) {
        res.NotFound(c, "The requested resource was not found")
    })

    v1 := router.Group("/api/watchtower/v1")
    {
        auth.Router(v1, app.AuthC, &app.Config.JWT)
        settings.Router(v1, app.SettingsC, &app.Config.JWT)
        recipient_lists.Router(v1, app.RecipientListC, &app.Config.JWT)
        // ... every feature
    }
    return router, nil
}
```

---

## 5. Response envelope (`internal/res`)

Every response uses one of two shapes; helpers set status + body:

```go
type SuccessResponse struct {
    Success bool        `json:"success"`
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

type ErrorsResponse struct {
    Success bool   `json:"success"`
    Code    int    `json:"code"`
    Error   string `json:"error,omitempty"`
}
```

Helpers: `Ok`, `OkPagination`, `Created`, `NoContent`, `BadRequest`,
`BadRequestValidation`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`,
`TooManyRequests`, `InternalServerError`.

Example usage:
```go
res.Ok(c, "email settings retrieved successfully", settings)
res.BadRequest(c, "Invalid request body")
res.NotFound(c, "The requested resource was not found")
res.OkPagination(c, "issues retrieved successfully", issues, &pageInfo)
```

---

## 6. Configuration (`config.LoadConfig`)

- Env-driven, `.env` loaded via `godotenv.Load()`.
- Typed nested structs (`Server`, `Database`, `JWT`, `RateLimit`, `CORS`,
  `LoggerConfig`, ...).
- Helpers: `getEnv`, `getEnvInt`, `getEnvDuration`, `getEnvSlice` — each prints
  the key when a default is used and returns the default on empty/missing.

```go
type DatabaseConfig struct {
    Host            string
    Port            string
    User            string
    Password        string
    Name            string
    SSLMode         string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}
```

WatchTower adds an admin section for env-based login:
```go
type AdminConfig struct {
    Username string
    Password string
}
```

---

## 7. Database (`internal/database`)

- bun over `pgdriver` (`sql.OpenDB(pgdriver.NewConnector(...))`).
- DSN: `postgres://user:pass@host:port/db?sslmode=...`.
- Pool settings: `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`.
- **Auto-creates the database** if missing: connect to the `postgres` admin DB
  and run `CREATE DATABASE <name>`, then reconnect.
- **Auto-migrates** from models: `db.RegisterModel(...)` for every model, then
  `NewCreateTable().Model(i).IfNotExists().Exec(ctx)` per model, plus idempotent
  raw-SQL index/column statements.

```go
func New(cfg config.DatabaseConfig) (*Database, error) {
    dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
        cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
    sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
    sqldb.SetMaxOpenConns(cfg.MaxOpenConns)
    sqldb.SetMaxIdleConns(cfg.MaxIdleConns)
    sqldb.SetConnMaxLifetime(cfg.ConnMaxLifetime)
    db := bun.NewDB(sqldb, pgdialect.New())

    if err := AutoMigration(db, ctx); err != nil {
        return nil, err
    }
    return &Database{DB: db}, nil
}
```

---

## 8. Models & bun tags (`internal/model`)

- Embed `bun.BaseModel` with `bun:"table:<name>,alias:<alias>"`.
- Field tags: `pk,autoincrement`, `notnull`, `unique`, `default:...`,
  `nullzero`, `rel:belongs-to,join:...`.
- `json` tags for the API; secrets use `json:"-"`.
- **Request and response structs live in the same file** as the model.
- Validation via `validate:"required,email,min=8"` tags.
- **Partial updates use pointer fields** (`*string`, `*int`) so callers can tell
  "not provided" from "zero value".

```go
type GeneralEmailSettings struct {
    bun.BaseModel `bun:"table:general_email_settings,alias:es"`

    ID        int64     `bun:"id,pk,autoincrement" json:"id"`
    Host      string    `bun:"host,notnull" json:"host"`
    Port      int       `bun:"port,notnull" json:"port"`
    Password  string    `bun:"password,notnull" json:"-"`
    CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
    UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
}

type UpdateEmailSettingsRequest struct {
    Host     *string `json:"host,omitempty"`
    Port     *int    `json:"port,omitempty"`
    Password *string `json:"password,omitempty"`
}
```

Partial update pattern:
```go
if req.Host != nil {
    settings.Host = *req.Host
}
settings.UpdatedAt = time.Now()
_, err = c.db.NewUpdate().Model(settings).Where("id = ?", settings.ID).Exec(ctx)
```

---

## 9. Auth (JWT + bcrypt)

- Passwords hashed with `bcrypt.GenerateFromPassword(..., bcrypt.DefaultCost)`;
  verified with `bcrypt.CompareHashAndPassword`.
- JWT signed HS256 with `golang-jwt/v5`; custom `Claims` struct embeds
  `jwt.RegisteredClaims` and carries `Id`, `Role`, `Username`, `Email`.
- **Two tokens**: access (short-lived) + refresh (long-lived), each signed with
  its own secret.
- `AuthMiddleware(cfg)` reads `Authorization: Bearer <token>`, validates,
  sets `c.Set("user_id"|"role"|"username"|"email", ...)`.

```go
func (c *Controller) generateToken(user *model.User, key string, duration time.Duration) (string, error) {
    claims := &model.Claims{
        Id:       user.ID,
        Username: user.Username,
        Email:    user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            ID:        uuid.New().String(),
            Issuer:    "watchtower",
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Subject:   fmt.Sprintf("%d", user.ID),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(key))
}
```

Login flow: fetch user → check active → `bcrypt` password → generate access +
refresh → return. Errors: sentinel `ErrInvalidCredentials` / `ErrAccountDisabled`
surfaced as 401; everything else logged + generic.

```go
grp.POST("/login", h.Login)
grp.POST("/refresh-token", h.RefreshToken)
grp.POST("/logout", middleware.AuthMiddleware(cfg), h.Logout)
```

---

## 10. Backoffice patterns

### Pagination (`model.PageInfo`)
```go
type PageInfo struct {
    CurrentPage     int  `json:"current_page"`
    Limit           int  `json:"limit"`
    Total           int  `json:"total"`
    TotalPages      int  `json:"total_pages"`
    HasNextPage     bool `json:"has_next_page"`
    HasPreviousPage bool `json:"has_previous_page"`
}
```

List pattern (controller): default page size, build a `countQ` for the total and
a `q` for rows, apply filters to both, then:
```go
totalPages := (count + pageSize - 1) / pageSize
pageInfo := &model.PageInfo{CurrentPage: page, Limit: pageSize, Total: count,
    TotalPages: totalPages, HasNextPage: page < totalPages, HasPreviousPage: page > 1}
err := q.Order("id ASC").Limit(pageSize).Offset((page-1)*pageSize).Scan(ctx, &rows)
```
Handler responds with `res.OkPagination`.

### Search / filter
`ILIKE '%term%'` across searchable columns, applied to both `q` and `countQ`;
exact-match filters via `Where("col = ?", value)`; switch-based status filters.

### Soft delete
Columns `is_deleted`, `is_active`, `deleted_at`. On delete: set flags, mangle
unique fields (`email = email_deleted_<ts>`), never hard-delete.

### Protected rows
`is_protected` flag; protected rows reject update/delete.

### Default seeding
`CreateDefault...()` checks existence, then inserts (e.g. default settings,
default admin). Called from `initDefaults()`.

---

## 11. Email sending (`gopkg.in/mail.v2`)

```go
func sendMail(host string, port int, username, password, fromName, fromEmail, to, subject, body string) error {
    d := mail.NewDialer(host, port, username, password)
    d.SSL = port == 465
    d.Timeout = 10 * time.Second
    if !d.SSL {
        d.TLSConfig = &tls.Config{ServerName: host}
    }

    m := mail.NewMessage()
    m.SetHeader("From", formatFrom(fromName, fromEmail))
    m.SetHeader("To", to)
    m.SetHeader("Subject", subject)
    m.SetBody("text/plain", body)

    return d.DialAndSend(m)
}
```

---

## 12. Validation (`internal/validator`)

- Wraps go-playground/validator: `validator.Validate(&req)` for struct-tag
  validation; `validator.SanitizeString(s)` to clean user input; converts
  validation errors into a user-friendly message via `res.BadRequestValidation`.

---

## 13. Logging (zerolog)

- Global logger initialized once from config.
- Request-scoped: `logger.Ctx(ctx).Error().Msgf("...")` uses the request context.
- Levels: `Fatal`, `Error`, `Warn`, `Info`, `Debug`, `Trace`.
- Every controller error is logged before returning a generic/user-facing error.

---

## 14. Cache patterns (simplified for WatchTower)

KU uses `cache.Singleton[T]` (single settings row cached in memory) and
`cache.Map[K, V]`. Pattern: read hits cache; mutation invalidates and reloads;
cross-process invalidation via NATS (dropped in WatchTower — single instance, so
plain in-memory invalidation on write suffices).

---

## 15. WatchTower Adaptation Map

### Carries over as-is
- Bootstrap flow (`main.go`) — §1
- Application wiring (`pkg/app.go`) — §2
- Feature package anatomy (controller/handler/router) — §3
- Router wiring (`router/app-router.go`) — §4
- Response envelope (`internal/res`) — §5
- Config (`internal/config`) — §6
- Database (`internal/database`) — §7
- Models & bun tags (`internal/model`) — §8
- Pagination, search/filter, soft delete, default seeding — §10
- Email sending (`gopkg.in/mail.v2`) — §11
- Validation — §12
- Logging — §13

### Adapted for WatchTower
- **Auth (§9)**: single admin, **not** multi-user. Credentials come from env
  vars `ADMIN_USERNAME` / `ADMIN_PASSWORD` (deployment requirement). On boot,
  ensure the admin exists in the `settings` store; `POST /auth/login` validates
  the password with bcrypt and issues a JWT **access + refresh** pair;
  `AuthMiddleware` guards all backoffice routes. No roles, no RBAC, no user CRUD.
- **Cache (§14)**: plain in-memory cache + invalidate-on-write. No NATS pub/sub.

### Dropped entirely
- NATS / message broker transport.
- i18n + translation middleware.
- Role/RBAC module.
- Audit log service.
- Swagger/swaggo.
- Excel/PDF export.
- Multi-project / multi-tenant management (projects are just labels).
