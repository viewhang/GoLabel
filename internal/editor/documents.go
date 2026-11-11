package editor

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/pengh/GoLabel/internal/i18n"
)

// documentFile 保存标签对应文件的路径、磁盘版本和未保存状态
type documentFile struct {
	path            string
	stamp           os.FileInfo
	edited, deleted bool
	conflict        bool
	untitled        int
	watcher         *mygo.FileWatcher
	recoveryPath    string
	recoveryDue     time.Time
}

// askDocumentChanges 显示文件修改和关闭标签时的原生确认弹框
var askDocumentChanges = mygo.Dialog.Message

// chooseLabelPath 显示标签文件的原生另存为弹框
var chooseLabelPath = mygo.Dialog.Save

// labelFileFilter 返回当前语言的标签文件筛选条件
func labelFileFilter() []mygo.FileFilter {
	return []mygo.FileFilter{{Name: i18n.Tr("app.templateType"), Extensions: []string{"golabel"}}}
}

// openDialog 将用户选择的文件依次打开为标签页
func (t *documentTabs) openDialog() error {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: t.window, Title: i18n.Tr("header.open"), Filters: labelFileFilter(), Multiple: true,
	})
	if err != nil {
		return err
	}
	for _, path := range paths {
		if err := t.open(path); err != nil {
			return err
		}
	}
	return nil
}

// open 读取标签文件并复用已经打开的同一文件标签
func (t *documentTabs) open(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	for _, e := range t.editors {
		if e.document.path == path || (e.document.stamp != nil && os.SameFile(e.document.stamp, info)) {
			t.selectTab(e)
			return nil
		}
	}
	l, err := readLabel(file)
	if err != nil {
		return err
	}
	initial := t.initial
	e := t.newTab(l)
	if initial != nil {
		t.removeInitial(initial)
	}
	e.document.path, e.document.stamp = path, info
	e.startWatching()
	t.recordRecent(path)
	t.syncWindow()
	return nil
}

// recordRecent 更新应用和系统的最近文件菜单
func (t *documentTabs) recordRecent(path string) {
	if t.window != nil {
		if err := mygo.App.AddRecentDocument(path); err != nil {
			log.Print(err)
		}
		t.installMenu()
	}
}

// save 保存当前标签，未命名或只读文件使用另存为
func (e *editor) save() error {
	e.flushTextEdit()
	path := e.document.path
	if e.document.stamp != nil && e.document.stamp.Mode().Perm()&0o222 == 0 {
		path = ""
	}
	return e.saveTo(path)
}

// saveAs 让用户选择当前标签的新保存路径
func (e *editor) saveAs() error {
	e.flushTextEdit()
	return e.saveTo("")
}

