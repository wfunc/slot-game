# Swagger UI 离线资源

为在离线环境中使用增强版 Swagger UI（/docs/ui），可将 swagger-ui-dist 的必要文件放置到本目录：

目标路径：

```
static/vendors/swagger-ui/swagger-ui.css
static/vendors/swagger-ui/swagger-ui-bundle.js
static/vendors/swagger-ui/swagger-ui-standalone-preset.js
```

获取方式（任选其一）：

1) 一键下载（需网络）

```bash
make fetch-swagger-ui
```

2) 手动下载

```bash
curl -fsSL https://unpkg.com/swagger-ui-dist@5/swagger-ui.css -o static/vendors/swagger-ui/swagger-ui.css
curl -fsSL https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js -o static/vendors/swagger-ui/swagger-ui-bundle.js
curl -fsSL https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js -o static/vendors/swagger-ui/swagger-ui-standalone-preset.js
```

完成后，访问：

- 增强版 Swagger UI: http://localhost:8080/docs/ui （将自动优先使用本地静态资源）

