package uklam

import (
	"github.com/eaciit/toolkit"
	"sync"
	"time"
)

type IFSWalker interface {
}

type WalkerStatusEnum int

const (
	WalkerIdle    WalkerStatusEnum = 0
	WalkerRunning                  = 1
	WalkerStop                     = 2
)

type FSWalker struct {
	sync.RWMutex

	Setting         *toolkit.M
	Path            string
	RefreshDuration time.Duration
	CheckFn         func(toolkit.M)*toolkit.Result
	WalkFn          func(toolkit.M)*toolkit.Result
	Status          WalkerStatusEnum
    
    chanCommand chan toolkit.M

	log *toolkit.LogEngine
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
	fs.Path = path
	fs.RefreshDuration = DefaultRefreshDuration()
	fs.log, _ = toolkit.NewLog(true, false, "", "", "")
    fs.chanCommand = make(chan toolkit.M)
	return fs
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

func (fs *FSWalker) Start() {
    go func(){
        for{
            select{
                case m := <-fs.chanCommand:
                    fs.processCommand(m)
                    if m.GetString("command")=="stop"{
                        return
                    }
                    
                case <-time.After(fs.RefreshDuration):
                    r:=fs.CheckFn(nil)
                    if r.Status!=toolkit.Status_OK{
                        fs.log.Error("Check Fail: " + r.Message)
                    }
                    fs.chanCommand <- toolkit.M{}.Set("command","walk").Set("data",r.Data)
                    
                default:
                    // do nothing
            }
        }
    }()
}

func (fs *FSWalker) processCommand(m toolkit.M){
    cmd := m.GetString("command")
    if cmd=="stop"{
        if fs.log!=nil{
            fs.log.Close()
        }
        return
    }
    
    if cmd=="walk"{
        if fs.WalkFn!=nil{
            r:=fs.WalkFn(m.Get("list",toolkit.M{}).(toolkit.M))
            if r.Status!=toolkit.Status_OK{
                fs.log.Error("Walking Fail: " + r.Message)
            }
        }
    }
}

func (fs *FSWalker) Stop() {
	fs.chanCommand <- toolkit.M{}.Set("command","stop")
}
