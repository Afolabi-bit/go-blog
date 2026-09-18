# Blog API — Step-by-Step Testing Guide & Verification Runbook

This runbook is structured to directly mirror your **API Client Collection (`Blog-api`)**, guiding you sequentially through every endpoint, test scenario, and validation check.

---

## Quick Reference: Collection Tree & Mapping

```text
Blog-api
│
├── 📁 Auth
│   ├── POST Login       -> /auth/login
│   ├── POST Register    -> /auth/register
│   └── POST Logout      -> /auth/logout
│
├── 📁 User
│   └── GET iam          -> /user/iam
│
├── 📁 Posts
│   ├── POST create      -> /api/posts
│   ├── GET List         -> /api/posts
│   ├── GET my-posts     -> /api/my-posts
│   ├── GET Single Post  -> /api/posts/:id
│   ├── PATCH Update     -> /api/posts/:id   (Note: see UI method callout)
│   └── DELETE Post      -> /api/posts/:id
│
├── 📁 Admin (Moderation)
│   ├── GET Admin Posts  -> /api/admin/posts
│   └── DELETE Admin Post-> /api/admin/posts/:id
│
└── GET health           -> /health
```

> [!IMPORTANT]
> **API Client Method Callout for "Update":**
> In your collection sidebar, if the request is named **`Update`** but shows a yellow/cyan **`GET`** tag, make sure to change the HTTP method dropdown in the request tab to **`PATCH`** and set the URL to `http://localhost:5000/api/posts/:id`. In RESTful APIs, `PATCH` is used for partial resource updates.

---

## Standard Error Conventions (Homogenous Errors)

All error responses across the API follow a strict, standardized format:

- Envelope: `{ "status": "error", "message": "<concise lowercase phrase>" }`
- Phrasing: **all-lowercase, concise, and no trailing punctuation** (no trailing periods or exclamation points).

### Sentinel Error Reference Table

| Status Code                 | Message                                    | Trigger Condition                                                                    |
| :-------------------------- | :----------------------------------------- | :----------------------------------------------------------------------------------- |
| `401 Unauthorized`          | `"authorization header required"`          | Missing `Authorization` header on protected route                                    |
| `401 Unauthorized`          | `"invalid or expired authorization token"` | Token is malformed, has invalid signature, or expired                                |
| `401 Unauthorized`          | `"invalid credentials"`                    | Incorrect email or password during login                                             |
| `401 Unauthorized`          | `"unauthorized"`                           | Unauthenticated context access                                                       |
| `403 Forbidden`             | `"permission denied"`                      | Insufficient role (e.g. `reader` creating a post, non-admin on admin route)          |
| `404 Not Found`             | `"post not found"`                         | Post does not exist, draft accessed without permission, or cross-tenant modification |
| `404 Not Found`             | `"user not found"`                         | User ID does not exist in database                                                   |
| `400 Bad Request`           | `"email already registered"`               | Email already in use during registration                                             |
| `400 Bad Request`           | `"invalid post id format"`                 | Post ID is not a valid 24-character hexadecimal MongoDB ObjectID                     |
| `400 Bad Request`           | `"invalid json payload"`                   | Broken, malformed, or unparseable JSON in request body                               |
| `400 Bad Request`           | `"invalid query parameters"`               | Malformed query parameter or cursor                                                  |
| `400 Bad Request`           | `"invalid limit parameter"`                | Limit query parameter is not an integer                                              |
| `400 Bad Request`           | `"invalid input: <reason>"`                | Domain validation failed (e.g. password length, missing title/content)               |
| `500 Internal Server Error` | `"internal server error"`                  | Unexpected server-side failure (detailed error is logged to console)                 |

---

## Test Personas Matrix

