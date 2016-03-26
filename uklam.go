package uklam

import (
	"github.com/eaciit/toolkit"
	"sync"
	"time"
    "os"
    "io/ioutil"
)

type IDataWalker interface {
    SetHost(string)
    Host() string
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
	RefreshDuration time.Duration
	CheckFn         func(IDataWalker, toolkit.M) *toolkit.Result
	WalkFn          func(IDataWalker, toolkit.M) *toolkit.Result
    EachFn          func(IDataWalker, toolkit.M, os.FileInfo, *toolkit.Result)
    Status          WalkerStatusEnum

	chanCommand chan toolkit.M

	_host string
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
	fs._host = path
	fs.RefreshDuration = DefaultRefreshDuration()
	fs.log, _ = toolkit.NewLog(true, false, "", "", "")
	fs.chanCommand = make(chan toolkit.M)
	return fs
}

func (fs *FSWalker) SetHost(h string){
    fs._host = h
}

func (fs *FSWalker) Host() string{
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

func checkFile(dw IDataWalker, in toolkit.M)*toolkit.Result{
    r := toolkit.NewResult()
    infos, e := ioutil.ReadDir(dw.Host())
    if e!=nil {
        return r.SetError(e)
    }
    //toolkit.Println("Files: ", len(infos))
    r.Data = infos
    return r
}

func (fs *FSWalker) Start() {
    if fs.CheckFn==nil {
        fs.CheckFn = checkFile
    }
    
    if fs.WalkFn==nil {
        fs.WalkFn = FSWalkFn
    }
    
    go func() {
		for {
			select {
			case m := <-fs.chanCommand:
				fs.processCommand(m)
				if m.GetString("command") == "stop" {
                    for fs.Status!=WalkerIdle{
                         time.Sleep(1*time.Millisecond)       
                    }
					return
				}

			default:
				// do nothing
			}
		}
	}()
    
	go func() {
		for {
			select {
			case <-time.After(fs.RefreshDuration):
				//--- check should only run when status is idle
                if fs.Status==WalkerIdle{
                    r := fs.CheckFn(fs, nil)
                    if r.Status != toolkit.Status_OK {
                        fs.log.Error("Check Fail: " + r.Message)
                    }
                    fs.chanCommand <- toolkit.M{}.Set("command", "walk").Set("data", r.Data)
                }

			//default:
				// do nothing
			}
		}
	}()
}

func (fs *FSWalker) processCommand(m toolkit.M) {
	cmd := m.GetString("command")
	if cmd == "stop" {
		if fs.log != nil {
			fs.log.Close()
		}
		return
	}

	if cmd == "walk" {
        //-- protect
        if fs.Status==WalkerRunning{
            return
        }
        
        //toolkit.Println("Walker Run")
        fs.Status = WalkerRunning
		if fs.WalkFn != nil {
			r := fs.WalkFn(fs, m)
			if r.Status != toolkit.Status_OK {
				fs.log.Error("Walking Fail: " + r.Message)
			}
		}
        fs.Status = WalkerIdle
	}
}

func (fs *FSWalker) Stop() {
	fs.chanCommand <- toolkit.M{}.Set("command", "stop")
}
