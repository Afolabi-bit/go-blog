# Step 6.2 Implementation Guide: MongoDB Posts Repository

This guide walks you step-by-step through completing the remaining database methods for the **Posts Repository** in `internal/posts/repo.go`. 

It is written in plain English without code snippets so you can follow the architectural intent, understand how each method works, and write the implementation yourself.

---

## 1. Architectural Overview & Multi-Tenancy Rules

In this project, the repository layer is responsible for translating domain requests into MongoDB queries while enforcing data isolation and security boundaries:

- **Public Audience**: Can only access posts whose status is marked as published. Drafts must never leak to public queries.
- **Authors**: Have their own dedicated dashboard query that retrieves both drafts and published articles, strictly filtered by their personal author ID.
- **Administrators**: Have system-wide moderation access and can browse, filter, inspect, update, or remove any post across all authors and statuses.
- **Ownership Scoping on Mutations**: When updating or deleting posts, if an author ID is provided, the query selector must include both the post ID and author ID. If a user tries to alter a post they do not own, the query finds zero matching records. If an administrator performs the operation, the author ID parameter is omitted, allowing them to modify any post.

---

## 2. Package Dependencies & Setup

Ensure your file header in `internal/posts/repo.go` imports the necessary standard library and driver packages:

1. **Context Package**: For deadline propagation and request cancellation across all database operations.
2. **Errors & Fmt Packages**: For detecting MongoDB-specific sentinel errors and creating wrapped, descriptive error messages.
3. **Strings Package**: For trimming whitespace from incoming search keywords.
4. **Time Package**: For generating current timestamps during update operations.
5. **MongoDB Driver Packages**:
   - `bson`: For constructing MongoDB query maps, update operators, and regular expression patterns.
   - `primitive`: For working with MongoDB Object IDs and parsing hexadecimal ID strings (note that the conversion function lives under `primitive`, not `bson`).
   - `mongo`: For accessing collection operations, database handles, and sentinel error values like the no-documents error.
   - `options`: For configuring sorting directions, pagination limits, and return-document preferences in find and update calls.

---

## 3. Detailed Method Specifications

### Method 1: `ListPublished` (Public Feed)

**Objective**: Retrieve published posts for public visitors, supporting cursor-based pagination, title keyword search, and tag filtering.

1. **Method Signature**:
   - Receiver: Pointer to the repository struct.
   - Parameters: Context, next cursor string, maximum limit integer, and the post filter struct defined in `model.go`.
   - Returns: A slice of post structs and an error.

2. **Construct the Query Filter**:
   - Initialize an empty BSON query document.
   - Set the `status` field strictly to the published status constant.
   - **Cursor Pagination**: If the incoming cursor string is not empty:
     - Convert the cursor string into a MongoDB Object ID using the hex parsing utility from the primitive package.
     - If parsing fails, return an informative error wrapping the failure.
     - If parsing succeeds, add a condition to the query specifying that the `_id` field must be less than the parsed Object ID. Using "less than" ensures that as a user scrolls forward, they retrieve posts created earlier in time when sorted newest-first.
   - **Search Query**: If the search string in the filter is not blank:
     - Trim surrounding whitespace.
     - Add a condition on the `title` field using a BSON regular expression pattern with case-insensitive matching enabled.
   - **Tag Filter**: If the tag string in the filter is not blank:
     - Add a condition matching documents where the `tags` array contains this tag string.

3. **Configure Options & Execute**:
   - Create a find options configuration.
   - Set the sort order on `_id` to descending (negative one), so newer posts always come first.
   - Set the query limit. If the incoming limit is zero or negative, apply a sensible default such as ten or twenty items.
   - Execute the find operation against the posts collection.
   - Ensure the returned cursor is closed using a deferred call when the method returns.

4. **Decode & Return**:
   - Initialize an empty slice of post structs (ensure you return an empty slice rather than nil if no documents are found).
   - Iterate through the cursor using its next method.
   - Decode each document into a post variable and append it to your slice.
   - After the loop finishes, verify that the cursor did not terminate prematurely due to an iteration error.
   - Return the populated slice and nil.

---

### Method 2: `ListByAuthor` (Author Personal Dashboard)

**Objective**: Allow an authenticated author to view all posts they have created, including both drafts and published articles.

1. **Method Signature**:
   - Receiver: Pointer to the repository struct.
   - Parameters: Context, author Object ID, next cursor string, and maximum limit integer.
   - Returns: A slice of post structs and an error.

2. **Construct the Query Filter**:
   - Initialize a query document.
   - Set the `author_id` field to strictly equal the provided author Object ID. This guarantees complete isolation between authors.
   - If the cursor string is not empty, parse it into an Object ID and add the "less than" condition on the `_id` field for chronological pagination.

3. **Configure Options, Execute & Decode**:
   - Apply descending sorting on `_id` and apply the query limit.
   - Run the find query against the collection.
   - Defer closing the cursor.
   - Iterate and decode all matching posts into a slice.
   - Return the slice and any potential error.

