package main
import (
    "database/sql"
    "fmt"
    _ "github.com/go-sql-driver/mysql" // This is standard for this library.
    "strconv"
    "gopkg.in/yaml.v2"
    "regexp"
    "errors"
    "os"
    "io"
    "io/ioutil"
    "bufio"
    "log"
    "strings"
)



// LogFilter is an anonymous function that
// that receives the log tag and string and
// allows you to filter out extraneous lines
// when trying to find bugs.
type LogFilter func (string,string)string
// SafeStringFilter is the function that escapes
// possible SQL Injection code. 
type SafeStringFilter func(string)string
// Adapter is the main Database interface which helps
// to separate the DB from the Models. This is not
// 100% just yet, and may never be. Eventually the
// Adapter will probably receive some arguments and
// a value map and build the Query internally
type Adapter interface {
    Open(string,string,string,string) error
    Close()
    Query(string) ([]map[string]DBValue,error)
    Execute(string) error
    LastInsertedId() int64
    AffectedRows() int64
    DatabasePrefix() string
    LogInfo(string)
    LogError(error)
    LogDebug(string)
    SetLogs(io.Writer)
    SetLogFilter(LogFilter)
    Oops(string) error
    SafeString(string)string
    NewDBValue() DBValue
}


// MysqlAdapter is the MySql implementation
type MysqlAdapter struct {
    // The host, localhost is valid here, or 127.0.0.1
    // if you use localhost, the system won't use TCP
    Host string `yaml:"host"`
    // The database username
    User string `yaml:"user"`
    // The database password
    Pass string `yaml:"pass"`
    // The database name
    Database string `yaml:"database"`
    // A prefix, if any - can be blank
    DBPrefix string `yaml:"prefix"`
    _infoLog *log.Logger
    _errorLog *log.Logger
    _debugLog *log.Logger
    _conn *sql.DB
    _lid int64
    _cnt int64
    _opened bool
    _logFilter LogFilter
    _safeStringFilter SafeStringFilter
}
// NewMysqlAdapter returns a pointer to MysqlAdapter
func NewMysqlAdapter(pre string) *MysqlAdapter {
    return &MysqlAdapter{DBPrefix: pre}
} 
// NewMysqlAdapterEx sets everything up based on your YAML config
// Args: fname is a string path to a YAML config file
// This function will attempt to Open the database
// defined in that file. Example file:
//     host: "localhost"
//     user: "dbuser"
//     pass: "dbuserpass"
//     database: "my_db"
//     prefix: "wp_"
func NewMysqlAdapterEx(fname string) (*MysqlAdapter,error) {
    a := NewMysqlAdapter(``)
    y,err := fileGetContents(fname)
    if err != nil {
        return nil,err
    }
    err = a.FromYAML(y)
    if err != nil {
        return nil,err
    }
    err = a.Open(a.Host,a.User,a.Pass,a.Database)
    if err != nil {
        return nil,err
    }
    a.SetLogs(ioutil.Discard)
    return a,nil
}
// SetLogFilter sets the LogFilter to a function. This is only
// useful if you are debugging, or you want to
// reformat the log data.
func (a *MysqlAdapter) SetLogFilter(f LogFilter) {
    a._logFilter = f
}
// SafeString Not implemented yet, but soon.
func (a *MysqlAdapter) SafeString(s string) string {
    return s
}
// SetInfoLog Sets the _infoLog to the io.Writer, use ioutil.Discard if you
// don't want this one at all.
func (a *MysqlAdapter) SetInfoLog(t io.Writer) {
    a._infoLog = log.New(t,`[INFO]:`,log.Ldate|log.Ltime|log.Lshortfile)
}
// SetErrorLog Sets the _errorLog to the io.Writer, use ioutil.Discard if you
// don't want this one at all.
func (a *MysqlAdapter) SetErrorLog(t io.Writer) {
    a._errorLog = log.New(t,`[ERROR]:`,log.Ldate|log.Ltime|log.Lshortfile)
}
// SetDebugLog Sets the _debugLog to the io.Writer, use ioutil.Discard if you
// don't want this one at all.
func (a *MysqlAdapter) SetDebugLog(t io.Writer) {
    a._debugLog = log.New(t,`[DEBUG]:`,log.Ldate|log.Ltime|log.Lshortfile)
}
// SetLogs Sets ALL logs to the io.Writer, use ioutil.Discard if you
// don't want this one at all.
func (a *MysqlAdapter) SetLogs(t io.Writer) {
    a.SetInfoLog(t)
    a.SetErrorLog(t)
    a.SetDebugLog(t)
}
// LogInfo Tags the string with INFO and puts it into _infoLog.
func (a *MysqlAdapter) LogInfo(s string) {
    if a._logFilter != nil {
        s = a._logFilter(`INFO`,s)
    }
    if s == "" {
        return
    }
    a._infoLog.Println(s)
}
// LogError Tags the string with ERROR and puts it into _errorLog.
func (a *MysqlAdapter) LogError(s error) {
    if a._logFilter != nil {
        ns := a._logFilter(`ERROR`,fmt.Sprintf(`%s`,s))
        if ns == `` {
            return
        }
        a._errorLog.Println(ns)
        return
    }
    a._errorLog.Println(s)
}
// LogDebug Tags the string with DEBUG and puts it into _debugLog.
func (a *MysqlAdapter) LogDebug(s string) {
    if a._logFilter != nil {
        s = a._logFilter(`DEBUG`,s)
    }
    if s == "" {
        return
    }
    a._debugLog.Println(s)
}
// NewDBValue Creates a new DBValue, mostly used internally, but
// you may wish to use it in special circumstances.
func (a *MysqlAdapter) NewDBValue() DBValue {
    return NewMysqlValue(a)
}
// DatabasePrefix Get the DatabasePrefix from the Adapter
func (a *MysqlAdapter) DatabasePrefix() string {
    return a.DBPrefix
}
// FromYAML Set the Adapter's members from a YAML file
func (a *MysqlAdapter) FromYAML(b []byte) error {
    return yaml.Unmarshal(b,a)
}
// Open Opens the database connection. Be sure to use 
// a.Close() as closing is NOT handled for you.
func (a *MysqlAdapter) Open(h,u,p,d string) error {
    if ( h != "localhost") {
        l := fmt.Sprintf("%s:%s@tcp(%s)/%s",u,p,h,d)
        tc, err := sql.Open("mysql",l)
        if err != nil {
            return a.Oops(fmt.Sprintf(`%s with %s`,err,l))
        }
        a._conn = tc
    } else {
        l := fmt.Sprintf("%s:%s@/%s",u,p,d)
        tc, err := sql.Open("mysql",l)
        if err != nil {
            return a.Oops(fmt.Sprintf(`%s with %s`,err,l))
        }
        a._conn = tc
    }
    err := a._conn.Ping()
    if err != nil {
        return err
    }
    a._opened = true
    return nil

}
// Close This should be called in your application with a defer a.Close() 
// or something similar. Closing is not automatic!
func (a *MysqlAdapter) Close() {
    a._conn.Close()
}
// Query The generay Query function, i.e. SQL that returns results, as
// opposed to an INSERT or UPDATE which uses Execute.
func (a *MysqlAdapter) Query(q string) ([]map[string]DBValue,error) {
    if a._opened != true {
        return nil,a.Oops(`you must first open the connection`)
    }
    results := new([]map[string]DBValue)
    a.LogInfo(q)
    rows, err := a._conn.Query(q)
    if err != nil {
        return nil,err
    }
    defer rows.Close()
    columns, err := rows.Columns()
    if err != nil {
        return nil, err
    }
    values := make([]sql.RawBytes, len(columns))
    scanArgs := make([]interface{},len(values))
    for i := range values {
        scanArgs[i] = &values[i]
    }
    for rows.Next() {
        err = rows.Scan(scanArgs...)
        if err != nil {
            return nil,err
        }
        res := make(map[string]DBValue)
        for i,col := range values {
            k := columns[i]
            res[k] = a.NewDBValue()
            res[k].SetInternalValue(k,string(col))
        }
        *results = append(*results,res)
    }
    return *results,nil
}
// Oops A function for catching errors generated by
// the library and funneling them to the log files
func (a *MysqlAdapter) Oops(s string) error {
    e := errors.New(s)
    a.LogError(e)
    return e
}
// Execute For UPDATE and INSERT calls, i.e. nothing that
// returns a result set.
func (a *MysqlAdapter) Execute(q string) error {
    if a._opened != true {
        return a.Oops(`you must first open the connection`)
    }
    tx, err := a._conn.Begin()
    if err != nil {
        return a.Oops(fmt.Sprintf(`could not Begin Transaction %s`,err))
    }
    defer tx.Rollback();
    stmt, err := tx.Prepare(q)
    if err != nil {
        return a.Oops(fmt.Sprintf(`could not Prepare Statement %s`,err))
    }
    defer stmt.Close()
    a.LogInfo(q)
    res,err := stmt.Exec()
    if err != nil {
        return a.Oops(fmt.Sprintf(`could not Exec stmt %s`,err))
    }
    a._lid,err = res.LastInsertId()
    a.LogInfo(fmt.Sprintf(`LastInsertedId is %d`,a._lid))
    if err != nil {
        return a.Oops(fmt.Sprintf(`could not get LastInsertId %s`,err))
    }
    a._cnt,err = res.RowsAffected()
    if err != nil {
        return a.Oops(fmt.Sprintf(`could not get RowsAffected %s`,err))
    }
    err = tx.Commit()
    if err != nil {
        return a.Oops(fmt.Sprintf(`could not Commit Transaction %s`,err))
    }
    return nil
}
// LastInsertedId Grab the last auto_incremented id
func (a *MysqlAdapter) LastInsertedId() int64 {
    return a._lid
}
// AffectedRows Grab the number of AffectedRows
func (a *MysqlAdapter) AffectedRows() int64 {
    return a._cnt
}