| Persona              | Email               |   Role   | Expected Capabilities                                                       |
| :------------------- | :------------------ | :------: | :-------------------------------------------------------------------------- |
| **Guest / Public**   | _(Unauthenticated)_ |   N/A    | Browse published posts, read single published post, check health.           |
| **Bob (Reader)**     | `bob@reader.com`    | `reader` | View own profile (`/user/iam`), cannot create/edit posts (`403 Forbidden`). |
| **Alice (Author 1)** | `alice@blog.com`    | `author` | Create posts, view draft & published own posts, edit/delete own posts only. |
| **Dave (Author 2)**  | `dave@author.com`   | `author` | Create posts, view own posts, forbidden from modifying Alice's posts.       |
| **Carol (Admin)**    | `admin@blog.com`    | `admin`  | Full moderation access across all authors and `/api/admin/*` endpoints.     |

---

# Step-by-Step Testing Walkthrough

---

### Step 1: Health Check (`GET health`)

Verify that the server is online and connected.

- **Method & URL:** `GET http://localhost:5000/health`
- **Headers:** None
- **Body:** None
- **PowerShell Command:**

```powershell
curl.exe -i -X GET http://localhost:5000/health
```

- **Expected Status:** `200 OK`
- **Expected Response:**

```json
{
  "ok": true,
  "service": "go-blog",
  "time": "2026-09-17T17:30:00.000000Z"
}
```

---

### Step 2: User Registration (`POST /auth/register`)

Register the four test personas, then verify input validation and duplicate prevention.

#### 2.1 Register Reader Bob

- **Method & URL:** `POST http://localhost:5000/auth/register`
- **Headers:** `Content-Type: application/json`
- **Body:**

```json
{
  "email": "bob@reader.com",
  "password": "password123",
  "firstName": "Bob",
  "lastName": "Reader"
}
```

- **PowerShell Command:**

```powershell
curl.exe -i -X POST http://localhost:5000/auth/register `
  -H "Content-Type: application/json" `
  -d '{"email":"bob@reader.com","password":"password123","firstName":"Bob","lastName":"Reader"}'
```

- **Expected Status:** `201 Created`
- **Expected Response:**

```json
{
  "status": "success",
  "message": "User registered successfully",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "6640b2f1e2a8b90123456789",
      "full_name": "Bob Reader",
      "email": "bob@reader.com",
      "role": "reader",
      "created_at": "2026-09-17T17:31:00Z",
      "updated_at": "2026-09-17T17:31:00Z"
    }
  }
}
```

#### 2.2 Register Author Alice

- **Method & URL:** `POST http://localhost:5000/auth/register`
- **Body:**

```json
{
  "email": "alice@blog.com",
  "password": "password123",
  "firstName": "Alice",
  "lastName": "Writer"
}
```

- **Expected Status:** `201 Created`

#### 2.3 Register Author Dave

- **Method & URL:** `POST http://localhost:5000/auth/register`
- **Body:**

```json
{
  "email": "dave@author.com",
  "password": "password123",
  "firstName": "Dave",
  "lastName": "Author"
}
```

- **Expected Status:** `201 Created`

#### 2.4 Register Admin Carol

- **Method & URL:** `POST http://localhost:5000/auth/register`
- **Body:**

```json
{
  "email": "admin@blog.com",
  "password": "password123",
  "firstName": "Carol",
  "lastName": "Admin"
}
```

- **Expected Status:** `201 Created`

#### 2.5 Validation: Duplicate Email Rejection

Send registration for `bob@reader.com` a second time.

- **Expected Status:** `400 Bad Request`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "email already registered"
}
```

#### 2.6 Validation: Short Password (< 6 Characters)

Send registration with `"password": "123"`.

- **Expected Status:** `400 Bad Request`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "invalid input: password must be at least 6 characters"
}
```

#### 2.7 Validation: Malformed JSON Body

Send broken JSON (e.g. `{"email": `)

- **Expected Status:** `400 Bad Request`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "invalid json payload"
}
```

---

### Step 3: Role Promotion (MongoDB)

All new users default to `role: "reader"`. Promote Alice, Dave, and Carol so their JWT tokens contain the correct permissions:

#### Option A: MongoDB Atlas / Compass UI

1. Open database `blog_api_db` $\rightarrow$ collection `users`.
2. Find `alice@blog.com` and `dave@author.com` $\rightarrow$ update `"role"` to `"author"`.
3. Find `admin@blog.com` $\rightarrow$ update `"role"` to `"admin"`.
4. Click **Update / Save**.

#### Option B: mongosh Command

```javascript
use blog_api_db;

