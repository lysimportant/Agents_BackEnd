package models

import "time"

// ServerMetrics 表示后端实际运行环境的完整服务器监控快照。
type ServerMetrics struct {
	// Scope 说明指标对应宿主机还是容器运行环境。
	Scope string `json:"scope"`
	// Hostname 表示运行环境主机名。
	Hostname string `json:"hostname"`
	// OS 表示操作系统名称。
	OS string `json:"os"`
	// Platform 表示操作系统发行版或平台名称。
	Platform string `json:"platform"`
	// PlatformVersion 表示操作系统发行版版本。
	PlatformVersion string `json:"platformVersion"`
	// KernelVersion 表示内核版本。
	KernelVersion string `json:"kernelVersion"`
	// Architecture 表示处理器架构。
	Architecture string `json:"architecture"`
	// UptimeSeconds 表示运行环境自启动后的秒数。
	UptimeSeconds uint64 `json:"uptimeSeconds"`
	// BootedAt 表示运行环境最近一次启动时间。
	BootedAt time.Time `json:"bootedAt"`
	// VirtualizationSystem 表示检测到的虚拟化或容器技术。
	VirtualizationSystem string `json:"virtualizationSystem"`
	// VirtualizationRole 表示当前环境在虚拟化体系中的角色。
	VirtualizationRole string `json:"virtualizationRole"`
	// CPU 表示处理器资源快照。
	CPU ServerCPUResource `json:"cpu"`
	// Load 表示系统负载和进程调度快照。
	Load ServerLoadResource `json:"load"`
	// Memory 表示内存与交换区资源快照。
	Memory ServerMemoryResource `json:"memory"`
	// Disk 表示后端工作目录所在磁盘资源快照。
	Disk ServerDiskResource `json:"disk"`
	// Partitions 表示可访问的物理文件系统分区。
	Partitions []ServerDiskResource `json:"partitions"`
	// DiskIO 表示块设备累计读写统计。
	DiskIO []ServerDiskIOResource `json:"diskIO"`
	// Network 表示运行环境网络流量、网卡和连接快照。
	Network ServerNetworkResource `json:"network"`
	// Process 表示后端进程运行状态。
	Process ServerProcessResource `json:"process"`
	// Temperatures 表示系统可读取的硬件温度。
	Temperatures []ServerTemperatureResource `json:"temperatures"`
	// Health 表示根据当前快照计算的健康状态。
	Health ServerHealthResource `json:"health"`
	// CollectionWarnings 表示当前平台或权限导致的非关键采集缺失。
	CollectionWarnings []string `json:"collectionWarnings"`
	// SampledAt 表示资源快照采集时间。
	SampledAt time.Time `json:"sampledAt"`
}

// ServerCPUResource 表示服务器处理器使用情况。
type ServerCPUResource struct {
	// LogicalCores 表示逻辑处理器数量。
	LogicalCores int `json:"logicalCores"`
	// PhysicalCores 表示物理处理器核心数量。
	PhysicalCores int `json:"physicalCores"`
	// UsagePercent 表示采样窗口内的总体处理器使用率。
	UsagePercent float64 `json:"usagePercent"`
	// PerCoreUsagePercent 表示每个逻辑核心的采样使用率。
	PerCoreUsagePercent []float64 `json:"perCoreUsagePercent"`
	// ModelName 表示处理器型号。
	ModelName string `json:"modelName"`
	// VendorID 表示处理器厂商标识。
	VendorID string `json:"vendorId"`
	// FrequencyMHz 表示处理器报告的当前或标称频率。
	FrequencyMHz float64 `json:"frequencyMHz"`
	// CacheSizeKB 表示处理器报告的缓存容量。
	CacheSizeKB int32 `json:"cacheSizeKB"`
	// Times 表示各类处理器状态的累计秒数。
	Times ServerCPUTimesResource `json:"times"`
}

// ServerCPUTimesResource 表示处理器各运行状态的累计时间。
type ServerCPUTimesResource struct {
	// UserSeconds 表示用户态累计秒数。
	UserSeconds float64 `json:"userSeconds"`
	// SystemSeconds 表示内核态累计秒数。
	SystemSeconds float64 `json:"systemSeconds"`
	// IdleSeconds 表示空闲累计秒数。
	IdleSeconds float64 `json:"idleSeconds"`
	// IOWaitSeconds 表示等待 I/O 的累计秒数。
	IOWaitSeconds float64 `json:"ioWaitSeconds"`
	// IRQSeconds 表示硬中断累计秒数。
	IRQSeconds float64 `json:"irqSeconds"`
	// SoftIRQSeconds 表示软中断累计秒数。
	SoftIRQSeconds float64 `json:"softIrqSeconds"`
	// StealSeconds 表示虚拟机被宿主机占用的累计秒数。
	StealSeconds float64 `json:"stealSeconds"`
}