// DBValue Provides a tidy way to convert string
// values from the DB into go values
type DBValue interface {
    AsInt() (int,error)
    AsInt32() (int32,error)
    AsInt64() (int64,error)
    AsFloat32() (float32,error)
    AsFloat64() (float64,error)
    AsString() (string,error)
    AsDateTime() (*DateTime,error)
    SetInternalValue(string,string)
}
// MysqlValue Implements DBValue for MySQL, you'll generally
// not interact directly with this type, but it
// is there for special cases.
type MysqlValue struct {
    _v string
    _k string
    _adapter Adapter
}
// SetInternalValue Sets the internal value of the DBValue to the string
// provided. key isn't really used, but it may be.
func (v *MysqlValue) SetInternalValue(key,value string) {
    v._v = value
    v._k = key

}
// AsString Simply returns the internal string representation.
func (v *MysqlValue) AsString() (string,error) {
    return v._v,nil
}
// AsInt Attempts to convert the internal string to an Int
func (v *MysqlValue) AsInt() (int,error) {
    i,err := strconv.ParseInt(v._v,10,32)
    return int(i),err
}
// AsInt32 Tries to convert the internal string to an int32
func (v *MysqlValue) AsInt32() (int32,error) {
    i,err := strconv.ParseInt(v._v,10,32)
    return int32(i),err
}
// AsInt64 Tries to convert the internal string to an int64 (i.e. BIGINT)
func (v *MysqlValue) AsInt64() (int64,error) {
    i,err := strconv.ParseInt(v._v,10,64)
    return i,err
}
// AsFloat32 Tries to convert the internal string to a float32
func (v *MysqlValue) AsFloat32() (float32,error) {
    i,err := strconv.ParseFloat(v._v,32)
    if err != nil {
        return 0.0,err
    }
    return float32(i),err
}
// AsFloat64 Tries to convert the internal string to a float64
func (v *MysqlValue) AsFloat64() (float64,error) {
    i,err := strconv.ParseFloat(v._v,64)
    if err != nil {
        return 0.0,err
    }
    return i,err
}
// AsDateTime Tries to convert the string to a DateTime,
// parsing may fail.
func (v *MysqlValue) AsDateTime() (*DateTime,error) {
    dt := NewDateTime(v._adapter)
    err := dt.FromString(v._v)
    if err != nil {
        return &DateTime{}, err
    }
    return dt,nil
}
// NewMysqlValue A function for largely internal use, but
// basically in order to use a DBValue, it 
// needs to have its Adapter setup, this is
// because some values have Adapter specific
// issues. The implementing adapter may need
// to provide some information, or logging etc
func NewMysqlValue(a Adapter) *MysqlValue {
    return &MysqlValue{_adapter: a}
}
// DateTime A simple struct to represent DateTime fields
type DateTime struct {
    // The day as an int
    Day int
    // the month, as an int
    Month int
    // The year, as an int
    Year int
    // the hours, in 24 hour format
    Hours int
    // the minutes
    Minutes int
    // the seconds
    Seconds int
    _adapter Adapter
}
// FromString Converts a string like 0000-00-00 00:00:00 into a DateTime
func (d *DateTime) FromString(s string) error {
    es := s
    re := regexp.MustCompile("(?P<year>[\\d]{4})-(?P<month>[\\d]{2})-(?P<day>[\\d]{2}) (?P<hours>[\\d]{2}):(?P<minutes>[\\d]{2}):(?P<seconds>[\\d]{2})")
    n1 := re.SubexpNames()
    ir2 := re.FindAllStringSubmatch(es, -1)
    if len(ir2) == 0 {
        return d._adapter.Oops(fmt.Sprintf("found no data to capture in %s",es))
    }
    r2 := ir2[0]
    for i, n := range r2 {
        if n1[i] == "year" {
            _Year,err := strconv.ParseInt(n,10,32)
            d.Year = int(_Year)
            if err != nil {
                return d._adapter.Oops(fmt.Sprintf("failed to convert %d in %v received %s",n[i],es,err))
            }
        }
        if n1[i] == "month" {
            _Month,err := strconv.ParseInt(n,10,32)
            d.Month = int(_Month)
            if err != nil {
                return d._adapter.Oops(fmt.Sprintf("failed to convert %d in %v received %s",n[i],es,err))
            }
        }
        if n1[i] == "day" {
            _Day,err := strconv.ParseInt(n,10,32)
            d.Day = int(_Day)
            if err != nil {
                return d._adapter.Oops(fmt.Sprintf("failed to convert %d in %v received %s",n[i],es,err))
            }
        }
        if n1[i] == "hours" {
            _Hours,err := strconv.ParseInt(n,10,32)
            d.Hours = int(_Hours)
            if err != nil {
                return d._adapter.Oops(fmt.Sprintf("failed to convert %d in %v received %s",n[i],es,err))
            }
        }
        if n1[i] == "minutes" {
            _Minutes,err := strconv.ParseInt(n,10,32)
            d.Minutes = int(_Minutes)
            if err != nil {
                return d._adapter.Oops(fmt.Sprintf("failed to convert %d in %v received %s",n[i],es,err))
            }
        }
        if n1[i] == "seconds" {
            _Seconds,err := strconv.ParseInt(n,10,32)
            d.Seconds = int(_Seconds)
            if err != nil {
                return d._adapter.Oops(fmt.Sprintf("failed to convert %d in %v received %s",n[i],es,err))
            }
        }
    }
    return nil
}
// ToString For backwards compat... Never use this, use String() instead.
func (d *DateTime) ToString() string {
    return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",d.Year,d.Month,d.Day,d.Hours,d.Minutes,d.Seconds)
}
// String The Stringer for DateTime to avoid having to call ToString all the time.
func (d *DateTime) String() string {
    return d.ToString()
}
// NewDateTime Returns a basic DateTime value
func NewDateTime(a Adapter) *DateTime {
    d := &DateTime{_adapter: a}
    return d
}
func fileExists(p string) bool {
    if _, err := os.Stat(p); os.IsNotExist(err) {
        return false
    }
    return true
}
func filePutContents(p string, txt string) error {
    f, err := os.Create(p)
    if err != nil {
        return err
    }
    w := bufio.NewWriter(f)
    _, err = w.WriteString(txt)
    w.Flush()
    return nil
}
func fileGetContents(p string) ([]byte, error) {
    return ioutil.ReadFile(p)
}

// Note is a Object Relational Mapping to
// the database table that represents it. In this case it is
// notes. The table name will be Sprintf'd to include
// the prefix you define in your YAML configuration for the
// Adapter.
type Note struct {
    _table string
    _adapter Adapter
    _pkey string // 0 The name of the primary key in this table
    _conds []string
    _new bool

    _select []string
    _where []string
    _cols []string
    _values []string
    _sets map[string]string
    _limit string
    _order string


    Id int64
    Value string
    PortfolioId int64
    PositionId int64
	// Dirty markers for smart updates
    IsIdDirty bool
    IsValueDirty bool
    IsPortfolioIdDirty bool
    IsPositionIdDirty bool
	// Relationships
}

