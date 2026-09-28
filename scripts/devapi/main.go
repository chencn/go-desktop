// 文件职责：开发期把桌面端挂在 /api 的本地 HTTP 门面单独跑起来，供浏览器预览调用。
//
// 为什么需要：wails3 dev 让 WebView 直接加载 Vite 源（默认 http://localhost:9245），
// 原生窗口里的前端靠 Wails 绑定调用 app.API，而绑定依赖 WebView 注入的原生桥；浏览器里没有这座桥，
// 相对请求 /api/* 又会被 Vite 的 SPA fallback 用 index.html 兜住。于是浏览器预览只能落到
// src/api/wails.ts 的假数据兜底，设置、日志、更新状态全都不是真后端的内容。
//
// 本工具只服务本地开发：新建一个 Runtime，把同一个 app.API 方法集挂到 127.0.0.1，并按 Vite
// 代理约定自己剥掉 /api 前缀。数据目录默认落在启动目录下的 .tmp/devapi（SQLite 配置库、每日日志
// 和更新缓存），预览期间改过的设置和产生的日志可以跨次启动保留；要看已安装应用的那份数据，
// 用 -data-dir 指向它的 data 目录。生产构建与原生窗口链路不经过这里。
//
// 用法：go run ./scripts/envrun go run ./scripts/devapi
// 然后让 Vite 代理指向它：GO_DESKTOP_LOCAL_API_URL=http://127.0.0.1:8081
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	desktopapp "github.com/chencn/go-desktop/app"
	"github.com/chencn/go-desktop/internal/adapters/githubrelease"
	"github.com/chencn/go-desktop/internal/desktopapp/metadata"
)

func main() {
	// addr 固定回环：门面暴露的是能改设置、清日志、退出应用的本机能力，不允许对外监听。
	addr := flag.String("addr", "127.0.0.1:8081", "本地 /api 门面监听地址（仅限回环；默认端口避开 gyt-treatment 的同名门面）")
	prefix := flag.String("prefix", "/api", "需要剥掉的本地路由前缀，与前端约定的 /api 调用路径一致")
	dataDir := flag.String("data-dir", filepath.Join(".tmp", "devapi"), "门面使用的数据目录（配置库、日志、更新缓存）")
	flag.Parse()

	if !isLoopbackHost(*addr) {
		log.Fatalf("devapi 只允许监听回环地址，收到 %q", *addr)
	}

	appRuntime := desktopapp.NewRuntime(desktopapp.ServiceOptions{
		AppName:      metadata.AppName,
		Version:      metadata.DefaultVersion,
		Description:  metadata.Description,
		Repository:   metadata.RepositoryURL,
		DatabasePath: filepath.Join(*dataDir, metadata.AppName+".db"),
		LogDirPath:   filepath.Join(*dataDir, "logs"),
		CachePath:    filepath.Join(*dataDir, "updates"),
		// 授权开关沿用构建期的环境变量注入，便于用授权版配置预览门禁页面。
		LicenseMode:      strings.TrimSpace(os.Getenv("GO_DESKTOP_LICENSE_MODE")),
		LicensePublicKey: strings.TrimSpace(os.Getenv("GO_DESKTOP_LICENSE_PUBLIC_KEY")),
		ReleaseChecker: githubrelease.NewChecker(githubrelease.Config{
			Owner:          metadata.GitHubOwner,
			Repo:           metadata.GitHubRepo,
			CurrentVersion: metadata.DefaultVersion,
			UserAgent:      metadata.UserAgent,
			APIVersion:     metadata.GitHubAPIVersion,
			AssetNames:     releaseAssetNames,
		}),
	})
	defer appRuntime.Shutdown()
	httpService := desktopapp.NewHTTPService(desktopapp.HTTPServiceOptions{API: appRuntime.API()})
	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		absDataDir = *dataDir
	}
	appRuntime.RecordLog("devapi", "本地 /api 门面已启动："+*addr+*prefix+"，数据目录 "+absDataDir)
	server := &http.Server{
		Addr:              *addr,
		Handler:           stripPrefix(*prefix, httpService),
		ReadHeaderTimeout: 5 * time.Second,
	}
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdown
		// 先停止接收新请求，再走 Runtime 收尾，保证日志缓冲落盘。
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("devapi 关闭监听失败：%v", err)
		}
	}()

	log.Printf("devapi 已启动 http://%s%s，暴露 %d 个方法（浏览器预览用，原生窗口仍走 Wails 绑定）", *addr, *prefix, len(httpService.Methods()))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("devapi 退出：%v", err)
	}
	log.Printf("devapi 已退出")
}

// stripPrefix 剥掉本地路由前缀后再交给门面 handler；门面内部的路由表以方法名开头。
func stripPrefix(prefix string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if trimmed := strings.TrimPrefix(r.URL.Path, prefix); trimmed != r.URL.Path {
			r.URL.Path = trimmed
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}
		handler.ServeHTTP(w, r)
	})
}

// isLoopbackHost 校验监听地址是回环，防止把本机控制能力暴露到局域网。
func isLoopbackHost(addr string) bool {
	host := addr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		host = addr[:i]
	}
	host = strings.Trim(host, "[]")
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// releaseAssetNames 与 main.go 保持同一组安装资产名，避免预览里的更新检查口径和真应用不一致。
func releaseAssetNames(version string) []string {
	return []string{
		metadata.WindowsInstallerAssetName(version),
		metadata.WindowsInstallerAssetNameWithoutV(version),
		metadata.WindowsSetupAssetName(version),
		metadata.WindowsSetupAssetNameWithoutV(version),
	}
}