// ServerLoadResource 表示负载、进程调度和上下文切换情况。
type ServerLoadResource struct {
	// Load1、Load5、Load15 分别表示 1、5、15 分钟平均负载。
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
	// ProcessTotal 表示系统进程总数。
	ProcessTotal int `json:"processTotal"`
	// ProcessRunning 表示处于运行状态的进程数。
	ProcessRunning int `json:"processRunning"`
	// ProcessBlocked 表示处于不可中断等待状态的进程数。
	ProcessBlocked int `json:"processBlocked"`
	// ProcessesCreated 表示系统启动后累计创建的进程数。
	ProcessesCreated int `json:"processesCreated"`
	// ContextSwitches 表示系统启动后累计上下文切换次数。
	ContextSwitches int `json:"contextSwitches"`
}

// ServerMemoryResource 表示服务器内存和交换区使用情况。
type ServerMemoryResource struct {
	// TotalBytes 表示可用内存总字节数。
	TotalBytes uint64 `json:"totalBytes"`
	// UsedBytes 表示已使用内存字节数。
	UsedBytes uint64 `json:"usedBytes"`
	// AvailableBytes 表示当前可用内存字节数。
	AvailableBytes uint64 `json:"availableBytes"`
	// FreeBytes 表示内核报告的完全空闲内存字节数。
	FreeBytes uint64 `json:"freeBytes"`
	// CachedBytes 表示文件缓存占用字节数。
	CachedBytes uint64 `json:"cachedBytes"`
	// BuffersBytes 表示块设备缓冲区占用字节数。
	BuffersBytes uint64 `json:"buffersBytes"`
	// ActiveBytes 表示活跃内存字节数。
	ActiveBytes uint64 `json:"activeBytes"`
	// InactiveBytes 表示非活跃内存字节数。
	InactiveBytes uint64 `json:"inactiveBytes"`
	// UsagePercent 表示内存使用率。
	UsagePercent float64 `json:"usagePercent"`
	// Swap 表示交换区使用情况。
	Swap ServerSwapResource `json:"swap"`
}

// ServerSwapResource 表示交换区容量和累计换入换出量。
type ServerSwapResource struct {
	// TotalBytes 表示交换区总容量。
	TotalBytes uint64 `json:"totalBytes"`
	// UsedBytes 表示交换区已使用容量。
	UsedBytes uint64 `json:"usedBytes"`
	// FreeBytes 表示交换区剩余容量。
	FreeBytes uint64 `json:"freeBytes"`
	// UsagePercent 表示交换区使用率。
	UsagePercent float64 `json:"usagePercent"`
	// BytesIn 表示累计换入字节数。
	BytesIn uint64 `json:"bytesIn"`
	// BytesOut 表示累计换出字节数。
	BytesOut uint64 `json:"bytesOut"`
}

// ServerDiskResource 表示一个文件系统分区的使用情况。
type ServerDiskResource struct {
	// Device 表示分区对应设备。
	Device string `json:"device"`
	// Path 表示文件系统挂载路径。
	Path string `json:"path"`
	// FileSystem 表示文件系统类型。
	FileSystem string `json:"fileSystem"`
	// TotalBytes 表示磁盘总字节数。
	TotalBytes uint64 `json:"totalBytes"`
	// UsedBytes 表示磁盘已使用字节数。
	UsedBytes uint64 `json:"usedBytes"`
	// FreeBytes 表示磁盘可用字节数。
	FreeBytes uint64 `json:"freeBytes"`
	// UsagePercent 表示磁盘使用率。
	UsagePercent float64 `json:"usagePercent"`
	// InodesTotal 表示 inode 总数；平台不支持时为零。
	InodesTotal uint64 `json:"inodesTotal"`
	// InodesUsed 表示已使用 inode 数；平台不支持时为零。
	InodesUsed uint64 `json:"inodesUsed"`
	// InodesUsagePercent 表示 inode 使用率；平台不支持时为零。
	InodesUsagePercent float64 `json:"inodesUsagePercent"`
}

// ServerDiskIOResource 表示一个块设备的累计 I/O 统计。
type ServerDiskIOResource struct {
	// Name 表示块设备名称。
	Name string `json:"name"`
	// ReadBytes、WriteBytes 分别表示累计读取和写入字节数。
	ReadBytes  uint64 `json:"readBytes"`
	WriteBytes uint64 `json:"writeBytes"`
	// ReadOperations、WriteOperations 分别表示累计读写操作数。
	ReadOperations  uint64 `json:"readOperations"`
	WriteOperations uint64 `json:"writeOperations"`
	// ReadTimeMs、WriteTimeMs 分别表示累计读写耗时毫秒数。
	ReadTimeMs  uint64 `json:"readTimeMs"`
	WriteTimeMs uint64 `json:"writeTimeMs"`
	// IOPerationsInProgress 表示采样时正在执行的 I/O 数。
	IOOperationsInProgress uint64 `json:"ioOperationsInProgress"`
}

