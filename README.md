# Go Authentication Lab

A hands-on authentication lab built in Go to understand authentication at the HTTP, application, and storage levels.

The project is intentionally developed through **two Git branches**, each representing a different authentication model:

- `main` → traditional **stateful session-based authentication**
- `jwt` → **JWT-based authentication with HttpOnly cookies, Redis revocation, and refresh-token rotation**

The goal is to understand the mechanics behind each approach and compare their trade-offs.

---

## Branches

### `main` — Stateful Session Authentication

The `main` branch implements classic server-side session authentication.

The browser receives an opaque session ID in an HttpOnly cookie:

```text
Browser
   │
   │ Cookie: sid=<session-id>
   ▼
Go Server
   │
   ▼
Redis
   │
   ▼
User / Session
```

The actual authentication state lives on the server.

Topics covered:

- HTTP client/server communication
- Go `net/http`
- HTTP handlers
- Password hashing with bcrypt
- Password verification
- Random session IDs
- Stateful authentication
- In-memory sessions
- Redis-backed sessions
- Redis session TTL
- HTTP cookies
- `HttpOnly`
- `Secure`
- `SameSite`
- Browser authentication
- Go `CookieJar`
- Authenticated requests
- Logout
- Redis inspection

Authentication flow:

```text
POST /login
     │
     ▼
username + password
     │
     ▼
bcrypt verification
     │
     ▼
generate session ID
     │
     ▼
store session in Redis
     │
     ▼
Set-Cookie: sid=<session-id>
     │
     ▼
GET /profile
     │
     ▼
Redis lookup
     │
     ▼
authenticated user
```

Logout invalidates the server-side session immediately by removing it from Redis and expiring the browser cookie.

---

### `jwt` — JWT Authentication

The `jwt` branch replaces the traditional session cookie with a JWT-based authentication flow.

The current architecture uses **HttpOnly cookies**, not `localStorage`.

```text
Browser
   │
   │ Cookie: access_token=<JWT>
   ▼
Go Server
   │
   ├── Verify JWT
   ├── Validate signature
   ├── Validate expiration
   ├── Read claims
   └── Check Redis revocation
```

#### Access Token

The access token is a signed JWT containing:

```text
user_id
user_name
role
exp
jti
```

The `jti` uniquely identifies each access token.

The `exp` claim controls JWT expiration.

#### JWT Middleware

Protected endpoints use middleware so authentication logic is not duplicated in every handler.

The middleware:

1. Reads the `access_token` cookie.
2. Verifies the JWT.
3. Validates the token and its claims.
4. Extracts the username, `jti`, and expiration.
5. Checks Redis for token revocation.
6. Stores request-scoped authentication data in `context`.
7. Calls the next handler.

Conceptually:

```text
Request
   │
   ▼
access_token cookie
   │
   ▼
JWT middleware
   │
   ├── VerifyJWT()
   ├── Validate claims
   ├── Check Redis
   │      │
   │      └── revoked:<jti>
   │
   ▼
request context
   │
   ▼
profileHandler / logoutHandler / ...
```

#### Redis Revocation

JWTs are normally stateless, but this lab adds server-side revocation using Redis.

When the user logs out, the JWT's `jti` is stored as:

```text
revoked:<jti>
```

The Redis key receives a TTL based on the JWT's remaining lifetime.

Therefore:

- revocation is immediate
- the JWT itself does not need to be deleted
- Redis does not store every active JWT
- the revocation entry disappears automatically when the JWT would have expired

---

## Refresh Tokens

The `jwt` branch also implements refresh tokens.

The refresh token is an opaque, cryptographically random value rather than a JWT.

It is stored in an HttpOnly cookie:

```text
Cookie: refresh_token=<opaque-token>
```

Refresh tokens are stored server-side in Redis.

A refresh-token record contains:

```text
refresh:<token>
    ├── username
    └── session_id
```

The Redis key has a TTL representing the refresh-token lifetime.

---

