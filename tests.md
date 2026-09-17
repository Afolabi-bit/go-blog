# Blog API — End-to-End Test Suite & Verification Runbook

This document contains extensive, production-grade test cases and **exact request/response payloads** for validating the **Blog API**, covering authentication, Role-Based Access Control (RBAC), multi-tenant data isolation, cursor pagination, and administrator moderation.

---

## 1. Test Environment & Setup

* **Base URL:** `http://localhost:5000`
* **Content-Type:** `application/json`

### Test Persona Matrix

| Persona | Email | Role | Expected Capabilities |
| :--- | :--- | :---: | :--- |
| **Guest / Public** | *(Unauthenticated)* | N/A | Browse published posts, read single published post, health check. |
| **Bob (Reader)** | `bob@reader.com` | `reader` | User profile (`/user/iam`), cannot create/edit posts (`403 Forbidden`). |
| **Alice (Author 1)** | `alice@blog.com` | `author` | Create posts, view draft & published own posts, edit/delete own posts only. |
| **Dave (Author 2)** | `dave@author.com` | `author` | Create posts, view own posts, **forbidden from editing Alice's posts**. |
| **Carol (Admin)** | `admin@blog.com` | `admin` | Full moderation access across all authors, access to `/api/admin/*`. |

---

## 2. Complete Request & Response Payload Reference

Use these exact JSON payloads to test in **Postman**, **Thunder Client**, **Insomnia**, or **curl**.

### A. Authentication Payloads

#### 1. Registration (`POST /auth/register`)
* **Request Body:**
```json
{
  "email": "alice@blog.com",
  "password": "password123",
  "firstName": "Alice",
  "lastName": "Writer"
}
```
* **Expected Response (`201 Created`):**
```json
{
  "status": "success",
  "message": "User registered successfully",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "6740b2f1e2a8b90123456789",
      "full_name": "Alice Writer",
      "email": "alice@blog.com",
      "role": "reader",
      "created_at": "2026-09-17T13:45:00.123456Z",
      "updated_at": "2026-09-17T13:45:00.123456Z"
    }
  }
}
```

#### 2. Login (`POST /auth/login`)
* **Request Body:**
```json
{
  "email": "alice@blog.com",
  "password": "password123"
}
```
* **Expected Response (`200 OK`):**
```json
{
  "status": "success",
  "message": "User logged in successfully",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "6740b2f1e2a8b90123456789",
      "full_name": "Alice Writer",
      "email": "alice@blog.com",
      "role": "author",
      "created_at": "2026-09-17T13:45:00.123456Z",
      "updated_at": "2026-09-17T13:45:00.123456Z"
    }
  }
}
```

#### 3. User Profile (`GET /user/iam`)
* **Request Headers:** `Authorization: Bearer <TOKEN>`
* **Expected Response (`200 OK`):**
```json
{
  "status": "success",
  "message": "User profile fetched successfully",
  "data": {
    "id": "6740b2f1e2a8b90123456789",
    "full_name": "Alice Writer",
    "email": "alice@blog.com",
    "role": "author",
    "created_at": "2026-09-17T13:45:00.123456Z",
    "updated_at": "2026-09-17T13:45:00.123456Z"
  }
}
```

#### 4. Logout (`POST /auth/logout`)
* **Request Headers:** `Authorization: Bearer <TOKEN>`
* **Request Body:** None
* **Expected Response (`200 OK`):**
```json
{
  "status": "success",
  "message": "User logged out successfully"
}
```

---

### B. Post Creation Payloads (`POST /api/posts`)

#### 1. Create Draft Post
* **Request Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Request Body:**
```json
{
  "title": "My Secret Work In Progress",
  "content": "This draft contains unpublished research and thoughts.",
  "status": "draft",
  "tags": ["golang", "research", "draft"]
}
```
* **Expected Response (`201 Created`):**
```json
{
  "status": "success",
  "message": "post successfully created",
  "data": {
    "id": "6740c5a2e2a8b90123456790",
    "author_id": "6740b2f1e2a8b90123456789",
    "author_name": "Alice Writer",
    "title": "My Secret Work In Progress",
    "slug": "my-secret-work-in-progress",
    "content": "This draft contains unpublished research and thoughts.",
    "status": "draft",
    "tags": ["golang", "research", "draft"],
    "created_at": "2026-09-17T14:00:00.123456Z",
    "updated_at": "2026-09-17T14:00:00.123456Z"
  }
}
```

