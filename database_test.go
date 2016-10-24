package main
import (
    "testing"
    "strconv"
    "math/rand"
    "os"
    "time"
    "bytes"
    "regexp"
    "bufio"
    "errors"
)
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomString(n int) string {
    rand.Seed(time.Now().UnixNano())
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}
func randomInteger() int {
    rand.Seed(time.Now().UnixNano())
    x := rand.Intn(10000) + 100
    if x == 0 {
        return randomInteger();
    }
    return x + 100
}
func randomFloat() float32 {
    rand.Seed(time.Now().UnixNano())
    return rand.Float32() * 100
}
func randomDateTime(a Adapter) *DateTime {
    rand.Seed(time.Now().UnixNano())
    d := NewDateTime(a)
    d.Year = rand.Intn(2017)
    d.Month = rand.Intn(11)
    d.Day = rand.Intn(28)
    d.Hours = rand.Intn(23)
    d.Minutes = rand.Intn(59)
    d.Seconds = rand.Intn(56)
    if d.Year < 1000 {
        d.Year = d.Year + 1000
    }
    return d
}


func TestNewNote(t *testing.T) {
    a := NewMysqlAdapter(``)
    o := NewNote(a)
    if o._table != "notes" {
        t.Errorf("failed creating %+v",o);
        return
    }
}
func TestNoteFromDBValueMap(t *testing.T) {
    a := NewMysqlAdapter(``)
    o := NewNote(a)
    m := make(map[string]DBValue)
	m["id"] = a.NewDBValue()
	m["id"].SetInternalValue("id",strconv.Itoa(999))
	m["value"] = a.NewDBValue()
	m["value"].SetInternalValue("value","AString")
	m["portfolio_id"] = a.NewDBValue()
	m["portfolio_id"].SetInternalValue("portfolio_id",strconv.Itoa(999))
	m["position_id"] = a.NewDBValue()
	m["position_id"].SetInternalValue("position_id",strconv.Itoa(999))

    err := o.FromDBValueMap(m)
    if err != nil {
        t.Errorf("FromDBValueMap failed %s",err)
    }

    if o.Id != 999 {
        t.Errorf("o.Id test failed %+v",o)
        return
    }    

    if o.Value != "AString" {
        t.Errorf("o.Value test failed %+v",o)
        return
    }    

    if o.PortfolioId != 999 {
        t.Errorf("o.PortfolioId test failed %+v",o)
        return
    }    

    if o.PositionId != 999 {
        t.Errorf("o.PositionId test failed %+v",o)
        return
    }    
}

func TestNoteCreate(t *testing.T) {
    if fileExists(`../gopaper-testing.db.yml`) {
    a,err := NewMysqlAdapterEx(`../gopaper-testing.db.yml`)
    defer a.Close()
    if err != nil {
        t.Errorf(`could not load ../gopaper-testing.db.yml %s`,err)
        return
    }
    file, err := os.OpenFile("adapter.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        t.Errorf(" Failed to open log file %s", err)
    }
    a.SetLogs(file)
    model := NewNote(a)
model.Value = randomString(25)
model.PortfolioId = int64(randomInteger())
model.PositionId = int64(randomInteger())

    err = model.Create()
    if err != nil {
        t.Errorf(` failed to create model %s`,err)
        return
    }

    model2 := NewNote(a)
    found,err := model2.Find(model.GetPrimaryKeyValue())
    if err != nil {
        t.Errorf(` did not find record for %s = %d because of %s`,model.GetPrimaryKeyName(),model.GetPrimaryKeyValue(),err)
        return
    }
    if found == false {
        t.Errorf(` did not find record for %s = %d because of %s`,model.GetPrimaryKeyName(),model.GetPrimaryKeyValue(),err)
        return
    }


    if model.Value != model2.Value {
        t.Errorf(` model.Value[%s] != model2.Value[%s]`,model.Value,model2.Value)
        return
    }

    if model.PortfolioId != model2.PortfolioId {
        t.Errorf(` model.PortfolioId[%d] != model2.PortfolioId[%d]`,model.PortfolioId,model2.PortfolioId)
        return
    }

    if model.PositionId != model2.PositionId {
        t.Errorf(` model.PositionId[%d] != model2.PositionId[%d]`,model.PositionId,model2.PositionId)
        return
    }
model2.SetValue(randomString(25))
model2.SetPortfolioId(int64(randomInteger()))
model2.SetPositionId(int64(randomInteger()))

    err = model2.Save()
    if err != nil {
        t.Errorf(`failed to save model2 %s`,err)
    }

    if model.Value == model2.Value {
        t.Errorf(`1: model.Value[%s] != model2.Value[%s]`,model.Value,model2.Value)
        return
    }

    if model.PortfolioId == model2.PortfolioId {
        t.Errorf(`1: model.PortfolioId[%d] != model2.PortfolioId[%d]`,model.PortfolioId,model2.PortfolioId)
        return
    }

    if model.PositionId == model2.PositionId {
        t.Errorf(`1: model.PositionId[%d] != model2.PositionId[%d]`,model.PositionId,model2.PositionId)
        return
    }

    res9,err := model.FindByValue(model2.GetValue())
    if err != nil {
        t.Errorf(`failed model.FindByValue(model2.GetValue())`)
    }
    if len(res9) == 0 {
        t.Errorf(`failed to find any Note`)
    }

    res10,err := model.FindByPortfolioId(model2.GetPortfolioId())
    if err != nil {
        t.Errorf(`failed model.FindByPortfolioId(model2.GetPortfolioId())`)
    }
    if len(res10) == 0 {
        t.Errorf(`failed to find any Note`)
    }

    res11,err := model.FindByPositionId(model2.GetPositionId())
    if err != nil {
        t.Errorf(`failed model.FindByPositionId(model2.GetPositionId())`)
    }
    if len(res11) == 0 {
        t.Errorf(`failed to find any Note`)
    }
} // end of if fileExists
};


func TestNoteUpdaters(t *testing.T) {
    if fileExists(`../gopaper-testing.db.yml`) == false {
        return
    }
    a,err := NewMysqlAdapterEx(`../gopaper-testing.db.yml`)
    defer a.Close()
    if err != nil {
        t.Errorf(`could not load ../gopaper-testing.db.yml %s`,err)
        return
    }
    file, err := os.OpenFile("adapter.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        t.Errorf("Failed to open log file %s", err)
        return
    }
    a.SetLogs(file)
    model := NewNote(a)

    model.SetValue(randomString(25))
    if model.GetValue() != model.Value {
        t.Errorf(`Note.GetValue() != Note.Value`)
    }
    if model.IsValueDirty != true {
        t.Errorf(`Note.IsValueDirty != true`)
        return
    }
    
    u0 := randomString(25)
    _,err = model.UpdateValue(u0)
    if err != nil {
        t.Errorf(`failed UpdateValue(u0) %s`,err)
        return
    }

    if model.GetValue() != u0 {
        t.Errorf(`Note.GetValue() != u0 after UpdateValue`)
        return
    }
    model.Reload()
    if model.GetValue() != u0 {
        t.Errorf(`Note.GetValue() != u0 after Reload`)
        return
    }

    model.SetPortfolioId(int64(randomInteger()))
    if model.GetPortfolioId() != model.PortfolioId {
        t.Errorf(`Note.GetPortfolioId() != Note.PortfolioId`)
    }
    if model.IsPortfolioIdDirty != true {
        t.Errorf(`Note.IsPortfolioIdDirty != true`)
        return
    }
    
    u1 := int64(randomInteger())
    _,err = model.UpdatePortfolioId(u1)
    if err != nil {
        t.Errorf(`failed UpdatePortfolioId(u1) %s`,err)
        return
    }

    if model.GetPortfolioId() != u1 {
        t.Errorf(`Note.GetPortfolioId() != u1 after UpdatePortfolioId`)
        return
    }
    model.Reload()
    if model.GetPortfolioId() != u1 {
        t.Errorf(`Note.GetPortfolioId() != u1 after Reload`)
        return
    }

    model.SetPositionId(int64(randomInteger()))
    if model.GetPositionId() != model.PositionId {
        t.Errorf(`Note.GetPositionId() != Note.PositionId`)
    }
    if model.IsPositionIdDirty != true {
        t.Errorf(`Note.IsPositionIdDirty != true`)
        return
    }
    
    u2 := int64(randomInteger())
    _,err = model.UpdatePositionId(u2)
    if err != nil {
        t.Errorf(`failed UpdatePositionId(u2) %s`,err)
        return
    }

    if model.GetPositionId() != u2 {
        t.Errorf(`Note.GetPositionId() != u2 after UpdatePositionId`)
        return
    }
    model.Reload()
    if model.GetPositionId() != u2 {
        t.Errorf(`Note.GetPositionId() != u2 after Reload`)
        return
    }

};


