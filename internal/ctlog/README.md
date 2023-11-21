# Setup

Run and setup Minio. Currently this is a bit janky because I dumped the minio creds directly into minio.go, but when it gets sorted, this should be enough automation.
```
docker run \
    -d \
    -p 9000:9000 \
    -p 9090:9090 \
    --user $(id -u):$(id -g) \
    -e "MINIO_ROOT_USER=ROOTUSER" \
    -e "MINIO_ROOT_PASSWORD=CHANGEME123" \
    -v ${HOME}/minio/data:/data \
    quay.io/minio/minio server /data \
    --console-address ":9090"

mc alias set litetlog-dev 'http://127.0.0.1:9000' ROOTUSER CHANGEME123 --api "s3v4" --path "off"
mc ready litetlog-dev
mc mb --ignore-existing litetlog-dev/rome2024h1
mc ls litetlog-dev

# Create an API token pair. The secret-key will only be shown once. The access-key is literally "changeme".
SECRET_KEY=$(mc admin user svcacct add litetlog-dev ROOTUSER --expiry $(date -d"+1 day" +%Y-%m-%dT%T) --access-key "changeme" | awk -F':' '/Secret Key: /{print $2}' | sed 's/[[:space:]]//g')

# Show existing tokens
mc admin user svcacct ls litetlog-dev ROOTUSER
```

Generate a key pair
```
cd litetlog/internal/ctlog
openssl req -new -x509 -nodes -newkey ec:<(openssl ecparam -name secp384r1) -keyout key.pem -out cert.pem -days 3650
```

Get some root CA certs
```
cd litetlog/internal/ctlog
curl -s https://letsencrypt.org/certs/lets-encrypt-e1.pem -o roots.pem
```

Run litetlog
```
cd litetlog/internal/ctlog
go run cmd/rome2024h1/main.go
```

Check metrics
```
curl -sLk https://127.0.0.1:8443/metrics
```