#### 2. Create Published Post
* **Request Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Request Body:**
```json
{
  "title": "Mastering Go Concurrency in 2026",
  "content": "Goroutines and buffered channels provide powerful concurrency primitives.",
  "status": "published",
  "tags": ["golang", "concurrency", "backend"]
}
```
* **Expected Response (`201 Created`):**
```json
{
  "status": "success",
  "message": "post successfully created",
  "data": {
    "id": "6740c6b3e2a8b90123456791",
    "author_id": "6740b2f1e2a8b90123456789",
    "author_name": "Alice Writer",
    "title": "Mastering Go Concurrency in 2026",
    "slug": "mastering-go-concurrency-in-2026",
    "content": "Goroutines and buffered channels provide powerful concurrency primitives.",
    "status": "published",
    "tags": ["golang", "concurrency", "backend"],
    "created_at": "2026-09-17T14:05:00.123456Z",
    "updated_at": "2026-09-17T14:05:00.123456Z"
  }
}
```

---

### C. Post Update Payloads (`PATCH /api/posts/:id`)

#### 1. Partial Update — Title Only
* **Request Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Request Body:**
```json
{
  "title": "Mastering Go Concurrency (Updated Edition)"
}
```
* **Expected Response (`200 OK`):**
```json
{
  "status": "success",
  "message": "Post updated successfully",
  "data": {
    "id": "6740c6b3e2a8b90123456791",
    "author_id": "6740b2f1e2a8b90123456789",
    "author_name": "Alice Writer",
    "title": "Mastering Go Concurrency (Updated Edition)",
    "slug": "mastering-go-concurrency-updated-edition",
    "content": "Goroutines and buffered channels provide powerful concurrency primitives.",
    "status": "published",
    "tags": ["golang", "concurrency", "backend"],
    "created_at": "2026-09-17T14:05:00.123456Z",
    "updated_at": "2026-09-17T14:15:00.123456Z"
  }
}
```

#### 2. Partial Update — Publish a Draft
* **Request Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Request Body:**
```json
{
  "status": "published",
  "tags": ["golang", "final", "production"]
}
```
* **Expected Response (`200 OK`):**
```json
{
  "status": "success",
  "message": "Post updated successfully",
  "data": {
    "id": "6740c5a2e2a8b90123456790",
    "status": "published",
    "tags": ["golang", "final", "production"],
    "updated_at": "2026-09-17T14:20:00.123456Z"
  }
}
```

---

### D. List & Feed Response Envelopes

#### 1. Public Feed (`GET /api/posts`)
* **Expected Response (`200 OK`):**
```json
{
  "status": "success",
  "message": "Fetched posts successfully",
  "data": {
    "posts": [
      {
        "id": "6740c6b3e2a8b90123456791",
        "author_id": "6740b2f1e2a8b90123456789",
        "author_name": "Alice Writer",
        "title": "Mastering Go Concurrency in 2026",
        "slug": "mastering-go-concurrency-in-2026",
        "content": "Goroutines and buffered channels provide powerful concurrency primitives.",
        "status": "published",
        "tags": ["golang", "concurrency", "backend"],
        "created_at": "2026-09-17T14:05:00.123456Z",
        "updated_at": "2026-09-17T14:05:00.123456Z"
      }
    ],
    "pagination": {
      "limit": 10,
      "has_next": false,
      "next_cursor": "",
      "count": 1
    }
  }
}
```

#### 2. Author Dashboard (`GET /api/my-posts`)
* **Request Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Expected Response (`200 OK`):**
```json
{
  "status": "success",
  "message": "Fetched posts successfully",
  "data": {
    "posts": [
      {
        "id": "6740c6b3e2a8b90123456791",
        "title": "Mastering Go Concurrency in 2026",
        "status": "published"
      },
      {
        "id": "6740c5a2e2a8b90123456790",
        "title": "My Secret Work In Progress",
        "status": "draft"
      }
    ],
    "pagination": {
      "limit": 10,
      "has_next": false,
      "next_cursor": "",
      "count": 2
    }
  }
}
```

---

### E. Standard Error Payloads

#### 1. Unauthorized (`401 Unauthorized`)
```json
{
  "status": "error",
  "message": "Authorization header required"
}
```

