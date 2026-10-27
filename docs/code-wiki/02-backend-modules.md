# 02 · 后端模块与关键代码

本文覆盖 `container/`、`module/**` 与 `api/` 公共层。HTTP 处理器逐个接口的说明见 [03-api-reference.md](03-api-reference.md)，文件级索引见 [07-code-map.md](07-code-map.md)。

---

## 1. container —— 应用容器层

`container` 是整个应用的“运行时容器”：装载应用配置、持有 SQLite、按资源 ID 查询连接配置并缓存连接。它不依赖 Gin，可被任何层调用。

### 1.1 应用配置（container/entry.go）

```go
type AppConfig struct {
    Port          int64  `env:"PORT" envDefault:"8080"`
    LoginPassword string `env:"LOGIN_PASSWORD"`
    SQLitePath    string `env:"SQLITE_PATH" envDefault:"./devops-console.db"`
    Env           string `env:"ENV"`
}
```

| 符号 | 说明 |
| --- | --- |
| `InitAppConfig() error` | `env.Parse` 解析环境变量（`.env` 由 godotenv autoload 提前加载），随后 `OpenDB(SQLitePath)`。失败返回错误，`main` 直接 panic。 |
| `GetAppConfig() *AppConfig` | 返回全局配置单例。 |
| `GetSQLiteDB() *gorm.DB` | 返回全局 SQLite 连接。 |
| `(*AppConfig).IsDevelop() bool` | `Env == "develop"`，决定模板/静态资源来自本地文件还是 embed。 |

### 1.2 SQLite 与资源模型（container/sqlite.go）

**表结构**（`AutoMigrate(&Resource{})`）：

| 列 | 类型 | 约束 | 含义 |
| --- | --- | --- | --- |
| `id` | INTEGER | PK, autoIncrement | 资源 ID（路由中的 `:id`） |
| `name` | varchar(128) | NOT NULL，与 `config_type` 联合唯一 | 展示名，如 `prod-mysql-main` |
| `config_type` | varchar(32) | NOT NULL，联合唯一 | `mysql` / `ftp` / `ssh` / `s3` / `redis` |
| `config` | text | NOT NULL default `{}` | 连接参数的 JSON 字符串 |
| `create_timestamp` | datetime | autoCreateTime | 创建时间 |
| `update_timestamp` | datetime | autoUpdateTime | 更新时间 |

**关键类型**

| 符号 | 说明 |
| --- | --- |
| `type Resource struct` | 表 `resource` 的 GORM 模型；`TableName()` 固定返回 `resource`。 |
| `type ConnectionConfig struct` | 通用连接参数：`Host/Port/Username/Password(密文)/Database/Prefix/Region/Bucket/DB`。写入时按类型只填需要的字段。 |
| `(*ConnectionConfig).SetExtra(key, value)` | 把类型扩展字段写进结构体：`database`/`prefix`/`region`/`bucket`/`db`。 |
| `(*ConnectionConfig).Extra() map[string]string` | 反向输出扩展字段，供接口返回。 |
| `(*Resource).DecodeConfig() (*ConnectionConfig, error)` | 解析 `Config` JSON。 |
| `(*Resource).EncodeConfig(*ConnectionConfig) error` | 序列化并写回 `Config`。 |

**CRUD 函数**

| 函数 | 行为 |
| --- | --- |
| `ListResources(db, configType)` | 按类型过滤（空则全部），`ORDER BY id DESC`。 |
| `GetResource(db, id)` | 按主键查询，未找到返回 `gorm.ErrRecordNotFound`。 |
| `CreateResource(db, record)` | 插入并回填主键/时间戳。 |
| `UpdateResource(db, record)` | 只更新 `name` 与 `config`（类型不可改）。 |
| `DeleteResource(db, id)` | 返回 `RowsAffected > 0` 表示是否真的删到。 |

**按类型解码的强类型配置**

| 类型 | 字段 |
| --- | --- |
| `MySQLConfig` | host, port, username, password, database |
| `FTPConfig` | host, port, username, password, prefix |
| `SSHConfig` | host, port, username, password |
| `S3Config` | access_key, secret_key, region, bucket, endpoint |
| `RedisConfig` | host, port, username, password, db |

查询入口：

```go
QueryMySQLConfig(id)                 // 仅配置
QueryMySQLResource(idStr)            // 资源 + 配置（页面渲染用）
QueryFTPConfig / QuerySSHConfig / QueryS3Config / QueryRedisConfig
```

