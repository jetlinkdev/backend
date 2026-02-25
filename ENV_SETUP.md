# Environment Variables Configuration

## Setup Instructions

1. **Copy the example environment file:**
   ```bash
   cp .env.example .env.local
   ```

2. **Edit `.env.local` with your actual credentials:**
   ```bash
   nano .env.local
   ```

3. **Update the following values:**
   - `MYSQL_DSN` - Your MySQL connection string
   - `REDIS_ADDR` - Your Redis server address (optional)
   - `REDIS_PASSWORD` - Your Redis password (if any)

## Environment Variables Reference

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_ADDR` | `:8080` | HTTP server bind address |

### Database Configuration (MySQL)

| Variable | Default | Description |
|----------|---------|-------------|
| `MYSQL_DSN` | `root:password@tcp(localhost:3306)/jetlink?...` | MySQL connection string |
| `MYSQL_HOST` | `localhost` | MySQL server host |
| `MYSQL_PORT` | `3306` | MySQL server port |
| `MYSQL_USER` | `root` | MySQL username |
| `MYSQL_PASSWORD` | `password` | MySQL password |
| `MYSQL_DATABASE` | `jetlink` | MySQL database name |

### Redis Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost:6379` | Redis server address |
| `REDIS_PASSWORD` | (empty) | Redis password (if any) |
| `REDIS_DB` | `0` | Redis database number |

### Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `ENV` | `development` | Environment mode (development/production) |

## File Priority

Environment variables are loaded in this order (later overrides earlier):

1. `.env` - Base configuration (can be committed to Git)
2. `.env.local` - Local overrides (IGNORED by Git - safe for secrets)
3. Command-line flags - Highest priority
4. System environment variables

## Usage Examples

### Development

```bash
# .env.local already has your local credentials
go run cmd/server/main.go
```

### Production

```bash
# Set environment variables directly
export MYSQL_DSN="user:pass@tcp(db:3306)/jetlink"
export REDIS_ADDR="redis:6379"
export SERVER_ADDR=":8080"

# Run server
./jetlink-server
```

### Docker

```bash
docker run -d \
  -e MYSQL_DSN="root:secret@tcp(mysql:3306)/jetlink" \
  -e REDIS_ADDR="redis:6379" \
  -p 8080:8080 \
  jetlink-backend
```

## Security Notes

- ✅ **DO** use `.env.local` for local development secrets
- ✅ **DO** set environment variables in production
- ❌ **DON'T** commit `.env.local` to Git (already in .gitignore)
- ❌ **DON'T** hardcode credentials in source code

## Migration from Hardcoded Values

Previously, credentials were hardcoded in `cmd/server/main.go`:

```go
// OLD (hardcoded)
mysqlDSN := "root:~Densus_88@tcp(localhost:3306)/jetlink?..."

// NEW (from environment)
mysqlDSN := utils.GetEnv("MYSQL_DSN", "default-value")
```

The application will still work with default values if no `.env` file is present.