#### 2. Forbidden (`403 Forbidden`)
```json
{
  "status": "error",
  "message": "you don't have the permission to perform this action."
}
```

#### 3. Not Found (`404 Not Found`)
```json
{
  "status": "error",
  "message": "post not found"
}
```

#### 4. Bad Request / Validation Failure (`400 Bad Request`)
```json
{
  "status": "error",
  "message": "Email already registered!"
}
```

---

## 3. Role Promotion in MongoDB

By default, all new registrations receive `role: "reader"`. To promote test accounts to `author` or `admin`:

### Option A: MongoDB Atlas Web UI
1. In the Atlas dashboard, navigate to **Database** $\rightarrow$ **Browse Collections**.
2. Open `blog_api_db` $\rightarrow$ `users`.
3. Find `alice@blog.com` and `dave@author.com` $\rightarrow$ edit `"role": "reader"` to `"role": "author"`.
4. Find `admin@blog.com` $\rightarrow$ edit `"role": "reader"` to `"role": "admin"`.
5. Click **Update**.

### Option B: mongosh Terminal Command
```javascript
use blog_api_db;

// Promote Authors
db.users.updateMany(
  { email: { $in: ["alice@blog.com", "dave@author.com"] } },
  { $set: { role: "author" } }
);

// Promote Admin
db.users.updateOne(
  { email: "admin@blog.com" },
  { $set: { role: "admin" } }
);
```

> **Crucial:** After changing roles in the database, log in again (`POST /auth/login`) so that the updated role is minted into your new JWT token.

---

## 4. End-to-End Test Cases (With Payloads & Commands)

### Section 1: Health & Connectivity

#### Test 1.1: Health Check
* **Method & URL:** `GET http://localhost:5000/health`
* **Headers:** None
* **Payload:** None
* **Command:**
```powershell
curl.exe -i -X GET http://localhost:5000/health
```
* **Expected Status:** `200 OK`
* **Expected Response:**
```json
{
  "ok": true,
  "service": "go-blog",
  "time": "2026-09-17T13:45:00.123456Z"
}
```

---

### Section 2: User Domain & Authentication Lifecycle

#### Test 2.1: Register Reader Bob
* **Method & URL:** `POST http://localhost:5000/auth/register`
* **Headers:** `Content-Type: application/json`
* **Request Payload:**
```json
{
  "email": "bob@reader.com",
  "password": "password123",
  "firstName": "Bob",
  "lastName": "Reader"
}
```
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/auth/register `
  -H "Content-Type: application/json" `
  -d '{"email":"bob@reader.com","password":"password123","firstName":"Bob","lastName":"Reader"}'
```
* **Expected Status:** `201 Created`
* **Verify:** Returns `role: "reader"`.

#### Test 2.2: Register Author Alice
* **Method & URL:** `POST http://localhost:5000/auth/register`
* **Request Payload:**
```json
{
  "email": "alice@blog.com",
  "password": "password123",
  "firstName": "Alice",
  "lastName": "Writer"
}
```
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/auth/register `
  -H "Content-Type: application/json" `
  -d '{"email":"alice@blog.com","password":"password123","firstName":"Alice","lastName":"Writer"}'
```
* **Expected Status:** `201 Created`

#### Test 2.3: Register Author Dave
* **Method & URL:** `POST http://localhost:5000/auth/register`
* **Request Payload:**
```json
{
  "email": "dave@author.com",
  "password": "password123",
  "firstName": "Dave",
  "lastName": "Author"
}
```
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/auth/register `
  -H "Content-Type: application/json" `
  -d '{"email":"dave@author.com","password":"password123","firstName":"Dave","lastName":"Author"}'
```
* **Expected Status:** `201 Created`

#### Test 2.4: Register Admin Carol
* **Method & URL:** `POST http://localhost:5000/auth/register`
* **Request Payload:**
```json
{
  "email": "admin@blog.com",
  "password": "password123",
  "firstName": "Carol",
  "lastName": "Admin"
}
```
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/auth/register `
  -H "Content-Type: application/json" `
  -d '{"email":"admin@blog.com","password":"password123","firstName":"Carol","lastName":"Admin"}'