func TestNewPlay(t *testing.T) {
    a := NewMysqlAdapter(``)
    o := NewPlay(a)
    if o._table != "plays" {
        t.Errorf("failed creating %+v",o);
        return
    }
}
func TestPlayFromDBValueMap(t *testing.T) {
    a := NewMysqlAdapter(``)
    o := NewPlay(a)
    m := make(map[string]DBValue)
	m["id"] = a.NewDBValue()
	m["id"].SetInternalValue("id",strconv.Itoa(999))
	m["position_id"] = a.NewDBValue()
	m["position_id"].SetInternalValue("position_id",strconv.Itoa(999))
	m["day"] = a.NewDBValue()
	m["day"].SetInternalValue("day","2016-01-01 10:50:23")
	m["open"] = a.NewDBValue()
	m["open"].SetInternalValue("open",strconv.Itoa(999))
	m["high"] = a.NewDBValue()
	m["high"].SetInternalValue("high",strconv.Itoa(999))
	m["low"] = a.NewDBValue()
	m["low"].SetInternalValue("low",strconv.Itoa(999))
	m["pvolume"] = a.NewDBValue()
	m["pvolume"].SetInternalValue("pvolume",strconv.Itoa(999))
	m["pchange"] = a.NewDBValue()
	m["pchange"].SetInternalValue("pchange",strconv.Itoa(999))
	m["pchange_percent"] = a.NewDBValue()
	m["pchange_percent"].SetInternalValue("pchange_percent",strconv.Itoa(999))
	m["adj_close"] = a.NewDBValue()
	m["adj_close"].SetInternalValue("adj_close",strconv.Itoa(999))
	m["data_source"] = a.NewDBValue()
	m["data_source"].SetInternalValue("data_source","AString")

    err := o.FromDBValueMap(m)
    if err != nil {
        t.Errorf("FromDBValueMap failed %s",err)
    }

    if o.Id != 999 {
        t.Errorf("o.Id test failed %+v",o)
        return
    }    

    if o.PositionId != 999 {
        t.Errorf("o.PositionId test failed %+v",o)
        return
    }    

    if o.Day.Year != 2016 {
        t.Errorf("year not set for %+v",o.Day)
        return
    }
    if (o.Day.Year != 2016 || 
        o.Day.Month != 1 ||
        o.Day.Day != 1 ||
        o.Day.Hours != 10 ||
        o.Day.Minutes != 50 ||
        o.Day.Seconds != 23 ) {
        t.Errorf(`fields don't match up for %+v`,o.Day)
    }
    r2,_ := m["day"].AsString()
    if o.Day.ToString() != r2 {
        t.Errorf(`restring of o.Day failed %s`,o.Day.ToString())
    }

    if o.Open != 999 {
        t.Errorf("o.Open test failed %+v",o)
        return
    }    

    if o.High != 999 {
        t.Errorf("o.High test failed %+v",o)
        return
    }    

    if o.Low != 999 {
        t.Errorf("o.Low test failed %+v",o)
        return
    }    

    if o.Pvolume != 999 {
        t.Errorf("o.Pvolume test failed %+v",o)
        return
    }    

    if o.Pchange != 999 {
        t.Errorf("o.Pchange test failed %+v",o)
        return
    }    

    if o.PchangePercent != 999 {
        t.Errorf("o.PchangePercent test failed %+v",o)
        return
    }    

    if o.AdjClose != 999 {
        t.Errorf("o.AdjClose test failed %+v",o)
        return
    }    

    if o.DataSource != "AString" {
        t.Errorf("o.DataSource test failed %+v",o)
        return
    }    
}

