## 1. Update the system and install prerequisites

SSH to vps
```bash
ssh root@0.0.0.0.0
```

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
    #volumes:
      # If your are using local m3u file instead of m3u remote file
      # put your m3u file in this folder
      #- ./iptv:/root/iptv
    container_name: "iptv-proxy"
    restart: on-failure
    ports:
      # have to be the same as ENV variable PORT
      - 8080:8080
    environment:
      # if you are using m3u remote file
      # M3U_URL: https://example.com/iptvfile.m3u
      # Port to expose the IPTVs endpoints
      PORT: 8080
      # Hostname or IP to expose the IPTVs endpoints (for machine not for docker)
      HOSTNAME: "0.0.0.0"
      GIN_MODE: release
      USER_AGENT: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
      REFERER: "http://....com" # change for iptv provider domain
      TIMEOUT: 30000
      ## Xtream-code proxy configuration
      # change for iptv provider username
      XTREAM_USER: username 
      # change for iptv provider password
      XTREAM_PASSWORD: password
      XTREAM_BASE_URL: "http://....com" # change for iptv provider domain
      ##### UNSAFE AUTH TODO ADD REAL AUTH
      #will be used for m3u and xtream auth poxy
      # change for custom username
      USER: my_custom_username
      # change for custom password
      PASSWORD: my_custom_password
```

## 5. Run it

```bash
cd ~/iptv-proxy
docker compose down
docker compose up -d --build
```
