# 07 代码地图与函数索引

本页按目录给出文件级地图、关键函数索引与调用关系，便于快速定位。行号对应当前 HEAD。

## 1. 仓库文件树

```text
.
├── main.go                     # 入口：embed、render、路由注册、页面处理器
├── go.mod / go.sum             # Go 1.24；依赖清单
├── .env.example                # 环境变量样例（部分已失效）
├── container/                  # 配置与连接管理层
│   ├── entry.go                # AppConfig、InitAppConfig、SQLite 全局句柄
│   ├── sqlite.go               # Resource 模型、CRUD、各类型配置、泛型解析
│   └── resource.go             # 密钥、MySQL/Redis 连接缓存、缓存失效
├── api/                        # HTTP 处理器
│   ├── gin.go                  # 参数收集、统一响应
│   ├── page.go                 # PageData / Breadcrumb / DataFunc
│   ├── resource/               # 资源配置 CRUD 与连通性测试
│   ├── mysql/                  # 单库查询、结构、DDL、SQL 执行
│   ├── file/                   # 本机文件管理
│   ├── k3s/                    # K3s/K8s 资源操作
│   ├── redis/                  # Redis Key 浏览
│   ├── ftp/                    # FTP 文件管理
│   └── s3client/               # S3 对象浏览与临时凭证
├── module/                     # 可复用实现层（与 Gin 解耦）
│   ├── crypto/                 # AES-GCM、哈希、Base64
│   ├── fsutil/                 # 文件、zip/tar
│   ├── ftp/                    # FTP 客户端封装
│   ├── ssh/                    # SSH 命令、上传下载、SFTP
│   ├── mysql/                  # GORM 查询构造、表结构、通用 CRUD
│   ├── redis/                  # Redis 客户端单例
│   ├── s3/                     # AWS SDK 客户端、STS
│   ├── jwt/                    # JWT（未接入）
│   ├── model/                  # 模型解析注册表（未接入）
│   └── common/                 # 通用转换与 env 读取
├── pages/                      # Go 模板（embed）
│   ├── layout.tmpl             # 全站骨架
│   ├── index.tmpl              # 首页资源配置管理
│   ├── error.tmpl              # 错误页
│   ├── mysql/ ftp/ ssh/ redis/ s3browse/
│   ├── file/ k3s/ system/ setting/ tool/ text/ http/ user/
│   └── k3s/*.yml               # K8s 备份/Service 清单（非 Go 读取）
├── assets/                     # 静态资源（embed）
│   ├── js/                     # Vue、axios、JSONEditor、my.js 等
│   ├── css/                    # Bootstrap 3.4、my.css 等
│   └── images/
└── docs/code-wiki/             # 本文档
```

## 2. 入口与页面层：`main.go`

