# Step 6.4 Implementation Guide: Posts HTTP Handlers

This guide walks you step-by-step through designing and implementing the **HTTP Handlers** for the Posts domain in `internal/posts/handler.go`.

Following our educational format, this document is written in plain English without complete code blocks so you can understand transport layer responsibilities, request binding, error mapping, and write the Go implementation yourself.

---

## 1. Architectural Overview & Responsibilities

The Handler layer is the transport boundary of your application. It acts as the bridge between incoming HTTP requests and your internal Service layer.

The Handler is strictly responsible for:
1. **Request Decoding & Binding**:
   - Extracting URL parameters (e.g. `:id`).
   - Binding and validating query parameters for search and cursor pagination (e.g. `?search=go&tag=web&limit=10&cursor=...`).
   - Parsing and validating JSON request bodies into DTOs (`CreatePostRequest`, `UpdatePostRequest`).
2. **Context Extraction**:
   - Reading the authenticated user's ID and Role stored in the `gin.Context` by the authentication middleware (`middleware.GetUserID(c)`, `middleware.GetRole(c)`).
   - Converting string IDs into MongoDB Object IDs when calling service methods.
3. **Delegation**:
   - Calling the appropriate Service method with sanitized inputs and context.
4. **Error Mapping & Response Formatting**:
   - Translating domain errors (`ErrNotFound`, `ErrForbidden`, `ErrInvalidInput`) into standard HTTP status codes (`400 Bad Request`, `403 Forbidden`, `404 Not Found`, `500 Internal Server Error`).
   - Returning consistent JSON response envelopes using `response.Success` and `response.Error`.

---

## 2. Package Dependencies & Setup

Ensure your file header in `internal/posts/handler.go` declares package `posts` and imports the necessary packages:

1. **Standard Library**:
   - `net/http`: For standard HTTP status constants (`http.StatusOK`, `http.StatusCreated`, `http.StatusBadRequest`, `http.StatusNotFound`, `http.StatusForbidden`, `http.StatusInternalServerError`).
   - `errors`: For checking sentinel errors using `errors.Is`.
   - `strconv`: Optional, if manually parsing string query parameters to integers.
2. **Framework & Project Packages**:
   - `github.com/gin-gonic/gin`: For HTTP request context, binding, and routing.
   - `blog-api/internal/middleware`: For retrieving authenticated user context (`GetUserID`, `GetRole`).
   - `blog-api/internal/response`: For emitting standardized JSON success and error envelopes.
   - `go.mongodb.org/mongo-driver/bson/primitive`: For parsing hex string IDs into Object IDs.

---

## 3. Handler Struct & Constructor

1. **Handler Struct**:
   - Defines a `Handler` struct that holds a pointer to `Service` (`svc *Service` or `service *Service`).
2. **Constructor Function (`NewHandler`)**:
   - Accepts `*Service` as an argument.
   - Returns a pointer to the initialized `Handler` struct.

---

## 4. Helper: Centralized Error Mapping

Because several handlers encounter the same domain errors, consider writing a small unexported helper method or `switch` block to map service errors to HTTP responses:

- If `errors.Is(err, ErrNotFound)`: Respond with `http.StatusNotFound` and the error message.
- If `errors.Is(err, ErrForbidden)`: Respond with `http.StatusForbidden` and the error message.
- If `errors.Is(err, ErrInvalidInput)`: Respond with `http.StatusBadRequest` and the error message.
- For any other unhandled error: Respond with `http.StatusInternalServerError` and a generic message like `"internal server error"`.

---

## 5. Detailed Handler Method Specifications

### Method 1: `CreatePost` (POST `/api/posts`)

**Objective**: Allow an authenticated author to create a new draft or published post.

1. **Extract Authentication**:
   - Call `middleware.GetUserID(c)` to retrieve the authenticated user ID string. If not found, return an unauthorized error (`401`).
   - Convert the user ID string to a `primitive.ObjectID`. If conversion fails, return an unauthorized or bad request error.
2. **Determine Author Display Name**:
   - The user's name can be fetched via your injected user service/repo using the author ID, or defaulted to a fallback string (e.g. `"Author"` or the user's email if available in context).
3. **Bind JSON Body**:
   - Create a `CreatePostRequest` variable.
   - Call `c.ShouldBindJSON(&input)`. If binding fails (e.g. missing required fields), return a `400 Bad Request` with `err.Error()`.
4. **Execute & Respond**:
   - Call `h.svc.CreateNewPost(c.Request.Context(), authorObjID, authorName, input)`.
   - If an error is returned, map it to the appropriate error status code.
   - On success, return `http.StatusCreated` (201) using `response.Success` with the created post object.

---

### Method 2: `GetPostByID` (GET `/api/posts/:id`)

**Objective**: Retrieve a single post. If the post is a draft, only allow access if the requester is the post owner or an admin.

1. **Extract URL Parameter**:
   - Retrieve the post ID from the URL path using `c.Param("id")`.
2. **Extract Optional Authentication**:
   - Attempt to read the user ID and role using `middleware.GetUserID(c)` and `middleware.GetRole(c)`.
   - **Note**: Because this endpoint is publicly reachable, unauthenticated visitors will not have tokens. If the getters return false, pass `nil` for both `requesterID` and `requesterRole` pointers to the service method. If authenticated, pass pointers to the extracted strings.
