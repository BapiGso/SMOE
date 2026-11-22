# 镜像里不带程序，每次容器启动时从 GitHub Release 下载 SMOE_linux_amd64 再运行，
# 所以重启容器就是升级。下载失败时沿用容器里上次下好的那份。
# 指定版本：-e SMOE_VERSION=v1.2.0；默认 latest。
FROM --platform=linux/amd64 alpine:latest

RUN apk add --no-cache ca-certificates

ENV SMOE_REPO=BapiGso/SMOE \
    SMOE_VERSION=latest

WORKDIR /app
VOLUME /app/usr
EXPOSE 95

ENTRYPOINT ["/bin/sh", "-c", "\
if [ \"$SMOE_VERSION\" = latest ]; then url=https://github.com/$SMOE_REPO/releases/latest/download/SMOE_linux_amd64; \
else url=https://github.com/$SMOE_REPO/releases/download/$SMOE_VERSION/SMOE_linux_amd64; fi; \
if wget -qO /tmp/SMOE \"$url\"; then chmod +x /tmp/SMOE && mv /tmp/SMOE /usr/local/bin/SMOE; \
else echo \"下载 $url 失败\" >&2; [ -x /usr/local/bin/SMOE ] || exit 1; fi; \
exec /usr/local/bin/SMOE \"$@\"", "--"]
