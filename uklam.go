package uklam

import (
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/eaciit/toolkit"
)

type IDataWalker interface {
	SetHost(string)
	Host() string
}

type WalkerStatusEnum int

const (
	WalkerIdle        WalkerStatusEnum = 0
	WalkerRunning                      = 1
	WalkerRunningDone                  = 10
	WalkerStop                         = 100
)

type FSWalker struct {
	sync.RWMutex

	Setting         *toolkit.M
	RefreshDuration time.Duration
	CheckFn         func(IDataWalker, toolkit.M) *toolkit.Result
	WalkFn          func(IDataWalker, toolkit.M) *toolkit.Result
	EachFn          func(IDataWalker, toolkit.M, os.FileInfo, *toolkit.Result)
	Status          WalkerStatusEnum

	chanCommand chan toolkit.M

	_host string
	log   *toolkit.LogEngine
}

var _defaultRefreshDuration time.Duration

func DefaultRefreshDuration() time.Duration {
	if _defaultRefreshDuration == 0 {
		_defaultRefreshDuration = 1 * time.Millisecond
	}
	return _defaultRefreshDuration
}

func SetDefaultRefreshDuration(t time.Duration) {
	_defaultRefreshDuration = t
}

func NewFS(path string) *FSWalker {
	fs := new(FSWalker)
	fs._host = path
	fs.RefreshDuration = DefaultRefreshDuration()
	fs.log, _ = toolkit.NewLog(true, false, "", "", "")
	fs.chanCommand = make(chan toolkit.M)
	return fs
}

func (fs *FSWalker) SetHost(h string) {
	fs._host = h
}

func (fs *FSWalker) Host() string {
	return fs._host
}

func (fs *FSWalker) Log() *toolkit.LogEngine {
	if fs.log == nil {
		fs.log, _ = toolkit.NewLog(true, false, "", "", "")
	}
	return fs.log
}

func (fs *FSWalker) SetLog(l *toolkit.LogEngine) {
	if fs.log != nil {
		fs.log.Close()
	}
	fs.log = l
}

func checkFile(dw IDataWalker, in toolkit.M) *toolkit.Result {
	r := toolkit.NewResult()
	infos, e := ioutil.ReadDir(dw.Host())
	if e != nil {
		return r.SetError(e)
	}
	//toolkit.Println("Files: ", len(infos))
	r.Data = infos
	return r
}

func (fs *FSWalker) Start() {
	if fs.CheckFn == nil {
		fs.CheckFn = checkFile
	}

	if fs.WalkFn == nil {
		fs.WalkFn = FSWalkFn
	}

	ticker := time.NewTicker(fs.RefreshDuration)
	go func() {
		for {
			select {
			case m := <-fs.chanCommand:
				cmd := m.GetString("command")
				if cmd == "stop" {
					ticker.Stop()
				}

			case <-ticker.C:
				if fs.Status == WalkerIdle {
					r := fs.CheckFn(fs, nil)
					if r.Status != toolkit.Status_OK {
						fs.log.Error("Check Fail: " + r.Message)
					}

					fs.Walk()
				}

				if fs.Status == WalkerRunningDone {
					fs.SetIdle()
				}
			}
		}
	}()
}

func (fs *FSWalker) NewData() bool {
	if fs.Status == WalkerRunning {
		return false
	}

	newData := false
	fs.Lock()

	fs.Unlock()
	return newData
}

func (fs *FSWalker) Walk() error {
	if fs.Status != WalkerIdle {
		return nil
	}

	fs.Lock()
	fs.Status = WalkerRunning
	fs.Unlock()

	if fs.WalkFn != nil {
		r := fs.CheckFn(fs, nil)
		if r.Status != toolkit.Status_OK {
			fs.log.Error("Check Fail: " + r.Message)
			return toolkit.Errorf("Check fail: %s", r.Message)
		}
		r = fs.WalkFn(fs, toolkit.M{}.Set("data", r.Data))
		if r.Status != toolkit.Status_OK {
			fs.log.Error("Walking Fail: " + r.Message)
		}
	}

	fs.Lock()
	fs.Status = WalkerRunningDone
	fs.Unlock()

	return nil
}

func (fs *FSWalker) SetIdle() error {
	fs.Lock()
	fs.Status = WalkerIdle
	fs.Unlock()
	return nil
}

func (fs *FSWalker) Stop() {
	fs.chanCommand <- toolkit.M{}.Set("command", "stop")
}
