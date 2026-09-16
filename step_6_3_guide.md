# Step 6.3 Implementation Guide: Posts Service Layer & Business Logic

This guide walks you step-by-step through designing and implementing the **Posts Service** in `internal/posts/service.go`.

Following the project's educational format, this document is written in plain English without complete code blocks so you understand the architectural responsibilities, data flow, access-control rules, and write the Go implementation yourself.

---

## 1. Architectural Overview & Responsibilities

In our 3-tier architecture, the Service layer acts as the coordinator between incoming HTTP transport concerns (handlers) and raw database access (repositories).

While the repository only cares about executing MongoDB queries, the Service layer is responsible for:
1. **Business Validation**: Ensuring titles and contents are not blank, statuses conform strictly to allowed constants (`draft` or `published`), and pagination limits stay within reasonable bounds.
2. **Slug Generation**: Transforming raw, human-readable post titles into clean, URL-friendly slugs (e.g. `"My First Go Post!"` -> `"my-first-go-post"`).
3. **Role-Based Access Control (RBAC) & Visibility Enforcement**:
   - **Public & Readers**: May only view posts whose status is `published`.
   - **Authors**: Can create posts, browse their own draft and published articles, and edit or delete only posts they personally authored.
   - **Admins**: Have unrestricted moderation rights to view any draft, update any post, or permanently delete any post across the platform.
4. **Pagination Metadata Calculation**: Inspecting the returned slice of posts, determining if a next page exists, and formulating the next cursor for the client.

---

## 2. Package Structure, Dependencies & Setup

Ensure your `internal/posts/service.go` file declares the `posts` package and imports the necessary packages:

1. **Context Package**: For propagating deadlines, cancellation signals, and request context across service and repo boundaries.
2. **Errors & Fmt Packages**: For defining sentinel domain errors (e.g., forbidden, not found, invalid input) and formatting dynamic error messages.
3. **Strings, Regexp & Unicode Packages**: For trimming whitespace, converting strings to lowercase, and sanitizing titles into URL slugs.
4. **Time Package**: For setting created and updated timestamps.
5. **MongoDB Primitives & Errors**:
   - `primitive`: For converting and handling MongoDB `ObjectID` types.
   - `mongo`: For recognizing the sentinel `mongo.ErrNoDocuments` and mapping it to domain errors.
6. **User Domain Integration (Optional / Recommended)**:
   - If your service looks up the author's full name from the user repository, import `blog-api/internal/user`. Alternatively, author full names can be passed directly from the handler/JWT claims into the service method.

---

## 3. Sentinel Domain Errors

At the top of `service.go`, define custom sentinel errors so your HTTP handler can easily check them and return the appropriate HTTP status codes (`400 Bad Request`, `403 Forbidden`, `404 Not Found`):

1. **Not Found Error**: Indicates that the requested post does not exist in the database.
2. **Forbidden Error**: Returned when a user attempts to view a draft they do not own, or modify/delete a post belonging to another author.
3. **Invalid Input Error**: Returned when required fields (like title or content) are blank, or an invalid status value is provided.

---

## 4. Service Struct & Constructor

1. **Service Struct**:
   - Holds a pointer to `Repo` (the posts repository).
   - Optionally holds a pointer to `user.Repo` if author profile details need to be verified or fetched during post creation.
2. **Constructor Function (`NewService`)**:
   - Accepts the repository dependencies.
   - Returns a pointer to the initialized `Service` struct.

---

## 5. Helper Function: URL Slug Generation

Create an unexported helper function (e.g., `generateSlug(title string) string`):

1. **Normalize**: Convert the title string to lowercase and trim any surrounding whitespace.
2. **Replace Non-Alphanumeric Characters**:
   - Replace any character that is not an ASCII letter or digit (spaces, punctuation, symbols) with a hyphen (`-`). A regular expression such as `[^a-z0-9]+` works reliably here.
3. **Deduplicate & Trim Hyphens**:
   - Collapse consecutive hyphens into a single hyphen.
   - Trim leading and trailing hyphens from both ends of the resulting string.
4. **Fallback**: If the resulting slug is completely empty (for instance, if the title consisted entirely of emojis or special characters), generate a fallback slug using a static prefix combined with the current Unix timestamp to guarantee a valid string.