func TestPlayCreate(t *testing.T) {
    if fileExists(`../gopaper-testing.db.yml`) {
    a,err := NewMysqlAdapterEx(`../gopaper-testing.db.yml`)
    defer a.Close()
    if err != nil {
        t.Errorf(`could not load ../gopaper-testing.db.yml %s`,err)
        return
    }
    file, err := os.OpenFile("adapter.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        t.Errorf(" Failed to open log file %s", err)
    }
    a.SetLogs(file)
    model := NewPlay(a)
model.PositionId = int64(randomInteger())
model.Day = randomDateTime(a)
model.Open = int(randomInteger())
model.High = int(randomInteger())
model.Low = int(randomInteger())
model.Pvolume = int(randomInteger())
model.Pchange = int(randomInteger())
model.PchangePercent = int(randomInteger())
model.AdjClose = int(randomInteger())
model.DataSource = randomString(19)

    err = model.Create()
    if err != nil {
        t.Errorf(` failed to create model %s`,err)
        return
    }

    model2 := NewPlay(a)
    found,err := model2.Find(model.GetPrimaryKeyValue())
    if err != nil {
        t.Errorf(` did not find record for %s = %d because of %s`,model.GetPrimaryKeyName(),model.GetPrimaryKeyValue(),err)
        return
    }
    if found == false {
        t.Errorf(` did not find record for %s = %d because of %s`,model.GetPrimaryKeyName(),model.GetPrimaryKeyValue(),err)
        return
    }


    if model.PositionId != model2.PositionId {
        t.Errorf(` model.PositionId[%d] != model2.PositionId[%d]`,model.PositionId,model2.PositionId)
        return
    }

    if (model.Day.Year != model2.Day.Year ||
        model.Day.Month != model2.Day.Month ||
        model.Day.Day != model2.Day.Day ||
        model.Day.Hours != model2.Day.Hours ||
        model.Day.Minutes != model2.Day.Minutes ||
        model.Day.Seconds != model2.Day.Seconds ) {
        t.Errorf(`2: model.Day != model2.Day %+v --- %+v`,model.Day,model2.Day)
        return
    }

    if model.Open != model2.Open {
        t.Errorf(` model.Open[%d] != model2.Open[%d]`,model.Open,model2.Open)
        return
    }

    if model.High != model2.High {
        t.Errorf(` model.High[%d] != model2.High[%d]`,model.High,model2.High)
        return
    }

    if model.Low != model2.Low {
        t.Errorf(` model.Low[%d] != model2.Low[%d]`,model.Low,model2.Low)
        return
    }

    if model.Pvolume != model2.Pvolume {
        t.Errorf(` model.Pvolume[%d] != model2.Pvolume[%d]`,model.Pvolume,model2.Pvolume)
        return
    }

    if model.Pchange != model2.Pchange {
        t.Errorf(` model.Pchange[%d] != model2.Pchange[%d]`,model.Pchange,model2.Pchange)
        return
    }

    if model.PchangePercent != model2.PchangePercent {
        t.Errorf(` model.PchangePercent[%d] != model2.PchangePercent[%d]`,model.PchangePercent,model2.PchangePercent)
        return
    }

    if model.AdjClose != model2.AdjClose {
        t.Errorf(` model.AdjClose[%d] != model2.AdjClose[%d]`,model.AdjClose,model2.AdjClose)
        return
    }

    if model.DataSource != model2.DataSource {
        t.Errorf(` model.DataSource[%s] != model2.DataSource[%s]`,model.DataSource,model2.DataSource)
        return
    }
model2.SetPositionId(int64(randomInteger()))
model2.SetDay(randomDateTime(a))
model2.SetOpen(int(randomInteger()))
model2.SetHigh(int(randomInteger()))
model2.SetLow(int(randomInteger()))
model2.SetPvolume(int(randomInteger()))
model2.SetPchange(int(randomInteger()))
model2.SetPchangePercent(int(randomInteger()))
model2.SetAdjClose(int(randomInteger()))
model2.SetDataSource(randomString(19))

    err = model2.Save()
    if err != nil {
        t.Errorf(`failed to save model2 %s`,err)
    }

    if model.PositionId == model2.PositionId {
        t.Errorf(`1: model.PositionId[%d] != model2.PositionId[%d]`,model.PositionId,model2.PositionId)
        return
    }

    if (model.Day.Year == model2.Day.Year) {
        t.Errorf(` model.Day.Year == model2.Day but should not!`)
        return
    }

    if model.Open == model2.Open {
        t.Errorf(`1: model.Open[%d] != model2.Open[%d]`,model.Open,model2.Open)
        return
    }

    if model.High == model2.High {
        t.Errorf(`1: model.High[%d] != model2.High[%d]`,model.High,model2.High)
        return
    }

    if model.Low == model2.Low {
        t.Errorf(`1: model.Low[%d] != model2.Low[%d]`,model.Low,model2.Low)
        return
    }

    if model.Pvolume == model2.Pvolume {
        t.Errorf(`1: model.Pvolume[%d] != model2.Pvolume[%d]`,model.Pvolume,model2.Pvolume)
        return
    }

    if model.Pchange == model2.Pchange {
        t.Errorf(`1: model.Pchange[%d] != model2.Pchange[%d]`,model.Pchange,model2.Pchange)
        return
    }

    if model.PchangePercent == model2.PchangePercent {
        t.Errorf(`1: model.PchangePercent[%d] != model2.PchangePercent[%d]`,model.PchangePercent,model2.PchangePercent)
        return
    }

    if model.AdjClose == model2.AdjClose {
        t.Errorf(`1: model.AdjClose[%d] != model2.AdjClose[%d]`,model.AdjClose,model2.AdjClose)
        return
    }

    if model.DataSource == model2.DataSource {
        t.Errorf(`1: model.DataSource[%s] != model2.DataSource[%s]`,model.DataSource,model2.DataSource)
        return
    }

    res28,err := model.FindByPositionId(model2.GetPositionId())
    if err != nil {
        t.Errorf(`failed model.FindByPositionId(model2.GetPositionId())`)
    }
    if len(res28) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res29,err := model.FindByDay(model2.GetDay())
    if err != nil {
        t.Errorf(`failed model.FindByDay(model2.GetDay())`)
    }
    if len(res29) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res30,err := model.FindByOpen(model2.GetOpen())
    if err != nil {
        t.Errorf(`failed model.FindByOpen(model2.GetOpen())`)
    }
    if len(res30) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res31,err := model.FindByHigh(model2.GetHigh())
    if err != nil {
        t.Errorf(`failed model.FindByHigh(model2.GetHigh())`)
    }
    if len(res31) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res32,err := model.FindByLow(model2.GetLow())
    if err != nil {
        t.Errorf(`failed model.FindByLow(model2.GetLow())`)
    }
    if len(res32) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res33,err := model.FindByPvolume(model2.GetPvolume())
    if err != nil {
        t.Errorf(`failed model.FindByPvolume(model2.GetPvolume())`)
    }
    if len(res33) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res34,err := model.FindByPchange(model2.GetPchange())
    if err != nil {
        t.Errorf(`failed model.FindByPchange(model2.GetPchange())`)
    }
    if len(res34) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res35,err := model.FindByPchangePercent(model2.GetPchangePercent())
    if err != nil {
        t.Errorf(`failed model.FindByPchangePercent(model2.GetPchangePercent())`)
    }
    if len(res35) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res36,err := model.FindByAdjClose(model2.GetAdjClose())
    if err != nil {
        t.Errorf(`failed model.FindByAdjClose(model2.GetAdjClose())`)
    }
    if len(res36) == 0 {
        t.Errorf(`failed to find any Play`)
    }

    res37,err := model.FindByDataSource(model2.GetDataSource())
    if err != nil {
        t.Errorf(`failed model.FindByDataSource(model2.GetDataSource())`)
    }
    if len(res37) == 0 {
        t.Errorf(`failed to find any Play`)
    }
} // end of if fileExists
};


