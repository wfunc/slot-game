package api

import (
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
)

// registerOpenAPIRoutes 提供 /openapi 与 /docs/redoc
func registerOpenAPIRoutes(engine *gin.Engine) {
    engine.GET("/openapi", serveOpenAPI)
    engine.GET("/openapi.yaml", serveOpenAPI)
    engine.GET("/docs/redoc", serveRedoc)
    engine.GET("/docs/ui", serveSwaggerUI)
}

func serveOpenAPI(c *gin.Context) {
    c.Header("Content-Type", "application/yaml; charset=utf-8")
    c.File("docs/api/openapi.yaml")
}

func serveRedoc(c *gin.Context) {
    // 优先使用本地 redoc 资源，离线可用；否则回退到 CDN
    useLocal := false
    if _, err := os.Stat("static/vendors/redoc/redoc.standalone.js"); err == nil {
        useLocal = true
    }

    scriptTag := "<script src=\"https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js\"></script>"
    note := "<div style=\"position:fixed;top:8px;right:8px;background:#fffae6;border:1px solid #f0e6b4;padding:6px 10px;border-radius:6px;font:12px/1.2 -apple-system,Segoe UI,Helvetica,Arial\">CDN fallback。离线环境请放置 static/vendors/redoc/redoc.standalone.js</div>"
    if useLocal {
        scriptTag = "<script src=\"/static/vendors/redoc/redoc.standalone.js\"></script>"
        note = ""
    }

    html := `<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Slot Game API - Redoc</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
      :root { --bg:#ffffff; --fg:#0f172a; --btn-bg:#f8fafc; --btn-br:#d1d5db; --btn-fg:#0f172a; }
      body{margin:0;padding:0;background:var(--bg);color:var(--fg);font-family:-apple-system,Segoe UI,Helvetica,Arial,sans-serif}
      .topbar{position:fixed;top:0;left:0;right:0;height:48px;display:flex;align-items:center;justify-content:space-between;padding:0 12px;background:linear-gradient(90deg,#ffffff,#f8fafc);border-bottom:1px solid #e5e7eb;z-index:9999}
      .brand{font-weight:600;letter-spacing:.2px;color:#0f172a}
      .nav a,.nav button{color:var(--btn-fg);text-decoration:none;margin-left:12px;padding:6px 10px;border-radius:6px;border:1px solid var(--btn-br);background:var(--btn-bg)}
      .nav a:hover,.nav button:hover{border-color:#94a3b8;color:#0f172a;cursor:pointer}
      .nav a.dim{opacity:.45;cursor:not-allowed}
      .wrap{height:100vh;margin-top:48px}
      /* dark theme overrides (Redoc content readability) */
      body.theme-dark{ --bg:#0b1324; --fg:#e2e8f0; --btn-bg:#0b1324; --btn-br:#1e293b; --btn-fg:#cbd5e1; }
      body.theme-dark .topbar{background:linear-gradient(90deg,#0b1324,#0f172a);border-bottom-color:#172036}
      body.theme-dark .brand{color:#cbd5e1}
      body.theme-dark .nav a, body.theme-dark .nav button{border-color:#1e293b;color:#cbd5e1}
      /* Content backgrounds per theme */
      body.theme-light .wrap{background:#ffffff}
      body.theme-dark .wrap{background:#0b1324}

      /* Improve menu & code contrast (strong overrides) */
      /* Left menu */
      body.theme-dark .redoc-wrap .menu-content{background:#0b1324 !important;color:#e6edf3 !important}
      body.theme-dark .redoc-wrap .menu-content a, 
      body.theme-dark .redoc-wrap .menu-content label, 
      body.theme-dark .redoc-wrap .menu-content span{color:#e6edf3 !important}
      body.theme-dark .redoc-wrap .menu-content .active,
      body.theme-dark .redoc-wrap .menu-content .active > label,
      body.theme-dark .redoc-wrap .menu-content .active a,
      body.theme-dark .redoc-wrap .menu-content .active span{color:#58a6ff !important}

      body.theme-light .redoc-wrap .menu-content{background:#ffffff !important;color:#0f172a !important}
      body.theme-light .redoc-wrap .menu-content a,
      body.theme-light .redoc-wrap .menu-content label,
      body.theme-light .redoc-wrap .menu-content span{color:#0f172a !important}

      /* Right/content area */
      body.theme-dark .redoc-wrap .api-content{background:#0b1324 !important;color:#e6edf3 !important}
      body.theme-light .redoc-wrap .api-content{background:#ffffff !important;color:#0f172a !important}

      /* Ensure all textual labels in content area are readable */
      body.theme-dark .redoc-wrap .api-content *:not(code):not(pre){color:#e6edf3 !important}
      body.theme-light .redoc-wrap .api-content *:not(code):not(pre){color:#0f172a !important}

      /* Chips / badges like content-type, http-verb, etc. */
      body.theme-dark .redoc-wrap .content-type{background:#0a0f1a !important;border:1px solid #1f2937 !important;color:#93c5fd !important}
      body.theme-light .redoc-wrap .content-type{background:#f3f4f6 !important;border:1px solid #e5e7eb !important;color:#1d4ed8 !important}
      body.theme-dark .redoc-wrap .http-verb, body.theme-dark .redoc-wrap .operation-type{color:#93c5fd !important}
      body.theme-light .redoc-wrap .http-verb, body.theme-light .redoc-wrap .operation-type{color:#1d4ed8 !important}

      /* Security/Authorization sections */
      body.theme-dark .redoc-wrap .security-requirements, 
      body.theme-dark .redoc-wrap .security-requirements * {color:#e6edf3 !important}
      body.theme-light .redoc-wrap .security-requirements, 
      body.theme-light .redoc-wrap .security-requirements * {color:#0f172a !important}

      /* Headings emphasis */
      body.theme-dark .redoc-wrap h1, body.theme-dark .redoc-wrap h2, body.theme-dark .redoc-wrap h3, body.theme-dark .redoc-wrap h4, body.theme-dark .redoc-wrap h5 {color:#e6edf3 !important}
      body.theme-light .redoc-wrap h1, body.theme-light .redoc-wrap h2, body.theme-light .redoc-wrap h3, body.theme-light .redoc-wrap h4, body.theme-light .redoc-wrap h5 {color:#0f172a !important}

      /* Code blocks / examples */
      body.theme-dark .redoc-wrap pre, 
      body.theme-dark .redoc-wrap code,
      body.theme-dark .redoc-wrap .example,
      body.theme-dark .redoc-wrap .redoc-json,
      body.theme-dark .redoc-wrap .tooltip-content{background:#0a0f1a !important;color:#e6edf3 !important;border-color:#1f2937 !important}

      body.theme-light .redoc-wrap pre,
      body.theme-light .redoc-wrap code,
      body.theme-light .redoc-wrap .example,
      body.theme-light .redoc-wrap .redoc-json,
      body.theme-light .redoc-wrap .tooltip-content{background:#ffffff !important;color:#111827 !important;border-color:#e5e7eb !important}

      /* Prism token colors to guarantee readability */
      /* Light */
      body.theme-light .redoc-wrap code[class*="language-"],
      body.theme-light .redoc-wrap pre[class*="language-"]{color:#111827 !important}
      body.theme-light .redoc-wrap .token.property{color:#111827 !important;font-weight:600}
      body.theme-light .redoc-wrap .token.string{color:#166534 !important}
      body.theme-light .redoc-wrap .token.number{color:#1d4ed8 !important}
      body.theme-light .redoc-wrap .token.boolean, 
      body.theme-light .redoc-wrap .token.null{color:#b91c1c !important}
      body.theme-light .redoc-wrap .token.punctuation{color:#475569 !important}

      /* Dark */
      body.theme-dark .redoc-wrap code[class*="language-"],
      body.theme-dark .redoc-wrap pre[class*="language-"]{color:#e6edf3 !important}
      body.theme-dark .redoc-wrap .token.property{color:#e6edf3 !important;font-weight:600}
      body.theme-dark .redoc-wrap .token.string{color:#22c55e !important}
      body.theme-dark .redoc-wrap .token.number{color:#93c5fd !important}
      body.theme-dark .redoc-wrap .token.boolean, 
      body.theme-dark .redoc-wrap .token.null{color:#f87171 !important}
      body.theme-dark .redoc-wrap .token.punctuation{color:#94a3b8 !important}

      /* Tables */
      body.theme-dark .redoc-wrap table{background:#0b1324 !important;border-color:#1f2937 !important}
      body.theme-dark .redoc-wrap table th, 
      body.theme-dark .redoc-wrap table td{color:#e6edf3 !important;border-color:#1f2937 !important}

      body.theme-light .redoc-wrap table{background:#ffffff !important;border-color:#e5e7eb !important}
      body.theme-light .redoc-wrap table th, 
      body.theme-light .redoc-wrap table td{color:#0f172a !important;border-color:#e5e7eb !important}

      /* Links */
      body.theme-dark .redoc-wrap a{color:#58a6ff !important}
      body.theme-light .redoc-wrap a{color:#0ea5e9 !important}
    </style>
  </head>
  <body>
    <div class="topbar">
      <div class="brand">Slot Game API</div>
      <div class="nav">
        <a href="/openapi" target="_blank">OpenAPI YAML</a>
        <a href="/docs/redoc">Redoc</a>
        <a id="swaggerLink" href="/swagger/index.html">Swagger UI</a>
        <button id="themeToggle" title="切换主题">切换主题</button>
      </div>
    </div>
    <div class="redoc-wrap wrap"></div>` + note + `
    ` + scriptTag + `
    <script>
      // Theme management
      const THEME_KEY = 'redoc_theme';
      function getTheme(){ try{ return localStorage.getItem(THEME_KEY) || 'light'; }catch(e){ return 'light'; } }
      function setTheme(v){ try{ localStorage.setItem(THEME_KEY, v); }catch(e){} }
      function applyBodyTheme(v){ document.body.classList.remove('theme-dark','theme-light'); document.body.classList.add('theme-'+v); }

      const themeLight = {
        colors:{
          primary:{ main:'#0ea5e9' },
          text:{ primary:'#0f172a', secondary:'#374151', disabled:'#9ca3af' },
          http:{ get:'#16a34a', post:'#2563eb', put:'#d97706', delete:'#dc2626' },
          border:{ light:'#e5e7eb' },
          background:{ default:'#ffffff', alternative:'#f9fafb' }
        },
        sidebar:{ backgroundColor:'#ffffff', textColor:'#0f172a', activeTextColor:'#0ea5e9', width:'280px' },
        rightPanel:{ backgroundColor:'#ffffff', width:'40%' },
        codeBlock:{ backgroundColor:'#f3f4f6', textColor:'#0f172a' },
        typography:{ fontSize:'14px' }
      };
      const themeDark = {
        colors:{
          primary:{ main:'#58a6ff' },
          text:{ primary:'#e6edf3', secondary:'#cbd5e1', disabled:'#64748b' },
          http:{ get:'#22c55e', post:'#60a5fa', put:'#eab308', delete:'#f87171' },
          border:{ light:'#1f2937' },
          background:{ default:'#0b1324', alternative:'#0a0f1a' }
        },
        sidebar:{ backgroundColor:'#0b1324', textColor:'#e6edf3', activeTextColor:'#58a6ff', width:'280px' },
        rightPanel:{ backgroundColor:'#0b1324', width:'40%' },
        codeBlock:{ backgroundColor:'#0a0f1a', textColor:'#e6edf3' },
        typography:{ fontSize:'14px' }
      };

      function renderRedoc(){
        const t = getTheme();
        applyBodyTheme(t);
        const themeObj = t==='light'? themeLight : themeDark;
        const container = document.querySelector('.redoc-wrap');
        container.innerHTML = '';
        Redoc.init('/openapi', { expandResponses: '200,201', theme: themeObj }, container);
      }

      // initial render
      renderRedoc();

      // toggle button
      (function(){
        const btn = document.getElementById('themeToggle');
        if(!btn) return;
        btn.addEventListener('click', function(){
          const cur = getTheme();
          const next = cur==='dark' ? 'light' : 'dark';
          setTheme(next);
          renderRedoc();
        });
      })();

      ;(function(){
        var link=document.getElementById('swaggerLink');
        function disable(){ if(!link) return; link.classList.add('dim'); link.title='需使用 make run-swagger 启用'; link.addEventListener('click',function(e){ e.preventDefault(); alert('Swagger UI 未启用\n请运行: make run-swagger'); }); }
        fetch('/swagger/index.html', {method:'GET'}).then(function(res){ if(!res.ok){ disable(); } }).catch(disable);
      })();
    </script>
  </body>
</html>`
    c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func serveSwaggerUI(c *gin.Context) {
    // 使用本地 swagger-ui-dist（若存在），否则回退 CDN
    useLocal := false
    if _, err := os.Stat("static/vendors/swagger-ui/swagger-ui.css"); err == nil {
        if _, err2 := os.Stat("static/vendors/swagger-ui/swagger-ui-bundle.js"); err2 == nil {
            useLocal = true
        }
    }
    cssHref := "https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"
    jsBundle := "https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"
    jsPreset := "https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"
    if useLocal {
        cssHref = "/static/vendors/swagger-ui/swagger-ui.css"
        jsBundle = "/static/vendors/swagger-ui/swagger-ui-bundle.js"
        // 预设文件若存在则使用本地
        if _, err := os.Stat("static/vendors/swagger-ui/swagger-ui-standalone-preset.js"); err == nil {
            jsPreset = "/static/vendors/swagger-ui/swagger-ui-standalone-preset.js"
        }
    }

    html := `<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <title>Slot Game API - Swagger UI</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="` + cssHref + `">
    <style>
      body{margin:0;background:#ffffff}
      .topbar{position:fixed;top:0;left:0;right:0;height:48px;display:flex;align-items:center;justify-content:space-between;padding:0 12px;background:linear-gradient(90deg,#0b1324,#0f172a);border-bottom:1px solid #172036;z-index:10}
      .brand{font:600 14px/1 -apple-system,Segoe UI,Helvetica,Arial;color:#0f172a}
      .nav a{color:#0f172a;text-decoration:none;margin-left:12px;padding:6px 10px;border-radius:6px;border:1px solid #d1d5db;background:#f8fafc}
      .nav a:hover{border-color:#94a3b8;color:#0f172a}
      .wrap{margin-top:48px}
      .tip{position:fixed;right:8px;top:52px;background:#eef2ff;border:1px solid #c7d2fe;color:#1f2937;padding:6px 10px;border-radius:6px;font:12px/1.2 -apple-system,Segoe UI,Helvetica,Arial}
    </style>
  </head>
  <body>
    <div class="topbar">
      <div class="brand">Slot Game API</div>
      <div class="nav">
        <a href="/openapi" target="_blank">OpenAPI YAML</a>
        <a href="/docs/redoc" title="Redoc 版 UI（支持深浅主题切换）">Redoc</a>
        <a href="/docs/index.html" title="gin-swagger 原生页面（启用 -tags swagger）">Swagger (original)</a>
      </div>
    </div>
    <div class="wrap">
      <div id="swagger-ui"></div>
    </div>
    <div class="tip">提示：此页面基于 CDN 渲染，如需完全离线可引入 swagger-ui-dist 静态资源。</div>
    <script src="` + jsBundle + `" crossorigin></script>
    <script src="` + jsPreset + `" crossorigin></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: '/openapi',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
        layout: 'BaseLayout'
      })
    </script>
  </body>
</html>`
    c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
