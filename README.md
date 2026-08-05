# borum-api

Single Go binary: PocketBase backend + chi custom API router.
Custom routes are declared in `main.go` (apiRoutes table).

## Build & run on Termux
    pkg install golang git tmux
    git clone https://github.com/borumbombum/borum-api.git
    cd borum-api
    CGO_ENABLED=0 go build -o borum-api .
    ./borum-api serve

Binds 127.0.0.1:8090 by default. For browser access from other
devices: ./borum-api serve --http 0.0.0.0:8090

Keep it alive with tmux (pkg install tmux).

## Access
- LAN: http://<phone-lan-ip>:8090
- Tailscale: http://<phone-tailscale-hostname>:8090
- Admin UI: http://<address>/_/

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