> 注意：`S3Config` 的 JSON 字段（`access_key/secret_key/endpoint`）与写库时使用的 `ConnectionConfig`（`host/username/password`）不一致，这是当前 S3 浏览链路的一个缺陷，见 [06-build-run-deploy.md](06-build-run-deploy.md#7-已知缺陷与待清理项)。

### 1.3 连接缓存与密码解密（container/resource.go）

```go
var (
    clientLock sync.Mutex
    gormMap    = map[string]*gorm.DB{}       // key = resource id
    redisMap   = map[string]*redisV8.Client{} // key = resource id
)
```

| 符号 | 说明 |
| --- | --- |
| `ConfigSecret() string` | 读取 `CONFIG_SECRET`，未设置时返回默认值 `devops-console-default-secret`。 |
| `GetMySQLDBByID(resourceID) (*gorm.DB, error)` | 加锁 → 查缓存 → `QueryMySQLConfig` → `AESGCMDecrypt` → 构造 DSN（`parseTime=true`、`charset=utf8mb4`）→ `gorm.Open` → 缓存。错误信息包含“配置不存在/解密失败/连接失败”。 |
| `GetRedisClientByID(resourceID) *redisV8.Client` | 同上，但失败返回 `nil`（无错误细节）；连接后 `PING` 成功才缓存。 |
| `ClearClientCache(resourceID)` | 关闭并删除该 ID 的 GORM/Redis 缓存；配置更新或删除后由 `api/resource` 调用。 |

设计要点：
- 锁是全局互斥锁，首次建立连接会串行化；连接建立后不再加锁。
- 缓存不会自动过期；配置变更必须走 `ClearClientCache`（已在 `Update`/`Delete` 中处理）。
- 密码在配置层是密文，只有 `container` 在建立连接时解密。

---

## 2. module —— 无框架依赖的基础能力

### 2.1 module/crypto

| 函数 | 说明 |
| --- | --- |
| `AESGCMEncrypt(key, plaintext) (string, error)` | key 经 SHA-256 → 32B；随机 12B nonce；返回 `base64(nonce||ciphertext)`。空串原样返回。 |
| `AESGCMDecrypt(key, ciphertext) (string, error)` | 反向操作；密文过短返回 `ciphertext too short`。 |
| `HMACSha1` / `Sha1Sum` / `MD5` / `GetFileMD5` | 摘要工具。 |
| `Base64Decode(value)` | 先 Base64 解码，再 `url.QueryUnescape`；`ExecSQL` 用它还原前端编码的 SQL。 |
| `JsonDecode(raws, dest)` | `json.Unmarshal` 包装。 |

### 2.2 module/fsutil

目录/路径：

| 函数 | 说明 |
| --- | --- |
| `GetNowTimeStamp()` | `2006-01-02-15-04-05` 格式时间戳，常用于临时文件命名。 |
| `MakeVarLogFile/MakeVarLogDir/MakeTmpWorkDir` | 在 `/var/log`、`/tmp` 下创建带时间戳的目录/文件。 |
| `GetFiles(folder)` / `GetFilesMap(folder)` / `GetDirFilesAsMap(folder)` | 递归列出文件；map 的 key 是相对路径。 |
| `GetFileMapping(path)` | `filepath.Walk` 生成“相对路径 → 绝对路径”映射（S3 目录上传使用）。 |

归档：

| 函数 | 说明 |
| --- | --- |
| `Zip(srcDir, dest)` | 文件或目录 → zip；目录先展开成文件映射，再用 `archiver.CompressedArchive{Zip}` 打包。 |
| `Unzip(srcFile, dest)` | 用 `archiver.Zip.Extract` 解压到目标目录，自动建父目录。 |
| `UnzipFrom(io.Reader, dest)` | 从流解压（S3 上传 zip 目录时使用）。 |
| `CreateTarFile(srcDir, dest)` / `TarWriteFile` / `TarWriteFiles` | tar 打包。 |
| `ExtractTarFile(tarFile, dest)` | tar 解包，跳过目录项、自动建目录。 |

其它：

| 函数 | 说明 |
| --- | --- |
| `ReadConfig(dest)` / `ReadFileAs(path, v)` | 从 `os.Args[1]` 或指定文件读 JSON。 |
| `JsonDecodeX(input)` | 解析任意 JSON。 |
| `ExtractURL(value, matchFunc)` / `ExtractURLToMap(...)` | 递归提取字符串中的 URL（历史工具使用）。 |

### 2.3 module/ftp —— FTPClient

```go
type FTPClient struct {
    serverConn *ftp.ServerConn
    prefix     string // 所有路径自动拼接的根目录
}
```

| 方法 | 说明 |
| --- | --- |
| `DefaultFTPClient(server, user, password)` | 60s 超时拨号 + 登录。 |
| `NewFTPClient(server, user, password, opts...)` | 可自定义 `ftp.DialOption`。 |
| `SetPrefix(prefix)` | 设置根目录；后续方法用 `filepath.Join(prefix, path)` 拼接。 |
| `Upload(localFile, remoteFile)` / `UploadReader(io.Reader, remoteFile)` | `STOR` 上传。 |
| `DownloadTo(path, localFile)` / `GetByte(path)` | `RETR` 下载到文件或字节切片。 |
| `List(path)` | 返回 `[]*ftp.Entry`。 |
| `Delete` / `Rename` / `MakeDir` / `Get` | 对应 FTP 命令；均自动拼接 prefix。 |
| `SetTransferTypeBinary()` | 二进制传输模式。 |
| `Close()` | `QUIT`。 |

### 2.4 module/mysql

| 符号 | 说明 |
| --- | --- |
| `BuildQueryRaws(where map[string]interface{}) (sql string, params []interface{})` | 把前端 `where` 映射为 SQL 条件。key 支持前缀：`not@col`→`not in`、`no@col`→`!=`、`gt/gte/lt/lte@col`→比较、`like@col`/`plike@col`→模糊；无 `@` 时默认 `col in (?)`。多个条件用 `and` 连接。 |
| `GetMySQLTables(db)` | `show tables` → `[]string`。 |
| `GetMySQLTableFields(db, table)` | `desc table`，返回 `[]TableDesc{Field,Type,Null,Key,Default,Extra}`。 |
| `GetMySQLTableIndexes(db, table)` | `show index`，按索引名聚合列顺序，返回 `map[string]TableIndex`。 |
| `GetCreateTableSQL(db, table)` | `show create table` 提取 `Create Table`。 |
| `GetGormDBFrom(user, password, host, port, database)` | 直接构造 GORM 连接（历史辅助，当前主链路使用 `container`）。 |
| `UpdateDataByID` / `DeleteDataByID` / `QueryList` / `CreateData` | 以 `map[string]interface{}` 为载体的通用 CRUD 辅助。 |

> 安全提示：`like`/`plike` 分支把值直接拼进 SQL（`fmt.Sprintf`），`BuildQueryRaws` 的表名也来自请求；生产使用需加白名单。

### 2.5 module/redis

| 函数 | 说明 |
| --- | --- |
| `GetRedisClient(addr, username, password, db)` | 10s 拨号/读/写超时。 |
| `GetDefaultBusinessRedisClient(addr, username, password)` | 空参数回落到 `REDIS_*` 环境变量。 |

当前 HTTP 链路使用 `container.GetRedisClientByID`，本包未被引用。

### 2.6 module/s3

| 符号 | 说明 |
| --- | --- |
| `GetS3Client(accessKey, secretKey, region, endpoint)` | 静态凭证；`BaseEndpoint` 支持 MinIO；`InsecureSkipVerify: true`（内网自签证书）；始终返回 `nil` error。 |
| `StsClient` / `NewSTSClient(...)` | 基于 STS 的临时凭证客户端，`Host` 存 endpoint。 |
| `(*StsClient).GetTempToken(bucket, prefix)` | `AssumeRole`（MinIO 忽略 RoleArn），有效期 3600s，返回 `TempToken{tmpSecretId,tmpSecretKey,sessionToken,expire,bucket,region,host,prefix}`。 |

### 2.7 module/ssh（当前未挂 HTTP 路由）

核心是“**所有远程操作都有超时边界**”：

| 常量/变量 | 值 | 作用 |
| --- | --- | --- |
| `sshDialTimeout` | 10s | TCP 拨号超时 |
| `sshHandshakeTimeout` | 15s | SSH 握手整体 deadline（banner/密钥交换/认证） |
| `sftpIdleTimeout` | 60s | SFTP 传输“无任何数据收发”的空闲超时（进度续期，不是总时长） |

| 符号 | 说明 |
| --- | --- |
| `dialSSH(addr, config, wrap)` | 先 `net.DialTimeout`，再给连接设握手 deadline，握手成功后清除 deadline。 |
| `progressConn` + `watchIdle` | 记录最近一次读写时间；连续 idle 无进展时关闭连接，让阻塞调用立即返回。 |
| `SSHExecCommand(host, port, user, password, command)` | 执行命令，返回 `SSHExecResult{Stdout,Stderr}`；默认不限时。 |
| `SSHExecCommandWithTimeout(..., timeout)` | 超时后关闭连接；`NewSession` 也有 15s 超时；命令执行期不设空闲 deadline（适配长时间静默命令）。 |
| `SSHUploadFile(..., reader, remotePath)` | SFTP 上传；失败时删除远端半截文件；卡死返回 `sftp transfer stalled`。 |
| `SSHOpenFile(...) (*SSHRemoteFile, error)` | 流式下载；`SSHRemoteFile.Read/Close`，`Close` 可重复调用。 |
| `OpenSFTP(host, port, user, password, timeout) (*SFTPSession, error)` | 一次性 SFTP 会话，用于 list/rename/delete/mkdir 等短操作；`Close` 关闭 SFTP 与 SSH。 |

### 2.8 module/jwt 与 module/model（预留）

| 包 | 符号 | 状态 |
| --- | --- | --- |
| `module/jwt` | `Claims`、`GenerateToken`、`ParseToken`、`GetExpireSeconds` | 未被任何包 import；`.env.example` 的 JWT 配置也未生效。 |
| `module/model` | `RegisterModelParse`、`GetModelParseFunc` | 设计为“库#表 → 强类型构造函数”的插件注册表，当前无注册者。 |

### 2.9 module/common

| 函数 | 说明 |
| --- | --- |
| `GetStringValFromMap(map, key)` | 把 `json.Number/int64/int/float/string/其它` 统一转字符串。 |
| `ToString(src)` | 反射转字符串。 |
| `GetEnv(key)` / `GetHost()` | 环境变量读取；`GF_HOST` 或默认 `127.0.0.1`。 |

---

## 3. api 公共层

### 3.1 参数与响应（api/gin.go）

| 函数 | 说明 |
| --- | --- |
| `QueryAllParams(ctx)` | 取 query 的**第一个值**。 |
| `AllGetParams(ctx)` | query 多值用逗号 join。 |
| `AllPostParams(ctx)` | 按 Content-Type 处理：表单、multipart（文件为 `[]*multipart.FileHeader`）、JSON（`UseNumber` 保精度，解析失败时把原始字节放入 `raw_data`）。 |
| `AllHeader(ctx)` | header 多值 join。 |
| `Query(ctx, keys...)` | 按顺序返回第一个存在的 query 值。 |
| `AllParams(ctx)` | GET + POST 合并。 |
| `Success(ctx, data)` / `Failure(ctx, code, message)` | 输出 `JSONResponse`；`ctx.IsAborted()` 时跳过。 |
| `type JSONResponse` | `{code, message, data}`。 |

### 3.2 页面数据（api/page.go）

```go
type Breadcrumb struct { Name, URL string }
type PageData struct {
    Title, Description string
    HideBreadcrumbs    bool
    Breadcrumbs        []Breadcrumb
    Extension          map[string]interface{} // 注入 window.PageData
}
type DataFunc func(ctx *gin.Context) (*PageData, error)
```

`main.go` 的 `pageBinding` 会把 `Title/Description/Extension/breadcrumbs/error` 转成模板变量，并自动在面包屑前补“首页”。

---

## 4. 跨模块调用关系（要点）

```text
api/file/*          -> module/fsutil
api/mysql/*         -> container.GetMySQLDBByID + module/mysql.BuildQueryRaws
api/mysql/exec_sql  -> module/crypto.Base64Decode
api/redis/query     -> container.GetRedisClientByID
api/ftp/*           -> container.QueryFTPConfig + module/ftp.FTPClient
api/s3client/*      -> container.QueryS3Config + module/s3（+ fsutil 处理 zip）
api/resource/*      -> container（SQLite/Resource/缓存） + module/crypto（加解密）
api/k3s/*           -> client-go（无 container/module 依赖）
```

补充：`api/ftp.GetFtpClient` 使用 `cfg.Password` 直接登录，**没有经过 AESGCMDecrypt**；`api/s3client.GetS3Client` 也未解密。二者与 `api/resource/test.go` 的正确做法不一致，详见缺陷清单。
