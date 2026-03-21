
FROM alpine

RUN apk add --no-cache curl jq \
    && DOWNLOAD_URL=$(curl -s https://api.github.com/repos/BapiGso/SMOE/releases/latest \
       | jq -r '.assets[] | select(.name | test("linux.*amd64")) | .browser_download_url') \
    && curl -L -o /usr/local/bin/SMOE "$DOWNLOAD_URL" \
    && chmod +x /usr/local/bin/SMOE \
    && apk del curl jq

WORKDIR /app

VOLUME /app/usr

EXPOSE 95

CMD ["SMOE"]
