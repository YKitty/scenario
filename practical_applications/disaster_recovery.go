package practical_applications

/*
异地容灾系统 - 多数据中心复制策略

原理：
异地容灾系统是为了应对地理位置上的灾难或大规模故障，通过在不同地理位置部署多个数据中心，
并在这些数据中心间实时或准实时地复制数据，确保在主数据中心发生故障时，业务可以快速切换到备用数据中心，
从而保证业务连续性和数据安全性。

关键特点：
1. 多中心部署：地理上分散的多个数据中心
2. 数据复制：多种复制模式（同步、异步、半同步）
3. 灾难检测：自动或手动检测主数据中心故障
4. 故障切换：自动或手动将业务切换到备用数据中心
5. 数据一致性：不同复制策略下的一致性保证不同

实现方式：
- 使用消息队列或日志复制技术进行数据传输
- 使用心跳机制监控数据中心健康状态
- 设计适合业务场景的复制策略和一致性模型

应用场景：
- 金融系统的交易数据备份
- 云服务提供商的多区域部署
- 核心业务系统的不间断服务保障
- 满足法规要求的数据保护和业务连续性

优缺点：
- 优点：提高系统可用性，保障业务连续性，满足合规需求
- 缺点：建设和维护成本高，复制延迟可能导致数据一致性问题

以下实现了一个基本的异地容灾系统模拟框架，包含多种复制策略和故障切换机制。
*/

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// 数据中心状态
const (
	StatusHealthy  = "健康"
	StatusDegraded = "性能下降"
	StatusFailed   = "故障"
)

// 复制策略
const (
	ReplicationSync     = "同步复制"
	ReplicationAsync    = "异步复制"
	ReplicationSemiSync = "半同步复制"
)

// DataCenter 数据中心结构
type DataCenter struct {
	ID            string            // 数据中心ID
	Name          string            // 数据中心名称
	Location      string            // 地理位置
	Status        string            // 当前状态
	IsActive      bool              // 是否为活跃的主数据中心
	Storage       map[string][]byte // 存储的数据
	lastHeartbeat time.Time         // 最后一次心跳时间
	mutex         sync.RWMutex      // 读写锁
}

// DisasterRecoverySystem 异地容灾系统
type DisasterRecoverySystem struct {
	dataCenters      map[string]*DataCenter // 所有数据中心
	primaryDC        *DataCenter            // 主数据中心
	replicationMode  string                 // 复制策略
	heartbeatTimeout time.Duration          // 心跳超时时间
	pendingWrites    map[string][]byte      // 待复制的写操作
	mutex            sync.RWMutex           // 读写锁
	ctx              context.Context        // 上下文
	cancel           context.CancelFunc     // 取消函数
}

// NewDataCenter 创建新的数据中心
func NewDataCenter(id, name, location string, isActive bool) *DataCenter {
	return &DataCenter{
		ID:            id,
		Name:          name,
		Location:      location,
		Status:        StatusHealthy,
		IsActive:      isActive,
		Storage:       make(map[string][]byte),
		lastHeartbeat: time.Now(),
	}
}

// NewDisasterRecoverySystem 创建新的异地容灾系统
func NewDisasterRecoverySystem(replicationMode string, heartbeatTimeout time.Duration) *DisasterRecoverySystem {
	ctx, cancel := context.WithCancel(context.Background())

	drs := &DisasterRecoverySystem{
		dataCenters:      make(map[string]*DataCenter),
		replicationMode:  replicationMode,
		heartbeatTimeout: heartbeatTimeout,
		pendingWrites:    make(map[string][]byte),
		ctx:              ctx,
		cancel:           cancel,
	}

	// 启动心跳检测和异步复制（如果是异步模式）
	go drs.heartbeatMonitor()
	if replicationMode == ReplicationAsync {
		go drs.asyncReplicationWorker()
	}

	return drs
}

