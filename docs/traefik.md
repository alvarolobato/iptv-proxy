# TLS / HTTPS with Traefik

This guide shows how to run iptv-proxy2 behind [Traefik](https://doc.traefik.io/traefik/) with HTTPS.

---

## Prerequisites

- Traefik v2 configured with a certificate resolver (e.g. DNS challenge).
- Docker and Docker Compose.

---

## Layout

Put the Traefik config and this app in the same tree, for example:

```text
.
├── docker-compose.yml   # iptv-proxy2 + Traefik
├── traefik/
│   ├── traefik.yaml
│   ├── etc/traefik/
│   └── log/
└── data/                # optional: replacements, etc.
    └── replacements.json
```

---

## docker-compose.yml

```yaml
version: "3"
services:
  iptv-proxy2:
    image: alobato/iptv-proxy2:latest
    container_name: iptv-proxy2
    restart: on-failure
    volumes:
      - ./data:/data
    environment:
      M3U_URL: "http://example.com/get.php?username=USER&password=PASS&type=m3u_plus&output=m3u8"
      PORT: 8080
      ADVERTISED_PORT: 443
      HOSTNAME: iptv.example.com
      HTTPS: "1"
      USER: myuser
      PASSWORD: mypass
      JSON_FOLDER: /data
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.iptv-proxy2.rule=Host(`iptv.example.com`)"
      - "traefik.http.routers.iptv-proxy2.entrypoints=websecure"
      - "traefik.http.routers.iptv-proxy2.tls.certresolver=mydnschallenge"
      - "traefik.http.services.iptv-proxy2.loadbalancer.server.port=8080"

  traefik:
    image: traefik:v2.4
    restart: always
    read_only: true
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik/traefik.yaml:/traefik.yaml:ro
      - ./traefik/etc/traefik:/etc/traefik/
      - ./traefik/log:/var/log/traefik/
```

Replace `iptv.example.com` and `mydnschallenge` with your domain and Traefik certificate resolver name.

---

## iptv-proxy2 settings

- **HOSTNAME** — Must match the Host rule in Traefik (e.g. `iptv.example.com`) so generated URLs are correct.
- **ADVERTISED_PORT** — Set to 443 when Traefik serves HTTPS on 443.
- **HTTPS** — Set to `1` so the proxy generates `https://` URLs.
- **JSON_FOLDER** — Use `/data` and mount your host folder so `replacements.json` is read from the mounted volume.

Start with:

```bash
docker-compose up -d
```

Then open `https://iptv.example.com/iptv.m3u?username=myuser&password=mypass` (or your configured path).