db.users.updateMany(
  { email: { $in: ["alice@blog.com", "dave@author.com"] } },
  { $set: { role: "author" } }
);

db.users.updateOne(
  { email: "admin@blog.com" },
  { $set: { role: "admin" } }
);
```

> [!NOTE]
> Roles are encoded into JWT claims upon login. You must log in (`POST /auth/login`) **after** modifying roles in the database to receive a token with the new role.

---

### Step 4: User Login (`POST /auth/login`)

Authenticate to obtain Bearer JWT tokens for each persona.

#### 4.1 Login with Incorrect Password

- **Method & URL:** `POST http://localhost:5000/auth/login`
- **Headers:** `Content-Type: application/json`
- **Body:**

```json
{
  "email": "alice@blog.com",
  "password": "wrongpassword"
}
```

- **Expected Status:** `401 Unauthorized`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "invalid credentials"
}
```

#### 4.2 Successful Login (Obtain Tokens)

Log in with each account and save the returned `data.token`:

```powershell
# Alice (Author)
curl.exe -i -X POST http://localhost:5000/auth/login `
  -H "Content-Type: application/json" `
  -d '{"email":"alice@blog.com","password":"password123"}'

# Bob (Reader)
curl.exe -i -X POST http://localhost:5000/auth/login `
  -H "Content-Type: application/json" `
  -d '{"email":"bob@reader.com","password":"password123"}'

# Dave (Author)
curl.exe -i -X POST http://localhost:5000/auth/login `
  -H "Content-Type: application/json" `
  -d '{"email":"dave@author.com","password":"password123"}'

# Carol (Admin)
curl.exe -i -X POST http://localhost:5000/auth/login `
  -H "Content-Type: application/json" `
  -d '{"email":"admin@blog.com","password":"password123"}'
```

- **Expected Status:** `200 OK`
- **Expected Response (`alice@blog.com`):**

```json
{
  "status": "success",
  "message": "User logged in successfully",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "6640b2f1e2a8b90123456789",
      "full_name": "Alice Writer",
      "email": "alice@blog.com",
      "role": "author",
      "created_at": "2026-09-17T17:31:00Z",
      "updated_at": "2026-09-17T17:31:00Z"
    }
  }
}
```

---

### Step 5: User Profile (`GET /user/iam`)

Inspect the currently authenticated user's profile and test token validation.

#### 5.1 Fetch Profile with Valid Token

- **Method & URL:** `GET http://localhost:5000/user/iam`
- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
- **Body:** None
- **PowerShell Command:**

```powershell
curl.exe -i -X GET http://localhost:5000/user/iam `
  -H "Authorization: Bearer <ALICE_TOKEN>"
```

- **Expected Status:** `200 OK`
- **Expected Response:**

```json
{
  "status": "success",
  "message": "User profile fetched successfully",
  "data": {
    "id": "6640b2f1e2a8b90123456789",
    "full_name": "Alice Writer",
    "email": "alice@blog.com",
    "role": "author",
    "created_at": "2026-09-17T17:31:00Z",
    "updated_at": "2026-09-17T17:31:00Z"
  }
}
```

#### 5.2 Error: Missing Authorization Header

Send request with no `Authorization` header.

- **Expected Status:** `401 Unauthorized`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "authorization header required"
}
```

#### 5.3 Error: Invalid or Expired Token

Send request with `Authorization: Bearer bad.token.here`.

- **Expected Status:** `401 Unauthorized`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "invalid or expired authorization token"
}
```

---

### Step 6: Post Creation (`POST /api/posts`)

Test role enforcement (RBAC), draft vs published creation, and field validations.

#### 6.1 RBAC Check: Reader Forbidden from Creating Post

Bob (`role: "reader"`) attempts to create a post.

- **Method & URL:** `POST http://localhost:5000/api/posts`
- **Headers:** `Authorization: Bearer <BOB_TOKEN>`, `Content-Type: application/json`
- **Body:**

```json
{
  "title": "Bob Reader Post",
  "content": "Readers should not be allowed to publish."
}
```