// saveTo 确认磁盘冲突并将标签完整写入指定文件
func (e *editor) saveTo(path string) error {
	if path == "" {
		var err error
		defaultPath := e.document.path
		if defaultPath == "" {
			defaultPath = e.documentName() + ".golabel"
		}
		path, err = chooseLabelPath(mygo.SaveDialogOptions{
			Parent: e.window, Title: i18n.Tr("header.save"), DefaultPath: defaultPath, Filters: labelFileFilter(),
		})
		if err != nil {
			return err
		}
		if path == "" {
			return mygo.ErrCanceled
		}
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, statErr := os.Stat(path)
	for _, other := range e.tabs.editors {
		if other != e && (other.document.path == path ||
			(statErr == nil && other.document.stamp != nil && os.SameFile(other.document.stamp, info))) {
			return errors.New(i18n.Tr("document.alreadyOpen"))
		}
	}
	if path == e.document.path && (e.document.conflict ||
		(statErr == nil && e.document.stamp != nil && !sameFileVersion(info, e.document.stamp))) {
		result, err := askDocumentChanges(mygo.MessageOptions{
			Parent: e.window, Type: mygo.MessageWarning, Title: "GoLabel",
			Message: i18n.Trf("document.overwrite", e.documentName()),
			Buttons: []string{i18n.Tr("document.saveAnyway"), i18n.Tr("new.cancel")}, CancelButton: 1,
		})
		if err != nil {
			return err
		}
		if result.Button != 0 {
			return mygo.ErrCanceled
		}
	}
	if err := writeDocumentFile(path, func(w io.Writer) error { return writeLabel(w, e.label) }); err != nil {
		return err
	}
	info, err = os.Stat(path)
	if err != nil {
		return err
	}
	e.document.path, e.document.stamp = path, info
	e.document.edited, e.document.deleted, e.document.conflict = false, false, false
	e.removeRecovery()
	e.startWatching()
	e.tabs.recordRecent(path)
	e.tabs.syncWindow()
	return nil
}

// confirmClose 在保存、放弃和取消之间确认标签的未保存修改
func (t *documentTabs) confirmClose(e *editor) bool {
	e.flushTextEdit()
	if !e.documentEdited() {
		return true
	}
	t.selectTab(e)
	result, err := askDocumentChanges(mygo.MessageOptions{
		Parent: t.window, Type: mygo.MessageQuestion, Title: "GoLabel",
		Message:      i18n.Trf("document.saveChanges", e.documentName()),
		Buttons:      []string{i18n.Tr("header.save"), i18n.Tr("document.discard"), i18n.Tr("new.cancel")},
		CancelButton: 2, DestructiveButtons: []int{1},
	})
	if err != nil {
		e.status = err.Error()
		return false
	}
	if result.Button == 0 {
		if err := e.save(); err != nil {
			if !errors.Is(err, mygo.ErrCanceled) {
				e.status = err.Error()
			}
			return false
		}
		return true
	}
	return result.Button == 1
}

// closeWindow 确认全部标签后释放文件监听和恢复副本
func (t *documentTabs) closeWindow(event *mygo.CloseEvent) {
	if t.closing {
		event.PreventDefault()
		return
	}
	t.closing = true
	defer func() { t.closing = false }()
	for _, e := range t.editors {
		if !t.confirmClose(e) {
			event.PreventDefault()
			return
		}
	}
	for _, e := range t.editors {
		e.stopWatching()
		e.removeRecovery()
	}
}

// sameFileVersion 比较文件身份、大小和修改时间
func sameFileVersion(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

// checkFile 检查外部修改并保留当前尚未保存的内容
func (e *editor) checkFile() {
	e.flushTextEdit()
	d := e.document
	info, err := os.Stat(d.path)
	if errors.Is(err, os.ErrNotExist) {
		if d.stamp != nil {
			entries, _ := os.ReadDir(filepath.Dir(d.path))
			for _, entry := range entries {
				candidate, err := entry.Info()
				if err == nil && os.SameFile(d.stamp, candidate) {
					d.path = filepath.Join(filepath.Dir(d.path), entry.Name())
					e.startWatching()
					e.tabs.recordRecent(d.path)
					e.checkFile()
					return
				}
			}
		}
		d.deleted = true
	} else if err != nil {
		e.status = err.Error()
	} else {
		d.deleted = false
		if d.stamp == nil || !sameFileVersion(info, d.stamp) {
			if d.edited {
				d.conflict = true
			} else {
				file, err := os.Open(d.path)
				if err == nil {
					defer file.Close()
					var l *label
					l, err = readLabel(file)
					if err == nil {
						e.loadLabel(l)
						d.stamp, d.conflict = info, false
					}
				}
				if err != nil {
					e.status = err.Error()
				}
			}
		}
	}
	e.tabs.syncWindow()
}

// startWatching 监听已保存标签的外部文件变更
func (e *editor) startWatching() {
	e.stopWatching()
	if e.window == nil {
		return
	}
	watcher, err := mygo.WatchFiles(context.Background(), e.document.path, mygo.FileWatchOptions{})
	if err != nil {
		e.status = err.Error()
		return
	}
	e.document.watcher = watcher
	go func() {
		for {
			if _, err := watcher.Next(context.Background()); err != nil {
				return
			}
			e.window.Update(func() {
				if e.document.watcher == watcher && slices.Contains(e.tabs.editors, e) {
					e.checkFile()
				}
			})
		}
	}()
}

// stopWatching 停止当前标签的文件监听
func (e *editor) stopWatching() {
	if e.document.watcher != nil {
		e.document.watcher.Close()
		e.document.watcher = nil
	}
}

// writeDocumentFile 将完整内容写入同目录临时文件后替换目标文件
func writeDocumentFile(path string, write func(io.Writer) error) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".golabel-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if info, err := os.Stat(path); err == nil {
		if err := file.Chmod(info.Mode().Perm()); err != nil {
			file.Close()
			return err
		}
	}
	err = write(file)
	if err == nil {
		err = file.Sync()
	}
	if err := errors.Join(err, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
