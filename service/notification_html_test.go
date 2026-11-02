package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramHTMLFormattingAndSafety(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		valid      bool
	}{
		{"加粗和引用", "🎫 <b>新工单</b>\n<blockquote><i>正文 &amp; 内容</i></blockquote>", true},
		{"折叠引用", "<blockquote expandable>多行\n内容</blockquote>", true},
		{"其他格式", "<u>下划线</u><s>删除</s><tg-spoiler>剧透</tg-spoiler><span class=\"tg-spoiler\">剧透</span>", true},
		{"代码语言", "<pre><code class=\"language-go\">fmt.Println(&#34;hello&#34;)</code></pre>", true},
		{"安全链接", "<a href=\"https://example.com\">查看</a>", true},
		{"交叉闭合", "<b><i>内容</b></i>", false},
		{"未闭合", "<b>内容", false},
		{"脚本", "<script>alert(1)</script>", false},
		{"事件属性", "<b onclick=\"evil()\">内容</b>", false},
		{"危险链接", "<a href=\"javascript:alert(1)\">链接</a>", false},
		{"链接属性不能用变量", "<a href=\"https://example.com/{{title}}\">链接</a>", false},
		{"嵌套引用", "<blockquote><blockquote>内容</blockquote></blockquote>", false},
		{"代码不可混用格式", "<b><code>内容</code></b>", false},
		{"不支持浏览器实体", "内容&nbsp;文本", false},
		{"纯文本符号需要转义", "1 < 2 & 3", false},
		{"自闭合标签", "<b/>", false},
		{"重复属性不可被浏览器丢弃", `<a href="https://example.com" href="javascript:evil()">链接</a>`, false},
		{"结束标签不能携带属性", `<b>正文</b title="{{title}}">`, false},
		{"结束标签不能自闭合", `<b>正文</b/>`, false},
		{"链接属性实体也需校验", `<a href="https://example.com/?q=&nbsp;">链接</a>`, false},
		{"链接属性裸符号也需转义", `<a href="https://example.com/?a=1&b=2">链接</a>`, false},
		{"链接合法实体保留", `<a href="https://example.com/?a=1&amp;b=2">链接</a>`, true},
		{"结束时不能残留半个标签", `hello<a`, false},
		{"属性中的尖括号也需转义", `<a href="https://example.com/?q=<test>">链接</a>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTelegramHTML(tc.text)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestTelegramDirectSendRejectsInvalidHTMLBeforeRequest(t *testing.T) {
	err := SendTelegramNotification("test-token", "-100", "<script>无效格式</script>")
	require.EqualError(t, err, "invalid Telegram HTML message")
}

func TestNotificationHTMLUserContentRemainsTextAndMentionCannotBecomeNestedLink(t *testing.T) {
	got, err := RenderNotificationTemplate("<blockquote>{{content}}</blockquote>", map[string]any{"content": "<script>恶意</script> & \"'"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "<blockquote>&lt;script&gt;恶意&lt;/script&gt; &amp; &#34;&#39;</blockquote>", got)
	_, err = RenderNotificationTemplate("<a href=\"https://example.com\">{{mention}}</a>", nil, &model.NotificationTarget{MentionUserId: "123", MentionName: "客服"})
	require.Error(t, err)
	_, err = RenderNotificationTemplate("<pre>{{mention}}</pre>", nil, &model.NotificationTarget{MentionUserId: "123", MentionName: "客服"})
	require.Error(t, err)
}