// NewNote binds an Adapter to a new instance
// of Note and sets up the _table and primary keys
func NewNote(a Adapter) *Note {
    var o Note
    o._table = fmt.Sprintf("%snotes",a.DatabasePrefix())
    o._adapter = a
    o._pkey = "id"
    o._new = false
    return &o
}


// GetPrimaryKeyValue returns the value, usually int64 of
// the PrimaryKey
func (o *Note) GetPrimaryKeyValue() int64 {
    return o.Id
}
// GetPrimaryKeyName returns the DB field name
func (o *Note) GetPrimaryKeyName() string {
    return `id`
}

// GetId returns the value of 
// Note.Id
func (o *Note) GetId() int64 {
    return o.Id
}
// SetId sets and marks as dirty the value of
// Note.Id
func (o *Note) SetId(arg int64) {
    o.Id = arg
    o.IsIdDirty = true
}

// GetValue returns the value of 
// Note.Value
func (o *Note) GetValue() string {
    return o.Value
}
// SetValue sets and marks as dirty the value of
// Note.Value
func (o *Note) SetValue(arg string) {
    o.Value = arg
    o.IsValueDirty = true
}

// GetPortfolioId returns the value of 
// Note.PortfolioId
func (o *Note) GetPortfolioId() int64 {
    return o.PortfolioId
}
// SetPortfolioId sets and marks as dirty the value of
// Note.PortfolioId
func (o *Note) SetPortfolioId(arg int64) {
    o.PortfolioId = arg
    o.IsPortfolioIdDirty = true
}

// GetPositionId returns the value of 
// Note.PositionId
func (o *Note) GetPositionId() int64 {
    return o.PositionId
}
// SetPositionId sets and marks as dirty the value of
// Note.PositionId
func (o *Note) SetPositionId(arg int64) {
    o.PositionId = arg
    o.IsPositionIdDirty = true
}

// Find searchs against the database table field id and will return bool,error
// This method is a programatically generated finder for Note
//  
// Note that Find returns a bool of true|false if found or not, not err, in the case of
// found == true, the instance data will be filled out!
//
// A call to find ALWAYS overwrites the model you call Find on
// i.e. receiver is a pointer!
//
//```go
//      m := NewNote(a)
//      found,err := m.Find(23)
//      .. handle err
//      if found == false {
//          // handle found
//      }
//      ... do what you want with m here
//```
//
func (o *Note) Find(_findById int64) (bool,error) {

    var _modelSlice []*Note
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "id", _findById)
    results, err := o._adapter.Query(q)
    if err != nil {
        return false,o._adapter.Oops(fmt.Sprintf(`%s`,err))
    }
    
    for _,result := range results {
        ro := NewNote(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return false,o._adapter.Oops(fmt.Sprintf(`%s`,err))
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return false, o._adapter.Oops(`not found`)
    }
    o.FromNote(_modelSlice[0])
    return true,nil

}
// FindByValue searchs against the database table field value and will return []*Note,error
// This method is a programatically generated finder for Note
//
//```go  
//    m := NewNote(a)
//    results,err := m.FindByValue(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Note
//    }
//```  
//
func (o *Note) FindByValue(_findByValue string) ([]*Note,error) {

    var _modelSlice []*Note
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%s'",o._table, "value", _findByValue)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewNote(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByPortfolioId searchs against the database table field portfolio_id and will return []*Note,error
// This method is a programatically generated finder for Note
//
//```go  
//    m := NewNote(a)
//    results,err := m.FindByPortfolioId(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Note
//    }
//```  
//
func (o *Note) FindByPortfolioId(_findByPortfolioId int64) ([]*Note,error) {

    var _modelSlice []*Note
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "portfolio_id", _findByPortfolioId)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewNote(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByPositionId searchs against the database table field position_id and will return []*Note,error
// This method is a programatically generated finder for Note
//
//```go  
//    m := NewNote(a)
//    results,err := m.FindByPositionId(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Note
//    }
//```  
//
func (o *Note) FindByPositionId(_findByPositionId int64) ([]*Note,error) {

    var _modelSlice []*Note
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "position_id", _findByPositionId)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewNote(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}

// FromDBValueMap Converts a DBValueMap returned from Adapter.Query to a Note
func (o *Note) FromDBValueMap(m map[string]DBValue) error {
	_Id,err := m["id"].AsInt64()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Id = _Id
	_Value,err := m["value"].AsString()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Value = _Value
	_PortfolioId,err := m["portfolio_id"].AsInt64()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.PortfolioId = _PortfolioId
	_PositionId,err := m["position_id"].AsInt64()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.PositionId = _PositionId

 	return nil
}
// FromNote A kind of Clone function for Note
func (o *Note) FromNote(m *Note) {
	o.Id = m.Id
	o.Value = m.Value
	o.PortfolioId = m.PortfolioId
	o.PositionId = m.PositionId

}
// Reload A function to forcibly reload Note
func (o *Note) Reload() error {
    _,err := o.Find(o.GetPrimaryKeyValue())
    return err
}

// Save is a dynamic saver 'inherited' by all models
func (o *Note) Save() error {
    if o._new == true {
        return o.Create()
    }
    var sets []string
    
    if o.IsValueDirty == true {
        sets = append(sets,fmt.Sprintf(`value = '%s'`,o._adapter.SafeString(o.Value)))
    }

    if o.IsPortfolioIdDirty == true {
        sets = append(sets,fmt.Sprintf(`portfolio_id = '%d'`,o.PortfolioId))
    }

    if o.IsPositionIdDirty == true {
        sets = append(sets,fmt.Sprintf(`position_id = '%d'`,o.PositionId))
    }

    frmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%d'",o._table,strings.Join(sets,`,`),o._pkey, o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return err
    }
    return nil
}
// Update is a dynamic updater, it considers whether or not
// a field is 'dirty' and needs to be updated. Will only work
// if you use the Getters and Setters
func (o *Note) Update() error {
    var sets []string
    
    if o.IsValueDirty == true {
        sets = append(sets,fmt.Sprintf(`value = '%s'`,o._adapter.SafeString(o.Value)))
    }

    if o.IsPortfolioIdDirty == true {
        sets = append(sets,fmt.Sprintf(`portfolio_id = '%d'`,o.PortfolioId))
    }

    if o.IsPositionIdDirty == true {
        sets = append(sets,fmt.Sprintf(`position_id = '%d'`,o.PositionId))
    }

    frmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%d'",o._table,strings.Join(sets,`,`),o._pkey, o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return err
    }
    return nil
}
// Create inserts the model. Calling Save will call this function
// automatically for new models
func (o *Note) Create() error {
    frmt := fmt.Sprintf("INSERT INTO %s (`value`, `portfolio_id`, `position_id`) VALUES ('%s', '%d', '%d')",o._table,o.Value, o.PortfolioId, o.PositionId)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return o._adapter.Oops(fmt.Sprintf(`%s led to %s`,frmt,err))
    }
    o.Id = o._adapter.LastInsertedId()
    o._new = false
    return nil
}


// UpdateValue an immediate DB Query to update a single column, in this
// case value
func (o *Note) UpdateValue(_updValue string) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `value` = '%s' WHERE `id` = '%d'",o._table,_updValue,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Value = _updValue
    return o._adapter.AffectedRows(),nil
}

// UpdatePortfolioId an immediate DB Query to update a single column, in this
// case portfolio_id
func (o *Note) UpdatePortfolioId(_updPortfolioId int64) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `portfolio_id` = '%d' WHERE `id` = '%d'",o._table,_updPortfolioId,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.PortfolioId = _updPortfolioId
    return o._adapter.AffectedRows(),nil
}

// UpdatePositionId an immediate DB Query to update a single column, in this
// case position_id
func (o *Note) UpdatePositionId(_updPositionId int64) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `position_id` = '%d' WHERE `id` = '%d'",o._table,_updPositionId,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.PositionId = _updPositionId
    return o._adapter.AffectedRows(),nil
}