// AddDataCenter 添加数据中心
func (drs *DisasterRecoverySystem) AddDataCenter(dc *DataCenter) {
	drs.mutex.Lock()
	defer drs.mutex.Unlock()

	drs.dataCenters[dc.ID] = dc

	// 如果是第一个添加的数据中心，或者明确指定为活跃，则设为主数据中心
	if drs.primaryDC == nil || dc.IsActive {
		if drs.primaryDC != nil {
			drs.primaryDC.IsActive = false
		}
		drs.primaryDC = dc
		dc.IsActive = true
	}
}

// Write 写入数据到系统
func (drs *DisasterRecoverySystem) Write(key string, data []byte) error {
	drs.mutex.Lock()
	defer drs.mutex.Unlock()

	if drs.primaryDC == nil {
		return errors.New("没有可用的主数据中心")
	}

	if drs.primaryDC.Status != StatusHealthy && drs.primaryDC.Status != StatusDegraded {
		return errors.New("主数据中心状态异常，无法写入")
	}

	// 按照不同的复制策略处理写入
	switch drs.replicationMode {
	case ReplicationSync:
		// 同步复制：先写入主数据中心，再同步复制到所有备份数据中心
		drs.primaryDC.mutex.Lock()
		drs.primaryDC.Storage[key] = data
		drs.primaryDC.mutex.Unlock()

		// 同步复制到所有其他数据中心
		for _, dc := range drs.dataCenters {
			if dc.ID != drs.primaryDC.ID && dc.Status == StatusHealthy {
				dc.mutex.Lock()
				dc.Storage[key] = data
				dc.mutex.Unlock()
			}
		}

	case ReplicationSemiSync:
		// 半同步复制：写入主数据中心，并至少等待一个备份数据中心确认
		drs.primaryDC.mutex.Lock()
		drs.primaryDC.Storage[key] = data
		drs.primaryDC.mutex.Unlock()

		// 至少复制到一个备份数据中心
		replicated := false
		for _, dc := range drs.dataCenters {
			if dc.ID != drs.primaryDC.ID && dc.Status == StatusHealthy {
				dc.mutex.Lock()
				dc.Storage[key] = data
				dc.mutex.Unlock()
				replicated = true
				break
			}
		}

		if !replicated {
			// 如果没有一个备份数据中心可用，加入待复制队列
			drs.pendingWrites[key] = data
			return errors.New("无法完成半同步复制，数据已写入主数据中心但未复制到备份数据中心")
		}

	case ReplicationAsync:
		// 异步复制：先写入主数据中心，再异步复制到备份数据中心
		drs.primaryDC.mutex.Lock()
		drs.primaryDC.Storage[key] = data
		drs.primaryDC.mutex.Unlock()

		// 将数据加入异步复制队列
		drs.pendingWrites[key] = data

	default:
		return errors.New("未知的复制策略")
	}

	return nil
}

// Read 从系统读取数据
func (drs *DisasterRecoverySystem) Read(key string) ([]byte, error) {
	drs.mutex.RLock()
	defer drs.mutex.RUnlock()

	var targetDC *DataCenter

	// 优先从主数据中心读取
	if drs.primaryDC != nil && (drs.primaryDC.Status == StatusHealthy || drs.primaryDC.Status == StatusDegraded) {
		targetDC = drs.primaryDC
	} else {
		// 主数据中心不可用，选择一个健康的备份数据中心
		for _, dc := range drs.dataCenters {
			if dc.Status == StatusHealthy {
				targetDC = dc
				break
			}
		}
	}

	if targetDC == nil {
		return nil, errors.New("没有可用的数据中心")
	}

	targetDC.mutex.RLock()
	defer targetDC.mutex.RUnlock()

	data, exists := targetDC.Storage[key]
	if !exists {
		return nil, errors.New("数据不存在")
	}

	return data, nil
}