```
* **Expected Status:** `201 Created`

*(Now promote Alice & Dave to `author`, and Carol to `admin` in MongoDB).*

#### Test 2.5: Duplicate Email Rejection
* **Method & URL:** `POST http://localhost:5000/auth/register`
* **Request Payload:** Same payload as Bob (`bob@reader.com`).
* **Expected Status:** `400 Bad Request`
* **Expected Response:**
```json
{
  "status": "error",
  "message": "Email already registered!"
}
```

#### Test 2.6: Validation — Password Under 6 Characters
* **Method & URL:** `POST http://localhost:5000/auth/register`
* **Request Payload:**
```json
{
  "email": "short@example.com",
  "password": "123",
  "firstName": "Short",
  "lastName": "Pass"
}
```
* **Expected Status:** `400 Bad Request`

#### Test 2.7: Login with Incorrect Password
* **Method & URL:** `POST http://localhost:5000/auth/login`
* **Request Payload:**
```json
{
  "email": "alice@blog.com",
  "password": "wrongpassword"
}
```
* **Expected Status:** `400 Bad Request`
* **Expected Response:**
```json
{
  "status": "error",
  "message": "Invalid Credentials"
}
```

#### Test 2.8: Login to Obtain JWT Tokens
Log in with each account and copy the `"token"` from `data.token`:
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

#### Test 2.9: Fetch Profile (`GET /user/iam`)
* **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Command:**
```powershell
curl.exe -i -X GET http://localhost:5000/user/iam `
  -H "Authorization: Bearer <ALICE_TOKEN>"
```
* **Expected Status:** `200 OK`
* **Verify:** Returns Alice's user profile with `role: "author"`.

#### Test 2.10: User Logout (`POST /auth/logout`)
* **Method & URL:** `POST http://localhost:5000/auth/logout`
* **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/auth/logout `
  -H "Authorization: Bearer <ALICE_TOKEN>"
```
* **Expected Status:** `200 OK`
* **Expected Response:**
```json
{
  "status": "success",
  "message": "User logged out successfully"
}
```

---

### Section 3: Role-Based Access Control (RBAC)

#### Test 3.1: Reader Forbidden from Creating Post
* **Method & URL:** `POST http://localhost:5000/api/posts`
* **Headers:** `Authorization: Bearer <BOB_TOKEN>`
* **Request Payload:**
```json
{
  "title": "Bob Illegal Post",
  "content": "Readers should not have authoring permissions."
}
```
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/api/posts `
  -H "Authorization: Bearer <BOB_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"title":"Bob Illegal Post","content":"Readers should not have authoring permissions."}'
```
* **Expected Status:** `403 Forbidden`
* **Expected Response:**
```json
{
  "status": "error",
  "message": "you don't have the permission to perform this action."
}
```

#### Test 3.2: Unauthenticated Visitor Forbidden
* **Method & URL:** `POST http://localhost:5000/api/posts`
* **Headers:** None (No Token)
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/api/posts `
  -H "Content-Type: application/json" `
  -d '{"title":"Anon Post","content":"No credentials."}'
```
* **Expected Status:** `401 Unauthorized`

#### Test 3.3: Author Alice Creates Draft Post
* **Method & URL:** `POST http://localhost:5000/api/posts`
* **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Request Payload:**
```json
{
  "title": "Alice Confidential Draft",
  "content": "This draft should be hidden from public and other authors.",
  "status": "draft",
  "tags": ["golang", "draft", "internal"]
}
```
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/api/posts `
  -H "Authorization: Bearer <ALICE_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"title":"Alice Confidential Draft","content":"This draft should be hidden from public and other authors.","status":"draft","tags":["golang","draft","internal"]}'
```
* **Expected Status:** `201 Created`
* **Action:** Save returned `data.id` as `$ALICE_DRAFT_ID`.

#### Test 3.4: Author Alice Creates Published Post
* **Method & URL:** `POST http://localhost:5000/api/posts`
* **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Request Payload:**
```json
{
  "title": "Mastering Go Concurrency",
  "content": "Goroutines and buffered channels provide powerful concurrency primitives.",
  "status": "published",
  "tags": ["golang", "concurrency", "backend"]
}
```
* **Command:**
```powershell
curl.exe -i -X POST http://localhost:5000/api/posts `
  -H "Authorization: Bearer <ALICE_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"title":"Mastering Go Concurrency","content":"Goroutines and buffered channels provide powerful concurrency primitives.","status":"published","tags":["golang","concurrency","backend"]}'
