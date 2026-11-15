# ConfigSource 配置源读取与解码

新应用使用 `ReadYAML` / `Document.Decode`，将应用 schema 放在应用仓库。

```go
import "github.com/wanglelecc/gokitbox/configsource"

doc, err := configsource.ReadYAML("./configs/app.yaml")
if err != nil { return err }
var opts AppConfig // 由应用定义，字段使用 yaml tags
if err := doc.Decode(&opts); err != nil { return err }
// 先 Validate，再构造模块；模块接收自己拥有的 Options。
```

严格拒绝重复 key、未知字段、错误类型、显式 null 字段、多文档及非 mapping 根。原生 sequence 不会先转为字符串再拆分。使用 YAML 1.2 语义，on/off 保持字符串，bool 写 true/false。

错误包含类别和行列位置，不包含源文件片段或配置值。ReadYAML 每次重新读取指定路径；Document 保存该次读取的快照。相对路径以当前工作目录为基准，不搜索可执行文件目录。Decode 不读取环境或创建网络资源，env/file 引用由应用实现。

## 兼容旧消费者

`Load(path)` 支持显式 `.ini/.yaml/.yml` 路径，包括裸文件名。每次重新读取文件，不再缓存文件路径。注册 source 仅使用精确名称匹配。

`NewSnapshot(sections)` 深拷贝已有 section 数据，返回旧 Config 接口；`Install(cfg)` 在应用启动时安装兼容视图，供 GetConf/GetConfDefault/GetConfArr/GetConfStringMap/GetConfArrayMap/ConfMapToStruct 使用。数组接口支持 YAML sequence，字符串形式列表仅作兼容。映射结构体的旧接口允许投影，严格 schema 请使用 Document.Decode。

旧 `SetConfigPath` / `InitConfig` 仍保留。默认路径仍为 `../config/app.ini`；新应用必须显式加载，不依赖该默认值。旧 InitConfig 无 error 返回且会记录加载错误，不适合应用 fail-fast。GetConfDefault 在加载失败时返回提供的默认值。

`Set` 修改 YAML 内存视图时保留同 section 的其它字段并加锁，不回写文件；INI 的 Set 仍不支持。不要用它更新运行中应用配置。`ClearConfigCache` 仅清除旧全局视图，不能重建 DB/Redis/RPC/SDK 等模块持有的配置或连接。

这些兼容入口不是完整应用热更新机制。应在加载/归一化/校验完成后安装一次，随后通过显式 Options 演进。
