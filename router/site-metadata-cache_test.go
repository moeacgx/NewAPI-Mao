package router

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	htmlnode "golang.org/x/net/html"
)

func TestSetThemeWebRouterVersionsLocalStylesheets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousTheme := common.GetTheme()
	previousVersion := common.Version
	previousRateLimit := common.GlobalWebRateLimitEnable
	common.Version = "v1.2.3+css&cache"
	common.GlobalWebRateLimitEnable = false
	t.Cleanup(func() {
		common.SetTheme(previousTheme)
		common.Version = previousVersion
		common.GlobalWebRateLimitEnable = previousRateLimit
	})

	page := `<!doctype html><html><head>
<link id="current" rel="stylesheet" href="/assets/index-THEME.css">
<link id="queried" rel="stylesheet" href="/assets/shared.css?keep=a%26b&amp;keep=two&amp;v=old&amp;v=older#section">
<link id="external" rel="stylesheet" href="https://cdn.example.com/assets/external.css?v=old#external">
<link id="network" rel="stylesheet" href="//cdn.example.com/assets/external.css">
<link id="relative" rel="stylesheet" href="assets/relative.css">
<link id="outside" rel="stylesheet" href="/other/style.css">
<link id="not-css" rel="stylesheet" href="/assets/theme.js">
<link id="preload" rel="preload" as="style" href="/assets/preload.css">
<script id="script" src="/assets/index-THEME.js"></script>
</head><body><div id="root"></div></body></html>`
	roots := map[string]string{"default": t.TempDir(), "classic": t.TempDir()}
	pages := make(map[string][]byte)
	for theme, root := range roots {
		require.NoError(t, os.MkdirAll(filepath.Join(root, "assets"), 0o755))
		for _, name := range []string{"index-" + theme + ".css", "shared.css"} {
			require.NoError(t, os.WriteFile(filepath.Join(root, "assets", name), []byte(theme+"-css"), 0o600))
		}
		// 重复预处理仍应只保留一个当前版本参数。
		pages[theme] = prepareSiteIndexPage(prepareSiteIndexPage([]byte(strings.ReplaceAll(page, "THEME", theme))))
	}
	engine := gin.New()
	setThemeWebRouter(engine, static.LocalFile(roots["default"], false), pages["default"], static.LocalFile(roots["classic"], false), pages["classic"])
	for _, theme := range []string{"default", "classic"} {
		t.Run(theme, func(t *testing.T) {
			common.SetTheme(theme)
			for _, entry := range []string{"/", "/index.html", "/console/log"} {
				t.Run(entry, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, entry, nil))
					require.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, "no-cache", recorder.Header().Get("Cache-Control"))
					document, err := htmlnode.Parse(strings.NewReader(recorder.Body.String()))
					require.NoError(t, err)
					head := findHTMLHead(document)
					require.NotNil(t, head)
					links := make(map[string]string)
					for child := head.FirstChild; child != nil; child = child.NextSibling {
						var id, target string
						for _, attribute := range child.Attr {
							if attribute.Key == "id" {
								id = attribute.Val
							} else if attribute.Key == "href" || attribute.Key == "src" {
								target = attribute.Val
							}
						}
						if id != "" {
							links[id] = target
						}
					}
					for _, id := range []string{"current", "queried"} {
						target, err := url.Parse(links[id])
						require.NoError(t, err)
						assert.Equal(t, []string{common.Version}, target.Query()["v"])
						if id == "queried" {
							assert.Equal(t, []string{"a&b", "two"}, target.Query()["keep"])
							assert.Equal(t, "section", target.Fragment)
						} else {
							assert.Equal(t, "/assets/index-"+theme+".css", target.Path)
						}
						// 静态服务必须接受版本参数，仍返回实际样式内容。
						target.Fragment = ""
						asset := httptest.NewRecorder()
						engine.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, target.String(), nil))
						require.Equal(t, http.StatusOK, asset.Code)
						assert.Equal(t, theme+"-css", asset.Body.String())
						assert.Equal(t, "max-age=604800", asset.Header().Get("Cache-Control"))
					}
					for id, target := range map[string]string{
						"external": "https://cdn.example.com/assets/external.css?v=old#external",
						"network":  "//cdn.example.com/assets/external.css",
						"relative": "assets/relative.css",
						"outside":  "/other/style.css",
						"not-css":  "/assets/theme.js",
						"preload":  "/assets/preload.css",
						"script":   "/assets/index-" + theme + ".js",
					} {
						assert.Equal(t, target, links[id], id)
					}
				})
			}
		})
	}
}