// UpdateDataCenterStatus 更新数据中心状态
func (drs *DisasterRecoverySystem) UpdateDataCenterStatus(dcID, status string) {
	drs.mutex.Lock()
	defer drs.mutex.Unlock()

	dc, exists := drs.dataCenters[dcID]
	if !exists {
		return
	}

	oldStatus := dc.Status
	dc.Status = status

	// 如果主数据中心发生故障，尝试故障切换
	if dc == drs.primaryDC && status == StatusFailed && oldStatus != StatusFailed {
		drs.failover()
	}
}

// 故障切换到备用数据中心
func (drs *DisasterRecoverySystem) failover() {
	// 旧主数据中心已经设为故障状态，现在寻找新的主数据中心
	drs.primaryDC.IsActive = false

	var newPrimary *DataCenter
	for _, dc := range drs.dataCenters {
		if dc.ID != drs.primaryDC.ID && dc.Status == StatusHealthy {
			newPrimary = dc
			break
		}
	}

	if newPrimary != nil {
		newPrimary.IsActive = true
		drs.primaryDC = newPrimary
		log.Printf("故障切换：主数据中心从 %s 切换到 %s", drs.primaryDC.ID, newPrimary.ID)
	} else {
		log.Printf("故障切换失败：没有可用的备份数据中心")
	}
}

// 心跳监控
func (drs *DisasterRecoverySystem) heartbeatMonitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-drs.ctx.Done():
			return
		case <-ticker.C:
			drs.checkHeartbeats()
		}
	}
}

// 检查所有数据中心的心跳
func (drs *DisasterRecoverySystem) checkHeartbeats() {
	drs.mutex.Lock()
	defer drs.mutex.Unlock()

	now := time.Now()

	for _, dc := range drs.dataCenters {
		// 模拟心跳检测：实际应该通过网络请求检测
		if now.Sub(dc.lastHeartbeat) > drs.heartbeatTimeout {
			// 心跳超时，标记为故障
			if dc.Status != StatusFailed {
				oldStatus := dc.Status
				dc.Status = StatusFailed

				// 如果是主数据中心故障，执行故障切换
				if dc == drs.primaryDC && oldStatus != StatusFailed {
					drs.failover()
				}
			}
		}
	}
}

// 模拟发送心跳
func (drs *DisasterRecoverySystem) SendHeartbeat(dcID string) {
	drs.mutex.Lock()
	defer drs.mutex.Unlock()

	dc, exists := drs.dataCenters[dcID]
	if !exists {
		return
	}

	dc.lastHeartbeat = time.Now()
}

// 异步复制工作器
func (drs *DisasterRecoverySystem) asyncReplicationWorker() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-drs.ctx.Done():
			return
		case <-ticker.C:
			drs.processAsyncReplications()
		}
	}
}

// 处理异步复制队列
func (drs *DisasterRecoverySystem) processAsyncReplications() {
	drs.mutex.Lock()

	// 复制待处理的写操作列表，然后释放主锁
	pendingCopy := make(map[string][]byte)
	for k, v := range drs.pendingWrites {
		pendingCopy[k] = v
	}

	// 清空待处理队列
	drs.pendingWrites = make(map[string][]byte)

	drs.mutex.Unlock()

	// 复制到所有健康的备份数据中心
	for key, data := range pendingCopy {
		for _, dc := range drs.dataCenters {
			if dc != drs.primaryDC && dc.Status == StatusHealthy {
				dc.mutex.Lock()
				dc.Storage[key] = data
				dc.mutex.Unlock()
			}
		}
	}
}

// Shutdown 关闭系统
func (drs *DisasterRecoverySystem) Shutdown() {
	drs.cancel()
}

