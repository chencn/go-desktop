// ============================================================================
// 文件: MainActivity.java
// 描述: Android 主活动
//
// 功能概述:
// - 承载 WebView 并管理 Wails 应用生命周期
// - 使用 WebViewAssetLoader 从 Go 库提供资源
// - 处理 Wails API 请求和 JavaScript 注入
// ============================================================================

package com.wails.app;

import android.annotation.SuppressLint;
import android.os.Bundle;
import android.util.Log;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

import androidx.annotation.Nullable;
import androidx.appcompat.app.AppCompatActivity;
import androidx.webkit.WebViewAssetLoader;
import com.wails.app.BuildConfig;

/**
 * 主活动类
 * 承载 WebView 并管理 Wails 应用生命周期
 * 使用 WebViewAssetLoader 从 Go 库提供资源，无需网络服务器
 */
public class MainActivity extends AppCompatActivity {
    // 日志标签
    private static final String TAG = "WailsActivity";
    // Wails 方案（https）
    private static final String WAILS_SCHEME = "https";
    // Wails 主机名
    private static final String WAILS_HOST = "wails.localhost";

    // WebView 实例
    private WebView webView;
    // Wails 桥接器
    private WailsBridge bridge;
    // 资源加载器
    private WebViewAssetLoader assetLoader;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        // 初始化原生 Go 库
        bridge = new WailsBridge(this);
        bridge.initialize();

        // 设置 WebView
        setupWebView();

        // 加载应用
        loadApplication();
    }

    @SuppressLint("SetJavaScriptEnabled")
    private void setupWebView() {
        webView = findViewById(R.id.webview);

        // 配置 WebView 设置
        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(true);
        settings.setDomStorageEnabled(true);
        settings.setDatabaseEnabled(true);
        settings.setAllowFileAccess(false);
        settings.setAllowContentAccess(false);
        settings.setMediaPlaybackRequiresUserGesture(false);
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_NEVER_ALLOW);

        // 在调试构建中启用调试
        if (BuildConfig.DEBUG) {
            WebView.setWebContentsDebuggingEnabled(true);
        }

        // 设置资源加载器
        assetLoader = new WebViewAssetLoader.Builder()
                .setDomain(WAILS_HOST)
                .addPathHandler("/", new WailsPathHandler(bridge))
                .build();

        // 设置 WebView 客户端以拦截请求
        webView.setWebViewClient(new WebViewClient() {
            @Nullable
            @Override
            public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
                String url = request.getUrl().toString();
                Log.d(TAG, "拦截请求: " + url);

                // 处理 wails.localhost 请求
                if (request.getUrl().getHost() != null &&
                        request.getUrl().getHost().equals(WAILS_HOST)) {

                    // 对于 Wails API 调用，需要传递完整 URL（包括查询字符串）
                    // 因为 WebViewAssetLoader.PathHandler 会剥离查询参数
                    String path = request.getUrl().getPath();
                    if (path != null && path.startsWith("/wails/")) {
                        // 获取包含查询字符串的完整路径
                        String fullPath = path;
                        String query = request.getUrl().getQuery();
                        if (query != null && !query.isEmpty()) {
                            fullPath = path + "?" + query;
                        }
                        Log.d(TAG, "检测到 Wails API 调用，完整路径: " + fullPath);

                        // 直接使用完整路径调用桥接器
                        byte[] data = bridge.serveAsset(fullPath, request.getMethod(), "{}");
                        if (data != null && data.length > 0) {
                            java.io.InputStream inputStream = new java.io.ByteArrayInputStream(data);
                            java.util.Map<String, String> headers = new java.util.HashMap<>();
                            headers.put("Access-Control-Allow-Origin", "*");
                            headers.put("Cache-Control", "no-cache");
                            headers.put("Content-Type", "application/json");

                            return new WebResourceResponse(
                                "application/json",
                                "UTF-8",
                                200,
                                "OK",
                                headers,
                                inputStream
                            );
                        }
                        // 如果数据为空，返回错误响应
                        return new WebResourceResponse(
                            "application/json",
                            "UTF-8",
                            500,
                            "内部错误",
                            new java.util.HashMap<>(),
                            new java.io.ByteArrayInputStream("{}".getBytes())
                        );
                    }

                    // 对于常规资源，使用资源加载器
                    return assetLoader.shouldInterceptRequest(request.getUrl());
                }

                return super.shouldInterceptRequest(view, request);
            }

            @Override
            public void onPageFinished(WebView view, String url) {
                super.onPageFinished(view, url);
                Log.d(TAG, "页面加载完成: " + url);
                // 注入 Wails 运行时
                bridge.injectRuntime(webView, url);
            }
        });

        // 添加 JavaScript 接口用于 Go 通信
        webView.addJavascriptInterface(new WailsJSBridge(bridge, webView), "wails");
    }

    /**
     * 从资产服务器加载主页
     */
    private void loadApplication() {
        String url = WAILS_SCHEME + "://" + WAILS_HOST + "/";
        Log.d(TAG, "加载 URL: " + url);
        webView.loadUrl(url);
    }

    /**
     * 从 Go 端在 WebView 中执行 JavaScript
     * 参数:
     *   - js: 要执行的 JavaScript 代码
     */
    public void executeJavaScript(final String js) {
        runOnUiThread(() -> {
            if (webView != null) {
                webView.evaluateJavascript(js, null);
            }
        });
    }

    @Override
    protected void onResume() {
        super.onResume();
        if (bridge != null) {
            bridge.onResume();
        }
    }

    @Override
    protected void onPause() {
        super.onPause();
        if (bridge != null) {
            bridge.onPause();
        }
    }

    @Override
    protected void onDestroy() {
        super.onDestroy();
        if (bridge != null) {
            bridge.shutdown();
        }
        if (webView != null) {
            webView.destroy();
        }
    }

    @Override
    public void onBackPressed() {
        if (webView != null && webView.canGoBack()) {
            webView.goBack();
        } else {
            super.onBackPressed();
        }
    }
}
