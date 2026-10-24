package service

import (
	"context"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const SystemTaskTypeTicketAutoArchive = "ticket_auto_archive"

type ticketArchiveHandler struct{}

var _ ScheduledSystemTaskHandler = ticketArchiveHandler{}

func (ticketArchiveHandler) Type() string { return SystemTaskTypeTicketAutoArchive }

func (ticketArchiveHandler) Enabled() bool {
	if model.DB == nil {
		return false
	}
	config, err := model.GetTicketAutoArchiveConfig()
	if err != nil {
		common.SysError("ticket auto-archive configuration unavailable")
		return false
	}
	return config.Enabled
}

func (ticketArchiveHandler) Interval() time.Duration { return time.Minute }
func (ticketArchiveHandler) NewPayload() any         { return nil }

func (ticketArchiveHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	passCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// 模型层重新读取当前配置，已入队的任务也必须遵守随后关闭的开关。
	count, runErr := model.ArchiveInactiveTickets(passCtx, time.Now().Unix(), 100)
	status, message := model.SystemTaskStatusSucceeded, ""
	if runErr != nil {
		status, message = model.SystemTaskStatusFailed, "ticket auto-archive failed"
		// 数据库错误可能包含工单正文，任务历史和通用日志均只记录固定错误。
		common.SysError(message)
	}
	if err := model.FinishSystemTask(task.TaskID, runnerID, status, map[string]int{"archived_count": count}, message); err != nil && !errors.Is(err, model.ErrSystemTaskLockLost) {
		common.SysError("ticket auto-archive task result could not be saved")
	}
}

// RegisterTicketArchiveSystemTask 复用主节点调度与数据库租约，避免多实例重复执行。
func RegisterTicketArchiveSystemTask() {
	RegisterSystemTaskHandler(ticketArchiveHandler{})
}
