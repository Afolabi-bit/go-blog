# Blog API — Render Deployment Guide

A step-by-step guide to deploying the **Go Blog REST API** to [Render](https://render.com).

---

## Step 1: Whitelist Outbound IPs in MongoDB Atlas

Cloud hosts use dynamic IP addresses. You must allow your MongoDB Atlas cluster to accept connections from Render:

1. Log in to [MongoDB Atlas](https://cloud.mongodb.com).
2. In the left sidebar, navigate to **Security** $\rightarrow$ **Network Access**.
3. Click **Add IP Address**.
4. Select **Allow Access from Anywhere** (`0.0.0.0/0`).
5. Click **Confirm**. Wait 1–2 minutes until the entry displays as **Active**.

---

## Step 2: Push Your Code to GitHub

Render builds your service directly from your GitHub repository:

```powershell
git add .
git commit -m "feat: complete blog api with dockerfile"
git push origin main
```

---

## Step 3: Create the Web Service on Render

1. Go to [dashboard.render.com](https://dashboard.render.com) and log in with your **GitHub** account.
2. In the top right corner, click **New +** $\rightarrow$ **Web Service**.
3. Under **Connect a repository**, find and select your **`go-blog`** repository.

---

## Step 4: Configure the Service

Fill out the configuration fields:

| Field | Value | Notes |
| :--- | :--- | :--- |
| **Name** | `go-blog-api` | Or any name you prefer |
| **Region** | *Closest to your Atlas cluster* | E.g., Frankfurt (EU) or Ohio (US) |
| **Branch** | `main` | Production branch |
| **Runtime** | **Docker** | Render will automatically detect your `Dockerfile` |
| **Instance Type** | **Free** | Free tier |

---

## Step 5: Add Environment Variables

Scroll down to the **Environment Variables** section and add these 4 keys:

| Key | Value | Purpose |
| :--- | :--- | :--- |
| **`GIN_MODE`** | `release` | Disables debug logs and optimizes performance |
| **`MONGO_URI`** | *(Your Atlas connection string)* | Copy from your local `.env` |
| **`MONGO_DB_NAME`** | `blog_api_db` | Your MongoDB database name |
| **`JWT_SECRET`** | *(Your 64-character secret)* | Copy from your local `.env` |

> [!NOTE]
> Render automatically assigns a dynamic port via `PORT=10000`, which our application automatically detects. You do not need to add a `PORT` variable.

---

## Step 6: Set the Health Check & Deploy

1. Scroll to the bottom and click **Advanced**.
2. Find the **Health Check Path** field and enter:
   ```text
   /health
   ```
3. Click the **Create Web Service** button.

---

## Step 7: Verification

### 1. Monitor the Build
Render will build the multi-stage Docker image and launch your API. Once the deployment finishes, the logs will show:
```text
Server listening on port 10000
Your service is live 🎉
```

Copy your public HTTPS URL from the top-left of the page (e.g. `https://go-blog-api-xxxx.onrender.com`).

### 2. Test the Live Health Endpoint
In your browser or PowerShell terminal:
```powershell
curl.exe -i -X GET https://<YOUR-RENDER-URL>/health
```
**Expected Response (`200 OK`):**
```json
{
  "ok": true,
  "service": "go-blog",
  "time": "2026-09-18T14:40:00.000000Z"
}
```

### 3. Update Your API Client Collection
In your API client (Postman / Thunder Client / Insomnia):
1. Open your **Blog-api** collection.
2. Update the `base_url` variable:
   - **From:** `http://localhost:5000`
   - **To:** `https://<YOUR-RENDER-URL>`
3. Run **POST Register**, **POST Login**, and **GET List** to verify that your live cloud API is fully functional!