func TestPlayUpdaters(t *testing.T) {
    if fileExists(`../gopaper-testing.db.yml`) == false {
        return
    }
    a,err := NewMysqlAdapterEx(`../gopaper-testing.db.yml`)
    defer a.Close()
    if err != nil {
        t.Errorf(`could not load ../gopaper-testing.db.yml %s`,err)
        return
    }
    file, err := os.OpenFile("adapter.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        t.Errorf("Failed to open log file %s", err)
        return
    }
    a.SetLogs(file)
    model := NewPlay(a)

    model.SetPositionId(int64(randomInteger()))
    if model.GetPositionId() != model.PositionId {
        t.Errorf(`Play.GetPositionId() != Play.PositionId`)
    }
    if model.IsPositionIdDirty != true {
        t.Errorf(`Play.IsPositionIdDirty != true`)
        return
    }
    
    u0 := int64(randomInteger())
    _,err = model.UpdatePositionId(u0)
    if err != nil {
        t.Errorf(`failed UpdatePositionId(u0) %s`,err)
        return
    }

    if model.GetPositionId() != u0 {
        t.Errorf(`Play.GetPositionId() != u0 after UpdatePositionId`)
        return
    }
    model.Reload()
    if model.GetPositionId() != u0 {
        t.Errorf(`Play.GetPositionId() != u0 after Reload`)
        return
    }

    model.SetDay(randomDateTime(a))
    if model.GetDay() != model.Day {
        t.Errorf(`Play.GetDay() != Play.Day`)
    }
    if model.IsDayDirty != true {
        t.Errorf(`Play.IsDayDirty != true`)
        return
    }
    
    u1 := randomDateTime(a)
    _,err = model.UpdateDay(u1)
    if err != nil {
        t.Errorf(`failed UpdateDay(u1) %s`,err)
        return
    }

    if model.GetDay() != u1 {
        t.Errorf(`Play.GetDay() != u1 after UpdateDay`)
        return
    }
    model.Reload()
    if model.GetDay() != u1 {
        t.Errorf(`Play.GetDay() != u1 after Reload`)
        return
    }

    model.SetOpen(int(randomInteger()))
    if model.GetOpen() != model.Open {
        t.Errorf(`Play.GetOpen() != Play.Open`)
    }
    if model.IsOpenDirty != true {
        t.Errorf(`Play.IsOpenDirty != true`)
        return
    }
    
    u2 := int(randomInteger())
    _,err = model.UpdateOpen(u2)
    if err != nil {
        t.Errorf(`failed UpdateOpen(u2) %s`,err)
        return
    }

    if model.GetOpen() != u2 {
        t.Errorf(`Play.GetOpen() != u2 after UpdateOpen`)
        return
    }
    model.Reload()
    if model.GetOpen() != u2 {
        t.Errorf(`Play.GetOpen() != u2 after Reload`)
        return
    }

    model.SetHigh(int(randomInteger()))
    if model.GetHigh() != model.High {
        t.Errorf(`Play.GetHigh() != Play.High`)
    }
    if model.IsHighDirty != true {
        t.Errorf(`Play.IsHighDirty != true`)
        return
    }
    
    u3 := int(randomInteger())
    _,err = model.UpdateHigh(u3)
    if err != nil {
        t.Errorf(`failed UpdateHigh(u3) %s`,err)
        return
    }

    if model.GetHigh() != u3 {
        t.Errorf(`Play.GetHigh() != u3 after UpdateHigh`)
        return
    }
    model.Reload()
    if model.GetHigh() != u3 {
        t.Errorf(`Play.GetHigh() != u3 after Reload`)
        return
    }

    model.SetLow(int(randomInteger()))
    if model.GetLow() != model.Low {
        t.Errorf(`Play.GetLow() != Play.Low`)
    }
    if model.IsLowDirty != true {
        t.Errorf(`Play.IsLowDirty != true`)
        return
    }
    
    u4 := int(randomInteger())
    _,err = model.UpdateLow(u4)
    if err != nil {
        t.Errorf(`failed UpdateLow(u4) %s`,err)
        return
    }

    if model.GetLow() != u4 {
        t.Errorf(`Play.GetLow() != u4 after UpdateLow`)
        return
    }
    model.Reload()
    if model.GetLow() != u4 {
        t.Errorf(`Play.GetLow() != u4 after Reload`)
        return
    }

    model.SetPvolume(int(randomInteger()))
    if model.GetPvolume() != model.Pvolume {
        t.Errorf(`Play.GetPvolume() != Play.Pvolume`)
    }
    if model.IsPvolumeDirty != true {
        t.Errorf(`Play.IsPvolumeDirty != true`)
        return
    }
    
    u5 := int(randomInteger())
    _,err = model.UpdatePvolume(u5)
    if err != nil {
        t.Errorf(`failed UpdatePvolume(u5) %s`,err)
        return
    }

    if model.GetPvolume() != u5 {
        t.Errorf(`Play.GetPvolume() != u5 after UpdatePvolume`)
        return
    }
    model.Reload()
    if model.GetPvolume() != u5 {
        t.Errorf(`Play.GetPvolume() != u5 after Reload`)
        return
    }

    model.SetPchange(int(randomInteger()))
    if model.GetPchange() != model.Pchange {
        t.Errorf(`Play.GetPchange() != Play.Pchange`)
    }
    if model.IsPchangeDirty != true {
        t.Errorf(`Play.IsPchangeDirty != true`)
        return
    }
    
    u6 := int(randomInteger())
    _,err = model.UpdatePchange(u6)
    if err != nil {
        t.Errorf(`failed UpdatePchange(u6) %s`,err)
        return
    }

    if model.GetPchange() != u6 {
        t.Errorf(`Play.GetPchange() != u6 after UpdatePchange`)
        return
    }
    model.Reload()
    if model.GetPchange() != u6 {
        t.Errorf(`Play.GetPchange() != u6 after Reload`)
        return
    }

    model.SetPchangePercent(int(randomInteger()))
    if model.GetPchangePercent() != model.PchangePercent {
        t.Errorf(`Play.GetPchangePercent() != Play.PchangePercent`)
    }
    if model.IsPchangePercentDirty != true {
        t.Errorf(`Play.IsPchangePercentDirty != true`)
        return
    }
    
    u7 := int(randomInteger())
    _,err = model.UpdatePchangePercent(u7)
    if err != nil {
        t.Errorf(`failed UpdatePchangePercent(u7) %s`,err)
        return
    }

    if model.GetPchangePercent() != u7 {
        t.Errorf(`Play.GetPchangePercent() != u7 after UpdatePchangePercent`)
        return
    }
    model.Reload()
    if model.GetPchangePercent() != u7 {
        t.Errorf(`Play.GetPchangePercent() != u7 after Reload`)
        return
    }

    model.SetAdjClose(int(randomInteger()))
    if model.GetAdjClose() != model.AdjClose {
        t.Errorf(`Play.GetAdjClose() != Play.AdjClose`)
    }
    if model.IsAdjCloseDirty != true {
        t.Errorf(`Play.IsAdjCloseDirty != true`)
        return
    }
    
    u8 := int(randomInteger())
    _,err = model.UpdateAdjClose(u8)
    if err != nil {
        t.Errorf(`failed UpdateAdjClose(u8) %s`,err)
        return
    }

    if model.GetAdjClose() != u8 {
        t.Errorf(`Play.GetAdjClose() != u8 after UpdateAdjClose`)
        return
    }
    model.Reload()
    if model.GetAdjClose() != u8 {
        t.Errorf(`Play.GetAdjClose() != u8 after Reload`)
        return
    }

    model.SetDataSource(randomString(19))
    if model.GetDataSource() != model.DataSource {
        t.Errorf(`Play.GetDataSource() != Play.DataSource`)
    }
    if model.IsDataSourceDirty != true {
        t.Errorf(`Play.IsDataSourceDirty != true`)
        return
    }
    
    u9 := randomString(19)
    _,err = model.UpdateDataSource(u9)
    if err != nil {
        t.Errorf(`failed UpdateDataSource(u9) %s`,err)
        return
    }

    if model.GetDataSource() != u9 {
        t.Errorf(`Play.GetDataSource() != u9 after UpdateDataSource`)
        return
    }
    model.Reload()
    if model.GetDataSource() != u9 {
        t.Errorf(`Play.GetDataSource() != u9 after Reload`)
        return
    }

};


func TestNewPortfolio(t *testing.T) {
    a := NewMysqlAdapter(``)
    o := NewPortfolio(a)
    if o._table != "portfolios" {
        t.Errorf("failed creating %+v",o);
        return
    }
}
func TestPortfolioFromDBValueMap(t *testing.T) {
    a := NewMysqlAdapter(``)
    o := NewPortfolio(a)
    m := make(map[string]DBValue)
	m["id"] = a.NewDBValue()
	m["id"].SetInternalValue("id",strconv.Itoa(999))
	m["name"] = a.NewDBValue()
	m["name"].SetInternalValue("name","AString")
	m["description"] = a.NewDBValue()
	m["description"].SetInternalValue("description","AString")
	m["value"] = a.NewDBValue()
	m["value"].SetInternalValue("value",strconv.Itoa(999))

    err := o.FromDBValueMap(m)
    if err != nil {
        t.Errorf("FromDBValueMap failed %s",err)
    }

    if o.Id != 999 {
        t.Errorf("o.Id test failed %+v",o)
        return
    }    

    if o.Name != "AString" {
        t.Errorf("o.Name test failed %+v",o)
        return
    }    

    if o.Description != "AString" {
        t.Errorf("o.Description test failed %+v",o)
        return
    }    

    if o.Value != 999 {
        t.Errorf("o.Value test failed %+v",o)
        return
    }    
}

