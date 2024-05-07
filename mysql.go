package gosf

/**
单条数据查询Get()方法
AccountData := DB("base").Table("auth_account").Where("phone=?", PhoneNumber).Get()
多条数据查询All()方法
UserData := DB("base").Table("auth_user").Where("account_id=? AND type=? AND status<>?", AccountId, UserType, 2).Select( "account_id", "id AS user_id", "last_login", "real_pwd", "status").Get()
*/
import (
	"database/sql"
	"fmt"
	"github.com/oyjz/gosf/config"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// 设置DB全部变量
var once sync.Once
var db *sql.DB
var dbMap map[string]*sql.DB

type Mysql struct {
}

// DbPool 数据库操作处理结构体
type DbPool struct {
	pool            *sql.DB                  `数据库连接池`
	tx              *sql.Tx                  `事务`
	tableName       string                   `数据表名字`
	selectCondition []string                 `选择条件`
	whereCondition  []map[string]interface{} `查询条件`
	groupCondition  []string                 `分组条件`
	orderCondition  []string                 `排序条件`
	lastSql         string
	limit           int
	page            int
}

// NewMysql 单例获取数据库实例
func NewMysql(config config.Config) *Mysql {
	once.Do(func() {
		dbMap = make(map[string]*sql.DB)

		var item map[string]string

		err := config.GetAs("mysql", &item)
		PanicErr(err, "config get error")

		for name, addr := range item {
			db, err = sql.Open("mysql", addr)
			PanicErr(err, "sql open error")
			// 设置连接池最大链接数--不能大于数据库设置的最大链接数
			db.SetMaxOpenConns(1000)
			// 设置最大空闲链接数--小于设置的链接数
			db.SetMaxIdleConns(10)
			// 设置数据库链接超时时间--不能大于数据库设置的超时时间
			db.SetConnMaxLifetime(time.Second * 5)
			dbMap[name] = db
		}
	})

	return &Mysql{}
}

// DB 单例获取数据库实例
func (p *Mysql) DB(name string) *DbPool {
	return &DbPool{
		pool: dbMap[name],
	}
}

func (p *DbPool) GetPool() *sql.DB {
	return p.pool
}

// Table 查询数据表获取
func (p *DbPool) Table(name string) *DbPool {
	p.tableName = name
	return p
}

// LastSql 获取最后执行SQL
func (p *DbPool) LastSql() string {
	return p.lastSql
}

// Select 查询select条件入参,入参类似python的args
func (p *DbPool) Select(params ...string) *DbPool {
	p.selectCondition = params
	return p
}

// Where 查询where条件入参,入参类似于python的args
func (p *DbPool) Where(query interface{}, values ...interface{}) *DbPool {
	p.whereCondition = append(p.whereCondition, map[string]interface{}{"query": query, "args": values})
	return p
}

// GroupBy 定义数据库分组函数,入参类似于python的args
func (p *DbPool) GroupBy(params ...string) *DbPool {
	p.groupCondition = params
	return p
}

// Limit
func (p *DbPool) Limit(limit int) *DbPool {
	p.limit = limit
	return p
}

// Page
func (p *DbPool) Page(page int) *DbPool {
	p.page = page
	return p
}

// Tx 设置事务
func (p *DbPool) Tx(tx *sql.Tx) *DbPool {
	p.tx = tx
	return p
}

// OrderBy 定义数据库排序函数,入参类似于python的args
func (p *DbPool) OrderBy(params ...string) *DbPool {
	p.orderCondition = params
	return p
}

// SQL拼接处理
func (p *DbPool) sql() string {
	// 处理select条件
	SelectFilter := strings.Join(p.selectCondition, ",")
	// 没有设置获取数据字段,默认查询全部
	if len(p.selectCondition) == 0 {
		SelectFilter = "*"
	}
	// 处理where条件
	WhereFilter := p.handlerWhere()
	// 处理分组条件
	GroupFilter := strings.Join(p.groupCondition, ",")
	// 处理排序条件
	OrderFilter := strings.Join(p.orderCondition, ",")
	if len(OrderFilter) > 0 {
		OrderFilter = " ORDER BY " + OrderFilter
	}
	LimitFileter := ""
	if p.limit > 0 {
		if p.page == 0 {
			p.page = 1
		}
		LimitFileter = fmt.Sprintf(" LIMIT %d, %d", (p.page-1)*p.limit, p.limit)
	}
	// 格式化生成SQL
	Sql := fmt.Sprintf("SELECT %v FROM `%v` %v %v %v %s", SelectFilter, p.tableName, WhereFilter, GroupFilter, OrderFilter, LimitFileter)
	return Sql
}

// 数据库返回数据处理,返回数据类型为slice,slice内层为map
func dealMysqlRows(rows *sql.Rows) []map[string]interface{} {
	defer closeRows(rows)
	// 获取列名
	columns, err := rows.Columns()
	PanicErr(err, "rows columns error")
	columnTypes, _ := rows.ColumnTypes()
	// 获取每列的数据类型
	ColumnTypeMap := make(map[string]string)
	for _, v := range columnTypes {
		ColumnTypeMap[v.Name()] = v.DatabaseTypeName()
	}
	// 定义返回参数的slice
	retValues := make([]sql.RawBytes, len(columns))
	// 定义数据列名的slice
	scanArgs := make([]interface{}, len(retValues))
	// 数据列赋值
	for i := range retValues {
		scanArgs[i] = &retValues[i]
	}
	// 定义返回数据类型slice
	var resList []map[string]interface{}
	// 返回数据赋值
	for rows.Next() {
		// 检测数据列是否超出
		err = rows.Scan(scanArgs...)
		PanicErr(err, "rows scan error")
		// 内层数据格式
		rowMap := make(map[string]interface{})
		for i, colVal := range retValues {
			if colVal != nil {
				keyName := columns[i]
				value := string(colVal)

				typeName := ColumnTypeMap[keyName]
				if strings.Contains(typeName, "INT") {
					newValue, _ := strconv.Atoi(value)
					rowMap[keyName] = newValue
				} else if strings.Contains(typeName, "DECIMAL") {
					newValue, _ := strconv.ParseFloat(value, 64)
					rowMap[keyName] = newValue
				} else {
					rowMap[keyName] = value
				}
			}
		}
		resList = append(resList, rowMap)
	}
	return resList
}

// Get 获取第一条数据,返回数据类型为map
func (p *DbPool) Get() map[string]interface{} {
	defer p.pool.Close()
	var RetOne map[string]interface{}
	p.Limit(1)
	GetSql := p.sql()
	p.lastSql = GetSql
	rows, err := p.pool.Query(GetSql)
	defer rows.Close()
	PanicErr(err, "query get error")
	// 数据获取
	RetMap := dealMysqlRows(rows)
	if len(RetMap) > 0 {
		RetOne = RetMap[0]
	}
	return RetOne
}

// All 获取多条数据,返回数据类型为slice,slice内层为map
func (p *DbPool) All() []map[string]interface{} {
	defer p.pool.Close()
	GetSql := p.sql()
	p.lastSql = GetSql
	rows, err := p.pool.Query(GetSql)
	defer rows.Close()
	PanicErr(err, "query all error")
	// 数据获取
	RetMap := dealMysqlRows(rows)
	return RetMap
}

// Insert 定义创建数据方法,返回最后的ID
func (p *DbPool) Insert(params map[string]interface{}) (lastId int, err error) {
	defer p.pool.Close()
	// defer func() {
	// 	fmt.Println(p.lastSql)
	// }()
	// 自定待创建的函数和参数
	InsertCols, InsertArgs := "", ""
	for k, v := range params {
		// 数据列只能为string类型
		if InsertCols == "" {
			InsertCols += fmt.Sprintf("%s", k)
		} else {
			InsertCols += fmt.Sprintf(",%s", k)
		}
		// 判断数据类型,类型断言判断
		switch v.(type) {
		case int:
			if InsertArgs == "" {
				InsertArgs += fmt.Sprintf("%d", v)
			} else {
				InsertArgs += fmt.Sprintf(",%d", v)
			}
		case string:
			temp := strings.Replace(v.(string), "'", "\\'", -1)
			if InsertArgs == "" {
				InsertArgs += fmt.Sprintf("'%s'", temp)
			} else {
				InsertArgs += fmt.Sprintf(",'%s'", temp)
			}
		case float64:
			if InsertArgs == "" {
				InsertArgs += fmt.Sprintf("%f", v)
			} else {
				InsertArgs += fmt.Sprintf(",%f", v)
			}
		default:
			if InsertArgs == "" {
				InsertArgs += fmt.Sprintf("%v", v)
			} else {
				InsertArgs += fmt.Sprintf(",%v", v)
			}
		}
	}
	// 组合数据写入SQL
	InsertSql := fmt.Sprintf("INSERT INTO `%v` (%v) VALUES (%v);", p.tableName, InsertCols, InsertArgs)
	p.lastSql = InsertSql

	// 执行，判断是否存在事务
	var retData sql.Result
	if p.tx == nil {
		retData, err = p.pool.Exec(InsertSql)
	} else {
		retData, err = p.tx.Exec(InsertSql)
	}

	if err != nil {
		return 0, err
	}
	LastId, err := retData.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(LastId), err
}

// Update 定义更新数据方法,返回影响的行数
func (p *DbPool) Update(params map[string]interface{}) (affectRows int, err error) {
	defer p.pool.Close()
	// defer func() {
	// 	fmt.Println(p.lastSql)
	// }()
	// 处理where条件
	WhereFilter := p.handlerWhere()
	// 定义待创建的函数和参数
	UpdateArgs := ""
	for k, v := range params {
		// 数据列只能为string类型
		if UpdateArgs == "" {
			// 判断数据类型,类型断言判断
			switch v.(type) {
			case int:
				UpdateArgs += fmt.Sprintf("%s=%d", k, v)
			case string:
				temp := strings.Replace(v.(string), "'", "\\'", -1)
				UpdateArgs += fmt.Sprintf("%s='%s'", k, temp)
			case float64:
				UpdateArgs += fmt.Sprintf("%s=%f", k, v)
			default:
				UpdateArgs += fmt.Sprintf("%v=%v", k, v)
			}
		} else {
			// 判断数据类型,类型断言判断
			switch v.(type) {
			case int:
				UpdateArgs += fmt.Sprintf(",%s=%d", k, v)
			case string:
				temp := strings.Replace(v.(string), "'", "\\'", -1)
				UpdateArgs += fmt.Sprintf(",%s='%s'", k, temp)
			case float64:
				UpdateArgs += fmt.Sprintf(",%s=%f", k, v)
			default:
				UpdateArgs += fmt.Sprintf(",%v=%v", k, v)
			}
		}
	}
	// 组合数据更新SQL
	UpdateSql := fmt.Sprintf("UPDATE `%v` SET %v %v;", p.tableName, UpdateArgs, WhereFilter)

	p.lastSql = UpdateSql
	// 执行，判断是否存在事务
	var retData sql.Result
	if p.tx == nil {
		retData, err = p.pool.Exec(UpdateSql)
	} else {
		retData, err = p.tx.Exec(UpdateSql)
	}
	if err != nil {
		return 0, err
	}
	ARows, err := retData.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(ARows), nil
}

// 处理where条件
func (p *DbPool) handlerWhere() string {
	// 匿名函数interface转slice--需要时调用
	fn := func(arr interface{}) []interface{} {
		v := reflect.ValueOf(arr)
		if v.Kind() != reflect.Slice {
			panic("The Where params Valid")
		}
		vLen := v.Len()
		ret := make([]interface{}, vLen)
		for i := 0; i < vLen; i++ {
			ret[i] = v.Index(i).Interface()
		}
		return ret
	}
	// 处理where条件
	WhereFilter := ""
	if len(p.whereCondition) > 0 && len(p.whereCondition[0]) > 0 {

		for _, whereItem := range p.whereCondition {

			FilterList := strings.Split(strings.ToLower(whereItem["query"].(string)), "and")
			// 匿名函数处理where入参, interface转slice
			WhereList := fn(whereItem["args"])
			// 组合where条件
			for index, value := range FilterList {
				// 参数分割,并去除空格
				NewValue := strings.TrimSpace(strings.Split(value, "?")[0])
				if len(WhereList) <= index {
					WhereFilter += fmt.Sprintf("WHERE %v", NewValue)
					continue
				}
				WhereValue := WhereList[index]
				// 入参类型断言
				switch reflect.ValueOf(WhereValue).Kind() {
				case reflect.Int:
					if WhereFilter == "" {
						WhereFilter += fmt.Sprintf("WHERE %v%d", NewValue, WhereList[index])
					} else {
						WhereFilter += fmt.Sprintf(" AND %v%d", NewValue, WhereList[index])
					}
				case reflect.String:
					temp := strings.Replace(WhereList[index].(string), "'", "\\'", -1)
					if WhereFilter == "" {
						WhereFilter += fmt.Sprintf("WHERE %v '%v'", NewValue, temp)
					} else {
						WhereFilter += fmt.Sprintf(" AND %v '%v'", NewValue, temp)
					}
				case reflect.Slice:
					// 匿名函数处理where入参, interface转slice
					NewList := fn(WhereValue)
					FilterWhere := ""
					for _, v := range NewList {
						switch reflect.ValueOf(v).Kind() {
						case reflect.Int:
							if FilterWhere == "" {
								FilterWhere += fmt.Sprintf("%d", v)
							} else {
								FilterWhere += fmt.Sprintf(",%d", v)
							}
						case reflect.String:
							if FilterWhere == "" {
								FilterWhere += fmt.Sprintf("'%v'", v)
							} else {
								FilterWhere += fmt.Sprintf(",'%v'", v)
							}
						default:
							panic("1001:The params Valid")
						}
					}
					if WhereFilter == "" {
						WhereFilter += fmt.Sprintf("WHERE %v (%v)", NewValue, FilterWhere)
					} else {
						WhereFilter += fmt.Sprintf(" AND %v (%v)", NewValue, FilterWhere)
					}
				default:
					panic("1002:The params Valid")
				}
			}
		}
	}
	return WhereFilter
}

// Delete 定义删除数据方法
func (p *DbPool) Delete() (affectRows int, err error) {
	defer p.pool.Close()
	// 处理where条件
	WhereFilter := p.handlerWhere()
	// 组合删除数据SQL
	DeleteSql := fmt.Sprintf("DELETE FROM `%v` %v", p.tableName, WhereFilter)
	p.lastSql = DeleteSql

	// 执行，判断是否存在事务
	var retData sql.Result
	if p.tx == nil {
		retData, err = p.pool.Exec(DeleteSql)
	} else {
		retData, err = p.tx.Exec(DeleteSql)
	}

	if err != nil {
		return 0, err
	}
	ARows, err := retData.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(ARows), nil
}

// Execute 查询执行SQL方法
func (p *DbPool) Execute(Sql string) (affectRows int, err error) {
	defer p.pool.Close()
	p.lastSql = Sql

	// 执行，判断是否存在事务
	var retData sql.Result
	if p.tx == nil {
		retData, err = p.pool.Exec(Sql)
	} else {
		retData, err = p.tx.Exec(Sql)
	}

	if err != nil {
		return 0, err
	}
	ARows, err := retData.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(ARows), nil
}

// FetchOne 定义执行SQL返回一条数据方法
func (p *DbPool) FetchOne(Sql string) map[string]interface{} {
	defer p.pool.Close()
	var RetOne map[string]interface{}
	p.lastSql = Sql
	rows, err := p.pool.Query(Sql)
	defer rows.Close()
	PanicErr(err, "fetch one error")
	// 数据获取
	RetMap := dealMysqlRows(rows)
	if len(RetMap) > 0 {
		RetOne = RetMap[0]
	}
	return RetOne
}

// FetchAll 定义执行SQL返回多条数据方法
func (p *DbPool) FetchAll(Sql string) []map[string]interface{} {
	defer p.pool.Close()
	p.lastSql = Sql
	rows, err := p.pool.Query(Sql)
	defer rows.Close()
	PanicErr(err, "fetch all error")
	// 数据获取
	RetMap := dealMysqlRows(rows)
	return RetMap
}

// 关闭行,释放链接
func closeRows(r *sql.Rows) {
	err := r.Close()
	PanicErr(err, "close rows error")
}

// BatchInsert 批量插入
func (p *DbPool) BatchInsert(params []map[string]interface{}) (affectRows int, err error) {
	defer p.pool.Close()
	// 自定待创建的函数和参数
	InsertCols, InsertArgsList := "", ""
	for k := range params[0] {
		// 数据列只能为string类型
		if InsertCols == "" {
			InsertCols += fmt.Sprintf("%s", k)
		} else {
			InsertCols += fmt.Sprintf(",%s", k)
		}
	}
	for _, data := range params {
		InsertArgs := ""
		for _, v := range data {
			// 判断数据类型,类型断言判断
			switch v.(type) {
			case int:
				if InsertArgs == "" {
					InsertArgs += fmt.Sprintf("%d", v)
				} else {
					InsertArgs += fmt.Sprintf(",%d", v)
				}
			case string:
				temp := strings.Replace(v.(string), "'", "\\'", -1)
				if InsertArgs == "" {
					InsertArgs += fmt.Sprintf("'%s'", temp)
				} else {
					InsertArgs += fmt.Sprintf(",'%s'", temp)
				}
			case float64:
				if InsertArgs == "" {
					InsertArgs += fmt.Sprintf("%f", v)
				} else {
					InsertArgs += fmt.Sprintf(",%f", v)
				}
			default:
				if InsertArgs == "" {
					InsertArgs += fmt.Sprintf("%v", v)
				} else {
					InsertArgs += fmt.Sprintf(",%v", v)
				}
			}
		}
		if InsertArgs != "" {
			if InsertArgsList == "" {
				InsertArgsList += fmt.Sprintf("(%v)", InsertArgs)
			} else {
				InsertArgsList += fmt.Sprintf(",(%v)", InsertArgs)
			}
		}
	}

	// 组合数据写入SQL
	InsertSql := fmt.Sprintf("INSERT INTO `%v` (%v) VALUES %v;", p.tableName, InsertCols, InsertArgsList)
	p.lastSql = InsertSql

	// 执行，判断是否存在事务
	var retData sql.Result
	if p.tx == nil {
		retData, err = p.pool.Exec(InsertSql)
	} else {
		retData, err = p.tx.Exec(InsertSql)
	}

	if err != nil {
		return 0, err
	}
	ARows, err := retData.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(ARows), nil
}

// Count 查询记录数
func (p *DbPool) Count() int {
	defer p.pool.Close()
	p.Select("count(*) as count")
	GetSql := p.sql()
	p.lastSql = GetSql
	count := 0
	err := p.pool.QueryRow(GetSql).Scan(&count)
	PanicErr(err, "query count error")
	return count
}
