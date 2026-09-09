# JWT Authentication Lab — CentOS Setup

Guide to running this Go lab with JWT, bcrypt, and Redis on CentOS/RHEL-compatible systems.

## Requirements

You need:

- Go
- Git
- curl
- Redis Server
- Internet access to download Go dependencies

Check:

```bash
go version
git --version
curl --version
redis-cli --version
```

## 1. Install basic packages

With `yum`:

```bash
sudo yum install -y git curl
```

With `dnf`:

```bash
sudo dnf install -y git curl
```

## 2. Install Go

The official Go installation for Linux uses the Go tarball and `/usr/local/go`.

Check first:

```bash
go version
```

If Go is not installed:

```bash
cd /tmp
curl -LO https://go.dev/dl/go1.25.1.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.1.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

Adjust `go1.25.1` to the version required by the project's `go.mod`.

## 3. Install Redis

Redis is required for the part of the lab that uses Redis-backed sessions.

The official Redis documentation provides RPM packages for RHEL-compatible distributions. For Rocky/Alma 9, for example:

```bash
sudo tee /etc/yum.repos.d/redis.repo > /dev/null <<'EOF'
[Redis]
name=Redis
baseurl=http://packages.redis.io/rpm/rockylinux9
enabled=1
gpgcheck=1
EOF
```

Import the signing key:

```bash
curl -fsSL https://packages.redis.io/gpg > /tmp/redis.key
sudo rpm --import /tmp/redis.key
```

Install Redis:

```bash
sudo yum install -y redis
```

or:

```bash
sudo dnf install -y redis
```

**Important:** `rockylinux9` is only an example. For CentOS/RHEL-compatible systems, use the repository corresponding to your operating system version.

## 4. Start Redis

### Normal CentOS installation with systemd

On a normal CentOS/RHEL-compatible installation:

```bash
sudo systemctl start redis
sudo systemctl enable redis
sudo systemctl status redis
```

Test it:

```bash
redis-cli ping
```

Expected result:

```text
PONG
```

Redis normally listens on:

```text
127.0.0.1:6379
```

### CentOS running inside WSL

If you are running CentOS inside **WSL** and your distribution does not use `systemd` as PID 1, `systemctl` will not work. You may see an error similar to:

```text
System has not been booted with systemd as init system
```

For this lab, you do not need to configure `systemd`. Start Redis directly:

```bash
redis-server --daemonize yes
```

Then check:

```bash
redis-cli ping
```

Expected result:

```text
PONG
```

You can verify that Redis is running:

```bash
ps aux | grep redis
```

or:

```bash
redis-cli INFO server | grep redis_version
```

In WSL, if Redis is not running, simply start it again with:

```bash
redis-server --daemonize yes
```

> You can configure `systemd` in WSL if you want Redis to start automatically, but this is **not required for this lab**.

## 5. Get the project

If the repository is on GitHub:

```bash
git clone <REPOSITORY_URL>
cd <REPOSITORY_NAME>
```

## 6. Download Go dependencies

There is no need to install the project's Go libraries manually:

```bash
go mod download
```

Check dependencies:

```bash
go list -m all
```

Optionally:

```bash
go mod tidy
```

## 7. Build the project

```bash
go build ./...
```

If tests exist:

```bash
go test ./...
```

## 8. Run the server

If `main.go` is inside `server`:

```bash
go run ./server
```

If it is in the project root:

```bash
go run .
```

You can also build a binary:

```bash
go build -o jwt-lab ./server
./jwt-lab
```

## 9. Test the login

The lab uses:

```http
POST /login
Content-Type: application/json
```

Example:

```bash
curl -X POST http://localhost:8080/login   -H "Content-Type: application/json"   -d '{"username":"alice","password":"YOUR_PASSWORD"}'
```

Response:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

## 10. Test `/profile`

Send the JWT as a Bearer token:

```bash
curl http://localhost:8080/profile   -H "Authorization: Bearer YOUR_TOKEN"
```

The flow is:

```text
POST /login
    ↓
CheckPassword()
    ↓
CreateJWT()
    ↓
JWT
    ↓
GET /profile
    ↓
Authorization: Bearer <JWT>
    ↓
VerifyJWT()
    ↓
signature + exp + claims
```

## 11. Redis during the lab

List keys:

```bash
redis-cli KEYS '*'
```

Read a key:

```bash
redis-cli GET <key>
```

Delete a key:

```bash
redis-cli DEL <key>
```

`KEYS *` is fine for this lab, but in production you should prefer `SCAN`.

## 12. JWT vs Redis

### Redis session

```text
Browser
   ↓
session ID
   ↓
Server
   ↓
Redis
   ↓
User/session
```

Deleting the key immediately invalidates the session.

### JWT

```text
Browser
   ↓
JWT
   ↓
Server
   ↓
signature + exp + claims
```

The server does not need to query Redis to validate the JWT.

A JWT becomes invalid when it expires or when its signature is invalid, unless an explicit revocation mechanism has been implemented.

## 13. Test expiration

For the lab, you can temporarily use:

```go
"exp": time.Now().Add(10 * time.Second).Unix(),
```

Then:

```text
Login
  ↓
JWT
  ↓
before 10 seconds → valid
after 10 seconds → Invalid token
```

After testing, restore a reasonable expiration time, for example:

```go
"exp": time.Now().Add(15 * time.Minute).Unix(),
```

## 14. Test JWT manipulation

A JWT consists of:

```text
HEADER.PAYLOAD.SIGNATURE
```

You can read the payload, but modifying it breaks the signature:

```text
HEADER.MODIFIED_PAYLOAD.ORIGINAL_SIGNATURE
                         ↓
                   invalid signature
                         ↓
                    Invalid token
```

## 15. Access the server from another computer

If you want to access the server from another machine, Go must listen on an accessible interface, for example:

```text
0.0.0.0:8080
```

If `firewalld` is enabled:

```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

Check:

```bash
sudo firewall-cmd --list-ports
```

For production, consider TLS, a reverse proxy, and appropriate firewall rules.

## 16. HTML

The lab can also be tested from `index.html`.

The flow is:

```text
Browser
   ↓
index.html
   ↓
POST /login
   ↓
JWT
   ↓
GET /profile
Authorization: Bearer JWT
```

This lets you compare browser-based requests with the API requests made using `curl`.

## Troubleshooting

### Go is not found

```bash
go version
echo $PATH
```

It should contain:

```text
/usr/local/go/bin
```

Reload:

```bash
source ~/.bashrc
```

### Redis does not start

```bash
sudo systemctl status redis
sudo journalctl -u redis --no-pager
ss -lntp | grep 6379
```

If running under WSL without `systemd`, use:

```bash
redis-server --daemonize yes
```

### Test Redis directly

```bash
redis-cli -h 127.0.0.1 -p 6379 ping
```

Expected:

```text
PONG
```

### The project does not compile

```bash
go mod download
go mod tidy
go build ./...
```

## Quick checklist

```bash
sudo yum install -y git curl

go version

redis-cli ping

git clone <REPO>
cd <REPO>

go mod download
go build ./...

go run ./server
```

For WSL without `systemd`, start Redis before running the Go application:

```bash
redis-server --daemonize yes
redis-cli ping
```

## Official references

- Go: https://go.dev/doc/install
- Redis: https://redis.io/docs/latest/operate/oss_and_stack/install/
