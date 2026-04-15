package uStruct

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// typeForMysqlToGo MySQL 数据类型到 Go 类型的映射表
var typeForMysqlToGo = map[string]string{
	"int":                "int",
	"integer":            "int",
	"tinyint":            "int",
	"smallint":           "int",
	"mediumint":          "int",
	"bigint":             "int64",
	"int unsigned":       "int",
	"integer unsigned":   "int",
	"tinyint unsigned":   "int",
	"smallint unsigned":  "int",
	"mediumint unsigned": "int",
	"bigint unsigned":    "int64",
	"bit":                "int",
	"bool":               "bool",
	"enum":               "string",
	"set":                "string",
	"varchar":            "string",
	"char":               "string",
	"tinytext":           "string",
	"mediumtext":         "string",
	"text":               "string",
	"longtext":           "string",
	"blob":               "string",
	"tinyblob":           "string",
	"mediumblob":         "string",
	"longblob":           "string",
	"date":               "time.Time",
	"datetime":           "time.Time",
	"timestamp":          "time.Time",
	"time":               "time.Time",
	"float":              "float64",
	"double":             "float64",
	"decimal":            "float64",
	"binary":             "string",
	"varbinary":          "string",
}

// Table2Struct 将 MySQL 表结构转换为 Go struct 的构建器
//
// 支持链式调用配置 DSN、表名、保存路径、包名等参数
//
// 使用示例：
//
//	err := uStruct.NewTable2Struct().
//	    Dsn("user:pass@tcp(127.0.0.1:3306)/db").
//	    Table("users").
//	    SavePath("model/user.go").
//	    Run()
type Table2Struct struct {
	dsn            string
	savePath       string
	db             *sql.DB
	table          string
	prefix         string
	config         *T2tConfig
	err            error
	realNameMethod string
	enableJsonTag  bool
	packageName    string
	tagKey         string
}

// T2tConfig Table2Struct 的配置选项
//
// 使用示例：
//
//	cfg := &uStruct.T2tConfig{
//	    UcFirstOnly: true,
//	    TagToLower:  true,
//	}
type T2tConfig struct {
	RmTagIfUcFirsted bool // 如果字段首字母大写，则移除 tag
	TagToLower       bool // tag 转为小写
	UcFirstOnly      bool // 仅首字母大写，其余小写
	SeperatFile      bool // 每个表单独生成文件
}

// NewTable2Struct 创建一个新的 Table2Struct 构建器
//
// 使用示例：
//
//	t2s := uStruct.NewTable2Struct()
func NewTable2Struct() *Table2Struct {
	return &Table2Struct{}
}

// Dsn 设置数据库连接串（Data Source Name）
//
// 使用示例：
//
//	t2s.Dsn("root:123456@tcp(127.0.0.1:3306)/test")
func (t *Table2Struct) Dsn(d string) *Table2Struct {
	t.dsn = d
	return t
}

// TagKey 设置结构体 tag 的 key 名称
//
// 使用示例：
//
//	t2s.TagKey("xorm")
func (t *Table2Struct) TagKey(r string) *Table2Struct {
	t.tagKey = r
	return t
}

// PackageName 设置生成文件的包名
//
// 使用示例：
//
//	t2s.PackageName("model")
func (t *Table2Struct) PackageName(r string) *Table2Struct {
	t.packageName = r
	return t
}

// RealNameMethod 设置表真实名称方法的名称
//
// 如果设置，则会为每个 struct 生成一个返回真实表名的方法
//
// 使用示例：
//
//	t2s.RealNameMethod("TableName")
func (t *Table2Struct) RealNameMethod(r string) *Table2Struct {
	t.realNameMethod = r
	return t
}

// SavePath 设置生成文件的保存路径
//
// 使用示例：
//
//	t2s.SavePath("internal/model/user.go")
func (t *Table2Struct) SavePath(p string) *Table2Struct {
	t.savePath = p
	return t
}

// DB 设置已存在的数据库连接
//
// 使用示例：
//
//	t2s.DB(db)
func (t *Table2Struct) DB(d *sql.DB) *Table2Struct {
	t.db = d
	return t
}

// Table 设置要转换的表名
//
// 使用示例：
//
//	t2s.Table("users")
func (t *Table2Struct) Table(tab string) *Table2Struct {
	t.table = tab
	return t
}

// Prefix 设置表名前缀
//
// 生成 struct 名称时会自动去掉此前缀
//
// 使用示例：
//
//	t2s.Prefix("t_")
func (t *Table2Struct) Prefix(p string) *Table2Struct {
	t.prefix = p
	return t
}

// EnableJsonTag 设置是否生成 json tag
//
// 使用示例：
//
//	t2s.EnableJsonTag(true)
func (t *Table2Struct) EnableJsonTag(p bool) *Table2Struct {
	t.enableJsonTag = p
	return t
}

// Config 设置转换配置
//
// 使用示例：
//
//	t2s.Config(&uStruct.T2tConfig{UcFirstOnly: true})
func (t *Table2Struct) Config(c *T2tConfig) *Table2Struct {
	t.config = c
	return t
}