## Refresh Token Rotation

Every successful refresh consumes the current refresh token and creates a new one.

```text
refresh A
    │
    │ /refresh
    ▼
used_refresh:A
    │
    ▼
refresh B
```

The new refresh token keeps the same `session_id` and inherits the remaining TTL.

This makes each refresh token effectively single-use.

---

## Refresh Token Reuse Detection

The lab also implements reuse detection.

If an already-used refresh token is presented again:

```text
refresh A
    │
    ▼
refresh:A does not exist
    │
    ▼
used_refresh:A exists
    │
    ▼
REUSE DETECTED
    │
    ▼
session:<session-id> = revoked
```

The whole refresh-token family/session is then considered compromised and revoked.

A newer refresh token belonging to the same session will subsequently fail with:

```text
Session revoked
```

This demonstrates how refresh-token rotation can detect reuse of an old token and revoke the associated token family.

---

## JWT Branch Authentication Flow

The complete flow is:

```text
                    LOGIN
                      │
                      ▼
               username/password
                      │
                      ▼
               CheckPassword()
                      │
                      ▼
             ┌────────┴────────┐
             │                 │
             ▼                 ▼
        Access JWT        Refresh Token
          + jti                 │
             │                  ▼
             │              Redis
             │          refresh:<token>
             │                  │
             ▼                  ▼
       HttpOnly cookie      session_id
             │
             └────────┬─────────┘
                      │
                      ▼
               Authenticated
                 requests
                      │
                      ▼
               JWT middleware
                      │
             ┌────────┴────────┐
             │                 │
             ▼                 ▼
        Verify JWT        Redis check
             │                 │
             │          revoked:<jti>
             │
             ▼
          context
             │
             ▼
         /profile
```

When the access token expires:

```text
Browser
   │
   │ refresh_token cookie
   ▼
POST /refresh
   │
   ▼
Redis validation
   │
   ▼
Rotate refresh token
   │
   ├── old → used_refresh
   └── new → refresh:<token>
   │
   ▼
Create new access JWT
   │
   ▼
Set new HttpOnly cookies
```

---

## Comparing the Two Branches

| Feature | `main` | `jwt` |
|---|---|---|
| Authentication model | Stateful session | JWT + refresh tokens |
| Access credential | Session ID | Signed JWT |
| Browser storage | HttpOnly cookie | HttpOnly cookies |
| Server-side session state | Yes | Limited to security state |
| Redis sessions | Yes | Refresh/revocation state |
| JWT | No | Yes |
| JWT claims | No | Yes |
| JWT `jti` | No | Yes |
| JWT expiration | Session TTL | JWT `exp` |
| Immediate access-token revocation | Delete session | Redis `revoked:<jti>` |
| Refresh token | No | Yes |
| Refresh-token rotation | No | Yes |
| Refresh-token reuse detection | No | Yes |
| Session/token-family revocation | No | Yes |
| Password hashing | bcrypt | bcrypt |
| Browser authentication | Yes | Yes |

---

## Security Concepts Demonstrated

### Password Security

Passwords are never stored in plaintext. Bcrypt is used for password hashing and verification.

### HttpOnly Cookies

Authentication cookies use:

```go
HttpOnly: true
```

This prevents JavaScript from directly reading the authentication cookies.

### Secure Cookies

Local development uses:

```go
Secure: false
```

because the lab runs over HTTP.

Production authentication should use HTTPS with:

```go
Secure: true
```

### SameSite

The cookies use:

```go
SameSite: http.SameSiteLaxMode
```

to provide additional protection against cross-site request scenarios.

### Redis TTL

Redis TTLs automatically remove temporary authentication state.

### Token Revocation

The JWT branch demonstrates how an otherwise stateless JWT system can gain immediate server-side revocation.

### Refresh Token Rotation

Each refresh token is consumed and replaced by a new token.

### Refresh Token Reuse Detection

Reuse of an old refresh token revokes the associated session/token family.