```
* **Expected Status:** `201 Created`
* **Action:** Save returned `data.id` as `$ALICE_PUBLISHED_ID`.

#### Test 3.5: Public Feed Hides Draft Posts
* **Method & URL:** `GET http://localhost:5000/api/posts`
* **Headers:** None
* **Command:**
```powershell
curl.exe -i -X GET http://localhost:5000/api/posts
```
* **Expected Status:** `200 OK`
* **Verify:**
  * `"Mastering Go Concurrency"` is present.
  * `"Alice Confidential Draft"` is **NOT** present.

#### Test 3.6: Public Cannot View Draft by ID
* **Method & URL:** `GET http://localhost:5000/api/posts/<ALICE_DRAFT_ID>`
* **Headers:** None
* **Expected Status:** `404 Not Found`

#### Test 3.7: Author Alice Views Her Own Draft by ID
* **Method & URL:** `GET http://localhost:5000/api/posts/<ALICE_DRAFT_ID>`
* **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Expected Status:** `200 OK`
* **Verify:** Alice can inspect her own draft.

#### Test 3.8: Admin Carol Views Alice's Draft by ID
* **Method & URL:** `GET http://localhost:5000/api/posts/<ALICE_DRAFT_ID>`
* **Headers:** `Authorization: Bearer <CAROL_TOKEN>`
* **Expected Status:** `200 OK`
* **Verify:** Admin has visibility into any author's draft.

---

### Section 4: Multi-Tenant Data Isolation (Cross-Author Security)

#### Test 4.1: Author Dave Creates His Own Post
* **Method & URL:** `POST http://localhost:5000/api/posts`
* **Headers:** `Authorization: Bearer <DAVE_TOKEN>`
* **Request Payload:**
```json
{
  "title": "Daves Architectural Insights",
  "content": "Designing resilient microservices in Go.",
  "status": "published",
  "tags": ["architecture", "microservices"]
}
```
* **Expected Status:** `201 Created`
* **Action:** Save `data.id` as `$DAVE_POST_ID`.

#### Test 4.2: Dave Forbidden from Updating Alice's Post
* **Method & URL:** `PATCH http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID>`
* **Headers:** `Authorization: Bearer <DAVE_TOKEN>`
* **Request Payload:**
```json
{
  "title": "Hacked by Dave"
}
```
* **Command:**
```powershell
curl.exe -i -X PATCH http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID> `
  -H "Authorization: Bearer <DAVE_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"title":"Hacked by Dave"}'
```
* **Expected Status:** `404 Not Found` or `403 Forbidden`
* **Verify:** Alice's post was **not** updated.

#### Test 4.3: Dave Forbidden from Deleting Alice's Post
* **Method & URL:** `DELETE http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID>`
* **Headers:** `Authorization: Bearer <DAVE_TOKEN>`
* **Expected Status:** `404 Not Found` or `403 Forbidden`
* **Verify:** Alice's post is intact.

#### Test 4.4: Author Dashboard Scoped to Owner (`GET /api/my-posts`)
* **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Command:**
```powershell
curl.exe -i -X GET http://localhost:5000/api/my-posts `
  -H "Authorization: Bearer <ALICE_TOKEN>"
```
* **Expected Status:** `200 OK`
* **Verify:** Returns Alice's draft and published posts. Dave's post does not appear.

#### Test 4.5: Author Alice Updates Her Own Post
* **Method & URL:** `PATCH http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID>`
* **Headers:** `Authorization: Bearer <ALICE_TOKEN>`
* **Request Payload:**
```json
{
  "title": "Mastering Go Concurrency in 2026",
  "tags": ["golang", "concurrency", "updated"]
}
```
* **Command:**
```powershell
curl.exe -i -X PATCH http://localhost:5000/api/posts/<ALICE_PUBLISHED_ID> `
  -H "Authorization: Bearer <ALICE_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"title":"Mastering Go Concurrency in 2026","tags":["golang","concurrency","updated"]}'
```
* **Expected Status:** `200 OK`
* **Verify:** `data.title == "Mastering Go Concurrency in 2026"`.

---

### Section 5: Administrator Moderation Privileges

#### Test 5.1: Non-Admins Forbidden from Admin Feed
* **Method & URL:** `GET http://localhost:5000/api/admin/posts`
* **Headers:** `Authorization: Bearer <BOB_TOKEN>` or `<ALICE_TOKEN>`
* **Expected Status:** `403 Forbidden`