// Run 执行表结构到 Go struct 的转换并写入文件
//
// 转换完成后会自动调用 gofmt 格式化生成的文件
//
// 使用示例：
//
//	err := uStruct.NewTable2Struct().
//	    Dsn("root:123456@tcp(127.0.0.1:3306)/test").
//	    Table("users").
//	    SavePath("model.go").
//	    Run()
func (t *Table2Struct) Run() error {
	if t.config == nil {
		t.config = new(T2tConfig)
	}
	t.dialMysql()
	if t.err != nil {
		return t.err
	}

	tableColumns, err := t.getColumns()
	if err != nil {
		return err
	}

	var packageName string
	if t.packageName == "" {
		packageName = "package model\n\n"
	} else {
		packageName = fmt.Sprintf("package %s\n\n", t.packageName)
	}

	var structContent string
	for tableRealName, item := range tableColumns {
		if t.prefix != "" {
			tableRealName = tableRealName[len(t.prefix):]
		}
		tableName := tableRealName

		switch len(tableName) {
		case 0:
		case 1:
			tableName = strings.ToUpper(tableName[0:1])
		default:
			tableName = strings.ToUpper(tableName[0:1]) + tableName[1:]
		}
		depth := 1
		structContent += "type " + camelCase(tableName) + " struct {\n"
		for _, v := range item {
			var clumnComment string
			if v.ColumnComment != "" {
				clumnComment = fmt.Sprintf(" // %s", v.ColumnComment)
			}
			structContent += fmt.Sprintf("%s%s %s %s%s\n",
				tab(depth), v.ColumnName, v.Type, v.Tag, clumnComment)
		}
		structContent += tab(depth-1) + "}\n\n"

		if t.realNameMethod != "" {
			structContent += fmt.Sprintf("func (*%s) %s() string {\n",
				camelCase(tableName), t.realNameMethod)
			structContent += fmt.Sprintf("%sreturn \"%s\"\n",
				tab(depth), tableRealName)
			structContent += "}\n\n"
		}
	}

	var importContent string
	if strings.Contains(structContent, "time.Time") {
		importContent = "import \"time\"\n\n"
	}

	savePath := t.savePath
	if savePath == "" {
		savePath = "model.go"
	}
	f, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(packageName + importContent + structContent)

	cmd := exec.Command("gofmt", "-w", savePath)
	cmd.Run()
	return nil
}

// dialMysql 建立 MySQL 数据库连接
func (t *Table2Struct) dialMysql() {
	if t.db == nil {
		if t.dsn == "" {
			t.err = errors.New("dsn数据库配置缺失")
			return
		}
		t.db, t.err = sql.Open("mysql", t.dsn)
	}
}

// column 表字段信息
type column struct {
	ColumnName    string
	Type          string
	Nullable      string
	TableName     string
	ColumnComment string
	Tag           string
}

// getColumns 从 information_schema.COLUMNS 读取表结构信息
func (t *Table2Struct) getColumns(table ...string) (tableColumns map[string][]column, err error) {
	tableColumns = make(map[string][]column)
	var sqlStr = `SELECT COLUMN_NAME,DATA_TYPE,IS_NULLABLE,TABLE_NAME,COLUMN_COMMENT FROM information_schema.COLUMNS WHERE table_schema = DATABASE()`
	if t.table != "" {
		sqlStr += fmt.Sprintf(" AND TABLE_NAME = '%s'", t.prefix+t.table)
	}
	sqlStr += " order by TABLE_NAME asc, ORDINAL_POSITION asc"

	rows, err := t.db.Query(sqlStr)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		col := column{}
		err = rows.Scan(&col.ColumnName, &col.Type, &col.Nullable, &col.TableName, &col.ColumnComment)
		if err != nil {
			return
		}
		col.Tag = col.ColumnName
		col.ColumnName = t.camelCaseField(col.ColumnName)
		col.Type = typeForMysqlToGo[col.Type]
		if col.Type == "int" && strings.HasSuffix(col.Tag, "_time") {
			col.Type = "int64"
		}
		if t.config.RmTagIfUcFirsted && col.ColumnName[0:1] == strings.ToUpper(col.ColumnName[0:1]) {
			col.Tag = "-"
		} else {
			if t.config.TagToLower {
				col.Tag = strings.ToLower(col.Tag)
			}
		}
		cTag := ""
		switch col.Tag {
		case "id":
			cTag = "autoincr pk "
		case "created_at":
			cTag = "created "
		case "updated_at":
			cTag = "updated "
		case "deleted_at":
			cTag = "deleted "
		}
		if t.enableJsonTag {
			col.Tag = fmt.Sprintf("`%s:\"%s'%s'\" json:\"%s\" label:\"%s\"`", t.tagKey, cTag, col.Tag, col.Tag, col.ColumnComment)
		} else {
			col.Tag = fmt.Sprintf("`%s:\"%s'%s'\" label:\"%s\"`", t.tagKey, cTag, col.Tag, col.ColumnComment)
		}
		if _, ok := tableColumns[col.TableName]; !ok {
			tableColumns[col.TableName] = []column{}
		}
		tableColumns[col.TableName] = append(tableColumns[col.TableName], col)
	}
	return
}

// camelCaseField 将下划线分隔的字段名转换为驼峰命名
func (t *Table2Struct) camelCaseField(str string) string {
	if t.prefix != "" {
		str = strings.Replace(str, t.prefix, "", 1)
	}
	var text string
	for _, p := range strings.Split(str, "_") {
		switch len(p) {
		case 0:
		case 1:
			text += strings.ToUpper(p[0:1])
		default:
			if t.config.UcFirstOnly {
				text += strings.ToUpper(p[0:1]) + strings.ToLower(p[1:])
			} else {
				text += strings.ToUpper(p[0:1]) + p[1:]
			}
		}
	}
	return text
}

// camelCase 将下划线分隔的字符串转换为驼峰命名
func camelCase(s string) string {
	var text string
	for _, p := range strings.Split(s, "_") {
		switch len(p) {
		case 0:
		case 1:
			text += strings.ToUpper(p[0:1])
		default:
			text += strings.ToUpper(p[0:1]) + p[1:]
		}
	}
	return text
}

// tab 生成指定深度的制表符缩进
func tab(depth int) string {
	return strings.Repeat("\t", depth)
}