- **Expected Status:** `403 Forbidden`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "permission denied"
}
```

#### 6.2 Author Alice Creates a Draft Post

- **Method & URL:** `POST http://localhost:5000/api/posts`
- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`, `Content-Type: application/json`
- **Body:**

```json
{
  "title": "Alice Confidential Draft",
  "content": "This draft contains unpublished research and thoughts.",
  "status": "draft",
  "tags": ["golang", "research", "draft"]
}
```

- **PowerShell Command:**

```powershell
curl.exe -i -X POST http://localhost:5000/api/posts `
  -H "Authorization: Bearer <ALICE_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"title":"Alice Confidential Draft","content":"This draft contains unpublished research and thoughts.","status":"draft","tags":["golang","research","draft"]}'
```

- **Expected Status:** `201 Created`
- **Expected Response:**

```json
{
  "status": "success",
  "message": "post successfully created",
  "data": {
    "id": "6640c5a2e2a8b90123456790",
    "author_id": "6640b2f1e2a8b90123456789",
    "author_name": "Alice Writer",
    "title": "Alice Confidential Draft",
    "slug": "alice-confidential-draft",
    "content": "This draft contains unpublished research and thoughts.",
    "status": "draft",
    "tags": ["golang", "research", "draft"],
    "created_at": "2026-09-17T17:35:00Z",
    "updated_at": "2026-09-17T17:35:00Z"
  }
}
```

> **Action:** Save returned `data.id` as **`$ALICE_DRAFT_ID`**.

#### 6.3 Author Alice Creates a Published Post

- **Method & URL:** `POST http://localhost:5000/api/posts`
- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`, `Content-Type: application/json`
- **Body:**

```json
{
  "title": "Mastering Go Concurrency in 2026",
  "content": "Goroutines and buffered channels provide powerful concurrency primitives.",
  "status": "published",
  "tags": ["golang", "concurrency", "backend"]
}
```

- **Expected Status:** `201 Created`
  > **Action:** Save returned `data.id` as **`$ALICE_PUBLISHED_ID`**.

#### 6.4 Author Dave Creates His Own Published Post

- **Method & URL:** `POST http://localhost:5000/api/posts`
- **Headers:** `Authorization: Bearer <DAVE_TOKEN>`, `Content-Type: application/json`
- **Body:**

```json
{
  "title": "Daves Architectural Insights",
  "content": "Designing resilient microservices in Go.",
  "status": "published",
  "tags": ["architecture", "microservices"]
}
```

- **Expected Status:** `201 Created`
  > **Action:** Save returned `data.id` as **`$DAVE_POST_ID`**.

#### 6.5 Validation: Missing Title or Content

Send request with empty `"title": ""`.

- **Expected Status:** `400 Bad Request`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "invalid input: title and content are required"
}
```

---

### Step 7: Public Posts Feed (`GET /api/posts` — "List")

Verify public feed visibility, draft isolation, search, and cursor pagination.

#### 7.1 Public Feed Hides Drafts

- **Method & URL:** `GET http://localhost:5000/api/posts`
- **Headers:** None (Public)
- **PowerShell Command:**

```powershell
curl.exe -i -X GET http://localhost:5000/api/posts
```

- **Expected Status:** `200 OK`
- **Verify:**
  - `"Mastering Go Concurrency in 2026"` (Published) is present.
  - `"Daves Architectural Insights"` (Published) is present.
  - `"Alice Confidential Draft"` (Draft) is **NOT** present.

#### 7.2 Search by Keyword

- **Method & URL:** `GET http://localhost:5000/api/posts?search=Concurrency`
- **Expected Status:** `200 OK`
- **Verify:** Only posts containing `"Concurrency"` in the title appear.

#### 7.3 Filter by Tag

- **Method & URL:** `GET http://localhost:5000/api/posts?tag=golang`
- **Expected Status:** `200 OK`
- **Verify:** Only posts tagged with `"golang"` appear.

#### 7.4 Cursor-Based Pagination

Request 1 item per page:

```powershell
curl.exe -i -X GET "http://localhost:5000/api/posts?limit=1"
```

