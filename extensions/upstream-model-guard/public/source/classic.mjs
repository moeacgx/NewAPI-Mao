const sdk = globalThis.__NEW_API_EXTENSION_NATIVE_SDK__;
if (!sdk || sdk.sdk !== "v1" || sdk.platform !== "classic") {
  throw new Error("Native extension SDK v1 for classic is unavailable.");
}
const { jsx, jsxs } = sdk.modules["react/jsx-runtime"];
const { Button, Card, Input, Table } = sdk.modules["@douyinfe/semi-ui"];
const helpers = sdk.modules["../../helpers"];

const ui = {
  api: () => helpers.getAPI(),
  notificationPath: "/notification-center",
  Button(props) {
    return jsx(Button, {
      htmlType: "button",
      theme: props.primary ? "solid" : "borderless",
      type: props.primary ? "primary" : "tertiary",
      disabled: props.disabled,
      onClick: props.onClick,
      "aria-label": props.ariaLabel || props.label,
      title: props.ariaLabel || props.label,
      children: props.label,
    });
  },
  Panel(props) {
    return (Array.isArray(props.children) ? jsxs : jsx)(Card, {
      ...props,
      className: `guard-panel ${props.className || ""}`,
      bodyStyle: { padding: 0 },
    });
  },
  Switch(props) {
    return jsxs("label", {
      className: "guard-switch-label",
      children: [
        jsx("input", {
          type: "checkbox",
          role: "switch",
          className: "guard-toggle",
          checked: props.checked,
          disabled: props.disabled,
          onChange: (event) => props.onChange(event.target.checked),
        }),
        jsx("span", { children: props.label }),
      ],
    });
  },
  Checkbox(props) {
    return jsxs("label", {
      className: "guard-checkbox-label",
      children: [
        jsx("input", {
          type: "checkbox",
          className: "guard-checkbox",
          checked: props.checked,
          disabled: props.disabled,
          onChange: (event) => props.onChange(event.target.checked),
        }),
        jsx("span", { children: props.label }),
      ],
    });
  },
  Input(props) {
    return jsx(Input, props);
  },
  Textarea(props) {
    return jsx("textarea", {
      ...props,
      className: "guard-textarea",
      onChange: (event) => props.onChange(event.target.value),
    });
  },
  RecordsTable(props) {
    return jsx("div", {
      className: "guard-table-scroll",
      children: jsx(Table, {
        rowKey: "id",
        pagination: false,
        dataSource: props.items,
        columns: props.columns.map((column) => ({
          key: column.key,
          title: column.title,
          render: (_, row) => column.render(row),
        })),
      }),
    });
  },
  Shell(props) {
    return jsxs("main", {
      className: "upstream-model-guard-shell",
      children: [
        jsxs("header", {
          className: "guard-page-header",
          children: [jsx("h1", { children: props.title }), props.actions],
        }),
        props.children,
      ],
    });
  },
};

export default createGuardPage(sdk, ui, guardTranslations);
