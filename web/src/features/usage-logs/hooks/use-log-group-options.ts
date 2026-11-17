import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

type LogGroup = { id?: number; code?: string; name?: string }
type LogGroupResponse = {
  success: boolean
  message?: string
  data: LogGroup[] | Record<string, LogGroup>
  group_names?: Record<string, string>
}

export function useLogGroupOptions(isAdmin: boolean) {
  const user = useAuthStore((state) => state.auth.user)
  const sessionID = useAuthStore((state) => state.auth.session?.sid)
  return useQuery({
    // 保留已有分组查询的失效前缀，同时隔离视图、用户和会话，避免沿用上个主体的选项。
    queryKey: [
      ...(isAdmin ? ['system-settings', 'group-details'] : ['user-groups']),
      'usage-log-options',
      user?.id,
      sessionID,
      user?.role,
      user?.group,
    ],
    enabled: Boolean(user),
    // 默认结构共享复制普通对象会丢失合法 __proto__ 自有键；保留完整名称响应。
    structuralSharing: false,
    queryFn: async ({ signal }) => {
      const { data: response } = await api
        .get<LogGroupResponse>(
          isAdmin ? '/api/group/details' : '/api/user/self/groups',
          // React Query 负责去重；跨挂载共享客户端请求会复用已取消的 signal。
          // 此控件自行显示错误与重试，正常取消请求不应弹出错误通知。
          { signal, disableDuplicate: true, skipErrorHandler: true }
        )
        .catch((cause: unknown) => {
          // 选项读取失败由控件恢复，避免全局 Axios 500 处理把日志页跳走。
          throw new Error('Failed to load groups', { cause })
        })
      if (!response.success) {
        throw new Error(response.message || 'Failed to load groups')
      }
      const entries = Array.isArray(response.data)
        ? response.data.map((group) => [group.code ?? '', group] as const)
        : Object.entries(response.data)
      const options = entries.map(([key, group]) => ({
        value: group.code || key,
        label: group.name || group.code || key,
      }))
      return { options, names: response.group_names }
    },
  })
}
