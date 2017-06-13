package uklam

import (
	"io"
	"os"

	"github.com/eaciit/toolkit"
)

/*FSCopy copy file from source to dst*/
func FSCopy(source, dst string, ismove bool) error {
	fsource, e := os.Open(source)
	if e != nil {
		return toolkit.Errorf("FSCopy: %s %s", source, e.Error())
	}
	defer fsource.Close()

	fdst, e := os.Create(dst)
	if e != nil {
		os.Remove(dst)
		fdst, e = os.Create(dst)
	}
	if e != nil {
		return toolkit.Errorf("FSCopy: %s %s", dst, e.Error())
	}
	defer fdst.Close()

	_, e = io.Copy(fdst, fsource)
	if e != nil {
		return toolkit.Errorf("FSCopy: Fail to copy %s", e.Error())
	}
	e = fdst.Sync()
	if e != nil {
		return toolkit.Errorf("FSCopy: Fail to sync %s", e.Error())
	}

	if ismove {
		os.Remove(source)
	}
	return nil
}

func FSWalkFn(dw IDataWalker, in toolkit.M) *toolkit.Result {
	r := toolkit.NewResult()
	infos := in.Get("data", []os.FileInfo{}).([]os.FileInfo)
	for _, info := range infos {
		dw.(*FSWalker).EachFn(dw, in, info, r)
	}
	return r
}