// ServerNetworkResource 表示服务器累计网络流量、网卡与连接统计。
type ServerNetworkResource struct {
	// BytesSent、BytesReceived 分别表示累计发送和接收字节数。
	BytesSent     uint64 `json:"bytesSent"`
	BytesReceived uint64 `json:"bytesReceived"`
	// PacketsSent、PacketsReceived 分别表示累计发送和接收数据包数。
	PacketsSent     uint64 `json:"packetsSent"`
	PacketsReceived uint64 `json:"packetsReceived"`
	// ErrorsIn、ErrorsOut 分别表示累计接收和发送错误数。
	ErrorsIn  uint64 `json:"errorsIn"`
	ErrorsOut uint64 `json:"errorsOut"`
	// DropsIn、DropsOut 分别表示累计接收和发送丢包数。
	DropsIn  uint64 `json:"dropsIn"`
	DropsOut uint64 `json:"dropsOut"`
	// Interfaces 表示网卡地址、状态和各自累计流量。
	Interfaces []ServerNetworkInterface `json:"interfaces"`
	// Connections 表示网络连接状态汇总。
	Connections ServerConnectionResource `json:"connections"`
}

// ServerNetworkInterface 表示一个网卡的地址、状态和流量。
type ServerNetworkInterface struct {
	// Name 表示网卡名称。
	Name string `json:"name"`
	// HardwareAddress 表示网卡硬件地址。
	HardwareAddress string `json:"hardwareAddress"`
	// MTU 表示最大传输单元。
	MTU int `json:"mtu"`
	// Flags 表示网卡状态标志。
	Flags []string `json:"flags"`
	// Addresses 表示网卡绑定的 IP 地址和掩码。
	Addresses []string `json:"addresses"`
	// BytesSent、BytesReceived 分别表示网卡累计发送和接收字节数。
	BytesSent     uint64 `json:"bytesSent"`
	BytesReceived uint64 `json:"bytesReceived"`
	// PacketsSent、PacketsReceived 分别表示网卡累计发送和接收数据包数。
	PacketsSent     uint64 `json:"packetsSent"`
	PacketsReceived uint64 `json:"packetsReceived"`
	// ErrorsIn、ErrorsOut 分别表示网卡累计接收和发送错误数。
	ErrorsIn  uint64 `json:"errorsIn"`
	ErrorsOut uint64 `json:"errorsOut"`
	// DropsIn、DropsOut 分别表示网卡累计接收和发送丢包数。
	DropsIn  uint64 `json:"dropsIn"`
	DropsOut uint64 `json:"dropsOut"`
}

// ServerConnectionResource 表示系统网络连接状态汇总。
type ServerConnectionResource struct {
	// Available 表示当前平台与权限是否允许读取连接统计。
	Available bool `json:"available"`
	// Sampled 表示本次实际统计的连接数量。
	Sampled int `json:"sampled"`
	// Truncated 表示连接数量是否达到采集上限。
	Truncated bool `json:"truncated"`
	// TCP、UDP 分别表示 TCP 和 UDP 套接字数量。
	TCP int `json:"tcp"`
	UDP int `json:"udp"`
	// Established、Listen、TimeWait、CloseWait 分别表示常见 TCP 状态数量。
	Established int `json:"established"`
	Listen      int `json:"listen"`
	TimeWait    int `json:"timeWait"`
	CloseWait   int `json:"closeWait"`
}

// ServerConnectionDetailsResource 表示按需查询的网络连接明细快照。
type ServerConnectionDetailsResource struct {
	// Available 表示当前平台和进程权限是否允许枚举连接明细。
	Available bool `json:"available"`
	// Warning 表示连接明细不可用时的用户可见原因。
	Warning string `json:"warning"`
	// Sampled 表示本次从系统枚举到的套接字数量。
	Sampled int `json:"sampled"`
	// Truncated 表示系统枚举结果是否达到五千条采集上限。
	Truncated bool `json:"truncated"`
	// DetailsTruncated 表示返回明细是否达到五百条响应上限。
	DetailsTruncated bool `json:"detailsTruncated"`
	// Summary 表示同一次系统枚举得到的协议与常见状态总数。
	Summary ServerConnectionResource `json:"summary"`
	// Connections 表示排序后的连接端点、状态和所属进程。
	Connections []ServerConnectionDetail `json:"connections"`
	// SampledAt 表示连接明细采样时间。
	SampledAt time.Time `json:"sampledAt"`
}

