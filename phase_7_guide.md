# Phase 7 Implementation Guide: Application Container & Router Assembly

This guide walks you step-by-step through wiring the entire application together in **Phase 7**: creating the **Application Container** in `internal/app/app.go` and updating the **Gin Router** in `internal/httpserver/router.go`.

Following our educational format, this document is written in plain English without complete code blocks so you understand dependency injection, application lifecycle management, and route grouping before writing the code yourself.

---

## 1. Architectural Overview & Responsibilities

Up to this point, each layer (Repository, Service, Handler) has been built in isolation. Phase 7 is where all pieces are assembled into a cohesive, runnable server:

1. **Dependency Injection Container (`internal/app/app.go`)**:
   - Rather than instantiating repositories, services, and handlers randomly inside `main.go`, production Go applications encapsulate setup inside an `App` struct.
   - It acts as the single source of truth for application dependencies: configuration, database connection pools, and HTTP router.
   - It manages application lifecycle: initializing resources on startup and cleanly tearing down database connections on shutdown.

2. **Route Grouping & RBAC Middleware Binding (`internal/httpserver/router.go`)**:
   - Organizes endpoints into clean, logical URL prefixes (`/auth`, `/user`, `/api`).
   - Attaches security middleware (`AuthRequired`, `RequireRoles`, `RequireAdmin`) strictly to the routes that need them.
   - Leaves public routes open while locking down author and admin actions.

---

## 2. Step 7.1: Application Container (`internal/app/app.go`)

### 1. Package & Dependencies
- Create a new directory `internal/app` and file `internal/app/app.go`.
- Declare package `app`.
- Import:
  - Standard library `context`, `fmt`.
  - Project packages: `blog-api/internal/config`, `blog-api/internal/db`, `blog-api/internal/httpserver`, `blog-api/internal/posts`, `blog-api/internal/user`.
  - Framework: `github.com/gin-gonic/gin`.

### 2. The `App` Struct Definition
Define an `App` struct with the fields needed to run and shut down the server:
- `Config`: The loaded `config.Config` value.
- `Router`: Pointer to `gin.Engine` for serving HTTP requests.
- `mongo`: Pointer to `db.Mongo` so the database connection can be closed cleanly on exit.

### 3. Constructor Function (`NewApp`)
Implement a constructor function `NewApp(ctx context.Context) (*App, error)`:

1. **Load Configuration**:
   - Call `config.LoadConfig()`.
   - If an error occurs, return an empty app pointer and a wrapped error indicating configuration loading failed.
2. **Connect to Database**:
   - Call `db.Connect(ctx, cfg)`.
   - If connecting fails, return the error.
3. **Instantiate User Domain**:
   - Create user repository: `user.NewRepo(mongo.Database)`.
   - Create user service: `user.NewService(userRepo, cfg.JWTSecret)`.
   - Create user handler: `user.NewHandler(userService)`.
4. **Instantiate Posts Domain**:
   - Create posts repository: `posts.NewRepo(mongo.Database)`.
   - Create posts service: `posts.NewService(postsRepo, userRepo)` *(note that `posts.Service` receives `userRepo` to look up author details during post creation)*.
   - Create posts handler: `posts.NewHandler(postsService)`.
5. **Assemble Router**:
   - Pass the user handler, posts handler, and JWT secret into your router constructor: `httpserver.NewRouter(userHandler, postsHandler, cfg.JWTSecret)`.
6. **Return**:
   - Return a pointer to the initialized `App` struct containing the config, router, and database handle.

### 4. Cleanup Method (`Close`)
Implement a `Close(ctx context.Context) error` method on `*App`:
- Calls `a.mongo.Disconnect(ctx)`.
- Returns any error encountered during disconnection.

---

## 3. Step 7.2: Router Assembly & RBAC Hierarchy (`internal/httpserver/router.go`)

### 1. Update `NewRouter` Signature
Update the function signature to accept both handlers:
- Parameters: `userHandler *user.Handler`, `postsHandler *posts.Handler`, and `jwtSecret string`.
- Returns: `*gin.Engine`.

