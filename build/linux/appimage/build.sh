#!/usr/bin/env bash
# Copyright (c) 2018-Present Lea Anthony
# SPDX-License-Identifier: MIT

# 任一步骤失败都立即退出，避免生成不完整 AppDir 后继续打包。
set -euxo pipefail

# AppDir 名称必须和 APP_NAME 对齐，linuxdeploy 会从该目录读取二进制、图标和 desktop 文件。
APP_DIR="${APP_NAME}.AppDir"

# 组装 AppImage 标准目录结构，调用方负责提前传入二进制、图标和 desktop 文件路径。
mkdir -p "${APP_DIR}/usr/bin"
cp -r "${APP_BINARY}" "${APP_DIR}/usr/bin/"
cp "${ICON_PATH}" "${APP_DIR}/"
cp "${DESKTOP_FILE}" "${APP_DIR}/"

if [[ $(uname -m) == *x86_64* ]]; then
    # x86_64 主机使用对应 linuxdeploy AppImage，-N 允许本地缓存复用。
    wget -q -4 -N https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage
    chmod +x linuxdeploy-x86_64.AppImage

    # linuxdeploy 根据 AppDir 内容补齐运行时依赖并输出 AppImage。
    ./linuxdeploy-x86_64.AppImage --appdir "${APP_DIR}" --output appimage
else
    # arm64 主机使用 aarch64 版本，避免在打包机上额外引入仿真层。
    wget -q -4 -N https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-aarch64.AppImage
    chmod +x linuxdeploy-aarch64.AppImage

    # arm64 路径和 x86_64 保持同样输入输出，便于上层 Taskfile 复用。
    ./linuxdeploy-aarch64.AppImage --appdir "${APP_DIR}" --output appimage
fi

# linuxdeploy 生成名包含架构和版本信息；项目发布链统一消费固定文件名。
mv "${APP_NAME}*.AppImage" "${APP_NAME}.AppImage"

