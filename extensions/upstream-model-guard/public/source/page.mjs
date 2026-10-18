export function createGuardPage(sdk, ui, translations) {
  const { jsx, jsxs, Fragment } = sdk.modules["react/jsx-runtime"];
  const { useEffect, useState } = sdk.modules.react;
  const I18n = sdk.modules["react-i18next"];
  const namespace = "upstream-model-guard";
  const useTranslation = () =>
    I18n.useTranslation(namespace, { useSuspense: false });
  const base = "/api/extensions/upstream-model-guard";

  function unwrap(response) {
    const body = response?.data ?? response;
    if (body?.success === false)
      throw new Error(body.message || "Request failed");
    return body?.data ?? body;
  }

  function errorText(error, t) {
    return (
      error?.response?.data?.message || error?.message || t("Request failed")
    );
  }

  function channelLabel(channel, t) {
    return `${channel.name?.trim() || t("Unavailable channel")} (#${channel.id})`;
  }

  function ChannelAllowlist(props) {
    const { t } = useTranslation();
    const onReady = props.onReady;
    const [search, setSearch] = useState("");
    const [query, setQuery] = useState({ keyword: "", page: 1, revision: 0 });
    const [state, setState] = useState({
      loading: true,
      error: "",
      items: [],
      total: 0,
    });
    const [idsText, setIdsText] = useState(() =>
      props.selected.map((channel) => channel.id).join("\n"),
    );
    const [idsError, setIdsError] = useState("");
    useEffect(() => {
      onReady(!state.loading && !state.error && !idsError);
    }, [state.loading, state.error, idsError, onReady]);
    useEffect(() => {
      let active = true;
      onReady(false);
      setState((current) => ({ ...current, loading: true, error: "" }));
      ui.api()
        .get(`${base}/channels`, {
          params: { keyword: query.keyword, page: query.page, page_size: 50 },
          skipErrorHandler: true,
        })
        .then(unwrap)
        .then((data) => {
          if (!active) return;
          if (!Array.isArray(data.items))
            throw new Error(t("Failed to load channels"));
          setState({
            loading: false,
            error: "",
            items: data.items,
            total: Number(data.total || 0),
          });
        })
        .catch((error) => {
          if (active)
            setState({
              loading: false,
              error: errorText(error, t),
              items: [],
              total: 0,
            });
        });
      return () => {
        active = false;
      };
    }, [query, t, onReady]);

    function changeIds(value) {
      if (props.disabled) return;
      setIdsText(value);
      const parts = value.split(/[\s,，;；]+/u).filter(Boolean);
      if (
        parts.some(
          (part) =>
            !/^\d+$/.test(part) ||
            !Number.isSafeInteger(Number(part)) ||
            Number(part) <= 0,
        )
      ) {
        setIdsError("Enter positive integer channel IDs.");
        props.onChange(props.selected);
        return;
      }
      const ids = [...new Set(parts.map(Number))];
      if (ids.length > 1000) {
        setIdsError("Select no more than 1000 channels.");
        props.onChange(props.selected);
        return;
      }
      setIdsError("");
      props.onChange(
        ids.map(
          (id) =>
            props.selected.find((channel) => channel.id === id) ||
            state.items.find((channel) => channel.id === id) || {
              id,
              name: t("Channel"),
              status: 0,
            },
        ),
      );
    }

    function toggle(channel, checked) {
      if (props.disabled) return;
      if (!checked) {
        const selected = props.selected.filter(
          (item) => item.id !== channel.id,
        );
        setIdsText(selected.map((item) => item.id).join("\n"));
        props.onChange(selected);
        return;
      }
      if (
        props.selected.length >= 1000 ||
        props.selected.some((item) => item.id === channel.id)
      ) {
        return;
      }
      const selected = [...props.selected, channel];
      setIdsText(selected.map((item) => item.id).join("\n"));
      props.onChange(selected);
    }

    const pages = Math.max(1, Math.ceil(state.total / 50));
    return jsxs(ui.Panel, {
      className: "guard-allowlist",
      children: [
        jsxs("div", {
          className: "guard-panel-header",
          children: [
            jsx("h2", { children: t("Channel allowlist") }),
            jsx("span", {
              className: "guard-muted",
              children: t("Selected {{count}} of 1000", {
                count: props.selected.length,
              }),
            }),
          ],
        }),
        jsxs("div", {
          className: "guard-panel-content",
          children: [
            jsx("p", {
              className: "guard-muted guard-help",
              children: t(
                "Allowlisted channels skip detection, counting, records, disabling, and notifications.",
              ),
            }),
            jsxs("div", {
              className: "guard-field",
              children: [
                jsx("label", {
                  htmlFor: "guard-channel-ids",
                  children: t("Allowlisted channel IDs"),
                }),
                jsx(ui.Textarea, {
                  id: "guard-channel-ids",
                  value: idsText,
                  disabled: props.disabled,
                  rows: 3,
                  maxLength: 30000,
                  "aria-invalid": Boolean(idsError),
                  "aria-describedby":
                    "guard-channel-ids-help" +
                    (idsError ? " guard-channel-ids-error" : ""),
                  onChange: changeIds,
                }),
                jsx("p", {
                  id: "guard-channel-ids-help",
                  className: "guard-muted guard-help",
                  children: t(
                    "Separate IDs with commas or newlines. Duplicates are removed. Clear the field to remove all allowlisted channels, then save settings.",
                  ),
                }),
                idsError &&
                  jsx("p", {
                    id: "guard-channel-ids-error",
                    role: "alert",
                    className: "guard-error",
                    children: t(idsError),
                  }),
              ],
            }),
            jsx("div", {
              className: "guard-selected-channels",
              children: props.selected.length
                ? props.selected.map((channel) =>
                    jsxs(
                      "div",
                      {
                        className: "guard-selected-channel",
                        children: [
                          jsx("span", { children: channelLabel(channel, t) }),
                          jsx(ui.Button, {
                            label: t("Remove"),
                            ariaLabel: t("Remove {{channel}} from allowlist", {
                              channel: channelLabel(channel, t),
                            }),
                            disabled: props.disabled || Boolean(idsError),
                            onClick: () => toggle(channel, false),
                          }),
                        ],
                      },
                      channel.id,
                    ),
                  )
                : jsx("p", {
                    className: "guard-muted",
                    children: t("No allowlisted channels"),
                  }),
            }),
            jsxs("form", {
              className: "guard-channel-search",
              onSubmit: (event) => {
                event.preventDefault();
                if (!props.disabled) {
                  setQuery((current) => ({
                    keyword: search.trim(),
                    page: 1,
                    revision: current.revision + 1,
                  }));
                }
              },
              children: [
                jsxs("div", {
                  className: "guard-field",
                  children: [
                    jsx("label", {
                      htmlFor: "guard-channel-search",
                      children: t("Search channels"),
                    }),
                    jsx(ui.Input, {
                      id: "guard-channel-search",
                      value: search,
                      disabled: props.disabled,
                      maxLength: 200,
                      onChange: setSearch,
                    }),
                  ],
                }),
                jsx(ui.Button, {
                  label: t("Search"),
                  disabled: props.disabled || state.loading,
                  onClick: () =>
                    setQuery((current) => ({
                      keyword: search.trim(),
                      page: 1,
                      revision: current.revision + 1,
                    })),
                }),
              ],
            }),
            jsxs("div", {
              role: "group",
              "aria-label": t("Available channels"),
              className: "guard-channel-results",
              children: [
                state.loading
                  ? jsx("p", {
                      role: "status",
                      className: "guard-muted",
                      children: t("Loading channels..."),
                    })
                  : null,
                state.error
                  ? jsxs("div", {
                      className: "guard-channel-error",
                      children: [
                        jsx("p", {
                          role: "alert",
                          className: "guard-error",
                          children: state.error,
                        }),
                        jsx(ui.Button, {
                          label: t("Retry channel search"),
                          disabled: props.disabled,
                          onClick: () =>
                            setQuery((current) => ({
                              ...current,
                              revision: current.revision + 1,
                            })),
                        }),
                      ],
                    })
                  : null,
                !state.loading && !state.error && state.items.length === 0
                  ? jsx("p", {
                      className: "guard-muted",
                      children: t("No channels found"),
                    })
                  : null,
                !state.loading && !state.error
                  ? jsx("div", {
                      className: "guard-channel-options",
                      children: state.items.map((channel) => {
                        const checked = props.selected.some(
                          (item) => item.id === channel.id,
                        );
                        return jsx(
                          ui.Checkbox,
                          {
                            label: channelLabel(channel, t),
                            checked,
                            disabled:
                              props.disabled ||
                              Boolean(idsError) ||
                              (!checked && props.selected.length >= 1000),
                            onChange: (value) => toggle(channel, value),
                          },
                          channel.id,
                        );
                      }),
                    })
                  : null,
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
                          label: t("Previous page"),
                          disabled:
                            props.disabled || state.loading || query.page <= 1,
                          onClick: () =>
                            setQuery((current) => ({
                              ...current,
                              page: current.page - 1,
                            })),
                        }),
                        jsx("span", {
                          className: "guard-page-number",
                          children: `${query.page} / ${pages}`,
                        }),
                        jsx(ui.Button, {
                          label: t("Next page"),
                          disabled:
                            props.disabled ||
                            state.loading ||
                            query.page >= pages,
                          onClick: () =>
                            setQuery((current) => ({
                              ...current,
                              page: current.page + 1,
                            })),
                        }),
                      ],
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
                            checked: props.rule.group_codes.includes(
                              group.code,
                            ),
                            disabled: props.disabled,
                            onChange: (checked) =>
                              changeGroup(group.code, checked),
                          },
                          group.code,
                        ),
                      ),
                    })
                  : jsx("p", {
                      className: "guard-muted",
                      children: t("No groups available"),
                    }),
              ],
            }),
            jsxs("div", {
              className: "guard-model-fields",
              children: [
                jsxs("div", {
                  className: "guard-field",
                  children: [
                    jsx("label", {
                      htmlFor: modelId,
                      children: t("Request model"),
                    }),
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
                    jsx("label", {
                      htmlFor: upstreamId,
                      children: t("Allowed upstream models"),
                    }),
                    jsx(ui.Textarea, {
                      id: upstreamId,
                      value: props.rule.upstreamText,
                      disabled: props.disabled,
                      rows: 4,
                      placeholder: t("One exact model name per line"),
                      onChange: (upstreamText) =>
                        props.onChange({ upstreamText }),
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
    const [state, setState] = useState({
      loading: true,
      error: "",
      config: null,
      groups: [],
    });
    const [draft, setDraft] = useState({
      enabled: false,
      rules: [],
      threshold: "2",
      excludedChannels: [],
    });
    const [channelsReady, setChannelsReady] = useState(false);
    const [saving, setSaving] = useState(false);
    const [saveError, setSaveError] = useState("");
    const [saved, setSaved] = useState(false);
    const [conflict, setConflict] = useState(false);

    useEffect(() => {
      let active = true;
      setChannelsReady(false);
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
            threshold: String(config.failure_threshold ?? 2),
            excludedChannels: (config.excluded_channel_ids || []).map(
              (id) =>
                (config.excluded_channels || []).find(
                  (channel) => channel.id === id,
                ) || {
                  id,
                  name: "",
                  status: 0,
                },
            ),
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
            setState((current) => ({
              ...current,
              loading: false,
              error: errorText(error, t),
            }));
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
      edit({
        rules: draft.rules.map((rule) =>
          rule.key === key ? { ...rule, ...patch } : rule,
        ),
      });
    }

    function addRule() {
      edit({
        rules: [
          ...draft.rules,
          {
            key: nextKey,
            enabled: true,
            group_codes: [],
            model: "",
            upstreamText: "",
          },
        ],
      });
      setNextKey((value) => value + 1);
    }

    async function save() {
      if (saving || conflict || !state.config || !channelsReady) return;
      const threshold = Number(draft.threshold);
      if (!Number.isInteger(threshold) || threshold < 1 || threshold > 100) {
        setSaveError(t("Enter an integer from 1 to 100."));
        return;
      }
      if (draft.excludedChannels.length > 1000) {
        setSaveError(t("Select no more than 1000 channels."));
        return;
      }
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
        (rule) =>
          !rule.group_codes.length ||
          !rule.model ||
          !rule.upstream_models.length,
      );
      if (invalid >= 0) {
        setSaveError(
          t("Complete groups and model names for rule {{number}}.", {
            number: invalid + 1,
          }),
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
              failure_threshold: threshold,
              excluded_channel_ids: draft.excludedChannels.map(
                (channel) => channel.id,
              ),
              rules,
            },
            { skipErrorHandler: true },
          ),
        );
        setState((current) => ({ ...current, config }));
        // 保存后使用宿主返回的渠道名称回显手动填写的 ID。
        setDraft((current) => ({
          ...current,
          excludedChannels: current.excludedChannels.map(
            (channel) =>
              (config.excluded_channels || []).find(
                (item) => item.id === channel.id,
              ) || channel,
          ),
        }));
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
            jsx("div", {
              role: "alert",
              className: "guard-error",
              children: state.error,
            }),
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
                      disabled: saving || conflict || !channelsReady,
                      onClick: save,
                    }),
                  ],
                }),
              ],
            }),
            jsxs("div", {
              className: "guard-panel-content guard-tolerance",
              children: [
                jsxs("div", {
                  className: "guard-field",
                  children: [
                    jsx("label", {
                      htmlFor: "guard-failure-threshold",
                      children: t("Consecutive mismatch threshold"),
                    }),
                    jsx(ui.Input, {
                      id: "guard-failure-threshold",
                      type: "number",
                      min: 1,
                      max: 100,
                      step: 1,
                      value: draft.threshold,
                      disabled: saving || conflict,
                      onChange: (threshold) => edit({ threshold }),
                    }),
                  ],
                }),
                jsx("p", {
                  className: "guard-muted guard-help",
                  children: t(
                    "All detected requests on the same channel share one count across models and groups. A matching request resets the count; missing model names are skipped.",
                  ),
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
        jsx(ChannelAllowlist, {
          selected: draft.excludedChannels,
          disabled: saving || conflict,
          onReady: setChannelsReady,
          onChange: (excludedChannels) => edit({ excludedChannels }),
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
              onRemove: () =>
                edit({
                  rules: draft.rules.filter((item) => item.key !== rule.key),
                }),
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
    const [state, setState] = useState({
      loading: true,
      error: "",
      items: [],
      total: 0,
    });
    useEffect(() => {
      let active = true;
      setState((current) => ({ ...current, loading: true, error: "" }));
      ui.api()
        .get(`${base}/records`, {
          params: { page, page_size: 20 },
          skipErrorHandler: true,
        })
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
            setState((current) => ({
              ...current,
              loading: false,
              error: errorText(error, t),
            }));
          }
        });
      return () => {
        active = false;
      };
    }, [page, refresh, t]);
    const columns = [
      {
        key: "created_at",
        title: t("Detected at"),
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
          props.groups
            .find((group) => group.code === row.group)
            ?.name?.trim() ||
          row.group ||
          "-",
      },
      {
        key: "model",
        title: t("Request model"),
        render: (row) => row.requested_model || "-",
      },
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
      {
        key: "count",
        title: t("Consecutive mismatches"),
        render: (row) =>
          `${row.consecutive_mismatches ?? 1} / ${row.failure_threshold ?? 1}`,
      },
      {
        key: "disabled",
        title: t("Channel status"),
        render: (row) =>
          row.channel_disabled === false
            ? t("Not disabled")
            : t("Channel disabled"),
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
        children: t("No model mismatch records"),
      });
    }
    const pages = Math.max(1, Math.ceil(state.total / 20));
    return jsxs(ui.Panel, {
      className: "guard-records",
      role: "region",
      "aria-label": t("Model mismatch records"),
      children: [
        jsxs("div", {
          className: "guard-panel-header",
          children: [
            jsx("h2", { children: t("Model mismatch records") }),
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
                jsx("span", {
                  className: "guard-page-number",
                  children: `${page} / ${pages}`,
                }),
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
        children: [
          jsx(Settings, { onGroupsLoaded: setGroups }),
          jsx(Records, { groups }),
        ],
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
      i18n.addResourceBundle(
        "zhTW",
        namespace,
        translations["zh-TW"],
        true,
        false,
      );
      setReady(true);
    }, [i18n]);
    return ready ? jsx(GuardContent, {}) : null;
  };
}