- **Expected Status:** `200 OK`
- **Expected Response Envelope:**

```json
{
  "status": "success",
  "message": "Fetched posts successfully",
  "data": {
    "posts": [ ... ],
    "pagination": {
      "limit": 1,
      "has_next": true,
      "next_cursor": "6640c6b3e2a8b90123456791",
      "count": 1
    }
  }
}
```

Fetch the next page using the returned `next_cursor`:

```powershell
curl.exe -i -X GET "http://localhost:5000/api/posts?limit=1&cursor=6640c6b3e2a8b90123456791"
```

---

### Step 8: Author Dashboard (`GET /api/my-posts`)

Verify that authors can see their own drafts and published posts, but not other authors' posts.

#### 8.1 Alice Views Her Own Dashboard

- **Method & URL:** `GET http://localhost:5000/api/my-posts`
- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
- **PowerShell Command:**

```powershell
curl.exe -i -X GET http://localhost:5000/api/my-posts `
  -H "Authorization: Bearer <ALICE_TOKEN>"
```

- **Expected Status:** `200 OK`
- **Verify:**
  - Contains Alice's draft (`"Alice Confidential Draft"`).
  - Contains Alice's published post (`"Mastering Go Concurrency in 2026"`).
  - **Does NOT** contain Dave's post (`"Daves Architectural Insights"`).

#### 8.2 Error: Missing Token

Send request with no `Authorization` header.

- **Expected Status:** `401 Unauthorized`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "authorization header required"
}
```

---

### Step 9: Get Single Post by ID (`GET /api/posts/:id`)

Verify privacy controls on drafts and error handling for malformed IDs.

#### 9.1 Public Views Published Post

- **Method & URL:** `GET http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID>`
- **Headers:** None
- **Expected Status:** `200 OK`

#### 9.2 Public Cannot View Draft Post

- **Method & URL:** `GET http://localhost:5000/api/posts/<ALICE_DRAFT_ID>`
- **Headers:** None
- **Expected Status:** `404 Not Found`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "post not found"
}
```

#### 9.3 Author Alice Views Her Own Draft

- **Method & URL:** `GET http://localhost:5000/api/posts/<ALICE_DRAFT_ID>`
- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
- **PowerShell Command:**

```powershell
curl.exe -i -X GET http://localhost:5000/api/posts/<ALICE_DRAFT_ID> `
  -H "Authorization: Bearer <ALICE_TOKEN>"
```

- **Expected Status:** `200 OK`
- **Verify:** Alice can view her confidential draft.

#### 9.4 Admin Carol Views Alice's Draft

- **Method & URL:** `GET http://localhost:5000/api/posts/<ALICE_DRAFT_ID>`
- **Headers:** `Authorization: Bearer <CAROL_TOKEN>`
- **PowerShell Command:**

```powershell
curl.exe -i -X GET http://localhost:5000/api/posts/<ALICE_DRAFT_ID> `
  -H "Authorization: Bearer <CAROL_TOKEN>"
```

- **Expected Status:** `200 OK`

#### 9.5 Error: Malformed Hex ID

- **Method & URL:** `GET http://localhost:5000/api/posts/not-a-valid-hex`
- **Expected Status:** `400 Bad Request`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "invalid post id format"
}
```

#### 9.6 Error: Non-Existent Post ID

- **Method & URL:** `GET http://localhost:5000/api/posts/65f1a2b3c4d5e6f7a8b9c0d1`
- **Expected Status:** `404 Not Found`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "post not found"
}
```

---

### Step 10: Update Post (`PATCH /api/posts/:id` — "Update")

Test partial updates, multi-tenant isolation, and administrator override.

> [!TIP]
> In your API client, ensure the method is set to **`PATCH`** (not `GET`).

#### 10.1 Author Alice Updates Her Own Post (Title & Tags)

- **Method & URL:** `PATCH http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID>`
- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`, `Content-Type: application/json`
- **Body:**

```json
{
  "title": "Mastering Go Concurrency (2026 Edition)",
  "tags": ["golang", "concurrency", "updated"]
}
```

- **PowerShell Command:**

