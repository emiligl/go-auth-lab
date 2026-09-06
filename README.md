# Go Authentication Lab

Hands-on authentication lab built in Go to understand how authentication works at HTTP, application and storage level.

The lab starts with password hashing and stateful session-based authentication, using both in-memory storage and Redis. The same authentication flow is exercised from a Go HTTP client and from a browser.

JWT authentication will be implemented as the next stage of the lab, using the same client/server structure to compare both approaches.

---

## 🎯 Goals

The objective of this lab is not only to implement authentication, but to understand what happens between client and server.

Topics covered:

- HTTP client/server communication
- HTTP handlers and requests/responses
- Password hashing with bcrypt
- Salt and password verification
- Session IDs
- Stateful authentication
- Server-side session storage
- In-memory sessions
- Redis-backed sessions
- Session TTL
- HTTP cookies
- `HttpOnly`
- `Secure`
- `SameSite`
- Browser authentication
- Go HTTP client with `CookieJar`
- Login / authenticated requests / logout
- Redis inspection using `redis-cli`
- Difference between cookies, `localStorage` and `sessionStorage`

---

# Architecture

The lab deliberately separates the client from the server.

```text
                 ┌─────────────────┐
                 │   Go Client     │
                 │                 │
                 │   HTTP Client   │
                 │   CookieJar     │
                 └────────┬────────┘
                          │
                          │ HTTP
                          ▼
                 ┌─────────────────┐
                 │   Go Server     │
                 │                 │
                 │   /health       │
                 │   /login        │
                 │   /profile      │
                 │   /logout       │
                 └────────┬────────┘
                          │
                          │ session
                          ▼
                 ┌─────────────────┐
                 │      Redis      │
                 │                 │
                 │ sess:<id>       │
                 │ TTL: 15 min     │
                 └─────────────────┘
```

The browser is also used as a real HTTP client:

```text
Browser
   │
   │ POST /login
   ▼
Go Server
   │
   ▼
Redis
   │
   ▼
Set-Cookie: sid=<session-id>
   │
   ▼
Browser
   │
   │ GET /profile
   │ Cookie: sid=<session-id>
   ▼
Go Server
   │
   ▼
Redis
```

---

# Project Structure

```text
.
├── go.mod
├── go.sum
│
├── server/
│   ├── main.go
│   ├── auth.go
│   ├── session_memory.go
│   ├── session_redis.go
│   └── index.html
│
└── client/
    └── main.go
```

The project intentionally keeps the memory and Redis implementations separately so they can be compared.

---

# 1. HTTP Server

The server exposes several endpoints:

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/health` | Health check |
| POST | `/login` | Authenticate user and create session |
| GET | `/profile` | Access authenticated user |
| POST | `/logout` | Invalidate session |
| GET | `/` | Browser client |

The handlers use the standard Go `net/http` package.

Example:

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
}
```

The handler receives:

```go
func handler(w http.ResponseWriter, r *http.Request)
```

where:

- `w` is the HTTP response writer
- `r` contains the incoming HTTP request

---

# 2. Password Hashing

Passwords are never stored as plaintext.

The lab uses bcrypt:

```go
func HashPassword(pw string) (string, error) {
    b, err := bcrypt.GenerateFromPassword(
        []byte(pw),
        bcrypt.DefaultCost,
    )

    return string(b), err
}
```

Password verification is performed with:

```go
func CheckPassword(hash, pw string) bool {
    return bcrypt.CompareHashAndPassword(
        []byte(hash),
        []byte(pw),
    ) == nil
}
```

## Important concepts

Bcrypt:

- generates a random salt
- includes the salt in the resulting hash
- stores the cost factor in the hash
- is deliberately expensive to compute
- does not allow recovering the original password

The stored value contains enough information to verify the password, but it is not the original password.

Conceptually:

```text
password
    │
    ▼
bcrypt
    │
    ├── salt
    ├── cost
    └── derived hash
    │
    ▼
stored hash
```

When the user logs in:

```text
password supplied
       │
       ▼
CompareHashAndPassword
       │
       ▼
stored bcrypt hash
       │
       ▼
match / reject
```

---

# 3. Session Authentication

After successful authentication, the server creates a random session ID.

```go
func newSessionID() string {
    b := make([]byte, 32)

    if _, err := rand.Read(b); err != nil {
        panic(err)
    }

    return hex.EncodeToString(b)
}
```

The session ID is not the user's identity itself.

It is an opaque identifier:

```text
session ID
    │
    ▼
server-side session
    │
    ▼
user
```

---

# 4. In-Memory Sessions

