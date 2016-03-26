package test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eaciit/dbox"
	_ "github.com/eaciit/dbox/dbc/mongo"
	"github.com/eaciit/orm"
	"github.com/eaciit/toolkit"
	"github.com/eaciit/uklam"
)

var conn dbox.IConnection
var db *orm.DataContext

var path = "/users/ariefdarmawan/Dropbox/pvt/Temp/bhesada"

func TestWalk(t *testing.T) {
	toolkit.Println("OK")

	w := uklam.NewFS(filepath.Join(path, "inbox"))
	w.EachFn = func(dw uklam.IDataWalker, in toolkit.M, info os.FileInfo, r *toolkit.Result) {
		sourcename := filepath.Join(path, "inbox", info.Name())
		dstname := filepath.Join(path, "running", info.Name())
		toolkit.Printf("Processing %s...", sourcename)
		e := uklam.FSCopy(sourcename, dstname, true)
		if e != nil {
			toolkit.Println(e.Error())
		} else {
			toolkit.Println("OK")
		}
	}
	w.Start()
	defer w.Stop()

	conn, _ := dbox.NewConnection("mongo", &dbox.ConnectionInfo{"localhost:27123", "ecwfmdemo", "", "", nil})
	conn.Connect()
	db = orm.New(conn)
	defer db.Close()

	w2 := uklam.NewFS(filepath.Join(path, "running"))
	w2.EachFn = func(dw uklam.IDataWalker, in toolkit.M, info os.FileInfo, r *toolkit.Result) {
		sourcename := filepath.Join(path, "running", info.Name())
		dstnameOK := filepath.Join(path, "success", info.Name())
		dstnameNOK := filepath.Join(path, "fail", info.Name())
		toolkit.Printf("Processing %s...", sourcename)
		e := streamit(sourcename)
		if e == nil {
			uklam.FSCopy(sourcename, dstnameOK, true)
			toolkit.Println("OK")
		} else {
			uklam.FSCopy(sourcename, dstnameNOK, true)
			toolkit.Println("NOK " + e.Error())
		}
	}
	w2.Start()
	defer w2.Stop()

	time.Sleep(20 * time.Second)
}

type Scada struct {
    orm.ModelBase `bson:"-" json:"-"`
    ID string `bson:"_id"`
	Timestamp                                                         time.Time
	Turbine                                                           string
	Speed, Direction, Nacel, Temp, FailureTime, ConnectTime, FullTime float32
	Power                                                             float64
}

func (s *Scada) TableName() string{
    return "scadas"
}

func (s *Scada) RecordID() interface{}{
    return s.ID
}

func (s *Scada) PreSave()error{
    s.ID = toolkit.Sprintf("%s-%s", s.Turbine, toolkit.Date2String(s.Timestamp, "YYYYMMddHHmmss"))
    return nil
}

func streamit(src string) error {
	f, _ := os.Open(src)
	defer f.Close()

	b := bufio.NewScanner(f)
	b.Split(bufio.ScanLines)

	i := 0
	for b.Scan() {
		if i > 0 {
			str := strings.Split(b.Text(), ",")
            scada := new(Scada)
			scada.Timestamp = toolkit.String2Date(str[0], "YYYYMMddHHmmss")
			scada.Turbine = str[1][len(str[1])-6:]
			scada.Speed = toolkit.ToFloat32(str[2], 2, toolkit.RoundingAuto)
			scada.Direction = toolkit.ToFloat32(str[3], 2, toolkit.RoundingAuto)
			if scada.Direction < 0 {
				scada.Direction = 360 + scada.Direction
			} else if scada.Direction >= 360 {
                scada.Direction = scada.Direction - 360
            }
			scada.Nacel = toolkit.ToFloat32(str[4], 2, toolkit.RoundingAuto)
			scada.FailureTime = toolkit.ToFloat32(str[6], 2, toolkit.RoundingAuto) / float32(60)
			scada.ConnectTime = toolkit.ToFloat32(str[7], 2, toolkit.RoundingAuto) / float32(60)
			scada.FullTime = scada.FailureTime + scada.ConnectTime
			scada.Power = toolkit.ToFloat64(str[5], 2, toolkit.RoundingAuto) * float64(scada.FullTime) / float64(60) / float64(1000)
			scada.Temp = toolkit.ToFloat32(str[8], 2, toolkit.RoundingAuto)
            db.Save(scada)
            toolkit.Println(toolkit.JsonString(scada))
		}
		i++
	}

	return nil
}