---

## 6. Detailed Method Specifications

### Method 1: `CreatePost`

**Objective**: Validate post input, generate a slug, assign ownership, and persist the post.

1. **Method Parameters**:
   - Context.
   - Author Object ID (`primitive.ObjectID`).
   - Author display name (`string`).
   - Create post request DTO (`CreatePostRequest`).
2. **Returns**:
   - The newly created `Post` struct and an error.
3. **Execution Steps**:
   - **Trim & Validate**: Trim whitespace from `Title` and `Content`. If either is empty, return your invalid input error.
   - **Default Status**: Inspect `req.Status`. If blank, default it to `StatusDraft`. If not blank, verify that it equals either `StatusDraft` or `StatusPublished`. If it is any other value, return an invalid input error.
   - **Sanitize Tags**: Iterate through the tags slice, trimming whitespace from each tag. Omit any tags that become empty strings after trimming.
   - **Generate Slug**: Pass the trimmed title into your slug generation helper.
   - **Assemble Entity**: Build the `Post` struct:
     - Set `AuthorID` to the provided author Object ID.
     - Set `AuthorName` to the author display name.
     - Set `Title`, `Content`, `Slug`, `Status`, and cleaned `Tags`.
     - Set `CreatedAt` and `UpdatedAt` to the current UTC timestamp.
   - **Persist**: Call the repository's `CreatePost` method and return the result.

---

### Method 2: `GetPostByID`

**Objective**: Retrieve a single post while strictly applying visibility rules between public users, authors, and admins.

1. **Method Parameters**:
   - Context.
   - Post ID string (hex representation from the URL parameter).
   - Optional requester user ID (passed as a string or pointer to Object ID; empty/nil if unauthenticated public visitor).
   - Optional requester role (`string`, e.g. `"reader"`, `"author"`, `"admin"`, or empty if guest).
2. **Returns**:
   - A `Post` struct and an error.
3. **Execution Steps**:
   - **Parse ID**: Convert the post ID string into a `primitive.ObjectID`. If conversion fails, return your invalid input or not found error.
   - **Retrieve from Repo**: Call the repository's `GetByID` method. If the repo returns `mongo.ErrNoDocuments`, translate and return your domain not found error.
   - **Visibility & RBAC Rules**:
     - If the post's `Status` is `StatusPublished`: return the post immediately (it is public).
     - If the post's `Status` is `StatusDraft`:
       - If the requester is an administrator (`role == RoleAdmin`): grant access and return the post.
       - If the requester is authenticated and their user ID matches the post's `AuthorID`: grant access and return the post.
       - Otherwise (the requester is a guest, a reader, or a different author): deny access. Return your domain not found error or forbidden error. *(Returning "not found" is common in secure systems to prevent leaking the existence of draft titles).*

---

### Method 3: `ListPublicPosts`

**Objective**: Retrieve a paginated list of published posts with search and tag filters for public readers.

1. **Method Parameters**:
   - Context.
   - Next cursor string.
   - Desired limit (`int64`).
   - Post filter DTO (`PostFilter`).
2. **Returns**:
   - A slice of `Post` structs, a `PaginationMeta` struct, and an error.
3. **Execution Steps**:
   - **Clamp Limits**: If limit is less than or equal to zero, default to 10. If limit exceeds a safe maximum (such as 50 or 100), clamp it to prevent excessive database load.
   - **Fetch**: Call the repository's `ListPublished` method.
   - **Build Pagination Metadata**:
     - Initialize `PaginationMeta` with the requested limit and count equal to the number of posts returned.
     - Determine `HasNext`: If the length of the returned slice equals the requested limit, set `HasNext` to true and populate `NextCursor` with the hex string of the last post's `ID`. Otherwise, set `HasNext` to false and `NextCursor` to an empty string.
   - Return the slice, pagination metadata, and nil.

---

### Method 4: `ListMyPosts`

**Objective**: Allow an authenticated author to view all their personal posts (both drafts and published).

1. **Method Parameters**:
   - Context.
   - Author Object ID (`primitive.ObjectID`).
   - Next cursor string.
   - Desired limit (`int64`).