// Play is a Object Relational Mapping to
// the database table that represents it. In this case it is
// plays. The table name will be Sprintf'd to include
// the prefix you define in your YAML configuration for the
// Adapter.
type Play struct {
    _table string
    _adapter Adapter
    _pkey string // 0 The name of the primary key in this table
    _conds []string
    _new bool

    _select []string
    _where []string
    _cols []string
    _values []string
    _sets map[string]string
    _limit string
    _order string


    Id int64
    PositionId int64
    Day *DateTime
    Open int
    High int
    Low int
    Pvolume int
    Pchange int
    PchangePercent int
    AdjClose int
    DataSource string
	// Dirty markers for smart updates
    IsIdDirty bool
    IsPositionIdDirty bool
    IsDayDirty bool
    IsOpenDirty bool
    IsHighDirty bool
    IsLowDirty bool
    IsPvolumeDirty bool
    IsPchangeDirty bool
    IsPchangePercentDirty bool
    IsAdjCloseDirty bool
    IsDataSourceDirty bool
	// Relationships
}

// NewPlay binds an Adapter to a new instance
// of Play and sets up the _table and primary keys
func NewPlay(a Adapter) *Play {
    var o Play
    o._table = fmt.Sprintf("%splays",a.DatabasePrefix())
    o._adapter = a
    o._pkey = "id"
    o._new = false
    return &o
}


// GetPrimaryKeyValue returns the value, usually int64 of
// the PrimaryKey
func (o *Play) GetPrimaryKeyValue() int64 {
    return o.Id
}
// GetPrimaryKeyName returns the DB field name
func (o *Play) GetPrimaryKeyName() string {
    return `id`
}

// GetId returns the value of 
// Play.Id
func (o *Play) GetId() int64 {
    return o.Id
}
// SetId sets and marks as dirty the value of
// Play.Id
func (o *Play) SetId(arg int64) {
    o.Id = arg
    o.IsIdDirty = true
}

// GetPositionId returns the value of 
// Play.PositionId
func (o *Play) GetPositionId() int64 {
    return o.PositionId
}
// SetPositionId sets and marks as dirty the value of
// Play.PositionId
func (o *Play) SetPositionId(arg int64) {
    o.PositionId = arg
    o.IsPositionIdDirty = true
}

// GetDay returns the value of 
// Play.Day
func (o *Play) GetDay() *DateTime {
    return o.Day
}
// SetDay sets and marks as dirty the value of
// Play.Day
func (o *Play) SetDay(arg *DateTime) {
    o.Day = arg
    o.IsDayDirty = true
}

// GetOpen returns the value of 
// Play.Open
func (o *Play) GetOpen() int {
    return o.Open
}
// SetOpen sets and marks as dirty the value of
// Play.Open
func (o *Play) SetOpen(arg int) {
    o.Open = arg
    o.IsOpenDirty = true
}

// GetHigh returns the value of 
// Play.High
func (o *Play) GetHigh() int {
    return o.High
}
// SetHigh sets and marks as dirty the value of
// Play.High
func (o *Play) SetHigh(arg int) {
    o.High = arg
    o.IsHighDirty = true
}

// GetLow returns the value of 
// Play.Low
func (o *Play) GetLow() int {
    return o.Low
}
// SetLow sets and marks as dirty the value of
// Play.Low
func (o *Play) SetLow(arg int) {
    o.Low = arg
    o.IsLowDirty = true
}

// GetPvolume returns the value of 
// Play.Pvolume
func (o *Play) GetPvolume() int {
    return o.Pvolume
}
// SetPvolume sets and marks as dirty the value of
// Play.Pvolume
func (o *Play) SetPvolume(arg int) {
    o.Pvolume = arg
    o.IsPvolumeDirty = true
}

// GetPchange returns the value of 
// Play.Pchange
func (o *Play) GetPchange() int {
    return o.Pchange
}
// SetPchange sets and marks as dirty the value of
// Play.Pchange
func (o *Play) SetPchange(arg int) {
    o.Pchange = arg
    o.IsPchangeDirty = true
}

// GetPchangePercent returns the value of 
// Play.PchangePercent
func (o *Play) GetPchangePercent() int {
    return o.PchangePercent
}
// SetPchangePercent sets and marks as dirty the value of
// Play.PchangePercent
func (o *Play) SetPchangePercent(arg int) {
    o.PchangePercent = arg
    o.IsPchangePercentDirty = true
}

// GetAdjClose returns the value of 
// Play.AdjClose
func (o *Play) GetAdjClose() int {
    return o.AdjClose
}
// SetAdjClose sets and marks as dirty the value of
// Play.AdjClose
func (o *Play) SetAdjClose(arg int) {
    o.AdjClose = arg
    o.IsAdjCloseDirty = true
}

// GetDataSource returns the value of 
// Play.DataSource
func (o *Play) GetDataSource() string {
    return o.DataSource
}
// SetDataSource sets and marks as dirty the value of
// Play.DataSource
func (o *Play) SetDataSource(arg string) {
    o.DataSource = arg
    o.IsDataSourceDirty = true
}