| 符号 | 行 | 说明 |
| --- | --- | --- |
| `tmplPage` / `assets` | [L26-L30](file:///workspace/main.go#L26-L30) | `go:embed pages` / `assets` |
| [newRenderClient](file:///workspace/main.go#L35-L53) | L35 | 构造 `unrolled/render`：layout、目录、分隔符、文件系统 |
| [main](file:///workspace/main.go#L55-L189) | L55 | 初始化配置 → 注册静态资源 → BasicAuth → 全部路由 → `Run` |
| [createPageHandlerV2](file:///workspace/main.go#L191-L195) | L191 | 静态 `PageData` 页面处理器 |
| [createPageHandlerV1](file:///workspace/main.go#L198-L213) | L198 | 动态 `DataFunc` 页面处理器，出错渲染 `error` |
| [pageBinding](file:///workspace/main.go#L216-L241) | L216 | 组装模板变量（title/description/breadcrumbs/extension/error） |

路由分组与注册顺序见 [main.go L77-L187](file:///workspace/main.go#L77-L187)：`/api/file` → `/api/mysql/:id` → `/api/k3s/:id` → `/api/redis/:id` → `/api/ftp/:id` → `/api/s3/:id` → `/api/config` → 页面路由 → `NoRoute`。

## 3. `container/`：配置与连接

| 文件 | 关键符号 | 说明 |
| --- | --- | --- |
| [entry.go](file:///workspace/container/entry.go) | [AppConfig](file:///workspace/container/entry.go#L15-L20)、[IsDevelop](file:///workspace/container/entry.go#L23-L25)、[InitAppConfig](file:///workspace/container/entry.go#L27-L39)、[GetSQLiteDB](file:///workspace/container/entry.go#L45-L47) | 环境变量解析、打开 SQLite、全局句柄 |
| [sqlite.go](file:///workspace/container/sqlite.go) | [OpenDB](file:///workspace/container/sqlite.go#L16-L32)、[ConnectionConfig](file:///workspace/container/sqlite.go#L42-L52)、[Resource](file:///workspace/container/sqlite.go#L95-L102) | 建库/迁移、连接参数模型、表模型 |
| 同上 | [DecodeConfig](file:///workspace/container/sqlite.go#L110-L119) / [EncodeConfig](file:///workspace/container/sqlite.go#L122-L129) | `config` 列与结构体互转 |
| 同上 | [ListResources](file:///workspace/container/sqlite.go#L132-L142)、[GetResource](file:///workspace/container/sqlite.go#L145-L151)、[CreateResource](file:///workspace/container/sqlite.go#L154-L156)、[UpdateResource](file:///workspace/container/sqlite.go#L159-L165)、[DeleteResource](file:///workspace/container/sqlite.go#L168-L174) | 仅操作 `resource` 表 |
| 同上 | [MySQLConfig](file:///workspace/container/sqlite.go#L177-L183)、[FTPConfig](file:///workspace/container/sqlite.go#L186-L192)、[SSHConfig](file:///workspace/container/sqlite.go#L195-L200)、[S3Config](file:///workspace/container/sqlite.go#L203-L209)、[RedisConfig](file:///workspace/container/sqlite.go#L212-L218) | 读取侧强类型视图（S3 与写入不一致） |
| 同上 | [decodeConfig](file:///workspace/container/sqlite.go#L221-L230)、[queryResourceByID](file:///workspace/container/sqlite.go#L233-L239) | 泛型解析、按类型+ID 查询 |
| 同上 | [QueryMySQLConfig](file:///workspace/container/sqlite.go#L242-L248)、[QueryMySQLResource](file:///workspace/container/sqlite.go#L251-L265)、[QueryRedisConfig](file:///workspace/container/sqlite.go#L268-L274)、[QueryFTPConfig](file:///workspace/container/sqlite.go#L277-L284)、[QuerySSHConfig](file:///workspace/container/sqlite.go#L287-L294)、[QueryS3Config](file:///workspace/container/sqlite.go#L297-L304) | 各类型读取入口 |
| [resource.go](file:///workspace/container/resource.go) | [ConfigSecret](file:///workspace/container/resource.go#L26-L31) | 加密密钥（含默认回退） |
| 同上 | [ClearClientCache](file:///workspace/container/resource.go#L34-L47) | 关闭并移除缓存连接 |
| 同上 | [GetRedisClientByID](file:///workspace/container/resource.go#L49-L78) | 解密 → 建连 → Ping → 缓存 |
| 同上 | [GetMySQLDBByID](file:///workspace/container/resource.go#L81-L114) | 解密 → GORM DSN → 缓存 |

## 4. `api/` 公共层

| 文件 | 符号 | 说明 |
| --- | --- | --- |
| [gin.go](file:///workspace/api/gin.go) | [QueryAllParams](file:///workspace/api/gin.go#L19-L35)、[AllGetParams](file:///workspace/api/gin.go#L37-L53)、[AllPostParams](file:///workspace/api/gin.go#L55-L94)、[AllHeader](file:///workspace/api/gin.go#L96-L111) | 合并 query/body/header 参数 |
| 同上 | [Query](file:///workspace/api/gin.go#L113-L124)、[AllParams](file:///workspace/api/gin.go#L126-L144) | 取单个/全部参数 |
| 同上 | [Success](file:///workspace/api/gin.go#L146-L160)、[Failure](file:///workspace/api/gin.go#L162-L172) | 统一 `{code,message,data}` |
| [page.go](file:///workspace/api/page.go) | `Breadcrumb`、`PageData`、`DataFunc` | 页面渲染数据契约 |

## 5. `api/resource/`：资源配置

| 文件 | 符号 | 说明 |
| --- | --- | --- |
| [define.go](file:///workspace/api/resource/define.go) | `TypeMySQL/FTP/SSH/S3/Redis`、`testTimeoutSeconds` | 常量 |
| 同上 | [FieldDef](file:///workspace/api/resource/define.go#L30-L36)、[TypeMeta](file:///workspace/api/resource/define.go#L38-L49)、[TypeMetas](file:///workspace/api/resource/define.go#L52-L118) | 驱动前端动态表单的类型元数据 |
| 同上 | [GetTypeMeta](file:///workspace/api/resource/define.go#L129-L133)、[SaveRequest](file:///workspace/api/resource/define.go#L135-L144)、[MaskedConfig](file:///workspace/api/resource/define.go#L146-L158) | 校验、请求体、脱敏响应 |
| 同上 | [Mask](file:///workspace/api/resource/define.go#L161-L178) | 去掉 `config` 与密文 |
| 同上 | [normalizeExtra](file:///workspace/api/resource/define.go#L181-L189)、[buildResource](file:///workspace/api/resource/define.go#L197-L229) | 扩展字段过滤、请求校验与建模 |
| [save.go](file:///workspace/api/resource/save.go) | [Create](file:///workspace/api/resource/save.go#L20-L52)、[Update](file:///workspace/api/resource/save.go#L59-L117)、[Delete](file:///workspace/api/resource/save.go#L124-L140)、[friendlyDbError](file:///workspace/api/resource/save.go#L143-L149) | CRUD；更新时保留旧密码、清理缓存 |
| [list.go](file:///workspace/api/resource/list.go) | [List](file:///workspace/api/resource/list.go#L16-L30) | 可选 `type` 过滤，全部脱敏 |
| [types.go](file:///workspace/api/resource/types.go) | [Types](file:///workspace/api/resource/types.go#L14-L20) | 返回 `TypeMetas()` |
| [test.go](file:///workspace/api/resource/test.go) | [Test](file:///workspace/api/resource/test.go#L31-L73) | 5 秒连通性测试入口 |
| 同上 | [testByType](file:///workspace/api/resource/test.go#L76-L94)、[testMySQL](file:///workspace/api/resource/test.go#L97-L113)、[testFTP](file:///workspace/api/resource/test.go#L116-L124)、[testSSH](file:///workspace/api/resource/test.go#L127-L139)、[testS3](file:///workspace/api/resource/test.go#L142-L162)、[testRedis](file:///workspace/api/resource/test.go#L165-L175) | 分类型探活 |

## 6. `api/mysql/`

| 文件 | 符号 | 说明 |
| --- | --- | --- |
| [db.go](file:///workspace/api/mysql/db.go) | [GetDB](file:///workspace/api/mysql/db.go#L9-L20) | 包装 `container.GetMySQLDBByID(ctx.Param("id"))` |
| [page.go](file:///workspace/api/mysql/page.go) | [PageData](file:///workspace/api/mysql/page.go#L12-L29) | `/mysql/:id` 的标题与 `config_id/config_name/database` |
| [table.go](file:///workspace/api/mysql/table.go) | [Table](file:///workspace/api/mysql/table.go#L12) | `SHOW TABLES` 封装 |
| [desc.go](file:///workspace/api/mysql/desc.go) | [Desc](file:///workspace/api/mysql/desc.go#L12) | 表字段列表 |
| [show_create_table.go](file:///workspace/api/mysql/show_create_table.go) | [ShowCreateTable](file:///workspace/api/mysql/show_create_table.go#L12) | 建表语句 |
| [show_index.go](file:///workspace/api/mysql/show_index.go) | [ShowIndex](file:///workspace/api/mysql/show_index.go#L12) | 索引信息 |
| [query.go](file:///workspace/api/mysql/query.go) | [Query](file:///workspace/api/mysql/query.go#L23) | 分页查询；`QueryRequest{where, order, offset, limit, fields}` |
| [count.go](file:///workspace/api/mysql/count.go) | [Count](file:///workspace/api/mysql/count.go#L13) | 条件计数 |
| [group_by_count.go](file:///workspace/api/mysql/group_by_count.go) | [GroupByCount](file:///workspace/api/mysql/group_by_count.go#L19) | 分组统计 |
| [distinct.go](file:///workspace/api/mysql/distinct.go) | [Distinct](file:///workspace/api/mysql/distinct.go#L15) | `distinct` 查询；**读错路由参数** |
| [create.go](file:///workspace/api/mysql/create.go) | [Create](file:///workspace/api/mysql/create.go#L14)、[BatchCreate](file:///workspace/api/mysql/create.go#L35) | 单条/批量插入 |
| [update.go](file:///workspace/api/mysql/update.go) | [Update](file:///workspace/api/mysql/update.go#L12) | 按 `row_id` 更新 |
| [delete.go](file:///workspace/api/mysql/delete.go) | [DeleteByID](file:///workspace/api/mysql/delete.go#L14) | 按 `row_id` 删除 |
| [exec_sql.go](file:///workspace/api/mysql/exec_sql.go) | [ExecSQL](file:///workspace/api/mysql/exec_sql.go#L18) | 任意 SQL，按语句类型返回行/影响行数 |
| [global_variables.go](file:///workspace/api/mysql/global_variables.go) | [GlobalVariables](file:///workspace/api/mysql/global_variables.go#L16) | `SHOW GLOBAL VARIABLES` |

## 7. `api/file/`

| 文件 | 符号 | 说明 |
| --- | --- | --- |
| [list.go](file:///workspace/api/file/list.go) | [ListFile](file:///workspace/api/file/list.go#L14) | 目录列表（大小、时间、是否目录） |
| [download.go](file:///workspace/api/file/download.go) | [DownloadFile](file:///workspace/api/file/download.go#L13) | 单文件下载 |
| [download_static.go](file:///workspace/api/file/download_static.go) | [DownloadStatic](file:///workspace/api/file/download_static.go#L17) | 静态文件下载（路径可控） |
| [upload.go](file:///workspace/api/file/upload.go) | [UploadFile](file:///workspace/api/file/upload.go#L16) | multipart 上传 |
| [zip.go](file:///workspace/api/file/zip.go) | [ZipDir](file:///workspace/api/file/zip.go#L17) | `async=1` 时起 goroutine；**分支逻辑有误** |
| [unzip.go](file:///workspace/api/file/unzip.go) | [Unzip](file:///workspace/api/file/unzip.go#L18) | 解压；**重复执行** |
| [delete.go](file:///workspace/api/file/delete.go) | [DeleteFile](file:///workspace/api/file/delete.go#L13) | 删除文件/目录 |
| [mkdir.go](file:///workspace/api/file/mkdir.go) | [MkdirAll](file:///workspace/api/file/mkdir.go#L13) | 递归建目录 |
| [rename.go](file:///workspace/api/file/rename.go) | [Rename](file:///workspace/api/file/rename.go#L14) | 重命名 |
| [create.go](file:///workspace/api/file/create.go) | [Create](file:///workspace/api/file/create.go#L14) | 创建空文件 |
| [detail.go](file:///workspace/api/file/detail.go) | [Detail](file:///workspace/api/file/detail.go#L10) | 文件元信息 |
| [directory_size.go](file:///workspace/api/file/directory_size.go) | [GetDirSize](file:///workspace/api/file/directory_size.go#L11) | 目录大小统计 |

## 8. `api/k3s/`

| 文件 | 符号 | 说明 |
| --- | --- | --- |
| [get.go](file:///workspace/api/k3s/get.go) | [getClientset](file:///workspace/api/k3s/get.go#L32-L47) | `sync.Once` 单例 clientset |
| 同上 | [getRestConfig](file:///workspace/api/k3s/get.go#L49-L55) | `KUBE_CONFIG` 或 InCluster |
| 同上 | [getIngressRouteGVRs](file:///workspace/api/k3s/get.go#L57-L62)、[getIngressRouteTCPGVRs](file:///workspace/api/k3s/get.go#L64-L71) | Traefik CRD 的双 Group 探测 |
| 同上 | [Get](file:///workspace/api/k3s/get.go#L73) | 单资源 YAML；**未注册路由** |
| [list.go](file:///workspace/api/k3s/list.go) | [List](file:///workspace/api/k3s/list.go#L17) | 资源别名 → GVR → 列表 |
| [describe.go](file:///workspace/api/k3s/describe.go) | [Describe](file:///workspace/api/k3s/describe.go#L27) | 目前只实现 `node` |
| [apply.go](file:///workspace/api/k3s/apply.go) | [Apply](file:///workspace/api/k3s/apply.go#L54) | YAML 多文档解析 + Server-Side Apply |
| [delete.go](file:///workspace/api/k3s/delete.go) | [Delete](file:///workspace/api/k3s/delete.go#L17) | 删除资源 |
| [exec.go](file:///workspace/api/k3s/exec.go) | [Exec](file:///workspace/api/k3s/exec.go#L26) | Pod 内命令执行（`/api/k3s/:id/exec/:pod_name`） |
| [log.go](file:///workspace/api/k3s/log.go) | [Log](file:///workspace/api/k3s/log.go#L19) | Pod 日志（支持 tail/follow） |
| [log_download.go](file:///workspace/api/k3s/log_download.go) | [LogDownload](file:///workspace/api/k3s/log_download.go#L33) | 打包下载 `maxDownloadLogPods=100` 个 Pod 日志 |
| [rollout.go](file:///workspace/api/k3s/rollout.go) | [RolloutRestart](file:///workspace/api/k3s/rollout.go#L17) | `kubectl rollout restart` 等价 |
| [scale.go](file:///workspace/api/k3s/scale.go) | [Scale](file:///workspace/api/k3s/scale.go#L17) | 副本数调整 |
| [patch.go](file:///workspace/api/k3s/patch.go) | [Patch](file:///workspace/api/k3s/patch.go#L22) | JSON merge patch |
| [node.go](file:///workspace/api/k3s/node.go) | [Node](file:///workspace/api/k3s/node.go#L28)、[NodeLabelAdd](file:///workspace/api/k3s/node.go#L110)、[NodeLabelRemove](file:///workspace/api/k3s/node.go#L166) | 节点列表与标签增删 |
| [event.go](file:///workspace/api/k3s/event.go) | [Event](file:///workspace/api/k3s/event.go#L8) | 事件查询；**未注册** |

## 9. `api/redis/`、`api/ftp/`、`api/s3client/`

| 文件 | 符号 | 说明 |
| --- | --- | --- |
| [redis/query.go](file:///workspace/api/redis/query.go) | `scanLimit=1000`、`valuePreviewLimit=20000`、`randomKeyLimit=100` | 扫描/预览/随机 Key 上限 |
| 同上 | [getRedisClient](file:///workspace/api/redis/query.go#L32-L36)、[reqContext](file:///workspace/api/redis/query.go#L38-L45) | 取缓存客户端、15s 超时上下文 |
| 同上 | [readValue](file:///workspace/api/redis/query.go#L47-L99)、[describeKey](file:///workspace/api/redis/query.go#L101-L126) | 按类型读值、Key 元信息 |
| 同上 | [Scan](file:///workspace/api/redis/query.go#L128-L182)、[Get](file:///workspace/api/redis/query.go#L184-L204)、[Random](file:///workspace/api/redis/query.go#L210-L262)、[Delete](file:///workspace/api/redis/query.go#L264-L280) | 四个 handler（均关闭共享客户端） |
| [ftp/base.go](file:///workspace/api/ftp/base.go) | [checkPath](file:///workspace/api/ftp/base.go#L14-L21)、[GetFtpClient](file:///workspace/api/ftp/base.go#L28-L39) | `..` 校验、**未解密**建客户端 |
| [ftp/list.go](file:///workspace/api/ftp/list.go) | [List](file:///workspace/api/ftp/list.go#L14) | 目录列表 |
| [ftp/download.go](file:///workspace/api/ftp/download.go) | [Download](file:///workspace/api/ftp/download.go#L19) | 下载 |
| [ftp/upload.go](file:///workspace/api/ftp/upload.go) | [Upload](file:///workspace/api/ftp/upload.go#L19) | 上传 |
| [ftp/delete.go](file:///workspace/api/ftp/delete.go) | [Delete](file:///workspace/api/ftp/delete.go#L21) | 删除 |
| [ftp/rename.go](file:///workspace/api/ftp/rename.go) | [Rename](file:///workspace/api/ftp/rename.go#L22) | 重命名 |
| [ftp/mkdir.go](file:///workspace/api/ftp/mkdir.go) | [Mkdir](file:///workspace/api/ftp/mkdir.go#L21) | 建目录 |
| [s3client/base.go](file:///workspace/api/s3client/base.go) | [GetS3Client](file:///workspace/api/s3client/base.go#L16-L23)、[GetSTSClient](file:///workspace/api/s3client/base.go#L25-L32) | 配置读取与客户端构造 |
| [s3client/list_bucket.go](file:///workspace/api/s3client/list_bucket.go) | [ListBucket](file:///workspace/api/s3client/list_bucket.go#L10) | 桶列表 |
| [s3client/list_object.go](file:///workspace/api/s3client/list_object.go) | [ListObject](file:///workspace/api/s3client/list_object.go#L17) | 前缀/分隔符列举 |
| [s3client/download.go](file:///workspace/api/s3client/download.go) | [Download](file:///workspace/api/s3client/download.go#L24) | 对象下载 |
| [s3client/upload.go](file:///workspace/api/s3client/upload.go) | [Upload](file:///workspace/api/s3client/upload.go#L21) | 对象上传 |
| [s3client/stat.go](file:///workspace/api/s3client/stat.go) | [Stat](file:///workspace/api/s3client/stat.go#L12) | 对象元信息 |
| [s3client/token.go](file:///workspace/api/s3client/token.go) | [Token](file:///workspace/api/s3client/token.go#L9) | STS 临时凭证 |
| [s3client/peek_zip.go](file:///workspace/api/s3client/peek_zip.go) | [PeekZip](file:///workspace/api/s3client/peek_zip.go#L16) | 不解压查看 ZIP 目录 |

## 10. `module/`：实现层

| 文件 | 关键符号 | 说明 |
| --- | --- | --- |
| [crypto/aes.go](file:///workspace/module/crypto/aes.go) | [AESGCMEncrypt](file:///workspace/module/crypto/aes.go#L16-L34)、[AESGCMDecrypt](file:///workspace/module/crypto/aes.go#L37-L62)、[aesKeyBytes](file:///workspace/module/crypto/aes.go#L65-L68) | 配置密码加解密 |
| [crypto/crypto.go](file:///workspace/module/crypto/crypto.go) | `HMACSha1`、`Sha1Sum`、`GetFileMD5`、`MD5`、`Base64Decode`、`JsonDecode` | 通用哈希/编码 |
| [fsutil/file.go](file:///workspace/module/fsutil/file.go) | [Zip](file:///workspace/module/fsutil/file.go#L124-L182)、[Unzip](file:///workspace/module/fsutil/file.go#L184-L214)、[UnzipFrom](file:///workspace/module/fsutil/file.go#L279-L304) | 压缩/解压（archiver v4） |
| 同上 | `GetFiles`、`GetFilesMap`、`GetDirFilesAsMap`、`ReadFileAs`、`ReadConfig`、`GetFileMapping` | 目录遍历与配置读取 |
| 同上 | `JsonDecodeX`、`ExtractURL`、`ExtractURLToMap` | JSON 与 URL 提取 |
| 同上 | `MakeVarLogFile/Dir`、`MakeTmpWorkDir`、`GetNowTimeStamp` | 临时目录与时间戳 |
| [fsutil/tar.go](file:///workspace/module/fsutil/tar.go) | [CreateTarFile](file:///workspace/module/fsutil/tar.go#L16-L36)、[TarWriteFile](file:///workspace/module/fsutil/tar.go#L38-L73)、[TarWriteFiles](file:///workspace/module/fsutil/tar.go#L75-L87)、[ExtractTarFile](file:///workspace/module/fsutil/tar.go#L89) | tar 打包/解包 |
| [ftp/client.go](file:///workspace/module/ftp/client.go) | [DefaultFTPClient](file:///workspace/module/ftp/client.go#L30-L50)、[NewFTPClient](file:///workspace/module/ftp/client.go#L52-L) | FTP 封装（prefix 限制、列表、上传下载） |
| [ssh/ssh.go](file:///workspace/module/ssh/ssh.go) | [SSHExecCommand](file:///workspace/module/ssh/ssh.go#L165-L175)、[SSHExecCommandWithTimeout](file:///workspace/module/ssh/ssh.go#L177-L273) | 命令执行；握手/空闲超时看门狗 |
| 同上 | [SSHUploadFile](file:///workspace/module/ssh/ssh.go#L275-L408)、[SSHOpenFile](file:///workspace/module/ssh/ssh.go#L410) | 上传与远程文件读取 |
| [ssh/sftp.go](file:///workspace/module/ssh/sftp.go) | [OpenSFTP](file:///workspace/module/ssh/sftp.go#L31-L) | SFTP 会话 |
| [mysql/mysql.go](file:///workspace/module/mysql/mysql.go) | [GetMySQLTables](file:///workspace/module/mysql/mysql.go#L17-L46)、[GetMySQLTableFields](file:///workspace/module/mysql/mysql.go#L48-L73)、[GetMySQLTableIndexes](file:///workspace/module/mysql/mysql.go#L75-L122)、[GetCreateTableSQL](file:///workspace/module/mysql/mysql.go#L124-L151) | 元数据查询 |
| 同上 | [GetGormDBFrom](file:///workspace/module/mysql/mysql.go#L153-L169)、[UpdateDataByID](file:///workspace/module/mysql/mysql.go#L171-L179)、[DeleteDataByID](file:///workspace/module/mysql/mysql.go#L181-L191)、[QueryList](file:///workspace/module/mysql/mysql.go#L193-L205)、[CreateData](file:///workspace/module/mysql/mysql.go#L207) | 通用 CRUD |
| [mysql/gorm.go](file:///workspace/module/mysql/gorm.go) | [BuildQueryRaws](file:///workspace/module/mysql/gorm.go#L13-L) | `where` map → SQL 片段（标识符拼接） |
| [redis/redis.go](file:///workspace/module/redis/redis.go) | [GetRedisClient](file:///workspace/module/redis/redis.go#L17-L30)、[GetDefaultBusinessRedisClient](file:///workspace/module/redis/redis.go#L32-L) | 全局单例客户端（按 addr 缓存） |
| [s3/base.go](file:///workspace/module/s3/base.go) | [GetS3Client](file:///workspace/module/s3/base.go#L12-L) | AWS SDK v2 S3 客户端 |
| [s3/sts.go](file:///workspace/module/s3/sts.go) | [NewSTSClient](file:///workspace/module/s3/sts.go#L58-L)、`StsClient` | STS 临时凭证 |
| [jwt/jwt.go](file:///workspace/module/jwt/jwt.go) | [GenerateToken](file:///workspace/module/jwt/jwt.go#L42-L56)、[ParseToken](file:///workspace/module/jwt/jwt.go#L58-L73)、[GetExpireSeconds](file:///workspace/module/jwt/jwt.go#L75) | 未接入鉴权 |
| [model/define.go](file:///workspace/module/model/define.go) | [RegisterModelParse](file:///workspace/module/model/define.go#L10-L14)、[GetModelParseFunc](file:///workspace/module/model/define.go#L16-L) | 表→解析函数注册表（无调用方） |
| [common/common.go](file:///workspace/module/common/common.go) | `GetStringValFromMap`、`ToString`、`GetEnv`、`GetHost` | 通用转换 |

## 11. 前端资源索引

| 路径 | 内容 |
| --- | --- |
| `pages/layout.tmpl` | 全局脚本引入、`window.PageData`、Vue 挂载、面包屑、`yield` |
| `pages/index.tmpl` | 资源配置管理（类型 Tab、CRUD 弹窗、测试） |
| `pages/mysql/index.tmpl` | MySQL 数据浏览（表列表、分页、结构、SQL 执行） |
| `assets/js/my.js` | 全局工具函数（见 [04-frontend-pages.md](04-frontend-pages.md#6-全局-js-工具函数assetsjsmyjs)） |
| `assets/js/json-to-go.js` | JSON → Go Struct（仅 `tool/json` 页） |
| `assets/js/vue.global.js` / `axios.min.js` / `jsoneditor9.8.js` | 核心运行时 |
| `assets/css/my.css` | 站点自定义样式 |
| `pages/k3s/*.yml` | kube-backup 模板、Grafana Service 示例（供人工使用） |

## 12. 调用关系速查

| 起点 | 调用链 |
| --- | --- |
| `GET /` | `NoRoute` → `createPageHandlerV2("index")` → `pageBinding` → `renderClient.HTML` |
| `GET /mysql/:id` | `PageData` → `QueryMySQLResource` → `resource` + `decodeConfig` → 渲染 |
| `POST /api/mysql/:id/query/:table` | `mysqlHandler.Query` → `GetDB` → `GetMySQLDBByID` → 解密 → GORM → `BuildQueryRaws` |
| `POST /api/config/create` | `resource.Create` → `buildResource` → `AESGCMEncrypt` → `CreateResource` |
| `POST /api/config/update/:id` | `Update` → `GetResource` → 保留/重加密密码 → `UpdateResource` → `ClearClientCache` |
| `POST /api/config/test/:id` | `Test` → `DecodeConfig` → `AESGCMDecrypt` → `testByType` → 各协议握手 |
| `GET /api/redis/:id/scan` | `Scan` → `getRedisClient` → `GetRedisClientByID` → 解密 → `Ping` → 缓存 |
| `GET /api/k3s/:id/list/:resource` | `List` → `getClientset`（`sync.Once`）→ `getRestConfig` → discovery → dynamic client |
| `GET /api/ftp/:id/list` | `List` → `GetFtpClient` → `QueryFTPConfig` → `DefaultFTPClient`（密文直传） |
| `GET /api/file/list` | `ListFile` → `module/fsutil` 目录遍历 |
| `GET /api/s3/:id/object/list` | `ListObject` → `GetS3Client` → `QueryS3Config`（字段不匹配）→ AWS SDK |

## 13. 推荐阅读路线

1. 先看 [README](README.md) 与 [01-architecture.md](01-architecture.md)，建立全局印象。
2. 按 `main.go` 的路由表读 [03-api-reference.md](03-api-reference.md)，明确“哪些能用”。
3. 想改资源配置：`api/resource/*` → `container/sqlite.go` → `container/resource.go`。
4. 想改页面：`pages/layout.tmpl` → 对应 `pages/*/index.tmpl` → [04-frontend-pages.md](04-frontend-pages.md)。
5. 想加一种新资源类型：`TypeMetas` 增加元数据 → `ConnectionConfig`/`SetExtra`/`Extra` → `Query*Config` → 新 handler → 缓存失效逻辑。
6. 部署前务必读 [06-build-run-deploy.md](06-build-run-deploy.md) 第 7 节缺陷清单。
