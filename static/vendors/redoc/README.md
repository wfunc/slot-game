# Redoc 离线资源

为保证在离线环境下也能浏览 API 文档，请将 `redoc.standalone.js` 放置到本目录：

目标路径：

```
static/vendors/redoc/redoc.standalone.js
```

获取方式（任选其一）：

1) 在线下载（需网络）

```bash
curl -fsSL https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js \
  -o static/vendors/redoc/redoc.standalone.js

# 或
wget -O static/vendors/redoc/redoc.standalone.js \
  https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js
```

2) 复制已有文件

从有网络的环境下载好后，拷贝到本目录。

完成后，浏览器访问：

- OpenAPI YAML: http://localhost:8080/openapi
- Redoc 页面: http://localhost:8080/docs/redoc

注意：你也可以继续使用在线 CDN（默认会自动回退到 CDN），但在离线环境下必须放置本地文件。

