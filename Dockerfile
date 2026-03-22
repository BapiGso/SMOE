FROM alpine AS dl
RUN wget -O /SMOE https://github.com/BapiGso/SMOE/releases/latest/download/SMOE_linux_amd64 \
    && chmod +x /SMOE

FROM scratch
COPY --from=dl /SMOE /SMOE
COPY --from=dl /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /app
VOLUME /app/usr
EXPOSE 95
CMD ["/SMOE"]
