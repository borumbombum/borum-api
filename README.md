# borum-api

Single Go binary: PocketBase backend + chi custom API router.
Custom routes are declared in `main.go` (apiRoutes table).

## Ports
- 8090 — PocketBase only: REST API `/api/*`, admin UI `/_/` (private)
- 8091 — custom chi API (the only thing you expose)

## Build & run on Termux
    pkg install golang git tmux
    git clone https://github.com/borumbombum/borum-api.git
    cd borum-api
    CGO_ENABLED=0 go build -o borum-api .
    ./borum-api serve

Binds 8090 (PocketBase) and 8091 (API) on 127.0.0.1 by default.
For browser access from other devices, bind both to all
interfaces: ./borum-api serve --http 0.0.0.0:8090 (API port is
configured in main.go, const apiPort).

Keep it alive with tmux (pkg install tmux).

## Access
- Custom API (8091): http://<host>:8091
- Admin UI (8090): http://<address>/_/
- REST API (8090): http://<address>/api/

## First superuser
    ./borum-api superuser upsert admin@example.com yourpassword

## Sync the database (over Tailscale)
On the phone (one-time):
    pkg install openssh rsync
    sshd              # SSH server on port 8022
    whoami            # note the username, e.g. u0_a123

On the source machine — stop the server if running (so data.db is
consistent), then sync the whole folder:

    pkill -x borum-api
    rsync -avz -e "ssh -p 8022" pb_data/ <phone-username>@<phone-tailscale-ip>:~/borum-api/pb_data/

## Expose to the internet (Cloudflare Tunnel)
Only the custom API (8091) is exposed; PocketBase and the DB stay
private on 8090.

Install: pkg install cloudflared tmux

**Quick (ephemeral, no account):**
    cloudflared tunnel --url http://127.0.0.1:8091
→ random https://xxx.trycloudflare.com. New URL each restart, no SLA.

**Named (persistent, needs a Cloudflare-managed domain):**
1. Dashboard → Zero Trust → Networks → Tunnels → Create → copy token
2. Add hostname (e.g. api.yourdomain.com) → http://127.0.0.1:8091
3. cloudflared tunnel run --token <TOKEN>

Keep the server + tunnel alive with tmux (Ctrl-B C new window,
Ctrl-B D detach).

## Keep Termux from sleeping (Android)

If your Termux CPU sleeps/suspends and the API stops, hold a Termux wake lock while the
server is running.

In one Termux session:
    termux-wake-lock

Leave that terminal running, then start your server in a second Termux session:
    ./borum-api serve

When you’re done:
    termux-wake-unlock

