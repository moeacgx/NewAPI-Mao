package service

import (
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// ValidateTelegramHTML 校验 Telegram 支持的 HTML 子集，不依赖浏览器自动修复格式。
// 预览、保存和实际发送共用同一边界；模板变量只能放在文本位置。
func ValidateTelegramHTML(content string) error {
	if !utf8.ValidString(content) || strings.ContainsRune(content, 0) {
		return errors.New("invalid Telegram message text")
	}
	tokenizer := html.NewTokenizer(strings.NewReader(content))
	var stack []string
	for {
		tokenType := tokenizer.Next()
		raw := string(tokenizer.Raw())
		if tokenType == html.ErrorToken {
			if tokenizer.Err() != io.EOF || raw != "" {
				return errors.New("invalid Telegram HTML")
			}
			if len(stack) != 0 {
				return errors.New("unclosed Telegram HTML tag")
			}
			return nil
		}
		if tokenType == html.TextToken {
			if strings.ContainsAny(raw, "<>") {
				return errors.New("escape literal angle brackets in Telegram HTML")
			}
			if err := validateTelegramEntities(raw); err != nil {
				return err
			}
			continue
		}
		if tokenType != html.StartTagToken && tokenType != html.EndTagToken {
			return errors.New("unsupported Telegram HTML markup")
		}
		tag, attrs, err := parseTelegramHTMLTag(raw, tokenType == html.EndTagToken)
		if err != nil {
			return err
		}
		switch tag {
		case "b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "span", "tg-spoiler", "a", "code", "pre", "blockquote":
		default:
			return fmt.Errorf("unsupported Telegram HTML tag: %s", tag)
		}
		if tokenType == html.EndTagToken {
			if len(stack) == 0 || stack[len(stack)-1] != tag {
				return errors.New("mismatched Telegram HTML closing tag")
			}
			stack = stack[:len(stack)-1]
			continue
		}
		for _, parent := range stack {
			if (tag == "blockquote" && parent == "blockquote") || (tag == "a" && parent == "a") {
				return errors.New("Telegram links and quotes cannot contain the same entity")
			}
			if parent == "code" || parent == "pre" || tag == "code" || tag == "pre" {
				if !(len(stack) == 1 && parent == "pre" && tag == "code") {
					return errors.New("Telegram code cannot contain other formatting")
				}
			}
		}
		seen := make(map[string]bool)
		for _, attr := range attrs {
			if seen[attr.Key] || notificationTemplateVariable.MatchString(attr.Val) {
				return errors.New("duplicate attributes or template variables in Telegram HTML attributes")
			}
			seen[attr.Key] = true
			valid := false
			switch tag {
			case "a":
				parsed, err := url.Parse(attr.Val)
				valid = attr.Key == "href" && err == nil && parsed != nil && (parsed.Scheme == "https" || parsed.Scheme == "http" || parsed.Scheme == "tg" || parsed.Scheme == "mailto")
			case "span":
				valid = attr.Key == "class" && attr.Val == "tg-spoiler"
			case "blockquote":
				valid = attr.Key == "expandable" && attr.Val == ""
			case "code":
				valid = len(stack) == 1 && stack[0] == "pre" && attr.Key == "class" && strings.HasPrefix(attr.Val, "language-") && len(attr.Val) > len("language-")
			}
			if !valid {
				return errors.New("unsupported Telegram HTML attribute")
			}
		}
		if (tag == "a" && !seen["href"]) || (tag == "span" && !seen["class"]) {
			return errors.New("missing Telegram HTML attribute")
		}
		stack = append(stack, tag)
		if len(stack) > 32 {
			return errors.New("Telegram HTML nesting is too deep")
		}
	}
}

// 浏览器 tokenizer 会吞掉重复属性和结束标签属性，因此必须检查原始标签。
func parseTelegramHTMLTag(raw string, closing bool) (string, []html.Attribute, error) {
	invalid := errors.New("invalid Telegram HTML tag syntax")
	if err := validateTelegramEntities(raw); err != nil {
		return "", nil, err
	}
	if !strings.HasPrefix(raw, "<") || !strings.HasSuffix(raw, ">") {
		return "", nil, invalid
	}
	inner := raw[1 : len(raw)-1]
	if closing {
		if !strings.HasPrefix(inner, "/") {
			return "", nil, invalid
		}
		inner = inner[1:]
	}
	nameEnd := strings.IndexAny(inner, " \t\r\n\f")
	if nameEnd < 0 {
		nameEnd = len(inner)
	}
	name, rest := strings.ToLower(inner[:nameEnd]), inner[nameEnd:]
	if name == "" || strings.ContainsAny(name, "/=<>\"'") {
		return "", nil, invalid
	}
	if closing {
		if strings.TrimSpace(rest) != "" {
			return "", nil, invalid
		}
		return name, nil, nil
	}
	var attrs []html.Attribute
	seen := make(map[string]bool)
	for strings.TrimSpace(rest) != "" {
		if !strings.ContainsAny(rest[:1], " \t\r\n\f") {
			return "", nil, invalid
		}
		rest = strings.TrimSpace(rest)
		attrEnd := strings.IndexAny(rest, "= \t\r\n\f")
		if attrEnd < 0 {
			attrEnd = len(rest)
		}
		key := strings.ToLower(rest[:attrEnd])
		if key == "" || seen[key] {
			return "", nil, invalid
		}
		seen[key] = true
		rest = rest[attrEnd:]
		afterKey := strings.TrimSpace(rest)
		value := ""
		if strings.HasPrefix(afterKey, "=") {
			afterKey = strings.TrimSpace(afterKey[1:])
			if len(afterKey) == 0 || (afterKey[0] != '\'' && afterKey[0] != '"') {
				return "", nil, invalid
			}
			valueEnd := strings.IndexByte(afterKey[1:], afterKey[0])
			if valueEnd < 0 {
				return "", nil, invalid
			}
			rawValue := afterKey[1 : valueEnd+1]
			if strings.ContainsAny(rawValue, "<>") {
				return "", nil, invalid
			}
			value = stdhtml.UnescapeString(rawValue)
			rest = afterKey[valueEnd+2:]
		} else if key != "expandable" || name != "blockquote" {
			return "", nil, invalid
		}
		attrs = append(attrs, html.Attribute{Key: key, Val: value})
	}
	return name, attrs, nil
}

func validateTelegramEntities(raw string) error {
	for {
		_, after, found := strings.Cut(raw, "&")
		if !found {
			return nil
		}
		entity, tail, terminated := strings.Cut(after, ";")
		if !terminated {
			return errors.New("escape literal ampersands in Telegram HTML")
		}
		switch entity {
		case "lt", "gt", "amp", "quot":
		default:
			if !strings.HasPrefix(entity, "#") {
				return errors.New("unsupported Telegram HTML entity")
			}
			number, base := entity[1:], 10
			if strings.HasPrefix(number, "x") || strings.HasPrefix(number, "X") {
				number, base = number[1:], 16
			}
			for _, digit := range number {
				if digit >= '0' && digit <= '9' || base == 16 && (digit >= 'a' && digit <= 'f' || digit >= 'A' && digit <= 'F') {
					continue
				}
				return errors.New("invalid Telegram HTML entity")
			}
			code, err := strconv.ParseInt(number, base, 32)
			if err != nil || code <= 0 || !utf8.ValidRune(rune(code)) {
				return errors.New("invalid Telegram HTML entity")
			}
		}
		raw = tail
	}
}
