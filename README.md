# Webhook Engine (Zoom fast-path demo)

Single-instance, high-throughput Zoom webhook receiver with:
- fasthttp server
- async HMAC validation
- sharded, batched Badger persistence
- Prometheus metrics, Loki logs, Tempo traces
- Docker Compose stack + Grafana dashboard
- Stress tool for signed Zoom traffic
- ngrok-ready for public HTTPS

## Quick start
```bash
export ZOOM_WEBHOOK_SECRET_TOKEN=supersecret
docker compose -f docker/docker-compose.yml up --build
# expose to the internet for Zoom CRC:
ngrok http 8080
```
Open Grafana at http://localhost:3000

Run local load:
```bash
make run-stress
```