3. **Execute & Respond**:
   - Call `h.svc.GetPostByID(c.Request.Context(), postIDStr, requesterIDPtr, requesterRolePtr)`.
   - If an error is returned, map it using your error helper (e.g. `ErrNotFound` -> 404, `ErrForbidden` -> 403).
   - On success, return `http.StatusOK` (200) with the post data.

---

### Method 3: `ListPublicPosts` (GET `/api/posts`)

**Objective**: Provide public visitors with a paginated, searchable feed of published articles.

1. **Bind Query Parameters**:
   - Declare a `PostFilter` struct variable.
   - Use Gin's `c.ShouldBindQuery(&filter)` to automatically bind query parameters into the struct (such as `tag`, `search`, `limit`, and `cursor`).
2. **Execute Service**:
   - Call `h.svc.ListPublicPosts(c.Request.Context(), filter.Cursor, filter.Limit, filter)`.
   - If an error occurs, return an internal server error or bad request error.
3. **Respond**:
   - Return `http.StatusOK` (200) with a payload wrapping both the slice of posts and the `PaginationMeta` struct (for example, in a `gin.H{"posts": posts, "pagination": pMeta}`).

---

### Method 4: `ListMyPosts` (GET `/api/my-posts`)

**Objective**: Provide an authenticated author with their personal dashboard of posts (both drafts and published).

1. **Extract Authentication**:
   - Read the author ID using `middleware.GetUserID(c)`. If missing, return `401 Unauthorized`.
   - Parse the author ID into a `primitive.ObjectID`.
2. **Extract Pagination Parameters**:
   - Read the cursor query parameter (`c.Query("cursor")`).
   - Read the limit query parameter using `c.DefaultQuery("limit", "10")`, and parse it into an `int64`.
3. **Execute & Respond**:
   - Call `h.svc.ListMyPosts(c.Request.Context(), authorObjID, cursorStr, limitInt)`.
   - Handle any returned service error.
   - On success, return `http.StatusOK` (200) with the posts and pagination metadata.

---

### Method 5: `ListAllAdmin` (GET `/api/admin/posts`)

**Objective**: Allow administrators to browse and search all posts across all users.

1. **Bind Query Parameters**:
   - Bind `PostFilter` using `c.ShouldBindQuery(&filter)`.
2. **Execute & Respond**:
   - Call `h.svc.ListAllAdmin(c.Request.Context(), filter.Cursor, filter.Limit, filter)`.
   - Handle any returned error.
   - On success, return `http.StatusOK` (200) with the posts and pagination metadata.

---

### Method 6: `UpdatePost` (PATCH `/api/posts/:id`)

**Objective**: Apply partial updates to a post while enforcing author ownership or admin override.

1. **Extract Parameters & Context**:
   - Retrieve the post ID from `c.Param("id")`.
   - Retrieve the authenticated user ID (`middleware.GetUserID(c)`) and role (`middleware.GetRole(c)`). Return `401 Unauthorized` if either is missing.
   - Convert the user ID to a `primitive.ObjectID`.
2. **Bind JSON Body**:
   - Declare an `UpdatePostRequest` variable.
   - Call `c.ShouldBindJSON(&input)`. If invalid JSON is sent, return `400 Bad Request`.
3. **Execute & Respond**:
   - Call `h.svc.UpdatePost(c.Request.Context(), postIDStr, userObjID, userRole, input)`.
   - Map domain errors (`ErrNotFound` -> 404, `ErrForbidden` -> 403, `ErrInvalidInput` -> 400).
   - On success, return `http.StatusOK` (200) with the updated post.

---

### Method 7: `DeletePost` (DELETE `/api/posts/:id`)

**Objective**: Delete a post while enforcing author ownership or admin override.

1. **Extract Parameters & Context**:
   - Retrieve the post ID from `c.Param("id")`.
   - Retrieve user ID and role from context. Return `401 Unauthorized` if missing.
   - Convert user ID to a `primitive.ObjectID`.
2. **Execute & Respond**:
   - Call `h.svc.DeletePost(c.Request.Context(), postIDStr, userObjID, userRole)`.
   - Map domain errors (`ErrNotFound` -> 404, `ErrForbidden` -> 403).
   - On success, return `http.StatusOK` (200) with a confirmation message (e.g. `"Post deleted successfully"`).

---

## 6. Key Gotchas & Best Practices

1. **Public vs Protected Route Handlers**:
   - `GetPostByID` and `ListPublicPosts` can be accessed without a token. Never assume `GetUserID(c)` succeeds on those routes.
   - For protected routes (`CreatePost`, `ListMyPosts`, `UpdatePost`, `DeletePost`), always verify authentication before proceeding.
2. **Standard Response Envelope**:
   - Always use `response.Success(c, statusCode, message, data)` and `response.Error(c, statusCode, message)` so API consumers always receive a predictable JSON shape.
3. **Avoid Exposing Internal Error Details**:
   - Never send raw database error strings to clients. Translate them through sentinel errors into clear, friendly messages.