#### Test 5.2: Admin Carol Views All Posts Across All Authors
* **Method & URL:** `GET http://localhost:5000/api/admin/posts`
* **Headers:** `Authorization: Bearer <CAROL_TOKEN>`
* **Command:**
```powershell
curl.exe -i -X GET http://localhost:5000/api/admin/posts `
  -H "Authorization: Bearer <CAROL_TOKEN>"
```
* **Expected Status:** `200 OK`
* **Verify:** Contains posts from Alice (both draft & published) and Dave.

#### Test 5.3: Admin Carol Updates Another Author's Post
* **Method & URL:** `PATCH http://localhost:5000/api/posts/<DAVE_POST_ID>`
* **Headers:** `Authorization: Bearer <CAROL_TOKEN>`
* **Request Payload:**
```json
{
  "title": "Daves Architectural Insights (Moderated by Admin)"
}
```
* **Command:**
```powershell
curl.exe -i -X PATCH http://localhost:5000/api/posts/<DAVE_POST_ID> `
  -H "Authorization: Bearer <CAROL_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"title":"Daves Architectural Insights (Moderated by Admin)"}'
```
* **Expected Status:** `200 OK`
* **Verify:** Admin overrides ownership successfully.

#### Test 5.4: Admin Carol Deletes Post for Moderation
* **Method & URL:** `DELETE http://localhost:5000/api/admin/posts/<DAVE_POST_ID>`
* **Headers:** `Authorization: Bearer <CAROL_TOKEN>`
* **Command:**
```powershell
curl.exe -i -X DELETE http://localhost:5000/api/admin/posts/<DAVE_POST_ID> `
  -H "Authorization: Bearer <CAROL_TOKEN>"
```
* **Expected Status:** `200 OK`
* **Verify:** Subsequent `GET /api/posts/<DAVE_POST_ID>` returns `404 Not Found`.

---

### Section 6: Search, Tag Filtering & Cursor Pagination

#### Test 6.1: Search by Title Keyword
* **Method & URL:** `GET http://localhost:5000/api/posts?search=Concurrency`
* **Expected Status:** `200 OK`
* **Verify:** Only posts containing `"Concurrency"` in the title appear.

#### Test 6.2: Filter by Tag
* **Method & URL:** `GET http://localhost:5000/api/posts?tag=golang`
* **Expected Status:** `200 OK`
* **Verify:** All returned posts contain `"golang"` in `tags`.

#### Test 6.3: Cursor Pagination
* **Method & URL:** `GET http://localhost:5000/api/posts?limit=1`
* **Expected Status:** `200 OK`
* **Verify:**
  * `data.pagination.limit == 1`
  * If next page exists: `data.pagination.has_next == true` with `data.pagination.next_cursor`.
* **Fetch Next Page:**
```powershell
curl.exe -i -X GET "http://localhost:5000/api/posts?limit=1&cursor=<NEXT_CURSOR>"
```

---

### Section 7: Edge Cases & Validation

#### Test 7.1: Malformed Hex ObjectID
* **Method & URL:** `GET http://localhost:5000/api/posts/not-a-valid-id`
* **Expected Status:** `400 Bad Request` or `404 Not Found`

#### Test 7.2: Non-Existent Post ID
* **Method & URL:** `GET http://localhost:5000/api/posts/65f1a2b3c4d5e6f7a8b9c0d1`
* **Expected Status:** `404 Not Found`
* **Expected Response:**
```json
{
  "status": "error",
  "message": "post not found"
}
```

#### Test 7.3: Broken JSON Payload
* **Method & URL:** `POST http://localhost:5000/api/posts`
* **Headers:** `Authorization: Bearer <ALICE_TOKEN>`, `Content-Type: application/json`
* **Payload:** `{"title": "Broken", ` (Malformed)
* **Expected Status:** `400 Bad Request`

#### Test 7.4: Tampered JWT Token
* **Method & URL:** `GET http://localhost:5000/user/iam`
* **Headers:** `Authorization: Bearer invalid.token.payload`
* **Expected Status:** `401 Unauthorized`