### 2. Base Middleware Setup
- Initialize the Gin engine (`gin.New()`).
- Enable `HandleMethodNotAllowed = true`.
- Attach standard global middleware: `gin.Logger()` and `gin.Recovery()`.
- Register the root health check endpoint: `GET /health` mapped to `health`.

### 3. Route Groups & Security Boundaries

Organize your routes into four distinct tiers:

#### Tier 1: Public Authentication Routes (`/auth`)
- Group prefix: `/auth`.
- No authentication middleware attached.
- Endpoints:
  - `POST /auth/register` $\rightarrow$ `userHandler.Register`
  - `POST /auth/login` $\rightarrow$ `userHandler.Login`

#### Tier 2: Public Post Browsing (`/api/posts`)
- Group prefix: `/api/posts`.
- No mandatory authentication middleware (open to guests, readers, authors, and admins alike).
- Endpoints:
  - `GET /api/posts` $\rightarrow$ `postsHandler.ListPublicPosts` (paginated public feed).
  - `GET /api/posts/:id` $\rightarrow$ `postsHandler.GetPostByID` *(reads token context optionally if present to resolve draft visibility)*.

#### Tier 3: Authenticated User & Author Operations
- Group prefix: `/api` (or `/user` for user profile).
- **Security**: Must use `middleware.AuthRequired(jwtSecret)`.
- Sub-group for User Profile:
  - `GET /user/iam` $\rightarrow$ `userHandler.Me`
- Sub-group for Author Actions:
  - Attach author RBAC: `middleware.RequireRoles(user.RoleAuthor, user.RoleAdmin)`. This ensures only authors and admins can create, modify, or view author dashboards. Readers and unauthenticated visitors receive `403 Forbidden`.
  - Endpoints:
    - `POST /api/posts` $\rightarrow$ `postsHandler.CreatePost` (create draft or published article).
    - `GET /api/my-posts` $\rightarrow$ `postsHandler.ListMyPosts` (author's private dashboard).
    - `PATCH /api/posts/:id` $\rightarrow$ `postsHandler.UpdatePost` (update own post, or any post if admin).
    - `DELETE /api/posts/:id` $\rightarrow$ `postsHandler.DeletePost` (delete own post, or any post if admin).

#### Tier 4: Administrator Moderation (`/api/admin`)
- Group prefix: `/api/admin`.
- **Security**: Requires **both** `middleware.AuthRequired(jwtSecret)` and `middleware.RequireAdmin()` (or `RequireRoles(user.RoleAdmin)`).
- Endpoints:
  - `GET /api/admin/posts` $\rightarrow$ `postsHandler.ListAllAdmin` (view all posts across all authors and statuses).
  - Optional: `DELETE /api/admin/posts/:id` $\rightarrow$ `postsHandler.DeletePost` (explicit moderation removal).

---

## 4. Key Gotchas & Best Practices

1. **Middleware Ordering Matters**:
   - Always apply `AuthRequired(jwtSecret)` **before** `RequireRoles(...)`. `RequireRoles` depends on the user ID and role being set in the context by `AuthRequired`. If placed in reverse, the role will always be missing, resulting in `401 Unauthorized`.
2. **Gin Group Route Conflicts**:
   - Gin does not allow registering overlapping dynamic path routes under the same HTTP verb (e.g. registering `/api/posts/:id` and `/api/posts/stats` on `GET`). In our design:
     - Public feed: `GET /api/posts`
     - Single post: `GET /api/posts/:id`
     - Author dashboard: `GET /api/my-posts`
     - Admin feed: `GET /api/admin/posts`
   - These distinct URL paths avoid any routing ambiguity.
3. **Public Route Token Extraction**:
   - `GetPostByID` needs to know if a user is logged in to allow draft viewing. In Gin, you can optionally apply a lightweight token extractor on public routes if desired, or let `GetPostByID` inspect the `Authorization` header directly if `AuthRequired` isn't attached to that route.
