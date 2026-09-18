export function createGuardPage(sdk, ui, translations) {
  const { jsx, jsxs, Fragment } = sdk.modules["react/jsx-runtime"];
  const { useEffect, useState } = sdk.modules.react;
  const I18n = sdk.modules["react-i18next"];
  const namespace = "upstream-model-guard";
  const useTranslation = () => I18n.useTranslation(namespace, { useSuspense: false });
  const base = "/api/extensions/upstream-model-guard";

  function unwrap(response) {
    const body = response?.data ?? response;
    if (body?.success === false) throw new Error(body.message || "Request failed");
    return body?.data ?? body;
  }

  function errorText(error, t) {
    return error?.response?.data?.message || error?.message || t("Request failed");
  }

  function RuleEditor(props) {
    const { t } = useTranslation();
    const label = t("Rule {{number}}", { number: props.index + 1 });
    const modelId = `model-guard-model-${props.rule.key}`;
    const upstreamId = `model-guard-upstream-${props.rule.key}`;
    const unavailable = props.rule.group_codes.filter(
      (code) => !props.groups.some((group) => group.code === code),
    );
    const groups = [...props.groups, ...unavailable.map((code) => ({ code }))];
    const changeGroup = (code, checked) =>
      props.onChange({
        group_codes: checked
          ? [...props.rule.group_codes, code]
          : props.rule.group_codes.filter((value) => value !== code),
      });
    return jsxs(ui.Panel, {
      role: "group",
      "aria-label": label,
      className: "guard-rule",
      children: [
        jsxs("div", {
          className: "guard-panel-header",
          children: [
            jsx("h3", { children: label }),
            jsxs("div", {
              className: "guard-actions",
              children: [
                jsx(ui.Switch, {
                  checked: props.rule.enabled,
                  disabled: props.disabled,
                  label: t("Rule enabled"),
                  onChange: (enabled) => props.onChange({ enabled }),
                }),
                jsx(ui.Button, {
                  icon: "Trash2",
                  label: t("Remove rule"),
                  iconOnly: true,
                  disabled: props.disabled,
                  onClick: props.onRemove,
                }),
              ],
            }),
          ],
        }),
        jsxs("div", {
          className: "guard-panel-content guard-rule-fields",
          children: [
            jsxs("fieldset", {
              className: "guard-groups",
              disabled: props.disabled,
              children: [
                jsx("legend", { children: t("Selected groups") }),
                groups.length
                  ? jsx("div", {
                      className: "guard-group-options",
                      children: groups.map((group) =>
                        jsx(
                          ui.Checkbox,
                          {
                            label: group.name?.trim() || group.code,
                            checked: props.rule.group_codes.includes(group.code),
                            disabled: props.disabled,
                            onChange: (checked) => changeGroup(group.code, checked),
                          },
                          group.code,
                        ),
                      ),
                    })
                  : jsx("p", { className: "guard-muted", children: t("No groups available") }),
              ],
            }),
            jsxs("div", {
              className: "guard-model-fields",
              children: [
                jsxs("div", {
                  className: "guard-field",
                  children: [
                    jsx("label", { htmlFor: modelId, children: t("Request model") }),
                    jsx(ui.Input, {
                      id: modelId,
                      value: props.rule.model,
                      disabled: props.disabled,
                      maxLength: 255,
                      onChange: (model) => props.onChange({ model }),
                    }),
                  ],
                }),
                jsxs("div", {
                  className: "guard-field",
                  children: [
                    jsx("label", { htmlFor: upstreamId, children: t("Allowed upstream models") }),
                    jsx(ui.Textarea, {
                      id: upstreamId,
                      value: props.rule.upstreamText,
                      disabled: props.disabled,
                      rows: 4,
                      placeholder: t("One exact model name per line"),
                      onChange: (upstreamText) => props.onChange({ upstreamText }),
                    }),
                  ],
                }),
              ],
            }),
          ],
        }),
      ],
    });
  }

  function Settings(props) {
    const { t } = useTranslation();
    const onGroupsLoaded = props.onGroupsLoaded;
    const [nextKey, setNextKey] = useState(1);
    const [reload, setReload] = useState(0);
    const [state, setState] = useState({ loading: true, error: "", config: null, groups: [] });
    const [draft, setDraft] = useState({ enabled: false, rules: [] });
    const [saving, setSaving] = useState(false);
    const [saveError, setSaveError] = useState("");
    const [saved, setSaved] = useState(false);
    const [conflict, setConflict] = useState(false);

    useEffect(() => {
      let active = true;
      setState((current) => ({ ...current, loading: true, error: "" }));
      Promise.all([
        ui.api().get(`${base}/config`, { skipErrorHandler: true }).then(unwrap),
        ui.api().get(`${base}/groups`, { skipErrorHandler: true }).then(unwrap),
      ])
        .then(([config, groups]) => {
          if (!active) return;
          setState({
            loading: false,
            error: "",
            config,
            groups: Array.isArray(groups) ? groups : [],
          });
          onGroupsLoaded(Array.isArray(groups) ? groups : []);
          setDraft({
            enabled: Boolean(config.enabled),
            rules: (config.rules || []).map((rule, index) => ({
              key: index + 1,
              enabled: Boolean(rule.enabled),
              group_codes: [...rule.group_codes],
              model: rule.model,
              upstreamText: rule.upstream_models.join("\n"),
            })),
          });
          setNextKey((config.rules || []).length + 1);
          setConflict(false);
          setSaveError("");
        })
        .catch((error) => {
          if (active) {
            setState((current) => ({ ...current, loading: false, error: errorText(error, t) }));
          }
        });
      return () => {
        active = false;
      };
    }, [reload, t, onGroupsLoaded]);

    function edit(patch) {
      setDraft((current) => ({ ...current, ...patch }));
      setSaved(false);
      setSaveError("");
    }

    function updateRule(key, patch) {
      edit({ rules: draft.rules.map((rule) => (rule.key === key ? { ...rule, ...patch } : rule)) });
    }

    function addRule() {
      edit({
        rules: [
          ...draft.rules,
          { key: nextKey, enabled: true, group_codes: [], model: "", upstreamText: "" },
        ],
      });
      setNextKey((value) => value + 1);
    }

    async function save() {
      if (saving || conflict || !state.config) return;
      const rules = draft.rules.map((rule) => ({
        enabled: rule.enabled,
        group_codes: [...rule.group_codes],
        model: rule.model.trim(),
        upstream_models: [
          ...new Set(
            rule.upstreamText
              .split(/\r?\n/)
              .map((value) => value.trim())
              .filter(Boolean),
          ),
        ],
      }));
      const invalid = rules.findIndex(
        (rule) => !rule.group_codes.length || !rule.model || !rule.upstream_models.length,
      );
      if (invalid >= 0) {
        setSaveError(
          t("Complete groups and model names for rule {{number}}.", { number: invalid + 1 }),
        );
        return;
      }
      setSaving(true);
      setSaved(false);
      setSaveError("");
      try {
        const config = unwrap(
          await ui.api().put(
            `${base}/config`,
            {
              expected_version: state.config.config_version,
              enabled: draft.enabled,
              rules,
            },
            { skipErrorHandler: true },
          ),
        );
        setState((current) => ({ ...current, config }));
        setSaved(true);
      } catch (error) {
        const stale = error?.response?.status === 409;
        setConflict(stale);
        setSaveError(
          stale
            ? t("Settings changed elsewhere. Reload before saving again.")
            : errorText(error, t),
        );
      } finally {
        setSaving(false);
      }
    }

    if (state.loading) {
      return jsx(ui.Panel, {
        children: jsx("div", {
          className: "guard-state",
          role: "status",
          children: t("Loading settings..."),
        }),
      });
    }
    if (state.error) {
      return jsx(ui.Panel, {
        children: jsxs("div", {
          className: "guard-state",
          children: [
            jsx("div", { role: "alert", className: "guard-error", children: state.error }),
            jsx(ui.Button, {
              icon: "RefreshCw",
              label: t("Reload settings"),
              onClick: () => setReload((value) => value + 1),
            }),
          ],
        }),
      });
    }
    return jsxs(Fragment, {
      children: [
        jsxs(ui.Panel, {
          children: [
            jsxs("div", {
              className: "guard-panel-header guard-config-toolbar",
              children: [
                jsx("h2", { children: t("Model matching rules") }),
                jsx(ui.Switch, {
                  checked: draft.enabled,
                  disabled: saving || conflict,
                  label: t("Detection enabled"),
                  onChange: (enabled) => edit({ enabled }),
                }),
                jsxs("div", {
                  className: "guard-actions",
                  children: [
                    jsx(ui.Button, {
                      icon: "Plus",
                      label: t("Add rule"),
                      disabled: saving || conflict,
                      onClick: addRule,
                    }),
                    jsx(ui.Button, {
                      icon: "Save",
                      primary: true,
                      label: saving ? t("Saving...") : t("Save settings"),
                      disabled: saving || conflict,
                      onClick: save,
                    }),
                  ],
                }),
              ],
            }),
            saveError &&
              jsx("div", {
                className: "guard-panel-content guard-error",
                role: "alert",
                children: saveError,
              }),
            conflict &&
              jsx("div", {
                className: "guard-panel-content",
                children: jsx(ui.Button, {
                  icon: "RefreshCw",
                  label: t("Reload settings"),
                  disabled: saving,
                  onClick: () => {
                    setSaved(false);
                    setReload((value) => value + 1);
                  },
                }),
              }),
            saved &&
              jsx("div", {
                className: "guard-panel-content guard-success",
                role: "status",
                children: t("Model guard settings saved"),
              }),
            draft.rules.length === 0 &&
              jsx("p", {
                className: "guard-state guard-muted",
                children: t("No model matching rules"),
              }),
          ],
        }),
        ...draft.rules.map((rule, index) =>
          jsx(
            RuleEditor,
            {
              rule,
              index,
              groups: state.groups,
              disabled: saving || conflict,
              onChange: (patch) => updateRule(rule.key, patch),
              onRemove: () => edit({ rules: draft.rules.filter((item) => item.key !== rule.key) }),
            },
            rule.key,
          ),
        ),
      ],
    });
  }

  function Records(props) {
    const { t } = useTranslation();
    const [page, setPage] = useState(1);
    const [refresh, setRefresh] = useState(0);
    const [state, setState] = useState({ loading: true, error: "", items: [], total: 0 });
    useEffect(() => {
      let active = true;
      setState((current) => ({ ...current, loading: true, error: "" }));
      ui.api()
        .get(`${base}/records`, { params: { page, page_size: 20 }, skipErrorHandler: true })
        .then(unwrap)
        .then((data) => {
          if (active) {
            setState({
              loading: false,
              error: "",
              items: data.items || [],
              total: Number(data.total || 0),
            });
          }
        })
        .catch((error) => {
          if (active) {
            setState((current) => ({ ...current, loading: false, error: errorText(error, t) }));
          }
        });
      return () => {
        active = false;
      };
    }, [page, refresh, t]);
    const columns = [
      {
        key: "created_at",
        title: t("Disabled at"),
        render: (row) => {
          const date = new Date(Number(row.created_at) * 1000);
          return Number.isNaN(date.getTime()) ? "-" : date.toLocaleString();
        },
      },
      {
        key: "channel",
        title: t("Channel"),
        render: (row) =>
          jsxs("div", {
            children: [
              jsx("span", { children: row.channel_name || "-" }),
              jsx("small", {
                className: "guard-muted guard-channel-id",
                children: `#${row.channel_id}`,
              }),
            ],
          }),
      },
      {
        key: "group",
        title: t("Group"),
        render: (row) =>
          row.group_name?.trim() ||
          props.groups.find((group) => group.code === row.group)?.name?.trim() ||
          row.group ||
          "-",
      },
      { key: "model", title: t("Request model"), render: (row) => row.requested_model || "-" },
      {
        key: "expected",
        title: t("Expected upstream models"),
        render: (row) => (row.expected_upstream_models || []).join("\n") || "-",
      },
      {
        key: "actual",
        title: t("Actual upstream model"),
        render: (row) => row.actual_upstream_model || "-",
      },
    ];
    let content = jsx(ui.RecordsTable, { columns, items: state.items });
    if (state.loading) {
      content = jsx("div", {
        className: "guard-state",
        role: "status",
        children: t("Loading records..."),
      });
    } else if (state.error) {
      content = jsx("div", {
        className: "guard-state guard-error",
        role: "alert",
        children: state.error,
      });
    } else if (!state.items.length) {
      content = jsx("div", {
        className: "guard-state guard-muted",
        children: t("No channel disable records"),
      });
    }
    const pages = Math.max(1, Math.ceil(state.total / 20));
    return jsxs(ui.Panel, {
      className: "guard-records",
      children: [
        jsxs("div", {
          className: "guard-panel-header",
          children: [
            jsx("h2", { children: t("Channel disable records") }),
            jsx(ui.Button, {
              icon: "RefreshCw",
              label: t("Refresh records"),
              disabled: state.loading,
              iconOnly: true,
              onClick: () => setRefresh((value) => value + 1),
            }),
          ],
        }),
        content,
        jsxs("div", {
          className: "guard-panel-footer",
          children: [
            jsx("span", {
              className: "guard-muted",
              children: t("Total {{count}}", { count: state.total }),
            }),
            jsxs("div", {
              className: "guard-actions",
              children: [
                jsx(ui.Button, {
                  icon: "ChevronLeft",
                  label: t("Previous page"),
                  iconOnly: true,
                  disabled: state.loading || page <= 1,
                  onClick: () => setPage((value) => value - 1),
                }),
                jsx("span", { className: "guard-page-number", children: `${page} / ${pages}` }),
                jsx(ui.Button, {
                  icon: "ChevronRight",
                  label: t("Next page"),
                  iconOnly: true,
                  disabled: state.loading || page >= pages,
                  onClick: () => setPage((value) => value + 1),
                }),
              ],
            }),
          ],
        }),
      ],
    });
  }

  function GuardContent() {
    const { t } = useTranslation();
    const [groups, setGroups] = useState([]);
    return jsx(ui.Shell, {
      title: t("Upstream model guard"),
      actions: jsx("a", {
        className: "guard-notification-link",
        href: ui.notificationPath,
        children: t("Notification Center"),
      }),
      children: jsxs("div", {
        className: "upstream-model-guard",
        children: [jsx(Settings, { onGroupsLoaded: setGroups }), jsx(Records, { groups })],
      }),
    });
  }

  return function UpstreamModelGuardPage() {
    const { i18n } = I18n.useTranslation();
    const [ready, setReady] = useState(false);
    useEffect(() => {
      for (const [locale, values] of Object.entries(translations)) {
        i18n.addResourceBundle(locale, namespace, values, true, false);
      }
      // 两套宿主使用不同的中文语言代码，别名只注册到插件命名空间。
      i18n.addResourceBundle("zhCN", namespace, translations.zh, true, false);
      i18n.addResourceBundle("zhTW", namespace, translations["zh-TW"], true, false);
      setReady(true);
    }, [i18n]);
    return ready ? jsx(GuardContent, {}) : null;
  };
}