---

### Method 3: `ListAllAdmin` (Admin Moderation Feed)

**Objective**: Give administrators system-wide visibility across all posts regardless of who wrote them.

1. **Method Signature**:
   - Receiver: Pointer to the repository struct.
   - Parameters: Context, next cursor string, maximum limit integer, and the post filter struct.
   - Returns: A slice of post structs and an error.

2. **Construct the Query Filter**:
   - Initialize an open query document.
   - Unlike the public listing, do not force the status to published.
   - If the filter specifies a particular status (such as draft or published), add that status requirement to the filter. If no status is specified, leave it open so all statuses match.
   - If a search query or tag filter is provided, apply the same regex and tag matching conditions used in `ListPublished`.
   - If a cursor string is provided, apply the cursor pagination condition.

3. **Configure Options, Execute & Decode**:
   - Set descending sorting on `_id` and apply the limit.
   - Execute the find operation and decode the matching documents into a slice of posts.
   - Return the slice and nil.

---

### Method 4: `GetByID` (Single Post Retrieval)

**Objective**: Retrieve a single post document by its unique identifier.

1. **Method Signature**:
   - Receiver: Pointer to the repository struct.
   - Parameters: Context and the post Object ID.
   - Returns: A single post struct and an error.

2. **Build Query & Execute**:
   - Create a query document matching `_id` against the incoming post Object ID.
   - Call the single-document find method on the collection.

3. **Decode & Handle Errors**:
   - Create a post struct variable and decode the result into it.
   - If the operation returns an error:
     - Check if the error matches MongoDB's sentinel "no documents" error. If it does, return that error directly so higher layers can translate it into an HTTP 404 response.
     - For any other database failure, return a wrapped error explaining that the lookup failed.
   - If successful, return the post and nil.

---

### Method 5: `Update` (Scoped Partial Update)

**Objective**: Apply partial updates to a post while strictly enforcing ownership when requested by an author.

1. **Method Signature**:
   - Receiver: Pointer to the repository struct.
   - Parameters: Context, post Object ID, an optional author Object ID (passed as a pointer), and the update request struct.
   - Returns: The updated post struct and an error.

2. **Construct the Filter with Ownership**:
   - Start with `_id` matching the target post Object ID.
   - **Ownership Check**: If the author ID pointer is not nil, dereference it and add `author_id` to the query filter. This ensures an author can only update their own post. If an administrator is updating, the pointer is nil, matching solely on `_id`.

3. **Construct the Update Document**:
   - Build a map representing fields to modify using MongoDB's set operator.
   - Inspect each pointer in the update request struct:
     - If the title pointer is not nil, add title to the update set.
     - If the content pointer is not nil, add content to the update set.
     - If the status pointer is not nil, add status to the update set.
     - If the tags pointer is not nil, add tags to the update set.
   - Always add the `updated_at` field set to the current UTC time.

4. **Execute Find-and-Modify**:
   - If no fields were provided for update, you can simply call `GetByID` and return the existing post.
   - Otherwise, call the collection's find-one-and-update method with the filter and update document.
   - Pass find-one-and-update options configured to return the document after the modification is applied (rather than before).
   - Decode the result into a post struct.
   - If no document is returned or if the error is "no documents found", return that error so the caller knows the post was either not found or the author lacked permission.
   - Return the updated post struct and nil.

---

### Method 6: `Delete` (Scoped Post Removal)

**Objective**: Permanently delete a post while enforcing author ownership.

1. **Method Signature**:
   - Receiver: Pointer to the repository struct.
   - Parameters: Context, post Object ID, and an optional author Object ID (passed as a pointer).
   - Returns: An error.

2. **Construct the Filter with Ownership**:
   - Add `_id` matching the post Object ID.
   - If the author ID pointer is not nil, add `author_id` matching the author Object ID. If nil, allow deletion by post ID alone.

3. **Execute Deletion & Verify**:
   - Call the collection's single-document delete method.
   - If an error occurred, return a wrapped error.
   - Inspect the delete result:
     - Check the deleted count. If the count is zero, it means no document matched the filter (either the post does not exist or the requester is not the author). Return MongoDB's "no documents" error.
     - If the count is one, return nil.

---

## 4. Key Gotchas & Best Practices

1. **Hex Parsing Function Location**:
   - In the official MongoDB Go driver, converting a string to an Object ID is done using the hex parsing function located in the `primitive` package, not the `bson` package.
2. **Resource Management**:
   - Every multi-document cursor returned by find operations must be closed with a defer statement to prevent database connection leaks.
3. **Empty Results Handling**:
   - For listing endpoints, always initialize the slice before decoding. If zero documents match, return an empty slice rather than nil so JSON responses serialize cleanly as an empty array rather than null.
4. **Pointer Checks**:
   - Always check whether optional pointer fields are nil before dereferencing them to prevent runtime panics.

---

## 5. Notes & Questions Workspace

You can use this section to write down questions or thoughts as you implement each method. Feel free to ask about any specific step!
