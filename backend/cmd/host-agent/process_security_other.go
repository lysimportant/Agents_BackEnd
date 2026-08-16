//go:build !linux

package main

// hardenAgentProcess 为非 Linux 编译检查保留无副作用实现；程序入口仍会拒绝运行。
func hardenAgentProcess() error {
	return nil
}
