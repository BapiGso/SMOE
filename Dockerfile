FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app
VOLUME /app/usr
EXPOSE 95

ENTRYPOINT ["sh", "-ec", "\
set -eu; \
repo=\"$${SMOE_REPO:-BapiGso/SMOE}\"; \
version=\"$${SMOE_VERSION:-latest}\"; \
bin_dir=\"$${SMOE_BIN_DIR:-/opt/smoe/bin}\"; \
bin_path=\"$${SMOE_BIN_PATH:-}\"; \
if [ -z \"$$bin_path\" ]; then bin_path=\"$$bin_dir/SMOE\"; fi; \
arch=\"$$(uname -m)\"; \
case \"$$arch\" in \
  x86_64|amd64) asset_arch=amd64 ;; \
  aarch64|arm64) asset_arch=arm64 ;; \
  riscv64) asset_arch=riscv64 ;; \
  *) echo \"Unsupported architecture: $$arch\" >&2; exit 1 ;; \
esac; \
download_url=\"$${SMOE_DOWNLOAD_URL:-}\"; \
if [ -z \"$$download_url\" ]; then \
  if [ \"$$version\" = \"latest\" ]; then \
    download_url=\"https://github.com/$$repo/releases/latest/download/SMOE_linux_$$asset_arch\"; \
  else \
    download_url=\"https://github.com/$$repo/releases/download/$$version/SMOE_linux_$$asset_arch\"; \
  fi; \
fi; \
mkdir -p \"$$(dirname \"$$bin_path\")\"; \
tmp_path=\"$$bin_path.download\"; \
if wget -O \"$$tmp_path\" \"$$download_url\"; then \
  chmod +x \"$$tmp_path\"; \
  mv \"$$tmp_path\" \"$$bin_path\"; \
  echo \"Using binary from $$download_url\" >&2; \
else \
  rm -f \"$$tmp_path\"; \
  if [ -x \"$$bin_path\" ]; then \
    echo \"Failed to download $$download_url, falling back to cached binary\" >&2; \
  else \
    echo \"Failed to download $$download_url and no cached binary is available\" >&2; \
    exit 1; \
  fi; \
fi; \
exec \"$$bin_path\" \"$$@\"", "--"]
