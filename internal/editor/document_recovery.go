package editor

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/ui"
)

// documentRecovery 保存未提交到原文件的标签内容和磁盘版本
type documentRecovery struct {
	Path     string          `json:"path,omitempty"`
	Title    string          `json:"title,omitempty"`
	Untitled int             `json:"untitled"`
	ModTime  time.Time       `json:"modTime"`
	Size     int64           `json:"size"`
	Content  json.RawMessage `json:"label"`
}

// documentRecoveryDirectory 返回当前应用的标签恢复目录
func documentRecoveryDirectory() string {
	return filepath.Join(filepath.Dir(settingsFile()), "label-recovery")
}

// checkpoint 在编辑暂停后保存各标签的恢复副本
func (t *documentTabs) checkpoint(c *ui.Context) {
	now := time.Now()
	for _, e := range t.editors {
		due := e.document.recoveryDue
		if due.IsZero() {
			continue
		}
		if due.After(now) {
			c.After(due.Sub(now))
			continue
		}
		e.document.recoveryDue = time.Time{}
		if err := e.saveRecovery(); err != nil {
			e.status = err.Error()
		}
	}
}

// saveRecovery 持久化当前标签的未保存内容
func (e *editor) saveRecovery() error {
	content, err := json.Marshal(e.label)
	if err != nil {
		return err
	}
	d := e.document
	recovery := documentRecovery{Path: d.path, Title: e.label.title, Untitled: d.untitled, Content: content}
	if d.stamp != nil {
		recovery.ModTime, recovery.Size = d.stamp.ModTime(), d.stamp.Size()
	}
	dir := documentRecoveryDirectory()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if d.recoveryPath == "" {
		d.recoveryPath = filepath.Join(dir, rand.Text()+".json")
	}
	return writeDocumentFile(d.recoveryPath, func(w io.Writer) error { return json.NewEncoder(w).Encode(recovery) })
}

// removeRecovery 删除用户已经保存或确认放弃的恢复副本
func (e *editor) removeRecovery() {
	d := e.document
	d.recoveryDue = time.Time{}
	if d.recoveryPath != "" {
		if err := os.Remove(d.recoveryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Print(err)
		}
		d.recoveryPath = ""
	}
}

// restoreRecovery 在启动时恢复上次异常退出留下的未保存标签
func (t *documentTabs) restoreRecovery() {
	entries, err := os.ReadDir(documentRecoveryDirectory())
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		log.Print(err)
		return
	}
	initial := t.initial
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(documentRecoveryDirectory(), entry.Name())
		if err := t.restoreDocument(path); err != nil {
			log.Printf("GoLabel: %s: %v", path, err)
		}
	}
	if initial != nil && len(t.editors) > 1 {
		t.removeInitial(initial)
	}
}

// restoreDocument 读取一份恢复副本并标记与原文件的冲突
func (t *documentTabs) restoreDocument(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var recovery documentRecovery
	if err := json.Unmarshal(data, &recovery); err != nil {
		return err
	}
	l, err := readLabel(bytes.NewReader(recovery.Content))
	if err != nil {
		return err
	}
	l.title = recovery.Title
	e := t.newTab(l)
	d := e.document
	d.path, d.untitled, d.edited, d.recoveryPath = recovery.Path, recovery.Untitled, true, path
	t.nextUntitled = max(t.nextUntitled, d.untitled)
	if d.path != "" {
		d.stamp, err = os.Stat(d.path)
		d.deleted = errors.Is(err, os.ErrNotExist)
		if err == nil {
			d.conflict = !d.stamp.ModTime().Equal(recovery.ModTime) || d.stamp.Size() != recovery.Size
			e.startWatching()
		}
	}
	t.syncWindow()
	return nil
}