func TestPortfolioCreate(t *testing.T) {
    if fileExists(`../gopaper-testing.db.yml`) {
    a,err := NewMysqlAdapterEx(`../gopaper-testing.db.yml`)
    defer a.Close()
    if err != nil {
        t.Errorf(`could not load ../gopaper-testing.db.yml %s`,err)
        return
    }
    file, err := os.OpenFile("adapter.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        t.Errorf(" Failed to open log file %s", err)
    }
    a.SetLogs(file)
    model := NewPortfolio(a)
model.Name = randomString(19)
model.Description = randomString(25)
model.Value = int(randomInteger())

    err = model.Create()
    if err != nil {
        t.Errorf(` failed to create model %s`,err)
        return
    }

    model2 := NewPortfolio(a)
    found,err := model2.Find(model.GetPrimaryKeyValue())
    if err != nil {
        t.Errorf(` did not find record for %s = %d because of %s`,model.GetPrimaryKeyName(),model.GetPrimaryKeyValue(),err)
        return
    }
    if found == false {
        t.Errorf(` did not find record for %s = %d because of %s`,model.GetPrimaryKeyName(),model.GetPrimaryKeyValue(),err)
        return
    }


    if model.Name != model2.Name {
        t.Errorf(` model.Name[%s] != model2.Name[%s]`,model.Name,model2.Name)
        return
    }

    if model.Description != model2.Description {
        t.Errorf(` model.Description[%s] != model2.Description[%s]`,model.Description,model2.Description)
        return
    }

    if model.Value != model2.Value {
        t.Errorf(` model.Value[%d] != model2.Value[%d]`,model.Value,model2.Value)
        return
    }
model2.SetName(randomString(19))
model2.SetDescription(randomString(25))
model2.SetValue(int(randomInteger()))

    err = model2.Save()
    if err != nil {
        t.Errorf(`failed to save model2 %s`,err)
    }

    if model.Name == model2.Name {
        t.Errorf(`1: model.Name[%s] != model2.Name[%s]`,model.Name,model2.Name)
        return
    }

    if model.Description == model2.Description {
        t.Errorf(`1: model.Description[%s] != model2.Description[%s]`,model.Description,model2.Description)
        return
    }

    if model.Value == model2.Value {
        t.Errorf(`1: model.Value[%d] != model2.Value[%d]`,model.Value,model2.Value)
        return
    }

    res9,err := model.FindByName(model2.GetName())
    if err != nil {
        t.Errorf(`failed model.FindByName(model2.GetName())`)
    }
    if len(res9) == 0 {
        t.Errorf(`failed to find any Portfolio`)
    }

    res10,err := model.FindByDescription(model2.GetDescription())
    if err != nil {
        t.Errorf(`failed model.FindByDescription(model2.GetDescription())`)
    }
    if len(res10) == 0 {
        t.Errorf(`failed to find any Portfolio`)
    }

    res11,err := model.FindByValue(model2.GetValue())
    if err != nil {
        t.Errorf(`failed model.FindByValue(model2.GetValue())`)
    }
    if len(res11) == 0 {
        t.Errorf(`failed to find any Portfolio`)
    }
} // end of if fileExists
};


func TestPortfolioUpdaters(t *testing.T) {
    if fileExists(`../gopaper-testing.db.yml`) == false {
        return
    }
    a,err := NewMysqlAdapterEx(`../gopaper-testing.db.yml`)
    defer a.Close()
    if err != nil {
        t.Errorf(`could not load ../gopaper-testing.db.yml %s`,err)
        return
    }
    file, err := os.OpenFile("adapter.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        t.Errorf("Failed to open log file %s", err)
        return
    }
    a.SetLogs(file)
    model := NewPortfolio(a)

    model.SetName(randomString(19))
    if model.GetName() != model.Name {
        t.Errorf(`Portfolio.GetName() != Portfolio.Name`)
    }
    if model.IsNameDirty != true {
        t.Errorf(`Portfolio.IsNameDirty != true`)
        return
    }
    
    u0 := randomString(19)
    _,err = model.UpdateName(u0)
    if err != nil {
        t.Errorf(`failed UpdateName(u0) %s`,err)
        return
    }

    if model.GetName() != u0 {
        t.Errorf(`Portfolio.GetName() != u0 after UpdateName`)
        return
    }
    model.Reload()
    if model.GetName() != u0 {
        t.Errorf(`Portfolio.GetName() != u0 after Reload`)
        return
    }

    model.SetDescription(randomString(25))
    if model.GetDescription() != model.Description {
        t.Errorf(`Portfolio.GetDescription() != Portfolio.Description`)
    }
    if model.IsDescriptionDirty != true {
        t.Errorf(`Portfolio.IsDescriptionDirty != true`)
        return
    }
    
    u1 := randomString(25)
    _,err = model.UpdateDescription(u1)
    if err != nil {
        t.Errorf(`failed UpdateDescription(u1) %s`,err)
        return
    }

    if model.GetDescription() != u1 {
        t.Errorf(`Portfolio.GetDescription() != u1 after UpdateDescription`)
        return
    }
    model.Reload()
    if model.GetDescription() != u1 {
        t.Errorf(`Portfolio.GetDescription() != u1 after Reload`)
        return
    }

    model.SetValue(int(randomInteger()))
    if model.GetValue() != model.Value {
        t.Errorf(`Portfolio.GetValue() != Portfolio.Value`)
    }
    if model.IsValueDirty != true {
        t.Errorf(`Portfolio.IsValueDirty != true`)
        return
    }
    
    u2 := int(randomInteger())
    _,err = model.UpdateValue(u2)
    if err != nil {
        t.Errorf(`failed UpdateValue(u2) %s`,err)
        return
    }

    if model.GetValue() != u2 {
        t.Errorf(`Portfolio.GetValue() != u2 after UpdateValue`)
        return
    }
    model.Reload()
    if model.GetValue() != u2 {
        t.Errorf(`Portfolio.GetValue() != u2 after Reload`)
        return
    }

};


func TestNewPosition(t *testing.T) {
    a := NewMysqlAdapter(``)
    o := NewPosition(a)
    if o._table != "positions" {
        t.Errorf("failed creating %+v",o);
        return
    }
}
func TestPositionFromDBValueMap(t *testing.T) {
    a := NewMysqlAdapter(``)
    o := NewPosition(a)
    m := make(map[string]DBValue)
	m["id"] = a.NewDBValue()
	m["id"].SetInternalValue("id",strconv.Itoa(999))
	m["portfolio_id"] = a.NewDBValue()
	m["portfolio_id"].SetInternalValue("portfolio_id",strconv.Itoa(999))
	m["started_at"] = a.NewDBValue()
	m["started_at"].SetInternalValue("started_at","2016-01-01 10:50:23")
	m["closed_at"] = a.NewDBValue()
	m["closed_at"].SetInternalValue("closed_at","2016-01-01 10:50:23")
	m["ptype"] = a.NewDBValue()
	m["ptype"].SetInternalValue("ptype","AString")
	m["buy"] = a.NewDBValue()
	m["buy"].SetInternalValue("buy",strconv.Itoa(999))
	m["sell"] = a.NewDBValue()
	m["sell"].SetInternalValue("sell",strconv.Itoa(999))
	m["stop_loss"] = a.NewDBValue()
	m["stop_loss"].SetInternalValue("stop_loss",strconv.Itoa(999))
	m["quantity"] = a.NewDBValue()
	m["quantity"].SetInternalValue("quantity",strconv.Itoa(999))

    err := o.FromDBValueMap(m)
    if err != nil {
        t.Errorf("FromDBValueMap failed %s",err)
    }

    if o.Id != 999 {
        t.Errorf("o.Id test failed %+v",o)
        return
    }    

    if o.PortfolioId != 999 {
        t.Errorf("o.PortfolioId test failed %+v",o)
        return
    }    

    if o.StartedAt.Year != 2016 {
        t.Errorf("year not set for %+v",o.StartedAt)
        return
    }
    if (o.StartedAt.Year != 2016 || 
        o.StartedAt.Month != 1 ||
        o.StartedAt.Day != 1 ||
        o.StartedAt.Hours != 10 ||
        o.StartedAt.Minutes != 50 ||
        o.StartedAt.Seconds != 23 ) {
        t.Errorf(`fields don't match up for %+v`,o.StartedAt)
    }
    r2,_ := m["started_at"].AsString()
    if o.StartedAt.ToString() != r2 {
        t.Errorf(`restring of o.StartedAt failed %s`,o.StartedAt.ToString())
    }

    if o.ClosedAt.Year != 2016 {
        t.Errorf("year not set for %+v",o.ClosedAt)
        return
    }
    if (o.ClosedAt.Year != 2016 || 
        o.ClosedAt.Month != 1 ||
        o.ClosedAt.Day != 1 ||
        o.ClosedAt.Hours != 10 ||
        o.ClosedAt.Minutes != 50 ||
        o.ClosedAt.Seconds != 23 ) {
        t.Errorf(`fields don't match up for %+v`,o.ClosedAt)
    }
    r3,_ := m["closed_at"].AsString()
    if o.ClosedAt.ToString() != r3 {
        t.Errorf(`restring of o.ClosedAt failed %s`,o.ClosedAt.ToString())
    }

    if o.Ptype != "AString" {
        t.Errorf("o.Ptype test failed %+v",o)
        return
    }    

    if o.Buy != 999 {
        t.Errorf("o.Buy test failed %+v",o)
        return
    }    

    if o.Sell != 999 {
        t.Errorf("o.Sell test failed %+v",o)
        return
    }    

    if o.StopLoss != 999 {
        t.Errorf("o.StopLoss test failed %+v",o)
        return
    }    

    if o.Quantity != 999 {
        t.Errorf("o.Quantity test failed %+v",o)
        return
    }    
}

