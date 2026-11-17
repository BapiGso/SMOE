SMOE
===

![smoe](assets/screenshot.png)

<p>
<img src="https://img.shields.io/github/go-mod/go-version/bapigso/smoe" alt="Go version" />
<img src="https://img.shields.io/badge/license-GPL-blue" alt="WP version" />
<img src="https://img.shields.io/github/actions/workflow/status/bapigso/smoe/release.yaml" alt="GitHub release" />
<img src="https://raw.githubusercontent.com/bapigso/smoe/main/assets/design/sign.svg" alt="signature" height="20" />
</p>

## About

[SMOE](https://smoe.cc)

## Run

站点目录 `usr/` 里要有 `config.yaml`（字段见 [config.go](internal/moe/config/config.go)），文章放在 `usr/content`。用 Docker 一行启动：

```sh
docker run -d --name smoe --restart unless-stopped -p 95:95 \
  -v /path/to/usr:/app/usr -v smoe-go:/go -w /app \
  -e GOCACHE=/go/.cache -e GOTOOLCHAIN=auto \
  golang:alpine sh -c 'go install github.com/BapiGso/SMOE@latest; exec SMOE'
```

每次启动都会更新到最新版本，更新失败就用上次装好的。首次启动要编译 Hugo，需要几分钟。

## License

This software is distributed under the GPL License