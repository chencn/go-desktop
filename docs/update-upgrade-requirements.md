# 更新升级需求说明

本文档记录 GitHub 升级和本地静态服务器升级的目标规则。当前内容是需求规划，不代表代码已经实现。

## 目标

- 同时支持 `github` 和 `local` 两种更新源。
- 更新源选择持久化到 SQLite 配置中，默认值为 `github`。
- 用户选择 `github` 时，只走 GitHub Release 更新链路。
- 用户选择 `local` 时，只走本地静态 HTTP 更新链路。
- 两种更新源最终复用同一套下载、SHA256 校验、pending 安装和静默安装流程。

## 默认配置

本地升级默认地址写入 `project.metadata.json`，方便后续修改。

```json
{
  "update": {
    "defaultSource": "github",
    "localBaseUrl": "http://www.xqchen.shop/exe/go-desktop",
    "localManifestPath": "releases/latest.json"
  }
}
```

本地 manifest 固定 URL：

```text
http://www.xqchen.shop/exe/go-desktop/releases/latest.json
```

## 本地服务器目录

本地服务器大概率使用 HTTP 静态服务，目录布局按 GitHub Release 下载路径设计。

```text
http://www.xqchen.shop/exe/go-desktop/
  releases/
    latest.json
    download/
      v1.0.0/
        go-desktop-v1.0.0-windows-amd64.exe
        go-desktop-v1.0.0-windows-amd64.exe.sha256
```

本地打包 staging 目录：

```text
bin/
  go-desktop/
    releases/
      latest.json
      download/
        v1.0.0/
          go-desktop-v1.0.0-windows-amd64.exe
          go-desktop-v1.0.0-windows-amd64.exe.sha256
```

## latest.json 内容

`latest.json` 使用 GitHub `List releases` API 的数组结构。地址不模拟 GitHub API，内容模拟 GitHub API。

最小可用内容如下：

```json
[
  {
    "tag_name": "v1.0.0",
    "name": "go-desktop v1.0.0",
    "html_url": "http://www.xqchen.shop/exe/go-desktop/releases/download/v1.0.0/",
    "body": "更新说明",
    "draft": false,
    "prerelease": false,
    "assets": [
      {
        "name": "go-desktop-v1.0.0-windows-amd64.exe",
        "size": 12345678,
        "digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
        "browser_download_url": "http://www.xqchen.shop/exe/go-desktop/releases/download/v1.0.0/go-desktop-v1.0.0-windows-amd64.exe"
      },
      {
        "name": "go-desktop-v1.0.0-windows-amd64.exe.sha256",
        "size": 94,
        "browser_download_url": "http://www.xqchen.shop/exe/go-desktop/releases/download/v1.0.0/go-desktop-v1.0.0-windows-amd64.exe.sha256"
      }
    ]
  }
]
```

字段要求：

- `tag_name`：发布版本标签，统一使用 `vX.Y.Z`。
- `name`：Release 名称，建议为 `go-desktop vX.Y.Z`。
- `html_url`：Release 展示地址；本地静态服务可指向版本目录。
- `body`：更新说明。
- `draft`：必须为 `false`，否则应被跳过。
- `prerelease`：必须为 `false`，否则应被跳过。
- `assets[].name`：资产文件名。
- `assets[].size`：资产大小，单位字节。
- `assets[].digest`：安装包 SHA256，格式为 `sha256:<64位hex>`。
- `assets[].browser_download_url`：安装包或 `.sha256` 文件下载地址。

`.sha256` 文件内容保持 GitHub workflow 当前格式：

```text
0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  go-desktop-v1.0.0-windows-amd64.exe
```

`digest` 和 `.sha256` 都要生成：`digest` 用于直接校验，`.sha256` 用于保持 GitHub Release 资产兜底兼容。

## 版本规则

- 版本号统一规范化为三段数字：`X.Y.Z`。
- `v1.0`、`1.0` 都视为 `1.0.0`。
- `v1`、`1` 都视为 `1.0.0`。
- 超过三段、包含非法字符、负数版本号都视为无效。
- 对外 tag 和目录使用 `vX.Y.Z`。
- 应用内部比较和注入版本使用 `X.Y.Z`。
- `project.metadata.json` 里的 `defaultVersion: "1.0.0"` 作为默认兜底版本；正式 GitHub 打包仍必须使用 tag 版本。

## 打包版本来源

GitHub 自动打包：

- 版本来自 GitHub tag。
- tag 允许 `v1`、`v1.0`、`v1.0.0`，打包前规范化为 `1.0.0` 这类三段版本。
- GitHub 打包必须使用 GitHub tag 规范化后的版本。

本地打包：

- 版本来自本地 `info.version`。
- 如果本地打包时能同时拿到 GitHub tag 版本，则比较 `info.version` 和 tag 版本。
- 本地打包使用二者规范化后较大的版本。
- 本地打包产物、Windows 版本资源、安装包文件名、`latest.json`、`.sha256` 都使用最终选出的版本。

## 更新检查和后台任务

- 更新检查只按当前选择的更新源执行，不同时检查 GitHub 和本地。
- 软件启动后 1 分钟自动发起一次后台更新检查。
- 如果发现新版本且 SHA256 完整，后台静默下载并校验安装包。
- 静默下载只到 `verified` 状态，不自动安装。
- 正常情况下，用户仍然通过更新弹窗选择“马上更新”或“下次启动再更新”。
- 用户点击右上角更新按键打开更新弹窗时，先自动检查本地是否已有下载完成且 SHA256 验证通过的安装包。
- 如果存在已下载且 SHA256 验证通过的安装包，则不再等待用户二次选择，直接强制进入升级安装流程。
- 增加定时检查任务，按配置里的检查间隔执行。
- 默认检查间隔为 3 小时。
- 间隔配置继续持久化到 SQLite。

## 实现边界

- 不拆两套下载器和安装器。
- GitHub checker 和 local checker 最终都输出统一的检查结果。
- 本地 checker 只负责读取 `releases/latest.json` 并解析 GitHub 兼容结构。
- 下载、进度、SHA256 校验、缓存路径、pending 安装和静默安装器复用现有更新管理器。
- 本地服务器只要求提供静态文件，不要求动态接口。