```powershell
curl.exe -i -X PATCH http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID> `
  -H "Authorization: Bearer <ALICE_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"title":"Mastering Go Concurrency (2026 Edition)","tags":["golang","concurrency","updated"]}'
```

- **Expected Status:** `200 OK`
- **Verify:** `data.title == "Mastering Go Concurrency (2026 Edition)"` and slug is updated.

#### 10.2 Author Alice Publishes Her Draft

- **Method & URL:** `PATCH http://localhost:5000/api/posts/<ALICE_DRAFT_ID>`
- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`, `Content-Type: application/json`
- **Body:**

```json
{
  "status": "published"
}
```

- **Expected Status:** `200 OK`
- **Verify:** `data.status == "published"`.

#### 10.3 Multi-Tenant Isolation: Dave Forbidden from Updating Alice's Post

Dave attempts to update Alice's post.

- **Method & URL:** `PATCH http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID>`
- **Headers:** `Authorization: Bearer <DAVE_TOKEN>`, `Content-Type: application/json`
- **Body:**

```json
{
  "title": "Hacked by Dave"
}
```

- **Expected Status:** `404 Not Found`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "post not found"
}
```

- **Verify:** Alice's post title was **not** changed.

#### 10.4 Admin Carol Overrides & Updates Dave's Post

Carol uses admin privileges to moderate Dave's post.

- **Method & URL:** `PATCH http://localhost:5000/api/posts/<DAVE_POST_ID>`
- **Headers:** `Authorization: Bearer <CAROL_TOKEN>`, `Content-Type: application/json`
- **Body:**

```json
{
  "title": "Daves Architectural Insights (Reviewed by Admin)"
}
```

- **Expected Status:** `200 OK`
- **Verify:** Title is successfully updated.

---

### Step 11: Post Deletion & Administration

Test author post deletion and administrator moderation endpoints.

#### 11.1 Author Dave Cannot Delete Alice's Post

- **Method & URL:** `DELETE http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID>`
- **Headers:** `Authorization: Bearer <DAVE_TOKEN>`
- **Expected Status:** `404 Not Found`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "post not found"
}
```

#### 11.2 Admin Carol Views All Posts Across Authors (`GET /api/admin/posts`)

- **Method & URL:** `GET http://localhost:5000/api/admin/posts`
- **Headers:** `Authorization: Bearer <CAROL_TOKEN>`
- **PowerShell Command:**

```powershell
curl.exe -i -X GET http://localhost:5000/api/admin/posts `
  -H "Authorization: Bearer <CAROL_TOKEN>"
```

- **Expected Status:** `200 OK`
- **Verify:** Contains posts from both Alice and Dave.

#### 11.3 Non-Admin Forbidden from Admin Endpoint

Bob or Alice calls `/api/admin/posts`.

- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
- **Expected Status:** `403 Forbidden`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "permission denied"
}
```

#### 11.4 Admin Carol Moderates & Deletes Dave's Post

- **Method & URL:** `DELETE http://localhost:5000/api/admin/posts/<DAVE_POST_ID>`
- **Headers:** `Authorization: Bearer <CAROL_TOKEN>`
- **Expected Status:** `200 OK`
- **Expected Response:**

```json
{
  "status": "success",
  "message": "Post successfully deleted"
}
```

- **Verify:** Subsequent `GET /api/posts/<DAVE_POST_ID>` returns `404 Not Found`.

---

### Step 12: User Logout (`POST /auth/logout`)

Invalidate or complete the user session.

#### 12.1 Successful Logout

- **Method & URL:** `POST http://localhost:5000/auth/logout`
- **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
- **Body:** None
- **PowerShell Command:**

```powershell
curl.exe -i -X POST http://localhost:5000/auth/logout `
  -H "Authorization: Bearer <ALICE_TOKEN>"
```

- **Expected Status:** `200 OK`
- **Expected Response:**

```json
{
  "status": "success",
  "message": "User logged out successfully"
}
```

#### 12.2 Logout Without Token

Send request without `Authorization` header.

- **Expected Status:** `401 Unauthorized`
- **Expected Response:**

```json
{
  "status": "error",
  "message": "authorization header required"
}
```
