// ============================================================================
// 文件: WailsBridge.java
// 描述: Wails 桥接器
//
// 功能概述:
// - 管理 Java/Android 端与 Go 原生库之间的连接
// - 处理资源请求、消息传递和回调管理
// - 负责原生 Go 库的加载和初始化
// ============================================================================

package com.wails.app;

import android.content.Context;
import android.util.Log;
import android.webkit.WebView;

import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Wails 桥接器类
 * 管理 Java/Android 端与 Go 原生库之间的连接
 * 功能包括:
 * - 加载和初始化原生 Go 库
 * - 从 Go 端提供资源请求
// - 在 JavaScript 和 Go 之间传递消息
// - 管理异步操作的回调
 */
public class WailsBridge {
    // 日志标签
    private static final String TAG = "WailsBridge";

    // 静态代码块：加载原生 Go 库
    static {
        System.loadLibrary("wails");
    }

    // Android 上下文
    private final Context context;
    // 回调 ID 生成器
    private final AtomicInteger callbackIdGenerator = new AtomicInteger(0);
    // 待处理的资源回调
    private final ConcurrentHashMap<Integer, AssetCallback> pendingAssetCallbacks = new ConcurrentHashMap<>();
    // 待处理的消息回调
    private final ConcurrentHashMap<Integer, MessageCallback> pendingMessageCallbacks = new ConcurrentHashMap<>();
    // WebView 实例
    private WebView webView;
    // 是否已初始化标志
    private volatile boolean initialized = false;

    // ============================================================================
    // 原生方法声明（在 Go 中实现）
    // ============================================================================

    private static native void nativeInit(WailsBridge bridge);
    private static native void nativeShutdown();
    private static native void nativeOnResume();
    private static native void nativeOnPause();
    private static native void nativeOnPageFinished(String url);
    private static native byte[] nativeServeAsset(String path, String method, String headers);
    private static native String nativeHandleMessage(String message);
    private static native String nativeGetAssetMimeType(String path);

    // ============================================================================
    // 构造函数
    // ============================================================================

    /**
     * 构造函数
     * 参数:
     *   - context: Android 上下文
     */
    public WailsBridge(Context context) {
        this.context = context;
    }

    // ============================================================================
    // 生命周期方法
    // ============================================================================

    /**
     * 初始化原生 Go 库
     */
    public void initialize() {
        if (initialized) {
            return;
        }

        Log.i(TAG, "正在初始化 Wails 桥接器...");
        try {
            nativeInit(this);
            initialized = true;
            Log.i(TAG, "Wails 桥接器初始化成功");
        } catch (Exception e) {
            Log.e(TAG, "初始化 Wails 桥接器失败", e);
        }
    }

    /**
     * 关闭原生 Go 库
     */
    public void shutdown() {
        if (!initialized) {
            return;
        }

        Log.i(TAG, "正在关闭 Wails 桥接器...");
        try {
            nativeShutdown();
            initialized = false;
        } catch (Exception e) {
            Log.e(TAG, "关闭时出错", e);
        }
    }

    /**
     * 活动恢复时调用
     */
    public void onResume() {
        if (initialized) {
            nativeOnResume();
        }
    }

    /**
     * 活动暂停时调用
     */
    public void onPause() {
        if (initialized) {
            nativeOnPause();
        }
    }

    // ============================================================================
    // 资源服务方法
    // ============================================================================

    /**
     * 从 Go 资产服务器获取资源
     * 参数:
     *   - path: 请求的 URL 路径
     *   - method: HTTP 方法
     *   - headers: 请求头（JSON 格式）
     * 返回:
     *   - byte[]: 资源数据，未找到时返回 null
     */
    public byte[] serveAsset(String path, String method, String headers) {
        if (!initialized) {
            Log.w(TAG, "桥接器未初始化，无法提供资源: " + path);
            return null;
        }

        Log.d(TAG, "提供资源: " + path);
        try {
            return nativeServeAsset(path, method, headers);
        } catch (Exception e) {
            Log.e(TAG, "提供资源出错: " + path, e);
            return null;
        }
    }

    /**
     * 获取资源的 MIME 类型
     * 参数:
     *   - path: 资源路径
     * 返回:
     *   - String: MIME 类型字符串
     */
    public String getAssetMimeType(String path) {
        if (!initialized) {
            return "application/octet-stream";
        }

        try {
            String mimeType = nativeGetAssetMimeType(path);
            return mimeType != null ? mimeType : "application/octet-stream";
        } catch (Exception e) {
            Log.e(TAG, "获取 MIME 类型出错: " + path, e);
            return "application/octet-stream";
        }
    }

    // ============================================================================
    // 消息处理方法
    // ============================================================================

    /**
     * 处理来自 JavaScript 的消息
     * 参数:
     *   - message: 来自 JavaScript 的消息（JSON 格式）
     * 返回:
     *   - String: 发送给 JavaScript 的响应（JSON 格式）
     */
    public String handleMessage(String message) {
        if (!initialized) {
            Log.w(TAG, "桥接器未初始化，无法处理消息");
            return "{\"error\":\"桥接器未初始化\"}";
        }

        Log.d(TAG, "处理来自 JS 的消息: " + message);
        try {
            return nativeHandleMessage(message);
        } catch (Exception e) {
            Log.e(TAG, "处理消息出错", e);
            return "{\"error\":\"" + e.getMessage() + "\"}";
        }
    }

    // ============================================================================
    // WebView 交互方法
    // ============================================================================

    /**
     * 向 WebView 注入 Wails 运行时 JavaScript
     * 页面加载完成时调用
     * 参数:
     *   - webView: 要注入的 WebView
     *   - url: 加载完成的 URL
     */
    public void injectRuntime(WebView webView, String url) {
        this.webView = webView;
        // 通知 Go 端页面已加载完成，以便注入运行时
        Log.d(TAG, "页面加载完成: " + url + "，通知 Go 端");
        if (initialized) {
            nativeOnPageFinished(url);
        }
    }

    /**
     * 在 WebView 中执行 JavaScript（从 Go 端调用）
     * 参数:
     *   - js: 要执行的 JavaScript 代码
     */
    public void executeJavaScript(String js) {
        if (webView != null) {
            webView.post(() -> webView.evaluateJavascript(js, null));
        }
    }

    /**
     * 从 Go 端触发事件时调用，将事件发送给 JavaScript
     * 参数:
     *   - eventName: 事件名称
     *   - eventData: 事件数据（JSON 格式）
     */
    public void emitEvent(String eventName, String eventData) {
        String js = String.format("window.wails && window.wails._emit('%s', %s);",
                escapeJsString(eventName), eventData);
        executeJavaScript(js);
    }

    /**
     * 转义 JavaScript 字符串
     * 参数:
     *   - str: 原始字符串
     * 返回:
     *   - String: 转义后的字符串
     */
    private String escapeJsString(String str) {
        return str.replace("\\", "\\\\")
                .replace("'", "\\'")
                .replace("\n", "\\n")
                .replace("\r", "\\r");
    }

    // ============================================================================
    // 回调接口
    // ============================================================================

    /**
     * 资源回调接口
     */
    public interface AssetCallback {
        void onAssetReady(byte[] data, String mimeType);
        void onAssetError(String error);
    }

    /**
     * 消息回调接口
     */
    public interface MessageCallback {
        void onResponse(String response);
        void onError(String error);
    }
}
