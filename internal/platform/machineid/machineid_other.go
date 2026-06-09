//go:build !windows

package machineid

import "os"

func readParts() []string {
	host, _ := os.Hostname()
	return []string{host}
}
