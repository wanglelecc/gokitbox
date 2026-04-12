package uSnowflake

import "context"

// NodeProvider 定义雪花算法节点号分配接口
// 实现方负责根据 project+service 维度分配唯一节点号（0~1023）
type NodeProvider interface {
	// MakeNode 返回分配的节点号，失败时返回错误
	// project: 项目标识，如 "order_platform"
	// service: 服务标识，如 "payment_service"
	MakeNode(ctx context.Context, project, service string) (int64, error)
}