// Find searchs against the database table field id and will return bool,error
// This method is a programatically generated finder for Play
//  
// Note that Find returns a bool of true|false if found or not, not err, in the case of
// found == true, the instance data will be filled out!
//
// A call to find ALWAYS overwrites the model you call Find on
// i.e. receiver is a pointer!
//
//```go
//      m := NewPlay(a)
//      found,err := m.Find(23)
//      .. handle err
//      if found == false {
//          // handle found
//      }
//      ... do what you want with m here
//```
//
func (o *Play) Find(_findById int64) (bool,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "id", _findById)
    results, err := o._adapter.Query(q)
    if err != nil {
        return false,o._adapter.Oops(fmt.Sprintf(`%s`,err))
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return false,o._adapter.Oops(fmt.Sprintf(`%s`,err))
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return false, o._adapter.Oops(`not found`)
    }
    o.FromPlay(_modelSlice[0])
    return true,nil

}
// FindByPositionId searchs against the database table field position_id and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByPositionId(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByPositionId(_findByPositionId int64) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "position_id", _findByPositionId)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByDay searchs against the database table field day and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByDay(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByDay(_findByDay *DateTime) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%s'",o._table, "day", _findByDay)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByOpen searchs against the database table field open and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByOpen(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByOpen(_findByOpen int) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "open", _findByOpen)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByHigh searchs against the database table field high and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByHigh(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByHigh(_findByHigh int) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "high", _findByHigh)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByLow searchs against the database table field low and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByLow(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByLow(_findByLow int) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "low", _findByLow)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByPvolume searchs against the database table field pvolume and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByPvolume(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByPvolume(_findByPvolume int) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "pvolume", _findByPvolume)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByPchange searchs against the database table field pchange and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByPchange(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByPchange(_findByPchange int) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "pchange", _findByPchange)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByPchangePercent searchs against the database table field pchange_percent and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByPchangePercent(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByPchangePercent(_findByPchangePercent int) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "pchange_percent", _findByPchangePercent)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByAdjClose searchs against the database table field adj_close and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByAdjClose(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByAdjClose(_findByAdjClose int) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "adj_close", _findByAdjClose)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByDataSource searchs against the database table field data_source and will return []*Play,error
// This method is a programatically generated finder for Play
//
//```go  
//    m := NewPlay(a)
//    results,err := m.FindByDataSource(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Play
//    }
//```  
//
func (o *Play) FindByDataSource(_findByDataSource string) ([]*Play,error) {

    var _modelSlice []*Play
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%s'",o._table, "data_source", _findByDataSource)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPlay(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}

// FromDBValueMap Converts a DBValueMap returned from Adapter.Query to a Play
func (o *Play) FromDBValueMap(m map[string]DBValue) error {
	_Id,err := m["id"].AsInt64()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Id = _Id
	_PositionId,err := m["position_id"].AsInt64()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.PositionId = _PositionId
	_Day,err := m["day"].AsDateTime()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Day = _Day
	_Open,err := m["open"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Open = _Open
	_High,err := m["high"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.High = _High
	_Low,err := m["low"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Low = _Low
	_Pvolume,err := m["pvolume"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Pvolume = _Pvolume
	_Pchange,err := m["pchange"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Pchange = _Pchange
	_PchangePercent,err := m["pchange_percent"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.PchangePercent = _PchangePercent
	_AdjClose,err := m["adj_close"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.AdjClose = _AdjClose
	_DataSource,err := m["data_source"].AsString()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.DataSource = _DataSource

 	return nil
}
// FromPlay A kind of Clone function for Play
func (o *Play) FromPlay(m *Play) {
	o.Id = m.Id
	o.PositionId = m.PositionId
	o.Day = m.Day
	o.Open = m.Open
	o.High = m.High
	o.Low = m.Low
	o.Pvolume = m.Pvolume
	o.Pchange = m.Pchange
	o.PchangePercent = m.PchangePercent
	o.AdjClose = m.AdjClose
	o.DataSource = m.DataSource

}
// Reload A function to forcibly reload Play
func (o *Play) Reload() error {
    _,err := o.Find(o.GetPrimaryKeyValue())
    return err
}

// Save is a dynamic saver 'inherited' by all models
func (o *Play) Save() error {
    if o._new == true {
        return o.Create()
    }
    var sets []string
    
    if o.IsPositionIdDirty == true {
        sets = append(sets,fmt.Sprintf(`position_id = '%d'`,o.PositionId))
    }

    if o.IsDayDirty == true {
        sets = append(sets,fmt.Sprintf(`day = '%s'`,o.Day))
    }

    if o.IsOpenDirty == true {
        sets = append(sets,fmt.Sprintf(`open = '%d'`,o.Open))
    }

    if o.IsHighDirty == true {
        sets = append(sets,fmt.Sprintf(`high = '%d'`,o.High))
    }

    if o.IsLowDirty == true {
        sets = append(sets,fmt.Sprintf(`low = '%d'`,o.Low))
    }

    if o.IsPvolumeDirty == true {
        sets = append(sets,fmt.Sprintf(`pvolume = '%d'`,o.Pvolume))
    }

    if o.IsPchangeDirty == true {
        sets = append(sets,fmt.Sprintf(`pchange = '%d'`,o.Pchange))
    }

    if o.IsPchangePercentDirty == true {
        sets = append(sets,fmt.Sprintf(`pchange_percent = '%d'`,o.PchangePercent))
    }

    if o.IsAdjCloseDirty == true {
        sets = append(sets,fmt.Sprintf(`adj_close = '%d'`,o.AdjClose))
    }

    if o.IsDataSourceDirty == true {
        sets = append(sets,fmt.Sprintf(`data_source = '%s'`,o._adapter.SafeString(o.DataSource)))
    }

    frmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%d'",o._table,strings.Join(sets,`,`),o._pkey, o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return err
    }
    return nil
}
// Update is a dynamic updater, it considers whether or not
// a field is 'dirty' and needs to be updated. Will only work
// if you use the Getters and Setters
func (o *Play) Update() error {
    var sets []string
    
    if o.IsPositionIdDirty == true {
        sets = append(sets,fmt.Sprintf(`position_id = '%d'`,o.PositionId))
    }

    if o.IsDayDirty == true {
        sets = append(sets,fmt.Sprintf(`day = '%s'`,o.Day))
    }

    if o.IsOpenDirty == true {
        sets = append(sets,fmt.Sprintf(`open = '%d'`,o.Open))
    }

    if o.IsHighDirty == true {
        sets = append(sets,fmt.Sprintf(`high = '%d'`,o.High))
    }

    if o.IsLowDirty == true {
        sets = append(sets,fmt.Sprintf(`low = '%d'`,o.Low))
    }

    if o.IsPvolumeDirty == true {
        sets = append(sets,fmt.Sprintf(`pvolume = '%d'`,o.Pvolume))
    }

    if o.IsPchangeDirty == true {
        sets = append(sets,fmt.Sprintf(`pchange = '%d'`,o.Pchange))
    }

    if o.IsPchangePercentDirty == true {
        sets = append(sets,fmt.Sprintf(`pchange_percent = '%d'`,o.PchangePercent))
    }

    if o.IsAdjCloseDirty == true {
        sets = append(sets,fmt.Sprintf(`adj_close = '%d'`,o.AdjClose))
    }

    if o.IsDataSourceDirty == true {
        sets = append(sets,fmt.Sprintf(`data_source = '%s'`,o._adapter.SafeString(o.DataSource)))
    }

    frmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%d'",o._table,strings.Join(sets,`,`),o._pkey, o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return err
    }
    return nil
}
// Create inserts the model. Calling Save will call this function
// automatically for new models
func (o *Play) Create() error {
    frmt := fmt.Sprintf("INSERT INTO %s (`position_id`, `day`, `open`, `high`, `low`, `pvolume`, `pchange`, `pchange_percent`, `adj_close`, `data_source`) VALUES ('%d', '%s', '%d', '%d', '%d', '%d', '%d', '%d', '%d', '%s')",o._table,o.PositionId, o.Day.ToString(), o.Open, o.High, o.Low, o.Pvolume, o.Pchange, o.PchangePercent, o.AdjClose, o.DataSource)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return o._adapter.Oops(fmt.Sprintf(`%s led to %s`,frmt,err))
    }
    o.Id = o._adapter.LastInsertedId()
    o._new = false
    return nil
}


// UpdatePositionId an immediate DB Query to update a single column, in this
// case position_id
func (o *Play) UpdatePositionId(_updPositionId int64) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `position_id` = '%d' WHERE `id` = '%d'",o._table,_updPositionId,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.PositionId = _updPositionId
    return o._adapter.AffectedRows(),nil
}

// UpdateDay an immediate DB Query to update a single column, in this
// case day
func (o *Play) UpdateDay(_updDay *DateTime) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `day` = '%s' WHERE `id` = '%d'",o._table,_updDay,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Day = _updDay
    return o._adapter.AffectedRows(),nil
}

// UpdateOpen an immediate DB Query to update a single column, in this
// case open
func (o *Play) UpdateOpen(_updOpen int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `open` = '%d' WHERE `id` = '%d'",o._table,_updOpen,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Open = _updOpen
    return o._adapter.AffectedRows(),nil
}

// UpdateHigh an immediate DB Query to update a single column, in this
// case high
func (o *Play) UpdateHigh(_updHigh int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `high` = '%d' WHERE `id` = '%d'",o._table,_updHigh,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.High = _updHigh
    return o._adapter.AffectedRows(),nil
}

// UpdateLow an immediate DB Query to update a single column, in this
// case low
func (o *Play) UpdateLow(_updLow int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `low` = '%d' WHERE `id` = '%d'",o._table,_updLow,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Low = _updLow
    return o._adapter.AffectedRows(),nil
}

// UpdatePvolume an immediate DB Query to update a single column, in this
// case pvolume
func (o *Play) UpdatePvolume(_updPvolume int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `pvolume` = '%d' WHERE `id` = '%d'",o._table,_updPvolume,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Pvolume = _updPvolume
    return o._adapter.AffectedRows(),nil
}

// UpdatePchange an immediate DB Query to update a single column, in this
// case pchange
func (o *Play) UpdatePchange(_updPchange int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `pchange` = '%d' WHERE `id` = '%d'",o._table,_updPchange,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Pchange = _updPchange
    return o._adapter.AffectedRows(),nil
}

// UpdatePchangePercent an immediate DB Query to update a single column, in this
// case pchange_percent
func (o *Play) UpdatePchangePercent(_updPchangePercent int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `pchange_percent` = '%d' WHERE `id` = '%d'",o._table,_updPchangePercent,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.PchangePercent = _updPchangePercent
    return o._adapter.AffectedRows(),nil
}

// UpdateAdjClose an immediate DB Query to update a single column, in this
// case adj_close
func (o *Play) UpdateAdjClose(_updAdjClose int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `adj_close` = '%d' WHERE `id` = '%d'",o._table,_updAdjClose,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.AdjClose = _updAdjClose
    return o._adapter.AffectedRows(),nil
}

// UpdateDataSource an immediate DB Query to update a single column, in this
// case data_source
func (o *Play) UpdateDataSource(_updDataSource string) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `data_source` = '%s' WHERE `id` = '%d'",o._table,_updDataSource,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.DataSource = _updDataSource
    return o._adapter.AffectedRows(),nil
}

// Portfolio is a Object Relational Mapping to
// the database table that represents it. In this case it is
// portfolios. The table name will be Sprintf'd to include
// the prefix you define in your YAML configuration for the
// Adapter.
type Portfolio struct {
    _table string
    _adapter Adapter
    _pkey string // 0 The name of the primary key in this table
    _conds []string
    _new bool

    _select []string
    _where []string
    _cols []string
    _values []string
    _sets map[string]string
    _limit string
    _order string


    Id int64
    Name string
    Description string
    Value int
	// Dirty markers for smart updates
    IsIdDirty bool
    IsNameDirty bool
    IsDescriptionDirty bool
    IsValueDirty bool
	// Relationships
}

// NewPortfolio binds an Adapter to a new instance
// of Portfolio and sets up the _table and primary keys
func NewPortfolio(a Adapter) *Portfolio {
    var o Portfolio
    o._table = fmt.Sprintf("%sportfolios",a.DatabasePrefix())
    o._adapter = a
    o._pkey = "id"
    o._new = false
    return &o
}


// GetPrimaryKeyValue returns the value, usually int64 of
// the PrimaryKey
func (o *Portfolio) GetPrimaryKeyValue() int64 {
    return o.Id
}
// GetPrimaryKeyName returns the DB field name
func (o *Portfolio) GetPrimaryKeyName() string {
    return `id`
}

// GetId returns the value of 
// Portfolio.Id
func (o *Portfolio) GetId() int64 {
    return o.Id
}
// SetId sets and marks as dirty the value of
// Portfolio.Id
func (o *Portfolio) SetId(arg int64) {
    o.Id = arg
    o.IsIdDirty = true
}

// GetName returns the value of 
// Portfolio.Name
func (o *Portfolio) GetName() string {
    return o.Name
}
// SetName sets and marks as dirty the value of
// Portfolio.Name
func (o *Portfolio) SetName(arg string) {
    o.Name = arg
    o.IsNameDirty = true
}

// GetDescription returns the value of 
// Portfolio.Description
func (o *Portfolio) GetDescription() string {
    return o.Description
}
// SetDescription sets and marks as dirty the value of
// Portfolio.Description
func (o *Portfolio) SetDescription(arg string) {
    o.Description = arg
    o.IsDescriptionDirty = true
}

// GetValue returns the value of 
// Portfolio.Value
func (o *Portfolio) GetValue() int {
    return o.Value
}
// SetValue sets and marks as dirty the value of
// Portfolio.Value
func (o *Portfolio) SetValue(arg int) {
    o.Value = arg
    o.IsValueDirty = true
}

// Find searchs against the database table field id and will return bool,error
// This method is a programatically generated finder for Portfolio
//  
// Note that Find returns a bool of true|false if found or not, not err, in the case of
// found == true, the instance data will be filled out!
//
// A call to find ALWAYS overwrites the model you call Find on
// i.e. receiver is a pointer!
//
//```go
//      m := NewPortfolio(a)
//      found,err := m.Find(23)
//      .. handle err
//      if found == false {
//          // handle found
//      }
//      ... do what you want with m here
//```
//
func (o *Portfolio) Find(_findById int64) (bool,error) {

    var _modelSlice []*Portfolio
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "id", _findById)
    results, err := o._adapter.Query(q)
    if err != nil {
        return false,o._adapter.Oops(fmt.Sprintf(`%s`,err))
    }
    
    for _,result := range results {
        ro := NewPortfolio(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return false,o._adapter.Oops(fmt.Sprintf(`%s`,err))
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return false, o._adapter.Oops(`not found`)
    }
    o.FromPortfolio(_modelSlice[0])
    return true,nil

}
// FindByName searchs against the database table field name and will return []*Portfolio,error
// This method is a programatically generated finder for Portfolio
//
//```go  
//    m := NewPortfolio(a)
//    results,err := m.FindByName(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Portfolio
//    }
//```  
//
func (o *Portfolio) FindByName(_findByName string) ([]*Portfolio,error) {

    var _modelSlice []*Portfolio
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%s'",o._table, "name", _findByName)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPortfolio(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByDescription searchs against the database table field description and will return []*Portfolio,error
// This method is a programatically generated finder for Portfolio
//
//```go  
//    m := NewPortfolio(a)
//    results,err := m.FindByDescription(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Portfolio
//    }
//```  
//
func (o *Portfolio) FindByDescription(_findByDescription string) ([]*Portfolio,error) {

    var _modelSlice []*Portfolio
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%s'",o._table, "description", _findByDescription)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPortfolio(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByValue searchs against the database table field value and will return []*Portfolio,error
// This method is a programatically generated finder for Portfolio
//
//```go  
//    m := NewPortfolio(a)
//    results,err := m.FindByValue(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Portfolio
//    }
//```  
//
func (o *Portfolio) FindByValue(_findByValue int) ([]*Portfolio,error) {

    var _modelSlice []*Portfolio
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "value", _findByValue)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPortfolio(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}

// FromDBValueMap Converts a DBValueMap returned from Adapter.Query to a Portfolio
func (o *Portfolio) FromDBValueMap(m map[string]DBValue) error {
	_Id,err := m["id"].AsInt64()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Id = _Id
	_Name,err := m["name"].AsString()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Name = _Name
	_Description,err := m["description"].AsString()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Description = _Description
	_Value,err := m["value"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Value = _Value

 	return nil
}
// FromPortfolio A kind of Clone function for Portfolio
func (o *Portfolio) FromPortfolio(m *Portfolio) {
	o.Id = m.Id
	o.Name = m.Name
	o.Description = m.Description
	o.Value = m.Value

}
// Reload A function to forcibly reload Portfolio
func (o *Portfolio) Reload() error {
    _,err := o.Find(o.GetPrimaryKeyValue())
    return err
}

// Save is a dynamic saver 'inherited' by all models
func (o *Portfolio) Save() error {
    if o._new == true {
        return o.Create()
    }
    var sets []string
    
    if o.IsNameDirty == true {
        sets = append(sets,fmt.Sprintf(`name = '%s'`,o._adapter.SafeString(o.Name)))
    }

    if o.IsDescriptionDirty == true {
        sets = append(sets,fmt.Sprintf(`description = '%s'`,o._adapter.SafeString(o.Description)))
    }

    if o.IsValueDirty == true {
        sets = append(sets,fmt.Sprintf(`value = '%d'`,o.Value))
    }

    frmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%d'",o._table,strings.Join(sets,`,`),o._pkey, o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return err
    }
    return nil
}
// Update is a dynamic updater, it considers whether or not
// a field is 'dirty' and needs to be updated. Will only work
// if you use the Getters and Setters
func (o *Portfolio) Update() error {
    var sets []string
    
    if o.IsNameDirty == true {
        sets = append(sets,fmt.Sprintf(`name = '%s'`,o._adapter.SafeString(o.Name)))
    }

    if o.IsDescriptionDirty == true {
        sets = append(sets,fmt.Sprintf(`description = '%s'`,o._adapter.SafeString(o.Description)))
    }

    if o.IsValueDirty == true {
        sets = append(sets,fmt.Sprintf(`value = '%d'`,o.Value))
    }

    frmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%d'",o._table,strings.Join(sets,`,`),o._pkey, o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return err
    }
    return nil
}
// Create inserts the model. Calling Save will call this function
// automatically for new models
func (o *Portfolio) Create() error {
    frmt := fmt.Sprintf("INSERT INTO %s (`name`, `description`, `value`) VALUES ('%s', '%s', '%d')",o._table,o.Name, o.Description, o.Value)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return o._adapter.Oops(fmt.Sprintf(`%s led to %s`,frmt,err))
    }
    o.Id = o._adapter.LastInsertedId()
    o._new = false
    return nil
}


// UpdateName an immediate DB Query to update a single column, in this
// case name
func (o *Portfolio) UpdateName(_updName string) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `name` = '%s' WHERE `id` = '%d'",o._table,_updName,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Name = _updName
    return o._adapter.AffectedRows(),nil
}

// UpdateDescription an immediate DB Query to update a single column, in this
// case description
func (o *Portfolio) UpdateDescription(_updDescription string) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `description` = '%s' WHERE `id` = '%d'",o._table,_updDescription,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Description = _updDescription
    return o._adapter.AffectedRows(),nil
}