// ServerConnectionDetail 表示一个系统网络套接字的可诊断字段。
type ServerConnectionDetail struct {
	// Protocol 表示 TCP、UDP 或系统返回的其他传输协议。
	Protocol string `json:"protocol"`
	// AddressFamily 表示 IPv4、IPv6 或未知地址族。
	AddressFamily string `json:"addressFamily"`
	// LocalAddress、LocalPort 表示本地监听或连接端点。
	LocalAddress string `json:"localAddress"`
	LocalPort    uint32 `json:"localPort"`
	// RemoteAddress、RemotePort 表示远端端点；监听套接字可能为空和零。
	RemoteAddress string `json:"remoteAddress"`
	RemotePort    uint32 `json:"remotePort"`
	// Status 表示 ESTABLISHED、LISTEN、TIME_WAIT 等系统连接状态。
	Status string `json:"status"`
	// PID 表示拥有套接字的进程标识；权限不足时可能为零。
	PID int32 `json:"pid"`
	// ProcessName 表示可读取到的进程名称；权限不足时为空。
	ProcessName string `json:"processName"`
}

// ServerProcessResource 表示后端进程自身的运行状态。
type ServerProcessResource struct {
	// PID 表示后端进程标识。
	PID int `json:"pid"`
	// GoVersion 表示后端使用的 Go 运行时版本。
	GoVersion string `json:"goVersion"`
	// Goroutines 表示后端当前协程数量。
	Goroutines int `json:"goroutines"`
	// Threads 表示后端进程线程数量。
	Threads int32 `json:"threads"`
	// CPUUsagePercent 表示后端进程处理器使用率。
	CPUUsagePercent float64 `json:"cpuUsagePercent"`
	// AllocatedBytes 表示 Go 堆当前已分配字节数。
	AllocatedBytes uint64 `json:"allocatedBytes"`
	// SystemBytes 表示 Go 运行时向系统申请的总字节数。
	SystemBytes uint64 `json:"systemBytes"`
	// HeapInUseBytes 表示 Go 堆正在使用的字节数。
	HeapInUseBytes uint64 `json:"heapInUseBytes"`
	// HeapObjects 表示 Go 堆存活对象数量。
	HeapObjects uint64 `json:"heapObjects"`
	// GCCycles 表示进程启动后完成的垃圾回收次数。
	GCCycles uint32 `json:"gcCycles"`
	// ResidentBytes 表示进程常驻内存字节数。
	ResidentBytes uint64 `json:"residentBytes"`
	// VirtualBytes 表示进程虚拟内存字节数。
	VirtualBytes uint64 `json:"virtualBytes"`
	// ReadBytes、WriteBytes 分别表示进程累计读写字节数。
	ReadBytes  uint64 `json:"readBytes"`
	WriteBytes uint64 `json:"writeBytes"`
	// OpenFileDescriptors 表示进程打开的文件描述符数量。
	OpenFileDescriptors int32 `json:"openFileDescriptors"`
	// UptimeSeconds 表示后端进程持续运行秒数。
	UptimeSeconds uint64 `json:"uptimeSeconds"`
}

// ServerTemperatureResource 表示一个温度传感器读数。
type ServerTemperatureResource struct {
	// SensorKey 表示传感器名称。
	SensorKey string `json:"sensorKey"`
	// TemperatureCelsius 表示当前摄氏温度。
	TemperatureCelsius float64 `json:"temperatureCelsius"`
	// HighCelsius 表示传感器高温阈值。
	HighCelsius float64 `json:"highCelsius"`
	// CriticalCelsius 表示传感器临界温度阈值。
	CriticalCelsius float64 `json:"criticalCelsius"`
}

// ServerHealthResource 表示服务器当前健康结论和触发项。
type ServerHealthResource struct {
	// Status 表示 healthy、warning 或 critical。
	Status string `json:"status"`
	// Score 表示 0 到 100 的即时健康分数。
	Score int `json:"score"`
	// Alerts 表示根据当前资源阈值生成的异常提示。
	Alerts []ServerHealthAlert `json:"alerts"`
}

// ServerHealthAlert 表示一个当前生效的服务器健康提示。
type ServerHealthAlert struct {
	// Code 表示稳定告警编码。
	Code string `json:"code"`
	// Severity 表示 warning 或 critical。
	Severity string `json:"severity"`
	// Title 表示告警标题。
	Title string `json:"title"`
	// Message 表示告警原因和当前数值。
	Message string `json:"message"`
}