func TestPositionCreate(t *testing.T) {
    if fileExists(`../gopaper-testing.db.yml`) {
    a,err := NewMysqlAdapterEx(`../gopaper-testing.db.yml`)
    defer a.Close()
    if err != nil {
        t.Errorf(`could not load ../gopaper-testing.db.yml %s`,err)
        return
    }
    file, err := os.OpenFile("adapter.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        t.Errorf(" Failed to open log file %s", err)
    }
    a.SetLogs(file)
    model := NewPosition(a)
model.PortfolioId = int64(randomInteger())
model.StartedAt = randomDateTime(a)
model.ClosedAt = randomDateTime(a)
model.Ptype = randomString(19)
model.Buy = int(randomInteger())
model.Sell = int(randomInteger())
model.StopLoss = int(randomInteger())
model.Quantity = int(randomInteger())

    err = model.Create()
    if err != nil {
        t.Errorf(` failed to create model %s`,err)
        return
    }

    model2 := NewPosition(a)
    found,err := model2.Find(model.GetPrimaryKeyValue())
    if err != nil {
        t.Errorf(` did not find record for %s = %d because of %s`,model.GetPrimaryKeyName(),model.GetPrimaryKeyValue(),err)
        return
    }
    if found == false {
        t.Errorf(` did not find record for %s = %d because of %s`,model.GetPrimaryKeyName(),model.GetPrimaryKeyValue(),err)
        return
    }


    if model.PortfolioId != model2.PortfolioId {
        t.Errorf(` model.PortfolioId[%d] != model2.PortfolioId[%d]`,model.PortfolioId,model2.PortfolioId)
        return
    }

    if (model.StartedAt.Year != model2.StartedAt.Year ||
        model.StartedAt.Month != model2.StartedAt.Month ||
        model.StartedAt.Day != model2.StartedAt.Day ||
        model.StartedAt.Hours != model2.StartedAt.Hours ||
        model.StartedAt.Minutes != model2.StartedAt.Minutes ||
        model.StartedAt.Seconds != model2.StartedAt.Seconds ) {
        t.Errorf(`2: model.StartedAt != model2.StartedAt %+v --- %+v`,model.StartedAt,model2.StartedAt)
        return
    }

    if (model.ClosedAt.Year != model2.ClosedAt.Year ||
        model.ClosedAt.Month != model2.ClosedAt.Month ||
        model.ClosedAt.Day != model2.ClosedAt.Day ||
        model.ClosedAt.Hours != model2.ClosedAt.Hours ||
        model.ClosedAt.Minutes != model2.ClosedAt.Minutes ||
        model.ClosedAt.Seconds != model2.ClosedAt.Seconds ) {
        t.Errorf(`2: model.ClosedAt != model2.ClosedAt %+v --- %+v`,model.ClosedAt,model2.ClosedAt)
        return
    }

    if model.Ptype != model2.Ptype {
        t.Errorf(` model.Ptype[%s] != model2.Ptype[%s]`,model.Ptype,model2.Ptype)
        return
    }

    if model.Buy != model2.Buy {
        t.Errorf(` model.Buy[%d] != model2.Buy[%d]`,model.Buy,model2.Buy)
        return
    }

    if model.Sell != model2.Sell {
        t.Errorf(` model.Sell[%d] != model2.Sell[%d]`,model.Sell,model2.Sell)
        return
    }

    if model.StopLoss != model2.StopLoss {
        t.Errorf(` model.StopLoss[%d] != model2.StopLoss[%d]`,model.StopLoss,model2.StopLoss)
        return
    }

    if model.Quantity != model2.Quantity {
        t.Errorf(` model.Quantity[%d] != model2.Quantity[%d]`,model.Quantity,model2.Quantity)
        return
    }
model2.SetPortfolioId(int64(randomInteger()))
model2.SetStartedAt(randomDateTime(a))
model2.SetClosedAt(randomDateTime(a))
model2.SetPtype(randomString(19))
model2.SetBuy(int(randomInteger()))
model2.SetSell(int(randomInteger()))
model2.SetStopLoss(int(randomInteger()))
model2.SetQuantity(int(randomInteger()))

    err = model2.Save()
    if err != nil {
        t.Errorf(`failed to save model2 %s`,err)
    }

    if model.PortfolioId == model2.PortfolioId {
        t.Errorf(`1: model.PortfolioId[%d] != model2.PortfolioId[%d]`,model.PortfolioId,model2.PortfolioId)
        return
    }

    if (model.StartedAt.Year == model2.StartedAt.Year) {
        t.Errorf(` model.StartedAt.Year == model2.StartedAt but should not!`)
        return
    }

    if (model.ClosedAt.Year == model2.ClosedAt.Year) {
        t.Errorf(` model.ClosedAt.Year == model2.ClosedAt but should not!`)
        return
    }

    if model.Ptype == model2.Ptype {
        t.Errorf(`1: model.Ptype[%s] != model2.Ptype[%s]`,model.Ptype,model2.Ptype)
        return
    }

    if model.Buy == model2.Buy {
        t.Errorf(`1: model.Buy[%d] != model2.Buy[%d]`,model.Buy,model2.Buy)
        return
    }

    if model.Sell == model2.Sell {
        t.Errorf(`1: model.Sell[%d] != model2.Sell[%d]`,model.Sell,model2.Sell)
        return
    }

    if model.StopLoss == model2.StopLoss {
        t.Errorf(`1: model.StopLoss[%d] != model2.StopLoss[%d]`,model.StopLoss,model2.StopLoss)
        return
    }

    if model.Quantity == model2.Quantity {
        t.Errorf(`1: model.Quantity[%d] != model2.Quantity[%d]`,model.Quantity,model2.Quantity)
        return
    }

    res20,err := model.FindByPortfolioId(model2.GetPortfolioId())
    if err != nil {
        t.Errorf(`failed model.FindByPortfolioId(model2.GetPortfolioId())`)
    }
    if len(res20) == 0 {
        t.Errorf(`failed to find any Position`)
    }

    res21,err := model.FindByStartedAt(model2.GetStartedAt())
    if err != nil {
        t.Errorf(`failed model.FindByStartedAt(model2.GetStartedAt())`)
    }
    if len(res21) == 0 {
        t.Errorf(`failed to find any Position`)
    }

    res22,err := model.FindByClosedAt(model2.GetClosedAt())
    if err != nil {
        t.Errorf(`failed model.FindByClosedAt(model2.GetClosedAt())`)
    }
    if len(res22) == 0 {
        t.Errorf(`failed to find any Position`)
    }

    res23,err := model.FindByPtype(model2.GetPtype())
    if err != nil {
        t.Errorf(`failed model.FindByPtype(model2.GetPtype())`)
    }
    if len(res23) == 0 {
        t.Errorf(`failed to find any Position`)
    }

    res24,err := model.FindByBuy(model2.GetBuy())
    if err != nil {
        t.Errorf(`failed model.FindByBuy(model2.GetBuy())`)
    }
    if len(res24) == 0 {
        t.Errorf(`failed to find any Position`)
    }

    res25,err := model.FindBySell(model2.GetSell())
    if err != nil {
        t.Errorf(`failed model.FindBySell(model2.GetSell())`)
    }
    if len(res25) == 0 {
        t.Errorf(`failed to find any Position`)
    }

    res26,err := model.FindByStopLoss(model2.GetStopLoss())
    if err != nil {
        t.Errorf(`failed model.FindByStopLoss(model2.GetStopLoss())`)
    }
    if len(res26) == 0 {
        t.Errorf(`failed to find any Position`)
    }

    res27,err := model.FindByQuantity(model2.GetQuantity())
    if err != nil {
        t.Errorf(`failed model.FindByQuantity(model2.GetQuantity())`)
    }
    if len(res27) == 0 {
        t.Errorf(`failed to find any Position`)
    }
} // end of if fileExists
};


