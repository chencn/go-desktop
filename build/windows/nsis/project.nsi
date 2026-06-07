Unicode true

####
## 由 scripts/sync_project_metadata.go 根据 project.metadata.json 生成；不要手工修改。
## 注意：这个文件里不能直接使用 Wails 模板替换，只能使用下面这些默认 define。
## 如果某个值没有在这里定义，wails_tools.nsh 会补默认值。
## 如果这里已经定义，wails_tools.nsh 不会覆盖，方便脱离 Wails 单独调试安装器。
## 
## 开发调试时先运行 Wails3 Windows 打包任务生成 wails_tools.nsh：
## > wails3 task windows:package
## 然后可以手动传入二进制路径调用 makensis。
## 仅 AMD64 安装器：
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## 仅 ARM64 安装器：
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## 同时包含两种架构的安装器：
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## 产品元数据由 project.metadata.json 生成，避免安装器和 Go/前端各写一遍。
####
!include "project_metadata.nsh"
!ifdef ARG_PRODUCT_VERSION
!define INFO_PRODUCTVERSION "${ARG_PRODUCT_VERSION}"
!else
!define INFO_PRODUCTVERSION "1.0.0"
!endif
###
####
!define REQUEST_EXECUTION_LEVEL "user"
!define UNINST_KEY_CURRENT_USER "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}"
####
## 引入 Wails 安装器辅助宏。
####
!include "wails_tools.nsh"

# Windows 版本资源必须是四段数字，这里给产品版本补最后一段。
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# 启用 HiDPI 支持。参考：https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" # 调试向导页时可加左侧图片，尺寸必须是 164x314。
!define MUI_FINISHPAGE_NOAUTOCLOSE # 只在关闭静默模式调试时可见。
!define MUI_ABORTWARNING # 只在关闭静默模式调试时可见。

SilentInstall silent
SilentUnInstall silent
AutoCloseWindow true
ShowInstDetails nevershow

!insertmacro MUI_PAGE_WELCOME # 安装器欢迎页。
# !insertmacro MUI_PAGE_LICENSE "resources\eula.txt" # 需要许可协议页时再启用。
!insertmacro MUI_PAGE_DIRECTORY # 安装目录页。
!insertmacro MUI_PAGE_INSTFILES # 安装进度页。
!insertmacro MUI_PAGE_FINISH # 安装完成页。

!insertmacro MUI_UNPAGE_INSTFILES # 卸载进度页。

!insertmacro MUI_LANGUAGE "SimpChinese" # 默认安装器语言。
!insertmacro MUI_LANGUAGE "English" # 兜底语言。

## 下面两行用于签名安装器和卸载器，%1 是待签名二进制路径。
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-v${INFO_PRODUCTVERSION}-windows-${ARCH}.exe" # 安装器输出文件名。
InstallDir "${APP_INSTALL_DIR}"

Function .onInit
   !insertmacro wails.checkArchitecture
   Call RequestRunningApplicationExit
   Call CloseRunningApplicationWindow
   Call ForceTerminateRunningApplication
FunctionEnd

Function .onInstSuccess
    IfFileExists "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 done
        DetailPrint "正在启动 ${INFO_PRODUCTNAME}..."
        Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
    done:
FunctionEnd

Function RequestRunningApplicationExit
    IfFileExists "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 done
        DetailPrint "正在请求已运行的 ${INFO_PRODUCTNAME} 退出..."
        Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}" --installer-exit'
        Sleep 1500
    done:
FunctionEnd

Function CloseRunningApplicationWindow
    FindWindow $0 "${APP_WINDOW_CLASS}" "${APP_WINDOW_TITLE}"
    ${If} $0 != 0
        DetailPrint "正在关闭已运行的 ${INFO_PRODUCTNAME}..."
        SendMessage $0 ${WM_CLOSE} 0 0 /TIMEOUT=5000
        Sleep 1500
    ${EndIf}
FunctionEnd

Function ForceTerminateRunningApplication
    FindWindow $0 "${APP_WINDOW_CLASS}" "${APP_WINDOW_TITLE}"
    ${If} $0 != 0
        System::Call 'user32::GetWindowThreadProcessId(p r0, *i .r1) i .r2'
        ${If} $1 != 0
            DetailPrint "正在确保已运行的 ${INFO_PRODUCTNAME} 退出..."
            nsExec::ExecToLog 'taskkill /PID $1 /T /F'
            Pop $2
            Sleep 500
        ${EndIf}
    ${EndIf}
FunctionEnd

Function WriteCurrentUserUninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    SetRegView 64
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "Publisher" "${INFO_COMPANYNAME}"
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "DisplayName" "${INFO_PRODUCTNAME}"
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKCU "${UNINST_KEY_CURRENT_USER}" "QuietUninstallString" "$\"$INSTDIR\uninstall.exe$\" /S"

    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKCU "${UNINST_KEY_CURRENT_USER}" "EstimatedSize" "$0"
FunctionEnd

Function un.RequestRunningApplicationExit
    IfFileExists "$INSTDIR\${PRODUCT_EXECUTABLE}" 0 done
        DetailPrint "正在请求已运行的 ${INFO_PRODUCTNAME} 退出..."
        Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}" --installer-exit'
        Sleep 1500
    done:
FunctionEnd

Function un.CloseRunningApplicationWindow
    FindWindow $0 "${APP_WINDOW_CLASS}" "${APP_WINDOW_TITLE}"
    ${If} $0 != 0
        DetailPrint "正在关闭已运行的 ${INFO_PRODUCTNAME}..."
        SendMessage $0 ${WM_CLOSE} 0 0 /TIMEOUT=5000
        Sleep 1500
    ${EndIf}
FunctionEnd

Function un.ForceTerminateRunningApplication
    FindWindow $0 "${APP_WINDOW_CLASS}" "${APP_WINDOW_TITLE}"
    ${If} $0 != 0
        System::Call 'user32::GetWindowThreadProcessId(p r0, *i .r1) i .r2'
        ${If} $1 != 0
            DetailPrint "正在确保已运行的 ${INFO_PRODUCTNAME} 退出..."
            nsExec::ExecToLog 'taskkill /PID $1 /T /F'
            Pop $2
            Sleep 500
        ${EndIf}
    ${EndIf}
FunctionEnd

Function un.DeleteCurrentUserUninstaller
    Delete "$INSTDIR\uninstall.exe"

    SetRegView 64
    DeleteRegKey HKCU "${UNINST_KEY_CURRENT_USER}"
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    
    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    Call WriteCurrentUserUninstaller
SectionEnd

Section "uninstall" 
    Call un.RequestRunningApplicationExit
    Call un.CloseRunningApplicationWindow
    Call un.ForceTerminateRunningApplication
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # 删除 WebView2 数据目录。

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    Call un.DeleteCurrentUserUninstaller
SectionEnd