// UpdateValue an immediate DB Query to update a single column, in this
// case value
func (o *Portfolio) UpdateValue(_updValue int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `value` = '%d' WHERE `id` = '%d'",o._table,_updValue,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Value = _updValue
    return o._adapter.AffectedRows(),nil
}

// Position is a Object Relational Mapping to
// the database table that represents it. In this case it is
// positions. The table name will be Sprintf'd to include
// the prefix you define in your YAML configuration for the
// Adapter.
type Position struct {
    _table string
    _adapter Adapter
    _pkey string // 0 The name of the primary key in this table
    _conds []string
    _new bool

    _select []string
    _where []string
    _cols []string
    _values []string
    _sets map[string]string
    _limit string
    _order string


    Id int64
    PortfolioId int64
    StartedAt *DateTime
    ClosedAt *DateTime
    Ptype string
    Buy int
    Sell int
    StopLoss int
    Quantity int
	// Dirty markers for smart updates
    IsIdDirty bool
    IsPortfolioIdDirty bool
    IsStartedAtDirty bool
    IsClosedAtDirty bool
    IsPtypeDirty bool
    IsBuyDirty bool
    IsSellDirty bool
    IsStopLossDirty bool
    IsQuantityDirty bool
	// Relationships
}

// NewPosition binds an Adapter to a new instance
// of Position and sets up the _table and primary keys
func NewPosition(a Adapter) *Position {
    var o Position
    o._table = fmt.Sprintf("%spositions",a.DatabasePrefix())
    o._adapter = a
    o._pkey = "id"
    o._new = false
    return &o
}