func TestPositionUpdaters(t *testing.T) {
    if fileExists(`../gopaper-testing.db.yml`) == false {
        return
    }
    a,err := NewMysqlAdapterEx(`../gopaper-testing.db.yml`)
    defer a.Close()
    if err != nil {
        t.Errorf(`could not load ../gopaper-testing.db.yml %s`,err)
        return
    }
    file, err := os.OpenFile("adapter.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        t.Errorf("Failed to open log file %s", err)
        return
    }
    a.SetLogs(file)
    model := NewPosition(a)

    model.SetPortfolioId(int64(randomInteger()))
    if model.GetPortfolioId() != model.PortfolioId {
        t.Errorf(`Position.GetPortfolioId() != Position.PortfolioId`)
    }
    if model.IsPortfolioIdDirty != true {
        t.Errorf(`Position.IsPortfolioIdDirty != true`)
        return
    }
    
    u0 := int64(randomInteger())
    _,err = model.UpdatePortfolioId(u0)
    if err != nil {
        t.Errorf(`failed UpdatePortfolioId(u0) %s`,err)
        return
    }

    if model.GetPortfolioId() != u0 {
        t.Errorf(`Position.GetPortfolioId() != u0 after UpdatePortfolioId`)
        return
    }
    model.Reload()
    if model.GetPortfolioId() != u0 {
        t.Errorf(`Position.GetPortfolioId() != u0 after Reload`)
        return
    }

    model.SetStartedAt(randomDateTime(a))
    if model.GetStartedAt() != model.StartedAt {
        t.Errorf(`Position.GetStartedAt() != Position.StartedAt`)
    }
    if model.IsStartedAtDirty != true {
        t.Errorf(`Position.IsStartedAtDirty != true`)
        return
    }
    
    u1 := randomDateTime(a)
    _,err = model.UpdateStartedAt(u1)
    if err != nil {
        t.Errorf(`failed UpdateStartedAt(u1) %s`,err)
        return
    }

    if model.GetStartedAt() != u1 {
        t.Errorf(`Position.GetStartedAt() != u1 after UpdateStartedAt`)
        return
    }
    model.Reload()
    if model.GetStartedAt() != u1 {
        t.Errorf(`Position.GetStartedAt() != u1 after Reload`)
        return
    }

    model.SetClosedAt(randomDateTime(a))
    if model.GetClosedAt() != model.ClosedAt {
        t.Errorf(`Position.GetClosedAt() != Position.ClosedAt`)
    }
    if model.IsClosedAtDirty != true {
        t.Errorf(`Position.IsClosedAtDirty != true`)
        return
    }
    
    u2 := randomDateTime(a)
    _,err = model.UpdateClosedAt(u2)
    if err != nil {
        t.Errorf(`failed UpdateClosedAt(u2) %s`,err)
        return
    }

    if model.GetClosedAt() != u2 {
        t.Errorf(`Position.GetClosedAt() != u2 after UpdateClosedAt`)
        return
    }
    model.Reload()
    if model.GetClosedAt() != u2 {
        t.Errorf(`Position.GetClosedAt() != u2 after Reload`)
        return
    }

    model.SetPtype(randomString(19))
    if model.GetPtype() != model.Ptype {
        t.Errorf(`Position.GetPtype() != Position.Ptype`)
    }
    if model.IsPtypeDirty != true {
        t.Errorf(`Position.IsPtypeDirty != true`)
        return
    }
    
    u3 := randomString(19)
    _,err = model.UpdatePtype(u3)
    if err != nil {
        t.Errorf(`failed UpdatePtype(u3) %s`,err)
        return
    }

    if model.GetPtype() != u3 {
        t.Errorf(`Position.GetPtype() != u3 after UpdatePtype`)
        return
    }
    model.Reload()
    if model.GetPtype() != u3 {
        t.Errorf(`Position.GetPtype() != u3 after Reload`)
        return
    }

    model.SetBuy(int(randomInteger()))
    if model.GetBuy() != model.Buy {
        t.Errorf(`Position.GetBuy() != Position.Buy`)
    }
    if model.IsBuyDirty != true {
        t.Errorf(`Position.IsBuyDirty != true`)
        return
    }
    
    u4 := int(randomInteger())
    _,err = model.UpdateBuy(u4)
    if err != nil {
        t.Errorf(`failed UpdateBuy(u4) %s`,err)
        return
    }

    if model.GetBuy() != u4 {
        t.Errorf(`Position.GetBuy() != u4 after UpdateBuy`)
        return
    }
    model.Reload()
    if model.GetBuy() != u4 {
        t.Errorf(`Position.GetBuy() != u4 after Reload`)
        return
    }

    model.SetSell(int(randomInteger()))
    if model.GetSell() != model.Sell {
        t.Errorf(`Position.GetSell() != Position.Sell`)
    }
    if model.IsSellDirty != true {
        t.Errorf(`Position.IsSellDirty != true`)
        return
    }
    
    u5 := int(randomInteger())
    _,err = model.UpdateSell(u5)
    if err != nil {
        t.Errorf(`failed UpdateSell(u5) %s`,err)
        return
    }

    if model.GetSell() != u5 {
        t.Errorf(`Position.GetSell() != u5 after UpdateSell`)
        return
    }
    model.Reload()
    if model.GetSell() != u5 {
        t.Errorf(`Position.GetSell() != u5 after Reload`)
        return
    }

    model.SetStopLoss(int(randomInteger()))
    if model.GetStopLoss() != model.StopLoss {
        t.Errorf(`Position.GetStopLoss() != Position.StopLoss`)
    }
    if model.IsStopLossDirty != true {
        t.Errorf(`Position.IsStopLossDirty != true`)
        return
    }
    
    u6 := int(randomInteger())
    _,err = model.UpdateStopLoss(u6)
    if err != nil {
        t.Errorf(`failed UpdateStopLoss(u6) %s`,err)
        return
    }

    if model.GetStopLoss() != u6 {
        t.Errorf(`Position.GetStopLoss() != u6 after UpdateStopLoss`)
        return
    }
    model.Reload()
    if model.GetStopLoss() != u6 {
        t.Errorf(`Position.GetStopLoss() != u6 after Reload`)
        return
    }

    model.SetQuantity(int(randomInteger()))
    if model.GetQuantity() != model.Quantity {
        t.Errorf(`Position.GetQuantity() != Position.Quantity`)
    }
    if model.IsQuantityDirty != true {
        t.Errorf(`Position.IsQuantityDirty != true`)
        return
    }
    
    u7 := int(randomInteger())
    _,err = model.UpdateQuantity(u7)
    if err != nil {
        t.Errorf(`failed UpdateQuantity(u7) %s`,err)
        return
    }

    if model.GetQuantity() != u7 {
        t.Errorf(`Position.GetQuantity() != u7 after UpdateQuantity`)
        return
    }
    model.Reload()
    if model.GetQuantity() != u7 {
        t.Errorf(`Position.GetQuantity() != u7 after Reload`)
        return
    }

};


