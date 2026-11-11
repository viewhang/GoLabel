package printing

import "github.com/pengh/GoLabel/internal/platform"

// Job 保存打印任务的标题和按实际尺寸渲染的页面
type Job struct {
	Title string
	Page  platform.Page
}
