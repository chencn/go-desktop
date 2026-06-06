// ============================================================================
// 文件: WailsPathHandler.java
// 描述: Wails 路径处理器
//
// 功能概述:
// - 实现 WebViewAssetLoader.PathHandler 以从 Go 资产服务器提供资源
// - 允许 WebView 加载资源而无需网络服务器
// - 类似于 iOS 的 WKURLSchemeHandler
// ============================================================================

package com.wails.app;

import android.net.Uri;
import android.util.Log;
import android.webkit.WebResourceResponse;

import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.webkit.WebViewAssetLoader;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.util.HashMap;
import java.util.Map;

/**
 * Wails 路径处理器类
 * 实现 WebViewAssetLoader.PathHandler 以从 Go 资产服务器提供资源
 * 允许 WebView 加载资源而无需网络服务器
 * 类似于 iOS 的 WKURLSchemeHandler
 */
public class WailsPathHandler implements WebViewAssetLoader.PathHandler {
    // 日志标签
    private static final String TAG = "WailsPathHandler";

    // Wails 桥接器实例
    private final WailsBridge bridge;

    /**
     * 构造函数
     * 参数:
     *   - bridge: Wails 桥接器实例
     */
    public WailsPathHandler(WailsBridge bridge) {
        this.bridge = bridge;
    }

    /**
     * 处理资源路径请求
     * 参数:
     *   - path: 资源路径
     * 返回:
     *   - WebResourceResponse: 资源响应，未找到时返回 null
     */
    @Nullable
    @Override
    public WebResourceResponse handle(@NonNull String path) {
        Log.d(TAG, "处理路径: " + path);

        // 标准化路径
        if (path.isEmpty() || path.equals("/")) {
            path = "/index.html";
        }

        // 从 Go 获取资源
        byte[] data = bridge.serveAsset(path, "GET", "{}");

        if (data == null || data.length == 0) {
            Log.w(TAG, "资源未找到: " + path);
            return null; // 返回 null 让 WebView 处理 404
        }

        // 确定 MIME 类型
        String mimeType = bridge.getAssetMimeType(path);
        Log.d(TAG, "提供 " + path + " 类型 " + mimeType + " (" + data.length + " 字节)");

        // 创建响应
        InputStream inputStream = new ByteArrayInputStream(data);
        Map<String, String> headers = new HashMap<>();
        headers.put("Access-Control-Allow-Origin", "*");
        headers.put("Cache-Control", "no-cache");

        return new WebResourceResponse(
                mimeType,
                "UTF-8",
                200,
                "OK",
                headers,
                inputStream
        );
    }

    /**
     * 根据文件扩展名确定 MIME 类型
     * 参数:
     *   - path: 文件路径
     * 返回:
     *   - String: MIME 类型字符串
     */
    private String getMimeType(String path) {
        String lowerPath = path.toLowerCase();

        if (lowerPath.endsWith(".html") || lowerPath.endsWith(".htm")) {
            return "text/html";
        } else if (lowerPath.endsWith(".js") || lowerPath.endsWith(".mjs")) {
            return "application/javascript";
        } else if (lowerPath.endsWith(".css")) {
            return "text/css";
        } else if (lowerPath.endsWith(".json")) {
            return "application/json";
        } else if (lowerPath.endsWith(".png")) {
            return "image/png";
        } else if (lowerPath.endsWith(".jpg") || lowerPath.endsWith(".jpeg")) {
            return "image/jpeg";
        } else if (lowerPath.endsWith(".gif")) {
            return "image/gif";
        } else if (lowerPath.endsWith(".svg")) {
            return "image/svg+xml";
        } else if (lowerPath.endsWith(".ico")) {
            return "image/x-icon";
        } else if (lowerPath.endsWith(".woff")) {
            return "font/woff";
        } else if (lowerPath.endsWith(".woff2")) {
            return "font/woff2";
        } else if (lowerPath.endsWith(".ttf")) {
            return "font/ttf";
        } else if (lowerPath.endsWith(".eot")) {
            return "application/vnd.ms-fontobject";
        } else if (lowerPath.endsWith(".xml")) {
            return "application/xml";
        } else if (lowerPath.endsWith(".txt")) {
            return "text/plain";
        } else if (lowerPath.endsWith(".wasm")) {
            return "application/wasm";
        } else if (lowerPath.endsWith(".mp3")) {
            return "audio/mpeg";
        } else if (lowerPath.endsWith(".mp4")) {
            return "video/mp4";
        } else if (lowerPath.endsWith(".webm")) {
            return "video/webm";
        } else if (lowerPath.endsWith(".webp")) {
            return "image/webp";
        }

        return "application/octet-stream";
    }
}
