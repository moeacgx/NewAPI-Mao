const sdk = globalThis.__NEW_API_EXTENSION_NATIVE_SDK__;
if (!sdk || sdk.sdk !== "v1" || sdk.platform !== "default") {
  throw new Error("Native extension SDK v1 for default is unavailable.");
}
const { jsx, jsxs } = sdk.modules["react/jsx-runtime"];
const { Button } = sdk.modules["@/components/ui/button"];
const { Card } = sdk.modules["@/components/ui/card"];
const { Input } = sdk.modules["@/components/ui/input"];
const { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } =
  sdk.modules["@/components/ui/table"];
const { SectionPageLayout } = sdk.modules["@/components/layout"];

const ui = {
  api: () => sdk.modules["@/lib/api"].api,
  notificationPath: "/notification-center",
  Button(props) {
    return jsx(Button, {
      type: "button",
      variant: props.primary ? "default" : "outline",
      size: "default",
      disabled: props.disabled,
      onClick: props.onClick,
      "aria-label": props.label,
      title: props.label,
      children: props.label,
    });
  },
  Panel(props) {
    return (Array.isArray(props.children) ? jsxs : jsx)(Card, {
      ...props,
      className: `guard-panel ${props.className || ""}`,
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
    return jsx(Input, { ...props, onChange: (event) => props.onChange(event.target.value) });
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
      children: jsxs(Table, {
        children: [
          jsx(TableHeader, {
            children: jsx(TableRow, {
              children: props.columns.map((column) =>
                jsx(TableHead, { children: column.title }, column.key),
              ),
            }),
          }),
          jsx(TableBody, {
            children: props.items.map((item) =>
              jsx(
                TableRow,
                {
                  children: props.columns.map((column) =>
                    jsx(TableCell, { children: column.render(item) }, column.key),
                  ),
                },
                item.id,
              ),
            ),
          }),
        ],
      }),
    });
  },
  Shell(props) {
    return jsxs(SectionPageLayout, {
      children: [
        jsx(SectionPageLayout.Title, { children: props.title }),
        jsx(SectionPageLayout.Actions, { children: props.actions }),
        jsx(SectionPageLayout.Content, { children: props.children }),
      ],
    });
  },
};

export default createGuardPage(sdk, ui, guardTranslations);