// 场景示例：金融交易系统的异地容灾
func DisasterRecoveryDemo() {
	fmt.Println("异地容灾系统示例 - 金融交易数据备份:")

	// 创建异地容灾系统（使用半同步复制策略，心跳超时5秒）
	drs := NewDisasterRecoverySystem(ReplicationSemiSync, 5*time.Second)

	// 添加多个数据中心
	primaryDC := NewDataCenter("dc-sh", "上海数据中心", "上海", true)
	drs.AddDataCenter(primaryDC)

	backupDCs := []*DataCenter{
		NewDataCenter("dc-bj", "北京数据中心", "北京", false),
		NewDataCenter("dc-gz", "广州数据中心", "广州", false),
		NewDataCenter("dc-cd", "成都数据中心", "成都", false),
	}

	for _, dc := range backupDCs {
		drs.AddDataCenter(dc)
	}

	fmt.Println("初始化数据中心配置:")
	fmt.Printf("  主数据中心: %s (%s)\n", primaryDC.Name, primaryDC.Location)
	fmt.Println("  备份数据中心:")
	for _, dc := range backupDCs {
		fmt.Printf("    - %s (%s)\n", dc.Name, dc.Location)
	}
	fmt.Printf("  复制策略: %s\n", drs.replicationMode)

	// 模拟正常业务操作
	fmt.Println("\n模拟正常业务操作:")

	// 模拟写入一些交易数据
	transactions := map[string][]byte{
		"tx-001": []byte("用户A转账到用户B：1000元"),
		"tx-002": []byte("用户C购买股票：500股"),
		"tx-003": []byte("用户D提现：2000元"),
	}

	for id, data := range transactions {
		err := drs.Write(id, data)
		if err != nil {
			fmt.Printf("交易 %s 写入失败: %v\n", id, err)
		} else {
			fmt.Printf("交易 %s 写入成功\n", id)
		}

		// 模拟发送心跳
		for _, dc := range append(backupDCs, primaryDC) {
			drs.SendHeartbeat(dc.ID)
		}
	}

	// 验证数据已同步到备份中心
	fmt.Println("\n验证数据同步情况:")
	for _, dc := range append(backupDCs, primaryDC) {
		fmt.Printf("  %s 数据情况:\n", dc.Name)
		dc.mutex.RLock()
		count := len(dc.Storage)
		dc.mutex.RUnlock()
		fmt.Printf("    - 存储交易数据: %d 条\n", count)
	}

	// 模拟主数据中心故障
	fmt.Println("\n模拟主数据中心故障:")
	drs.UpdateDataCenterStatus(primaryDC.ID, StatusFailed)
	fmt.Printf("  %s 状态更新为: %s\n", primaryDC.Name, primaryDC.Status)

	// 等待故障切换完成
	time.Sleep(100 * time.Millisecond)

	// 获取新的主数据中心
	drs.mutex.RLock()
	newPrimary := drs.primaryDC
	drs.mutex.RUnlock()

	fmt.Printf("  故障切换结果: 新的主数据中心是 %s\n", newPrimary.Name)

	// 尝试在故障切换后读取数据
	fmt.Println("\n故障切换后读取数据:")
	for id := range transactions {
		data, err := drs.Read(id)
		if err != nil {
			fmt.Printf("  交易 %s 读取失败: %v\n", id, err)
		} else {
			fmt.Printf("  交易 %s 读取成功: %s\n", id, string(data))
		}
	}

	// 模拟主数据中心恢复
	fmt.Println("\n模拟原主数据中心恢复:")
	drs.UpdateDataCenterStatus(primaryDC.ID, StatusHealthy)
	drs.SendHeartbeat(primaryDC.ID)
	fmt.Printf("  %s 状态更新为: %s\n", primaryDC.Name, primaryDC.Status)
	fmt.Println("  注意: 原主数据中心恢复后不会自动切回，避免频繁切换")

	// 数据中心状态报告
	fmt.Println("\n数据中心最终状态:")
	for _, dc := range append(backupDCs, primaryDC) {
		fmt.Printf("  %s: 状态=%s, 是否为主=%v\n", dc.Name, dc.Status, dc.IsActive)
	}

	// 关闭系统
	drs.Shutdown()
}
