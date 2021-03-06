package pcscommand

import (
	"fmt"
	"github.com/iikira/BaiduPCS-Go/pcstable"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	// BgMap 后台
	BgMap = BgTasks{
		tasks: sync.Map{},
		sig:   make(chan struct{}),
	}
)

type (
	// BgTasks 后台任务
	BgTasks struct {
		lastID  int64
		tasks   sync.Map
		started bool
		sig     chan struct{}
	}
	// BgDTaskItem 后台任务详情
	BgDTaskItem struct {
		id              int
		downloadOptions *DownloadOptions
		pcspaths        []string
		done            <-chan struct{}
	}
)

func (b *BgTasks) checkDoneTask() {
	b.tasks.Range(func(id, v interface{}) bool {
		task := v.(*BgDTaskItem)
		select {
		case <-task.done:
			fmt.Printf("任务：%d 已完成\n", id.(int64))
			b.tasks.Delete(id)
			return true
		default:
			return true
		}
	})
}

// NewID 返回生成的 ID
func (b *BgTasks) NewID() int64 {
	id := atomic.AddInt64(&b.lastID, 1)
	return id
}

// TaskID 返回后台任务 id
func (t *BgDTaskItem) TaskID() int {
	return t.id
}

// PrintAllBgTask 输出所有的后台任务
func (b *BgTasks) PrintAllBgTask() {
	tb := pcstable.NewTable(os.Stdout)
	tb.SetHeader([]string{"task_id", "files"})
	b.tasks.Range(func(id, v interface{}) bool {
		tb.Append([]string{strconv.FormatInt(id.(int64), 10), strings.Join(v.(*BgDTaskItem).pcspaths, ",")})
		return true
	})
	tb.Render()
}

// RunBgDownload 执行后台下载
func RunBgDownload(paths []string, options *DownloadOptions) {
	if !BgMap.started {
		go func() {
			for {
				select {
				case <-BgMap.sig:
					BgMap.checkDoneTask()
				}
			}
		}()
	} else {
		BgMap.started = true
	}

	if options.Out == nil {
		options.Out, _ = os.Open(os.DevNull)
	}

	task := new(BgDTaskItem)
	task.pcspaths = paths

	dchan := make(chan struct{})
	task.done = dchan

	BgMap.tasks.Store(BgMap.NewID(), task)

	go func(dchan chan struct{}) {
		RunDownload(paths, options)
		close(dchan)
		BgMap.sig <- struct{}{}
	}(dchan)
}