The first implementation stores sessions directly inside the Go process:

```go
var sessions = make(map[string]User)
```

Conceptually:

```text
sid=ABC123
    │
    ▼
Go process memory
    │
    ▼
User Alice
```

This is simple and useful for understanding stateful authentication.

However, the data disappears when the process stops.

It also becomes problematic with multiple server instances:

```text
             Load Balancer
              /          \
             ▼            ▼
        Server A       Server B
        memory A       memory B
```

A session created on Server A does not automatically exist on Server B.

---

# 5. Redis Sessions

The second implementation stores sessions in Redis.

```go
err = rdb.Set(
    ctx,
    "sess:"+sessionID,
    data,
    15*time.Minute,
).Err()
```

The session is stored as:

```text
sess:<session-id>
```

Example:

```text
sess:8f7c2e...
```

with a TTL of 15 minutes.

Retrieving the session:

```go
data, err := rdb.Get(
    ctx,
    "sess:"+sessionID,
).Result()
```

The flow becomes:

```text
Cookie
  │
  ▼
session ID
  │
  ▼
Redis GET
  │
  ▼
JSON
  │
  ▼
User
```

Redis allows multiple application instances to share the same session store:

```text
             Load Balancer
              /          \
             ▼            ▼
        Go Server A   Go Server B
              \          /
               \        /
                 Redis
                   │
             shared sessions
```

---

# 6. Session TTL

Sessions stored in Redis have a 15-minute TTL:

```go
15 * time.Minute
```

Redis automatically expires the session when the TTL reaches zero.

This provides automatic session expiration.

The TTL can be inspected with:

```bash
redis-cli
```

```redis
TTL sess:<session-id>
```

Example:

```text
899
```

---

# 7. HTTP Cookies

The session ID is sent to the browser as a cookie:

```go
http.SetCookie(w, &http.Cookie{
    Name:     "sid",
    Value:    sessionID,
    HttpOnly: true,
    Secure:   false,
    SameSite: http.SameSiteLaxMode,
    Path:     "/",
    MaxAge:   15 * 60,
})
```

The browser receives:

```http
Set-Cookie: sid=<session-id>
```

On subsequent requests it automatically sends:

```http
Cookie: sid=<session-id>
```

The server reads it with:

```go
cookie, err := r.Cookie("sid")
```

---

# 8. Cookie Security Attributes

### HttpOnly

```go
HttpOnly: true
```

Prevents JavaScript from reading the cookie.

This helps reduce the impact of some XSS attacks against authentication cookies.

---

### Secure

```go
Secure: true
```

The browser only sends the cookie over HTTPS.

For local development over:

```text
http://localhost
```

the lab uses:

```go
Secure: false
```

Production authentication should use HTTPS and `Secure: true`.

---

### SameSite

The lab uses:

```go
SameSite: http.SameSiteLaxMode
```

This controls when browsers send the cookie in cross-site requests and helps mitigate CSRF-related attacks.

---

# 9. Go HTTP Client

The lab also includes a Go client.

A `CookieJar` allows the client to behave similarly to a browser:

```go
jar, _ := cookiejar.New(nil)

client := &http.Client{
    Jar: jar,
}
```

After login, the client stores:

```text
sid=<session-id>
```

and automatically sends it with subsequent requests.

This allows the following flow:

```text
POST /login
     │
     ▼
Set-Cookie
     │
     ▼
CookieJar
     │
     ▼
GET /profile
     │
     ▼
Cookie automatically attached
```

---

# 10. Browser Client

The server also serves a small HTML client.

The browser can perform:

```text
Login
Profile
Logout
```

The login request is:

```javascript
fetch("/login", {
    method: "POST",
    headers: {
        "Content-Type": "application/json"
    },
    body: JSON.stringify({
        username: username,
        password: password
    })
});
```

The browser automatically manages the authentication cookie.

This makes it possible to inspect the authentication flow using browser DevTools.

---

# 11. Observing HTTP

Using browser DevTools → Network, the authentication flow can be inspected directly.

### Login

```http
POST /login
Content-Type: application/json
```

Request body:

```json
{
    "username": "alice",
    "password": "secret123"
}
```

Response:

```http
Set-Cookie: sid=<session-id>
```

### Authenticated request

```http
GET /profile
Cookie: sid=<session-id>
```

This makes the client/server boundary visible instead of hiding it behind a framework.

---

# 12. Redis Inspection

Redis can be inspected manually.

List sessions:

```bash
redis-cli
```

```redis
KEYS sess:*
```

Inspect a session:

```redis
GET sess:<session-id>
```

Inspect its TTL:

