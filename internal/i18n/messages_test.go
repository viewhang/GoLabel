package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMessagesComplete 验证代码里用到的文案 key 都在文案表里，且两种语言都不为空
func TestMessagesComplete(t *testing.T) {
	used := make(map[string]bool)
	sources := new(strings.Builder)
	pattern := regexp.MustCompile(`\bTrf?\("([^"]+)"`)
	for _, dir := range []string{"../editor", "../platform", "../printing"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range files {
			data, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			sources.Write(data)
			for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
				used[match[1]] = true
			}
		}
	}
	for key := range used {
		// 以点结尾的 key 是拼接用的前缀，不是完整文案
		if strings.HasSuffix(key, ".") {
			continue
		}
		if _, ok := messages[key]; !ok {
			t.Errorf("文案表缺少 key %q", key)
		}
	}
	source := sources.String()
	for key := range messages {
		if used[key] {
			continue
		}
		// 模板分类的 key 是拼出来的，按前缀判断
		if strings.HasPrefix(key, "template.category.") && strings.Contains(source, `"template.category.`) {
			continue
		}
		if !strings.Contains(source, `"`+key+`"`) {
			t.Errorf("文案 %q 没有被使用", key)
		}
	}
	for key, message := range messages {
		if strings.TrimSpace(message.ZH) == "" || strings.TrimSpace(message.EN) == "" {
			t.Errorf("文案 %q 缺少中文或英文", key)
		}
	}
}