---

## Project Structure

The exact files may evolve as the lab progresses. The JWT branch currently revolves around the Go server, JWT/authentication code, and browser client.

Typical structure:

```text
.
├── go.mod
├── go.sum
│
└── server/
    ├── main.go
    ├── jwt.go
    ├── auth.go
    └── index.html
```

---

## Running the Lab

Start Redis first:

```bash
redis-cli ping
```

Expected:

```text
PONG
```

Then run the Go application from the project root:

```bash
go run .
```

The server is available at:

```text
http://localhost:8080
```

The browser client can be used to test:

- Login
- Profile
- Logout
- Refresh token

---

## Inspecting Redis

During the lab, Redis can be inspected manually:

```bash
redis-cli
```

List keys:

```redis
KEYS *
```

Useful JWT-branch keys include:

```text
refresh:<token>
used_refresh:<token>
revoked:<jti>
session:<session-id>
```

Inspect a key:

```redis
GET <key>
```

For a refresh-token Hash:

```redis
HGETALL refresh:<token>
```

Check TTL:

```redis
TTL <key>
```

For this local lab, `KEYS *` is useful for learning and debugging. In production, prefer `SCAN`.

---

## Learning Approach

The project is intentionally built incrementally.

Instead of starting with a framework that hides authentication internals, the lab implements the mechanisms directly with Go's HTTP stack and Redis.

The learning path is:

```text
Password
   ↓
bcrypt
   ↓
Authentication
   ↓
Stateful sessions
   ↓
Cookies
   ↓
Redis
   ↓
JWT
   ↓
JWT middleware
   ↓
JWT revocation
   ↓
Refresh tokens
   ↓
Refresh token rotation
   ↓
Reuse detection
   ↓
Session/token-family revocation
```

The two branches allow the implementations to be compared directly:

```text
main
  │
  └── Stateful authentication
          │
          └── Server-side session state


jwt
  │
  └── JWT authentication
          │
          ├── Stateless access token
          ├── Redis revocation
          ├── Refresh tokens
          ├── Token rotation
          └── Reuse detection
```

---

## Current Status

### `main` — Stateful Authentication

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
- [x] Secure
- [x] SameSite
- [x] Browser authentication
- [x] Authenticated `/profile`
- [x] Logout
- [x] Redis inspection
- [x] CookieJar

### `jwt` — JWT Authentication

- [x] JWT generation
- [x] JWT claims
- [x] JWT signature
- [x] JWT verification
- [x] JWT expiration
- [x] Unique JWT `jti`
- [x] JWT authentication middleware
- [x] HttpOnly access-token cookie
- [x] Redis token revocation
- [x] Revocation TTL based on JWT expiration
- [x] Request context for authenticated identity
- [x] Logout with access-token revocation
- [x] Opaque refresh tokens
- [x] Refresh tokens stored in Redis
- [x] Refresh-token TTL
- [x] Refresh-token rotation
- [x] Shared `session_id`
- [x] Used refresh-token tracking
- [x] Refresh-token reuse detection
- [x] Session/token-family revocation
- [x] Browser refresh flow
- [x] Redis inspection

---

## Branch Strategy

The branches are intentionally kept separate so each authentication model can be studied independently.

Switch to the original stateful implementation:

```bash
git checkout main
```

Switch to the JWT implementation:

```bash
git checkout jwt
```

The `main` branch represents the baseline authentication model.

The `jwt` branch represents the extended JWT implementation built on top of the concepts learned in `main`.

---

## Disclaimer

This is a learning lab, not a production-ready authentication system.

Some implementation choices are intentionally simplified to make the authentication mechanics visible and easy to experiment with.

Production systems should additionally consider:

- HTTPS everywhere
- Secret/key management
- CSRF protection strategy
- Secure cookie configuration
- Redis high availability
- Atomic refresh-token rotation
- Rate limiting
- Audit logging
- Key rotation
- Distributed deployment
- Proper session lifecycle management