// GetPrimaryKeyValue returns the value, usually int64 of
// the PrimaryKey
func (o *Position) GetPrimaryKeyValue() int64 {
    return o.Id
}
// GetPrimaryKeyName returns the DB field name
func (o *Position) GetPrimaryKeyName() string {
    return `id`
}

// GetId returns the value of 
// Position.Id
func (o *Position) GetId() int64 {
    return o.Id
}
// SetId sets and marks as dirty the value of
// Position.Id
func (o *Position) SetId(arg int64) {
    o.Id = arg
    o.IsIdDirty = true
}

// GetPortfolioId returns the value of 
// Position.PortfolioId
func (o *Position) GetPortfolioId() int64 {
    return o.PortfolioId
}
// SetPortfolioId sets and marks as dirty the value of
// Position.PortfolioId
func (o *Position) SetPortfolioId(arg int64) {
    o.PortfolioId = arg
    o.IsPortfolioIdDirty = true
}

// GetStartedAt returns the value of 
// Position.StartedAt
func (o *Position) GetStartedAt() *DateTime {
    return o.StartedAt
}
// SetStartedAt sets and marks as dirty the value of
// Position.StartedAt
func (o *Position) SetStartedAt(arg *DateTime) {
    o.StartedAt = arg
    o.IsStartedAtDirty = true
}

// GetClosedAt returns the value of 
// Position.ClosedAt
func (o *Position) GetClosedAt() *DateTime {
    return o.ClosedAt
}
// SetClosedAt sets and marks as dirty the value of
// Position.ClosedAt
func (o *Position) SetClosedAt(arg *DateTime) {
    o.ClosedAt = arg
    o.IsClosedAtDirty = true
}

// GetPtype returns the value of 
// Position.Ptype
func (o *Position) GetPtype() string {
    return o.Ptype
}
// SetPtype sets and marks as dirty the value of
// Position.Ptype
func (o *Position) SetPtype(arg string) {
    o.Ptype = arg
    o.IsPtypeDirty = true
}

// GetBuy returns the value of 
// Position.Buy
func (o *Position) GetBuy() int {
    return o.Buy
}
// SetBuy sets and marks as dirty the value of
// Position.Buy
func (o *Position) SetBuy(arg int) {
    o.Buy = arg
    o.IsBuyDirty = true
}

// GetSell returns the value of 
// Position.Sell
func (o *Position) GetSell() int {
    return o.Sell
}
// SetSell sets and marks as dirty the value of
// Position.Sell
func (o *Position) SetSell(arg int) {
    o.Sell = arg
    o.IsSellDirty = true
}

// GetStopLoss returns the value of 
// Position.StopLoss
func (o *Position) GetStopLoss() int {
    return o.StopLoss
}
// SetStopLoss sets and marks as dirty the value of
// Position.StopLoss
func (o *Position) SetStopLoss(arg int) {
    o.StopLoss = arg
    o.IsStopLossDirty = true
}

// GetQuantity returns the value of 
// Position.Quantity
func (o *Position) GetQuantity() int {
    return o.Quantity
}
// SetQuantity sets and marks as dirty the value of
// Position.Quantity
func (o *Position) SetQuantity(arg int) {
    o.Quantity = arg
    o.IsQuantityDirty = true
}

