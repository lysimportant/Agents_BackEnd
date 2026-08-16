//go:build linux

package main

import "golang.org/x/sys/unix"

// hardenAgentProcess 禁止同一系统账号通过 ptrace 或 /proc 读取代理令牌内存。
func hardenAgentProcess() error {
	return unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)
}
