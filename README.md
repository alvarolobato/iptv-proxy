## SSH to vps
```bash
ssh root@0.0.0.0.0
```

## 1. Update the system and install prerequisites



```bash
apt update && apt upgrade -y
apt install -y docker.io docker-compose curl nano git jq
```

## 2. Make sure docker & docker-compose are running

```bash
systemctl enable --now docker
docker --version
docker-compose --version
```

## 3. Clone iptv proxy repo

```bash
git clone https://github.com/chernandezweb/iptv-proxy.git
```

## 4. Create provider logins

```bash
sudo nano ~/iptv-proxy/docker-compose.yml
```

```yaml
version: "3"
services:
  iptv-proxy:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: "iptv-proxy"
    restart: on-failure
    ports:
      - 8080:8080
    environment:
      # Port to expose the IPTVs endpoints
      PORT: 8080
      # Hostname or IP to expose the IPTVs endpoints (for machine not for docker)
      HOSTNAME: "0.0.0.0" # change for ipof the server proxy
      GIN_MODE: release
      USER_AGENT: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
      REFERER: "http://....com" # change for iptv provider domain
      TIMEOUT: 30000
      ## Xtream-code proxy configuration
      XTREAM_USER: username  # change for iptv provider username
      XTREAM_PASSWORD: password  # change for iptv provider password
      XTREAM_BASE_URL: "http://....com" # change for iptv provider domain
      USER: my_custom_username  # change for custom username
      PASSWORD: my_custom_password  # change for custom password
      METADATA_CACHE_TTL: 5m # cache heavy Xtream metadata (get_series, get_series_info)
      XMLTV_CACHE_TTL: 30m   # cache xmltv.php responses to avoid repeated guide downloads
```

## 5. Run it

```bash
cd ~/iptv-proxy
docker compose down
docker compose up -d --build
```
## ------------------------------------------------
## To see logs
## ------------------------------------------------

```bash
docker logs iptv-proxy
```

## ------------------------------------------------
## To change provider and account info
## ------------------------------------------------

```bash
sudo nano ~/iptv-proxy/docker-compose.yml
```

```bash
cd ~/iptv-proxy
docker compose down
docker compose up -d --build
```