// Find searchs against the database table field id and will return bool,error
// This method is a programatically generated finder for Position
//  
// Note that Find returns a bool of true|false if found or not, not err, in the case of
// found == true, the instance data will be filled out!
//
// A call to find ALWAYS overwrites the model you call Find on
// i.e. receiver is a pointer!
//
//```go
//      m := NewPosition(a)
//      found,err := m.Find(23)
//      .. handle err
//      if found == false {
//          // handle found
//      }
//      ... do what you want with m here
//```
//
func (o *Position) Find(_findById int64) (bool,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "id", _findById)
    results, err := o._adapter.Query(q)
    if err != nil {
        return false,o._adapter.Oops(fmt.Sprintf(`%s`,err))
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return false,o._adapter.Oops(fmt.Sprintf(`%s`,err))
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return false, o._adapter.Oops(`not found`)
    }
    o.FromPosition(_modelSlice[0])
    return true,nil

}
// FindByPortfolioId searchs against the database table field portfolio_id and will return []*Position,error
// This method is a programatically generated finder for Position
//
//```go  
//    m := NewPosition(a)
//    results,err := m.FindByPortfolioId(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Position
//    }
//```  
//
func (o *Position) FindByPortfolioId(_findByPortfolioId int64) ([]*Position,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "portfolio_id", _findByPortfolioId)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByStartedAt searchs against the database table field started_at and will return []*Position,error
// This method is a programatically generated finder for Position
//
//```go  
//    m := NewPosition(a)
//    results,err := m.FindByStartedAt(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Position
//    }
//```  
//
func (o *Position) FindByStartedAt(_findByStartedAt *DateTime) ([]*Position,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%s'",o._table, "started_at", _findByStartedAt)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByClosedAt searchs against the database table field closed_at and will return []*Position,error
// This method is a programatically generated finder for Position
//
//```go  
//    m := NewPosition(a)
//    results,err := m.FindByClosedAt(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Position
//    }
//```  
//
func (o *Position) FindByClosedAt(_findByClosedAt *DateTime) ([]*Position,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%s'",o._table, "closed_at", _findByClosedAt)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByPtype searchs against the database table field ptype and will return []*Position,error
// This method is a programatically generated finder for Position
//
//```go  
//    m := NewPosition(a)
//    results,err := m.FindByPtype(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Position
//    }
//```  
//
func (o *Position) FindByPtype(_findByPtype string) ([]*Position,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%s'",o._table, "ptype", _findByPtype)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByBuy searchs against the database table field buy and will return []*Position,error
// This method is a programatically generated finder for Position
//
//```go  
//    m := NewPosition(a)
//    results,err := m.FindByBuy(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Position
//    }
//```  
//
func (o *Position) FindByBuy(_findByBuy int) ([]*Position,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "buy", _findByBuy)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindBySell searchs against the database table field sell and will return []*Position,error
// This method is a programatically generated finder for Position
//
//```go  
//    m := NewPosition(a)
//    results,err := m.FindBySell(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Position
//    }
//```  
//
func (o *Position) FindBySell(_findBySell int) ([]*Position,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "sell", _findBySell)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByStopLoss searchs against the database table field stop_loss and will return []*Position,error
// This method is a programatically generated finder for Position
//
//```go  
//    m := NewPosition(a)
//    results,err := m.FindByStopLoss(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Position
//    }
//```  
//
func (o *Position) FindByStopLoss(_findByStopLoss int) ([]*Position,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "stop_loss", _findByStopLoss)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}
// FindByQuantity searchs against the database table field quantity and will return []*Position,error
// This method is a programatically generated finder for Position
//
//```go  
//    m := NewPosition(a)
//    results,err := m.FindByQuantity(...)
//    // handle err
//    for i,r := results {
//      // now r is an instance of Position
//    }
//```  
//
func (o *Position) FindByQuantity(_findByQuantity int) ([]*Position,error) {

    var _modelSlice []*Position
    q := fmt.Sprintf("SELECT * FROM %s WHERE `%s` = '%d'",o._table, "quantity", _findByQuantity)
    results, err := o._adapter.Query(q)
    if err != nil {
        return _modelSlice,err
    }
    
    for _,result := range results {
        ro := NewPosition(o._adapter)
        err = ro.FromDBValueMap(result)
        if err != nil {
            return _modelSlice,err
        }
        _modelSlice = append(_modelSlice,ro)
    }

    if len(_modelSlice) == 0 {
        // there was an error!
        return nil, o._adapter.Oops(`no results`)
    }
    return _modelSlice,nil

}

// FromDBValueMap Converts a DBValueMap returned from Adapter.Query to a Position
func (o *Position) FromDBValueMap(m map[string]DBValue) error {
	_Id,err := m["id"].AsInt64()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Id = _Id
	_PortfolioId,err := m["portfolio_id"].AsInt64()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.PortfolioId = _PortfolioId
	_StartedAt,err := m["started_at"].AsDateTime()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.StartedAt = _StartedAt
	_ClosedAt,err := m["closed_at"].AsDateTime()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.ClosedAt = _ClosedAt
	_Ptype,err := m["ptype"].AsString()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Ptype = _Ptype
	_Buy,err := m["buy"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Buy = _Buy
	_Sell,err := m["sell"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Sell = _Sell
	_StopLoss,err := m["stop_loss"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.StopLoss = _StopLoss
	_Quantity,err := m["quantity"].AsInt()
	if err != nil {
 		return o._adapter.Oops(fmt.Sprintf(`%s`,err))
	}
	o.Quantity = _Quantity

 	return nil
}
// FromPosition A kind of Clone function for Position
func (o *Position) FromPosition(m *Position) {
	o.Id = m.Id
	o.PortfolioId = m.PortfolioId
	o.StartedAt = m.StartedAt
	o.ClosedAt = m.ClosedAt
	o.Ptype = m.Ptype
	o.Buy = m.Buy
	o.Sell = m.Sell
	o.StopLoss = m.StopLoss
	o.Quantity = m.Quantity

}
// Reload A function to forcibly reload Position
func (o *Position) Reload() error {
    _,err := o.Find(o.GetPrimaryKeyValue())
    return err
}

// Save is a dynamic saver 'inherited' by all models
func (o *Position) Save() error {
    if o._new == true {
        return o.Create()
    }
    var sets []string
    
    if o.IsPortfolioIdDirty == true {
        sets = append(sets,fmt.Sprintf(`portfolio_id = '%d'`,o.PortfolioId))
    }

    if o.IsStartedAtDirty == true {
        sets = append(sets,fmt.Sprintf(`started_at = '%s'`,o.StartedAt))
    }

    if o.IsClosedAtDirty == true {
        sets = append(sets,fmt.Sprintf(`closed_at = '%s'`,o.ClosedAt))
    }

    if o.IsPtypeDirty == true {
        sets = append(sets,fmt.Sprintf(`ptype = '%s'`,o._adapter.SafeString(o.Ptype)))
    }

    if o.IsBuyDirty == true {
        sets = append(sets,fmt.Sprintf(`buy = '%d'`,o.Buy))
    }

    if o.IsSellDirty == true {
        sets = append(sets,fmt.Sprintf(`sell = '%d'`,o.Sell))
    }

    if o.IsStopLossDirty == true {
        sets = append(sets,fmt.Sprintf(`stop_loss = '%d'`,o.StopLoss))
    }

    if o.IsQuantityDirty == true {
        sets = append(sets,fmt.Sprintf(`quantity = '%d'`,o.Quantity))
    }

    frmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%d'",o._table,strings.Join(sets,`,`),o._pkey, o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return err
    }
    return nil
}
// Update is a dynamic updater, it considers whether or not
// a field is 'dirty' and needs to be updated. Will only work
// if you use the Getters and Setters
func (o *Position) Update() error {
    var sets []string
    
    if o.IsPortfolioIdDirty == true {
        sets = append(sets,fmt.Sprintf(`portfolio_id = '%d'`,o.PortfolioId))
    }

    if o.IsStartedAtDirty == true {
        sets = append(sets,fmt.Sprintf(`started_at = '%s'`,o.StartedAt))
    }

    if o.IsClosedAtDirty == true {
        sets = append(sets,fmt.Sprintf(`closed_at = '%s'`,o.ClosedAt))
    }

    if o.IsPtypeDirty == true {
        sets = append(sets,fmt.Sprintf(`ptype = '%s'`,o._adapter.SafeString(o.Ptype)))
    }

    if o.IsBuyDirty == true {
        sets = append(sets,fmt.Sprintf(`buy = '%d'`,o.Buy))
    }

    if o.IsSellDirty == true {
        sets = append(sets,fmt.Sprintf(`sell = '%d'`,o.Sell))
    }

    if o.IsStopLossDirty == true {
        sets = append(sets,fmt.Sprintf(`stop_loss = '%d'`,o.StopLoss))
    }

    if o.IsQuantityDirty == true {
        sets = append(sets,fmt.Sprintf(`quantity = '%d'`,o.Quantity))
    }

    frmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = '%d'",o._table,strings.Join(sets,`,`),o._pkey, o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return err
    }
    return nil
}
// Create inserts the model. Calling Save will call this function
// automatically for new models
func (o *Position) Create() error {
    frmt := fmt.Sprintf("INSERT INTO %s (`portfolio_id`, `started_at`, `closed_at`, `ptype`, `buy`, `sell`, `stop_loss`, `quantity`) VALUES ('%d', '%s', '%s', '%s', '%d', '%d', '%d', '%d')",o._table,o.PortfolioId, o.StartedAt.ToString(), o.ClosedAt.ToString(), o.Ptype, o.Buy, o.Sell, o.StopLoss, o.Quantity)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return o._adapter.Oops(fmt.Sprintf(`%s led to %s`,frmt,err))
    }
    o.Id = o._adapter.LastInsertedId()
    o._new = false
    return nil
}


// UpdatePortfolioId an immediate DB Query to update a single column, in this
// case portfolio_id
func (o *Position) UpdatePortfolioId(_updPortfolioId int64) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `portfolio_id` = '%d' WHERE `id` = '%d'",o._table,_updPortfolioId,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.PortfolioId = _updPortfolioId
    return o._adapter.AffectedRows(),nil
}

// UpdateStartedAt an immediate DB Query to update a single column, in this
// case started_at
func (o *Position) UpdateStartedAt(_updStartedAt *DateTime) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `started_at` = '%s' WHERE `id` = '%d'",o._table,_updStartedAt,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.StartedAt = _updStartedAt
    return o._adapter.AffectedRows(),nil
}

// UpdateClosedAt an immediate DB Query to update a single column, in this
// case closed_at
func (o *Position) UpdateClosedAt(_updClosedAt *DateTime) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `closed_at` = '%s' WHERE `id` = '%d'",o._table,_updClosedAt,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.ClosedAt = _updClosedAt
    return o._adapter.AffectedRows(),nil
}

// UpdatePtype an immediate DB Query to update a single column, in this
// case ptype
func (o *Position) UpdatePtype(_updPtype string) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `ptype` = '%s' WHERE `id` = '%d'",o._table,_updPtype,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Ptype = _updPtype
    return o._adapter.AffectedRows(),nil
}

// UpdateBuy an immediate DB Query to update a single column, in this
// case buy
func (o *Position) UpdateBuy(_updBuy int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `buy` = '%d' WHERE `id` = '%d'",o._table,_updBuy,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Buy = _updBuy
    return o._adapter.AffectedRows(),nil
}

// UpdateSell an immediate DB Query to update a single column, in this
// case sell
func (o *Position) UpdateSell(_updSell int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `sell` = '%d' WHERE `id` = '%d'",o._table,_updSell,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Sell = _updSell
    return o._adapter.AffectedRows(),nil
}

// UpdateStopLoss an immediate DB Query to update a single column, in this
// case stop_loss
func (o *Position) UpdateStopLoss(_updStopLoss int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `stop_loss` = '%d' WHERE `id` = '%d'",o._table,_updStopLoss,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.StopLoss = _updStopLoss
    return o._adapter.AffectedRows(),nil
}

// UpdateQuantity an immediate DB Query to update a single column, in this
// case quantity
func (o *Position) UpdateQuantity(_updQuantity int) (int64,error) {
    frmt := fmt.Sprintf("UPDATE %s SET `quantity` = '%d' WHERE `id` = '%d'",o._table,_updQuantity,o.Id)
    err := o._adapter.Execute(frmt)
    if err != nil {
        return 0,err
    }
    o.Quantity = _updQuantity
    return o._adapter.AffectedRows(),nil
}

