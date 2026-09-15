package relay

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	pluginadaptor "github.com/QuantumNous/new-api/relay/channel/task/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

// GetPinnedTaskAdaptor 只按任务的持久版本解析，不回退当前插件或原生适配器。
func GetPinnedTaskAdaptor(task *model.Task) (channel.TaskAdaptor, error) {
	if task.PrivateData.Execution == nil || task.PrivateData.Execution.TaskPlugin == nil {
		return GetTaskAdaptor(task.Platform), nil
	}
	pin := task.PrivateData.Execution.TaskPlugin
	if string(task.Platform) != pin.Key {
		return nil, errors.New("任务插件来源不匹配")
	}
	loaded, err := service.LoadPinnedTaskPlugin(pin)
	if err != nil {
		return nil, err
	}
	return pluginadaptor.NewLegacy(loaded, task), nil
}

// AuthorizePluginTaskAccess 同时用于源任务与查询/资源读取，复用本地分组规则。
func AuthorizePluginTaskAccess(c *gin.Context, task *model.Task, key string) error {
	if task == nil || task.UserId != c.GetInt("id") || string(task.Platform) != key || task.PrivateData.Execution == nil || task.PrivateData.Execution.TaskPlugin == nil || task.PrivateData.Execution.TaskPlugin.Key != key {
		return errors.New("任务不存在或无权访问")
	}
	group, err := resolveAuthorizedOriginTaskGroup(c, task.Group)
	if err != nil {
		return err
	}
	name := task.Properties.OriginModelName
	if name == "" {
		return errors.New("源任务缺少模型")
	}
	if common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled) {
		limits, _ := common.GetContextKeyType[map[string]bool](c, constant.ContextKeyTokenModelLimit)
		if _, ok := limits[ratio_setting.FormatMatchingModelName(name)]; !ok {
			return errors.New("令牌无权访问源任务模型")
		}
	}
	ch, err := model.GetChannelById(task.ChannelId, true)
	if err != nil {
		return err
	}
	if ch.Type != constant.ChannelTypeTaskPlugin || ch.GetSetting().TaskPluginKey != key || ch.Status != common.ChannelStatusEnabled || !model.IsChannelEnabledForGroupModel(group, name, ch.Id) {
		return errors.New("源任务渠道已停用或授权已撤销")
	}
	return nil
}

// ResolvePluginOriginTasks 在提示词审计前固定全部依赖的授权渠道和计费分组。
func ResolvePluginOriginTasks(c *gin.Context, info *relaycommon.RelayInfo) error {
	var body map[string]any
	if err := common.UnmarshalBodyReusable(c, &body); err != nil {
		return err
	}
	raw, exists := body["originTaskIds"]
	if !exists {
		return nil
	}
	ids, ok := raw.([]any)
	if !ok || len(ids) > 16 {
		return errors.New("originTaskIds 必须是最多 16 个公开任务 ID")
	}
	pinValue, _ := c.Get("task_plugin_snapshot")
	pin, ok := pinValue.(*model.TaskPluginSnapshot)
	if !ok {
		return errors.New("缺少插件版本")
	}
	for _, value := range ids {
		id, ok := value.(string)
		if !ok || strings.TrimSpace(id) == "" {
			return errors.New("源任务 ID 无效")
		}
		task, found, err := model.GetByTaskId(info.UserId, id)
		if err != nil {
			return err
		}
		if !found {
			return errors.New("源任务不存在")
		}
		if err := AuthorizePluginTaskAccess(c, task, pin.Key); err != nil {
			return err
		}
		if task.PrivateData.Execution.TaskPlugin.SourceHash != pin.SourceHash {
			return errors.New("暂不支持跨插件版本引用源任务")
		}
		group, err := resolveAuthorizedOriginTaskGroup(c, task.Group)
		if err != nil {
			return err
		}
		if locked, ok := info.LockedChannel.(*model.Channel); ok && (locked.Id != task.ChannelId || info.UsingGroup != group) {
			return errors.New("多个源任务必须属于同一渠道和分组")
		}
		ch, err := model.GetChannelById(task.ChannelId, true)
		if err != nil {
			return err
		}
		if !model.IsChannelEnabledForGroupModel(group, info.OriginModelName, ch.Id) {
			return fmt.Errorf("源渠道不支持当前请求模型")
		}
		info.LockedChannel = ch
		info.UsingGroup = group
		common.SetContextKey(c, constant.ContextKeyUsingGroup, group)
		common.SetContextKey(c, constant.ContextKeySelectedChannelGroup, group)
		info.OriginTasks = append(info.OriginTasks, relaycommon.OriginTaskRef{TaskID: task.TaskID, UpstreamTaskID: task.GetUpstreamTaskID(), Action: task.Action, Status: string(task.Status), Data: task.PrivateData.PluginData})
	}
	if len(info.OriginTasks) > 0 {
		info.OriginTaskID = info.OriginTasks[0].UpstreamTaskID
	}
	return nil
}
