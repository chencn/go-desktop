// 文件职责：从 .env 文件加载环境变量后执行指定命令。
// 说明：本文件的注释覆盖文件、实体、方法和关键状态，不改变任何运行逻辑。

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// main 是命令入口，负责解析启动上下文、装配依赖并启动核心流程。
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: envrun <command> [args...]")
		os.Exit(2)
	}

	command := os.Args[1]
	if runtime.GOOS == "windows" && filepath.Ext(command) == "" {
		if path, err := exec.LookPath(command + ".cmd"); err == nil {
			command = path
		}
	}

	cmd := exec.Command(command, os.Args[2:]...)
	cmd.Env = windowsEnvWithDefaults(os.Environ())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// windowsEnvWithDefaults 封装 从 .env 文件加载环境变量后执行指定命令 中的一段独立逻辑，调用方通过它复用同一业务规则。
func windowsEnvWithDefaults(env []string) []string {
	if runtime.GOOS != "windows" {
		return env
	}

	systemRoot := envValue(env, "SystemRoot")
	if systemRoot == "" {
		systemRoot = `C:\WINDOWS`
	}

	userProfile := envValue(env, "USERPROFILE")
	if userProfile == "" {
		userProfile = envValue(env, "HOMEDRIVE") + envValue(env, "HOMEPATH")
	}

	// Some Windows automation shells start with drive-level variables stripped.
	// Without these, Windows runtimes can create ./frontend/%SystemDrive%/ProgramData caches.
	systemDrive := envValue(env, "SystemDrive")
	if systemDrive == "" {
		systemDrive = filepath.VolumeName(systemRoot)
	}
	if systemDrive == "" && userProfile != "" {
		systemDrive = filepath.VolumeName(userProfile)
	}
	if systemDrive == "" {
		systemDrive = "C:"
	}
	programData := envValue(env, "ProgramData")
	if programData == "" {
		programData = filepath.Join(driveRoot(systemDrive), "ProgramData")
	}

	env = setDefaultEnv(env, "SystemDrive", systemDrive)
	env = setDefaultEnv(env, "ProgramData", programData)
	env = setDefaultEnv(env, "SystemRoot", systemRoot)
	env = setDefaultEnv(env, "WINDIR", systemRoot)
	env = setDefaultEnv(env, "ComSpec", filepath.Join(systemRoot, "System32", "cmd.exe"))

	if userProfile != "" {
		localAppData := filepath.Join(userProfile, "AppData", "Local")
		env = setDefaultEnv(env, "LOCALAPPDATA", localAppData)
		env = setDefaultEnv(env, "APPDATA", filepath.Join(userProfile, "AppData", "Roaming"))
		env = setDefaultEnv(env, "GOCACHE", filepath.Join(localAppData, "go-build"))
	}

	return env
}

// driveRoot 封装 从 .env 文件加载环境变量后执行指定命令 中的一段独立逻辑，调用方通过它复用同一业务规则。
func driveRoot(drive string) string {
	if strings.HasSuffix(drive, `\`) || strings.HasSuffix(drive, `/`) {
		return drive
	}
	return drive + `\`
}

// envValue 封装 从 .env 文件加载环境变量后执行指定命令 中的一段独立逻辑，调用方通过它复用同一业务规则。
func envValue(env []string, name string) string {
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

// setDefaultEnv 修改 从 .env 文件加载环境变量后执行指定命令 管理的状态、文件或外部副作用，并把失败原因向上返回。
func setDefaultEnv(env []string, name, value string) []string {
	if value == "" {
		return env
	}
	for index, entry := range env {
		key, existing, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, name) {
			if existing == "" {
				env[index] = name + "=" + value
			}
			return env
		}
	}
	return append(env, name+"="+value)
}