2. **Returns**:
   - A slice of `Post` structs, a `PaginationMeta` struct, and an error.
3. **Execution Steps**:
   - **Clamp Limits**: Apply default and maximum boundary checks on the limit.
   - **Fetch**: Call the repository's `ListByAuthor` method with the author Object ID.
   - **Build Pagination Metadata**: Calculate `HasNext` and `NextCursor` using the same logic as public listing.
   - Return the slice, pagination metadata, and nil.

---

### Method 5: `ListAllAdmin`

**Objective**: Allow administrators to browse, search, and filter all posts across all authors and statuses.

1. **Method Parameters**:
   - Context.
   - Next cursor string.
   - Desired limit (`int64`).
   - Post filter DTO (`PostFilter`).
2. **Returns**:
   - A slice of `Post` structs, a `PaginationMeta` struct, and an error.
3. **Execution Steps**:
   - Clamp the limit.
   - Call the repository's `ListAllAdmin` method.
   - Assemble `PaginationMeta` with `HasNext` and `NextCursor`.
   - Return the slice and metadata.

---

### Method 6: `UpdatePost`

**Objective**: Apply partial modifications to a post while strictly enforcing author ownership or admin override.

1. **Method Parameters**:
   - Context.
   - Post ID string.
   - Requester user ID (`primitive.ObjectID`).
   - Requester role (`string`).
   - Update request DTO (`UpdatePostRequest`).
2. **Returns**:
   - The updated `Post` struct and an error.
3. **Execution Steps**:
   - **Parse ID**: Convert the post ID string to a `primitive.ObjectID`. Return an error if invalid.
   - **Validate Input**:
     - If `req.Status` is provided (not nil), verify that its value is either `StatusDraft` or `StatusPublished`.
     - If `req.Title` is provided (not nil), verify that trimming it does not produce an empty string.
     - If `req.Content` is provided (not nil), verify that trimming it does not produce an empty string.
   - **Determine Ownership Scope**:
     - If the requester's role is `RoleAdmin`: set the author ID pointer to `nil` (allowing the query to match solely on post ID).
     - If the requester's role is `RoleAuthor`: pass a pointer to the requester's user ID (`&requesterID`) so the query strictly matches both `_id` and `author_id`.
   - **Invoke Repo**: Call the repository's `Update` method.
   - **Handle Result**:
     - If the repo returns `mongo.ErrNoDocuments`, return your domain not found or forbidden error (the post either doesn't exist or the author is not the owner).
     - Return the updated post and nil.

---

### Method 7: `DeletePost`

**Objective**: Permanently remove a post while enforcing author ownership or admin override.

1. **Method Parameters**:
   - Context.
   - Post ID string.
   - Requester user ID (`primitive.ObjectID`).
   - Requester role (`string`).
2. **Returns**:
   - An error.
3. **Execution Steps**:
   - **Parse ID**: Convert the post ID string to a `primitive.ObjectID`.
   - **Determine Ownership Scope**:
     - If `role == RoleAdmin`: author ID pointer is `nil`.
     - If `role == RoleAuthor`: author ID pointer is `&requesterID`.
   - **Invoke Repo**: Call the repository's `Delete` method.
   - **Handle Result**:
     - If the repo returns `mongo.ErrNoDocuments`, return your domain not found or forbidden error.
     - Return nil on success.

---

## 7. Key Gotchas & Best Practices

1. **Never Trust Raw Client Limits**: Always check `limit <= 0` to apply a sensible default (e.g., 10), and cap upper limits (e.g., `limit > 100`) so an external client cannot trigger an unindexed full-table scan.
2. **Draft Leakage Prevention**: Never return a draft post to unauthenticated users or users with the `reader` role. If an unprivileged user asks for a draft by ID, return a generic `404 Not Found` rather than `403 Forbidden` to prevent enumeration attacks.
3. **Slice vs Nil Pagination**: If zero posts match a query, always ensure the returned slice is empty (`[]Post{}`) rather than `nil` before packaging it into HTTP responses.
4. **Pointer Dereferencing**: In `UpdatePost`, always confirm pointers inside `UpdatePostRequest` are not `nil` before reading their contents.
