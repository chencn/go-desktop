// ============================================================================
// 文件: WailsJSBridge.java
// 描述: Wails JavaScript 桥接器
//
// 功能概述:
// - 提供 JavaScript 接口，允许网页前端与 Go 后端通信
// - 暴露给 JavaScript 的 `window.wails` 对象
// - 支持同步调用、异步调用、日志记录等功能
// ============================================================================

package com.wails.app;

import android.util.Log;
import android.webkit.JavascriptInterface;
import android.webkit.WebView;
import com.wails.app.BuildConfig;

/**
 * Wails JavaScript 桥接器类
 * 提供 JavaScript 接口，允许网页前端与 Go 后端通信
 * 暴露给 JavaScript 的 `window.wails` 对象
 * 类似于 iOS 的 WKScriptMessageHandler，使用 Android 的 addJavascriptInterface
 */
public class WailsJSBridge {
    // 日志标签
    private static final String TAG = "WailsJSBridge";

    // Wails 桥接器实例
    private final WailsBridge bridge;
    // WebView 实例
    private final WebView webView;

    /**
     * 构造函数
     * 参数:
     *   - bridge: Wails 桥接器实例
     *   - webView: WebView 实例
     */
    public WailsJSBridge(WailsBridge bridge, WebView webView) {
        this.bridge = bridge;
        this.webView = webView;
    }

    /**
     * 同步发送消息给 Go 并返回响应
     * JavaScript 调用: wails.invoke(message)
     * 参数:
     *   - message: 要发送的消息（JSON 字符串）
     * 返回:
     *   - String: Go 的响应（JSON 字符串）
     */
    @JavascriptInterface
    public String invoke(String message) {
        Log.d(TAG, "Invoke 被调用: " + message);
        return bridge.handleMessage(message);
    }

    /**
     * 异步发送消息给 Go
     * 响应将通过回调返回
     * JavaScript 调用: wails.invokeAsync(callbackId, message)
     * 参数:
     *   - callbackId: 用于响应的回调 ID
     *   - message: 要发送的消息（JSON 字符串）
     */
    @JavascriptInterface
    public void invokeAsync(final String callbackId, final String message) {
        Log.d(TAG, "InvokeAsync 被调用: " + message);

        // 在后台线程处理，避免阻塞 JavaScript
        new Thread(() -> {
            try {
                String response = bridge.handleMessage(message);
                sendCallback(callbackId, response, null);
            } catch (Exception e) {
                Log.e(TAG, "异步调用出错", e);
                sendCallback(callbackId, null, e.getMessage());
            }
        }).start();
    }

    /**
     * 将 JavaScript 的日志消息输出到 Android logcat
     * JavaScript 调用: wails.log(level, message)
     * 参数:
     *   - level: 日志级别（debug, info, warn, error）
     *   - message: 要记录的消息
     */
    @JavascriptInterface
    public void log(String level, String message) {
        switch (level.toLowerCase()) {
            case "debug":
                Log.d(TAG + "/JS", message);
                break;
            case "info":
                Log.i(TAG + "/JS", message);
                break;
            case "warn":
                Log.w(TAG + "/JS", message);
                break;
            case "error":
                Log.e(TAG + "/JS", message);
                break;
            default:
                Log.v(TAG + "/JS", message);
                break;
        }
    }

    /**
     * 获取平台名称
     * JavaScript 调用: wails.platform()
     * 返回:
     *   - String: "android"
     */
    @JavascriptInterface
    public String platform() {
        return "android";
    }

    /**
     * 检查是否运行在调试模式
     * JavaScript 调用: wails.isDebug()
     * 返回:
     *   - boolean: 如果是调试构建返回 true，否则返回 false
     */
    @JavascriptInterface
    public boolean isDebug() {
        return BuildConfig.DEBUG;
    }

    /**
     * 向 JavaScript 发送回调响应
     * 参数:
     *   - callbackId: 回调 ID
     *   - result: 响应结果
     *   - error: 错误信息（如果有）
     */
    private void sendCallback(String callbackId, String result, String error) {
        final String js;
        if (error != null) {
            js = String.format(
                    "window.wails && window.wails._callback('%s', null, '%s');",
                    escapeJsString(callbackId),
                    escapeJsString(error)
            );
        } else {
            js = String.format(
                    "window.wails && window.wails._callback('%s', %s, null);",
                    escapeJsString(callbackId),
                    result != null ? result : "null"
            );
        }

        webView.post(() -> webView.evaluateJavascript(js, null));
    }

    /**
     * 转义 JavaScript 字符串
     * 参数:
     *   - str: 原始字符串
     * 返回:
     *   - String: 转义后的字符串
     */
    private String escapeJsString(String str) {
        if (str == null) return "";
        return str.replace("\\", "\\\\")
                .replace("'", "\\'")
                .replace("\n", "\\n")
                .replace("\r", "\\r");
    }
}
