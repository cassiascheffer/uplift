# Deploying Uplift to Fly.io

This guide walks through deploying Uplift to Fly.io.

## Prerequisites

1. Install the Fly.io CLI:
   ```bash
   curl -L https://fly.io/install.sh | sh
   ```

2. Sign up or log in to Fly.io:
   ```bash
   fly auth signup
   # or if you already have an account:
   fly auth login
   ```

## Deployment Files

The following files have been created for Fly.io deployment:

- **Dockerfile**: Multi-stage build that:
  1. Builds frontend assets with Node.js
  2. Compiles Go binary
  3. Creates minimal Alpine Linux production image

- **fly.toml**: Fly.io configuration with:
  - WebSocket support (connections concurrency type)
  - Auto-scaling with sleep when idle
  - 256MB RAM, 1 shared CPU
  - HTTPS enforcement

- **.dockerignore**: Excludes development files from the build

## Initial Deployment

1. **Create the Fly.io app** (if not already created):
   ```bash
   fly launch --no-deploy
   ```

   This will:
   - Use the existing `fly.toml` configuration
   - Create the app on Fly.io
   - **Note**: If it asks to overwrite `fly.toml`, say **no** to keep your configuration

2. **Review and customise fly.toml** (optional):
   ```bash
   # Edit app name if needed
   app = 'uplift'  # Change to your preferred name

   # Change region if desired (default: yyz - Toronto)
   primary_region = 'yyz'  # See: fly platform regions
   ```

3. **Deploy the application**:
   ```bash
   fly deploy
   ```

   This will:
   - Build the Docker image (frontend + backend)
   - Push to Fly.io registry
   - Deploy to your primary region
   - Start the application

4. **Open your app**:
   ```bash
   fly open
   ```

## Deployment Architecture

### Multi-Stage Docker Build

1. **Frontend Stage** (`node:18-alpine`):
   - Installs npm dependencies
   - Runs `npm run build` to create production assets
   - Output: `/app/dist` directory

2. **Backend Stage** (`golang:1.25-alpine`):
   - Downloads Go dependencies
   - Compiles the Go binary
   - Output: `/app/uplift` binary

3. **Production Stage** (`alpine:latest`):
   - Minimal image (~50MB)
   - Copies compiled binary and built frontend
   - Serves static files from `./static`
   - Runs on port 8080

### WebSocket Support

The `fly.toml` is configured for WebSocket connections:

```toml
[http_service.concurrency]
  type = "connections"  # Required for WebSocket support
  hard_limit = 1000
  soft_limit = 1000
```

### Auto-Scaling

The app is configured to automatically sleep when idle and wake on requests:

```toml
auto_stop_machines = "stop"
auto_start_machines = true
min_machines_running = 0
```

This means:
- **Zero cost when idle** (machines stop automatically)
- **Fast wake-up** when users visit (typically < 1 second)
- Perfect for low-traffic apps

## Post-Deployment

### Check Application Status
```bash
fly status
```

### View Logs
```bash
fly logs
```

### Scale Resources (if needed)
```bash
# Increase memory
fly scale memory 512

# Add more machines for high availability
fly scale count 2
```

### Set Secrets (if needed)
```bash
fly secrets set SECRET_KEY=value
```

## Updating the Application

After making code changes:

```bash
fly deploy
```

This rebuilds and redeploys automatically.

## Monitoring

### View Metrics
```bash
fly dashboard
```

Or visit: https://fly.io/apps/[your-app-name]

### SSH into Running Machine
```bash
fly ssh console
```

## Troubleshooting

### Build Fails

Check the Dockerfile syntax:
```bash
docker build -t uplift:test .
```

### App Doesn't Start

Check logs:
```bash
fly logs
```

Common issues:
- Port mismatch (ensure app listens on PORT env var or 8080)
- Missing static files (ensure `dist` directory is built)

### WebSocket Connection Issues

Verify the app is using `connections` concurrency type in `fly.toml`:
```toml
[http_service.concurrency]
  type = "connections"
```

## Cost Estimate

With the current configuration (256MB RAM, 1 shared CPU, auto-scaling):

- **Idle**: $0 (machine stopped)
- **Active**: ~$3-5/month for moderate usage
- **Free tier**: Fly.io offers free allowances for small apps

See: https://fly.io/docs/about/pricing/

## Additional Resources

- [Fly.io Go Documentation](https://fly.io/docs/languages-and-frameworks/golang/)
- [Fly.io Configuration Reference](https://fly.io/docs/reference/configuration/)
- [WebSocket Support](https://fly.io/docs/app-guides/websockets/)