func TestMysqlAdapterFromYAML(t *testing.T) {
    a := NewMysqlAdapter(`pw_`)
    y,err := fileGetContents(`test_data/adapter.yml`)
    if err != nil {
        t.Errorf(`failed to load yaml %s`,err)
        return
    }
    err = a.FromYAML(y)
    if err != nil {
        t.Errorf(`failed to apply yaml %s`,err)
        return
    }

    if (a.User != `root` ||
        a.Pass != `rootpass` ||
        a.Host != `localhost` ||
        a.Database != `my_db` ||
        a.DBPrefix != `wp_`) {
        t.Errorf(`did not fully apply yaml file %+v`,a)
    }
}
func TestAdapterFailures(t *testing.T) {
    _,err := NewMysqlAdapterEx(`file_that_does_not_exist123323`)
    if err == nil {
        t.Errorf(`Did not receive an error when file should not exist!`)
        return
    }
    // Load a nonsense yaml file
    _,err = NewMysqlAdapterEx(`test_data/nonsenseyaml.yml`)
    if err == nil {
        t.Errorf(`this should fail to load a nonsense yaml file`)
        return
    }
    // Load a test yaml with wrong Open
    _, err = NewMysqlAdapterEx(`test_data/adapter.yml`)
    if err == nil {
        t.Errorf(`this should fail with wrong login info`)
        return
    }
    // Load a silly yml file with wrong data
    _, err = NewMysqlAdapterEx(`test_data/silly.yml`)
    if err == nil {
        t.Errorf(`this should fail with wrong login info`)
        return
    }
}

func TestDBValue(t *testing.T) {
    a := NewMysqlAdapter(``)

    v0 := a.NewDBValue()
    v0.SetInternalValue(`x`,`999`)
    c0,err := v0.AsInt32()
    if err != nil {
        t.Errorf(`failed to convert with AsInt32() %+v`,v0)
        return
    }
    if c0 != 999 {
        t.Errorf(`values don't match `)
        return
    }

    v1 := a.NewDBValue()
    v1.SetInternalValue(`x`,`666`)
    c1,err := v1.AsInt()
    if err != nil {
        t.Errorf(`failed to convert with AsInt() %+v`,v1)
        return
    }
    if c1 != 666 {
        t.Errorf(`values don't match `)
        return
    }

    v2 := a.NewDBValue()
    v2.SetInternalValue(`x`,`hello world`)
    c2,err := v2.AsString()
    if err != nil {
        t.Errorf(`failed to convert with AsString() %+v`,v2)
        return
    }
    if c2 != "hello world" {
        t.Errorf(`values don't match `)
        return
    }

    v3 := a.NewDBValue()
    v3.SetInternalValue(`x`,`3.14`)
    c3,err := v3.AsFloat32()
    if err != nil {
        t.Errorf(`failed to convert with AsFloat32() %+v`,v3)
        return
    }
    if c3 != 3.14 {
        t.Errorf(`values don't match `)
        return
    }

    v4 := a.NewDBValue()
    v4.SetInternalValue(`x`,`67859.58686`)
    c4,err := v4.AsFloat64()
    if err != nil {
        t.Errorf(`failed to convert with AsFloat64() %+v`,v4)
        return
    }
    if c4 != 67859.58686 {
        t.Errorf(`values don't match `)
        return
    }

    dvar := a.NewDBValue()
    dvar.SetInternalValue(`x`,`2016-01-09 23:24:50`)
    dc,err := dvar.AsDateTime()
    if err != nil {
        t.Errorf(`failed to convert datetime %+v`,dc)
    }

    if (dc.Year != 2016 || 
        dc.Month != 1 ||
        dc.Day != 9 ||
        dc.Hours != 23 ||
        dc.Minutes != 24 ||
        dc.Seconds != 50 ) {
        t.Errorf(`fields don't match up for %+v`,dc)
    }
    r,_ := dvar.AsString()
    if dc.ToString() != r {
        t.Errorf(`restring of dvar failed %s`,dc.ToString())
    }

}

func TestAdapterInfoLogging(t *testing.T) {
    a := NewMysqlAdapter(``)
    var b bytes.Buffer
    r, err := regexp.Compile(`\[INFO\]:.+Hello World`)
    if err != nil {
        t.Errorf(`could not compile regex`)
        return
    }
    wr := bufio.NewWriter(&b)
    a.SetLogs(wr)
    a.LogInfo(`Hello World`)
    wr.Flush()
    if r.MatchString(b.String()) == false {
        t.Errorf(`failed to match info line`)
        return
    }
}
func TestAdapterEmptyInfoLogging(t *testing.T) {
    a := NewMysqlAdapter(``)
    var b bytes.Buffer
    wr := bufio.NewWriter(&b)
    a.SetLogs(wr)
    a.LogInfo(``)
    wr.Flush()
    if b.String() != `` {
        t.Errorf(`Info should not occur in this case`)
        return
    }
    a.SetLogFilter(func (tag string,val string) string {
        return ``
    })
    a.LogInfo(`Hello World`)
    wr.Flush()
    if b.String() != `` {
        t.Errorf(`Info should not occur due to filter in this case`)
        return
    }
}

func TestAdapterDebugLogging(t *testing.T) {
    a := NewMysqlAdapter(``)
    var b bytes.Buffer
    r, err := regexp.Compile(`\[DEBUG\]:.+Hello World`)
    if err != nil {
        t.Errorf(`could not compile regex`)
        return
    }
    wr := bufio.NewWriter(&b)
    a.SetLogs(wr)
    a.LogDebug(`Hello World`)
    wr.Flush()
    if r.MatchString(b.String()) == false {
        t.Errorf(`failed to match info line`)
        return
    }
}
func TestAdapterEmptyDebugLogging(t *testing.T) {
    a := NewMysqlAdapter(``)
    var b bytes.Buffer
    wr := bufio.NewWriter(&b)
    a.SetLogs(wr)
    a.LogDebug(``)
    wr.Flush()
    if b.String() != `` {
        t.Errorf(`Info should not occur in this case`)
        return
    }
    a.SetLogFilter(func (tag string,val string) string {
        return ``
    })
    a.LogDebug(`Hello World`)
    wr.Flush()
    if b.String() != `` {
        t.Errorf(`Info should not occur due to filter in this case`)
        return
    }
}

func TestAdapterErrorLogging(t *testing.T) {
    a := NewMysqlAdapter(``)
    
    r, err := regexp.Compile(`\[ERROR\]:.+Hello World`)
    if err != nil {
        t.Errorf(`could not compile regex`)
        return
    }
    var b bytes.Buffer
    wr := bufio.NewWriter(&b)
    a.SetLogs(wr)
    a.LogError(errors.New(`Hello World`))
    wr.Flush()
    if r.MatchString(b.String()) == false {
        t.Errorf(`failed to match info line`)
        return
    }

    var b2 bytes.Buffer
    wr2 := bufio.NewWriter(&b2)
    a.SetLogs(wr2)
    a.SetLogFilter(func (tag string,val string) string {
        return ``
    })
    a.LogError(errors.New(`Hello World`))
    wr2.Flush()
    if b2.String() != `` {
        t.Errorf(`Info should not occur due to filter in this case but equals %s`,b2.String())
        return
    }
}


