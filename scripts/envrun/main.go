// 文件职责：加载仓库 .env 并为子命令补齐 Windows 自动化环境变量。

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// main 保留子命令的 stdin/stdout/stderr，并把子命令退出码透传给调用方。
// Windows 上无扩展名命令会优先解析为 .cmd，兼容 npm、wails3 等 shim。
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
	cmd.Env = windowsEnvWithDefaults(mergeDotEnv(os.Environ(), findDotEnv()))
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

// findDotEnv 从当前目录向上查找仓库级 .env，兼容 frontend、build 等子目录执行场景。
func findDotEnv() string {
	dir, err := os.Getwd()
	if err != nil {
		return ".env"
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Join(dir, ".env")
		}
		dir = parent
	}
}

// mergeDotEnv 读取可选 .env；进程环境变量优先于 .env，避免本地密钥或 CI 变量被覆盖。
func mergeDotEnv(env []string, path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return env
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || envValue(env, key) != "" {
			continue
		}
		env = append(env, key+"="+value)
	}
	return env
}

// windowsEnvWithDefaults 补齐 Windows shell 里常见缺失变量。
// 这些默认值防止 Go/npm/Wails 把缓存目录解析到仓库内的 %SystemDrive% 或空路径。
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

// driveRoot 把 C: 形式补成可参与 filepath.Join 的根路径 C:\。
func driveRoot(drive string) string {
	if strings.HasSuffix(drive, `\`) || strings.HasSuffix(drive, `/`) {
		return drive
	}
	return drive + `\`
}

// envValue 在环境变量列表里按 Windows 规则大小写不敏感查找值。
func envValue(env []string, name string) string {
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

// setDefaultEnv 只填补缺失或空值，不覆盖已有非空环境变量。
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