```redis
TTL sess:<session-id>
```

Delete a session manually:

```redis
DEL sess:<session-id>
```

Delete everything in the current Redis database:

```redis
FLUSHDB
```

`FLUSHDB` should only be used carefully, especially outside a local development environment.

---

# 13. Logout

Logout invalidates the session immediately.

The server:

1. Reads the `sid` cookie.
2. Deletes the corresponding Redis session.
3. Expires the browser cookie.

Conceptually:

```text
POST /logout
      │
      ▼
Cookie sid
      │
      ▼
Redis DEL sess:<sid>
      │
      ▼
Expire cookie
```

After logout:

```text
GET /profile
      │
      ▼
session not found
      │
      ▼
401 Unauthorized
```

This is different from TTL expiration:

- Logout invalidates the session immediately.
- TTL automatically invalidates an inactive session after its lifetime.

---

# 14. Cookies vs Web Storage

The lab also explores the difference between:

- Cookies
- `localStorage`
- `sessionStorage`

Cookies are automatically included in HTTP requests.

```text
Browser
   │
   │ Cookie: sid=ABC
   ▼
Server
```

`localStorage` and `sessionStorage` are browser-side storage mechanisms and are not automatically sent to the server.

JavaScript can read both:

```javascript
localStorage.getItem("token")
sessionStorage.getItem("token")
```

but an `HttpOnly` cookie cannot be read by JavaScript.

This distinction becomes particularly important when discussing JWT storage.

---

# 15. Stateful Authentication

The current implementation is stateful.

The client stores only an identifier:

```text
sid=ABC
```

The actual session state lives on the server side:

```text
ABC → Alice
```

With Redis:

```text
ABC → Redis → Alice
```

The server must therefore maintain session state.

---

# 16. Authentication Flow

Complete session-based authentication:

```text
                 LOGIN
                   │
                   ▼
              username/password
                   │
                   ▼
                bcrypt
                   │
              password OK
                   │
                   ▼
             generate SID
                   │
                   ▼
                Redis
            sess:<sid> → User
                   │
                   ▼
              Set-Cookie
                   │
                   ▼
                CLIENT
                   │
                   │ Cookie: sid
                   ▼
              GET /profile
                   │
                   ▼
                Redis
                   │
                   ▼
                  User
                   │
                   ▼
               200 OK
```

Logout:

```text
POST /logout
     │
     ▼
Cookie sid
     │
     ▼
Redis DEL
     │
     ▼
Expire cookie
```

---

# 17. Running the Lab

## Start Redis

Make sure Redis is running:

```bash
sudo systemctl start redis-server
```

Check:

```bash
redis-cli ping
```

Expected:

```text
PONG
```

---

## Start the Go server

From the project root:

```bash
go run ./server
```

The server runs on:

```text
http://localhost:8080
```

---

## Browser

Open:

```text
http://localhost:8080
```

Use the login form and then access the profile.

---

## Go Client

In another terminal:

```bash
go run ./client
```

The client performs the authentication flow programmatically.

---

# 18. Session vs JWT

The next stage of the lab will implement JWT authentication using the same client/server structure.

Current session-based model:

```text
Client
  │
  │ sid
  ▼
Server
  │
  ▼
Redis
  │
  ▼
User
```

JWT model:

```text
Client
  │
  │ JWT
  ▼
Server
  │
  ├── verify signature
  ├── verify expiration
  └── read claims
```

The goal is to compare the two approaches through code rather than treating JWT as a purely theoretical concept.

---

# Learning Notes

This project is intentionally built incrementally.

The main objective is understanding the mechanics behind authentication:

```text
Password
   ↓
Hash
   ↓
Authentication
   ↓
Session
   ↓
Cookie
   ↓
Authenticated request
   ↓
Session store
```

Rather than relying on a framework to hide these details.

---

# Status

## Session Authentication

- [x] HTTP server
- [x] HTTP client
- [x] Health endpoint
- [x] Password hashing
- [x] Password verification
- [x] Random session IDs
- [x] In-memory sessions
- [x] Redis sessions
- [x] Redis TTL
- [x] HTTP cookies
- [x] HttpOnly
- [x] SameSite
- [x] Browser client
- [x] Authenticated `/profile`
- [x] Logout
- [x] Redis inspection
- [x] CookieJar

## JWT Authentication

- [ ] JWT generation
- [ ] JWT claims
- [ ] JWT signature
- [ ] JWT verification
- [ ] JWT expiration
- [ ] JWT authentication middleware
- [ ] JWT client
- [ ] Browser JWT flow
- [ ] Session vs JWT comparison
