# 03 · HTTP API 参考

> 所有路由都在 [main.go](../../main.go) 中注册。除特别说明外，返回 HTTP 200，响应体为 `{"code":0,"message":"success","data":...}`；业务失败时 `code` 为负数、`message` 为错误描述、`data` 为 `null`。

## 0. 通用约定

### 鉴权

```go
router.Use(gin.Recovery(), gin.Logger(), gin.BasicAuth(gin.Accounts{
    "admin": appConfig.LoginPassword,
}))
```

- 用户名固定 `admin`，密码取 `LOGIN_PASSWORD`。
- 中间件挂在根路由上，**静态资源 `/assets` 与页面同样需要 Basic Auth**。
- 没有 session、JWT 或角色体系；`LOGIN_USER`/`TOKEN_KEY`/`JWT_SECRET` 等环境变量并未生效。

### 响应包络

```go
type JSONResponse struct {
    Code    int64       `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}
```

```bash
curl -u admin:$LOGIN_PASSWORD http://127.0.0.1:8874/api/config/types
```

### 参数来源

| 位置 | 示例 | 解析方式 |
| --- | --- | --- |
| Query | `?dir=/data` | `ctx.Query/DefaultQuery/QueryArray` |
| Path | `/api/mysql/:id/query/:table` | `ctx.Param` |
| JSON Body | `{"where":{...}}` | `ctx.ShouldBindJSON` / `api.AllPostParams`（保留数字精度） |
| multipart | `file=@x.zip` | `ctx.FormFile` |
| 原始 Body | YAML / SQL / 文件内容 | `ctx.GetRawData()` |

---

## 1. 页面路由

| 方法 | 路径 | 模板 | 说明 |
| --- | --- | --- | --- |
| GET | `/` | `index` | 没有显式 `/` 路由，由 `NoRoute` 兜底渲染。标题“首页”，描述“页面不存在”，隐藏面包屑。 |
| GET | `/mysql/:id` | `mysql/index` | `id` 为 **resource ID**；`PageData` 注入 `config_id/config_name/database`，配置不存在时渲染 `error` 模板（HTTP 200）。 |
| GET | `/ftp/:id` | `service/ftp` | **模板不存在**，实测 HTTP 500：`html/template: "service/ftp" is undefined`。 |
| GET | `/ssh/:id` | `service/ssh` | 同上，500。 |
| GET | `/s3/:id` | `service/s3` | 同上，500。 |
| GET | `/redis/:id` | `service/redis` | 同上，500。 |
| GET | `/tool/json` | `tool/json` | JSON 工具页。 |
| GET | `/assets/*` | — | 静态资源；生产模式来自 `go:embed`，开发模式来自 `./assets`。 |
| ANY | 未匹配路径 | `index` | `NoRoute`，HTTP 200 + “页面不存在”首页。 |

> `pages/` 下还有 `file/manage`、`ftp/index`、`k3s/index`、`k3s/image`、`k3s/image2`、`redis/index`、`s3browse/index`、`setting/index`、`ssh/file`、`system/info`、`system/state`、`text/*`、`http/request`、`user/login` 等模板，但**当前 main.go 未注册对应路由**（详见 [04-frontend-pages.md](04-frontend-pages.md)）。

---

## 2. 资源配置管理 `/api/config`

统一管理 MySQL / FTP / SSH / S3 / Redis 连接配置，存储于 SQLite（见 [05-data-and-security.md](05-data-and-security.md)）。

### 2.1 类型元数据

`GET /api/config/types`

返回 `[]TypeMeta`，驱动前端动态表单：

```json
[{
  "type": "mysql", "label": "MySQL",
  "host_label": "主机", "host_placeholder": "10.0.0.1",
  "user_label": "用户名", "password_label": "密码",
  "require_host": true, "default_port": 3306,
  "extra_fields": [{"key":"database","label":"数据库","placeholder":"连接的库名","required":true}]
}]
```

各类型默认值：mysql 3306（database 必填）、ftp 21（prefix 可选）、ssh 22、s3 无端口（region 必填，bucket 可选）、redis 6379（db 可选）。

### 2.2 列表

`GET /api/config/list?type=mysql`

- `type` 为空返回全部；非法类型返回 `code=-1`。
- 返回 `[]MaskedConfig`，**密码不回传**，只给 `has_password`：

```json
[{
  "id": 1, "type": "mysql", "name": "prod-mysql",
  "host": "10.0.0.1", "port": 3306, "username": "root",
  "has_password": true, "extra": {"database": "app"},
  "created_at": "2026-10-27 10:00:00", "updated_at": "2026-10-27 10:00:00"
}]
```

### 2.3 新建 / 更新 / 删除

| 方法 | 路径 | 请求体 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/config/create` | `SaveRequest` | 新建；密码 `AESGCMEncrypt` 后入库；返回 `MaskedConfig` |
| POST | `/api/config/update/:id` | `SaveRequest` | `type` 不允许修改；**密码留空表示保留原密码**；更新后 `ClearClientCache(id)` |
| POST | `/api/config/delete/:id` | — | 删除并清理连接缓存；返回 `data=null` |

`SaveRequest`：

```json
{
  "type": "mysql",
  "name": "prod-mysql",
  "host": "10.0.0.1",
  "port": 3306,
  "username": "root",
  "password": "plain-text",
  "extra": {"database": "app"}
}
```

校验规则：
- `type` 必须是 5 种之一，`name` 必填。
- `require_host=true` 的类型必须给 `host`。
- `port <= 0` 时取类型默认端口。
- `extra` 只保留该类型声明的字段；`required` 字段为空则报错。
- `(config_type, name)` 唯一；冲突返回“同名配置已存在”。

### 2.4 连通性测试

`POST /api/config/test/:id`（body 可为空）

按类型执行，统一 5s 超时：

| 类型 | 测试方式 |
| --- | --- |
| mysql | `sql.Open` + `PingContext`（先解密密码） |
| ftp | `ftp.Dial` + `Login` |
| ssh | `ssh.Dial` 建立会话 |
| s3 | `ListBuckets`（`InsecureSkipVerify`） |
| redis | `PING` |

返回：

```json
{"id":1,"ok":true,"message":"连接成功","cost_ms":12,"tested_at":"2026-10-27 10:00:00"}
```

失败也是 HTTP 200，`ok=false`，`message` 为具体原因。

---

## 3. 本机文件管理 `/api/file`

> 直接操作**运行进程所在主机的文件系统**，没有根目录限制；仅有 Basic Auth 保护。

| 方法 | 路径 | 参数 | 返回/说明 |
| --- | --- | --- | --- |
| GET | `/api/file/list` | `dir`（默认 `/`） | `[{name,is_dir,path,size,modify}]` |
| GET | `/api/file/download` | `file`（必填） | 文件流（`Content-Disposition: attachment`） |
| POST | `/api/file/upload` | query `dir`；multipart `file` | `{dest}`；未给 `dir` 时写入 `/tmp/tmp_upload_<unix>`；重名自动加时间戳前缀 |
| GET | `/api/file/zip` | `dir`（必填，且不能为 `/`）、`target_zip`、`async=1` | `{dest}`；默认命名为 `<dir>-<unix>.zip` |
| GET | `/api/file/unzip` | `file`（必填）、`target_dir`、`async=1` | `{dest}`；默认解压到 `<fileDir>/<name>-<unix>` |
| GET | `/api/file/delete` | `file`（必填） | `os.RemoveAll`；`data=null` |
| GET | `/api/file/mkdirall` | `dir`（必填） | 递归建目录；`data=null` |
| GET | `/api/file/rename` | `file`、`name`（均必填） | 同目录重命名；`{dest}` |
| POST | `/api/file/create` | query `file`；Body 为文件内容 | 写入文件（自动建父目录）；`{dest}` |
| GET | `/api/file/detail` | `file`（必填） | `{name,size,last_modified}` |
| GET | `/api/file/dir_size` | `dir`（必填） | `{dir,size}`，递归累加文件大小 |
| GET | `/api/file/download_static` | `url`（必填）、`root_dir`（默认 `/data1/vrdata/vrlab-public-1`） | 把 URL 内容下载到 `root_dir + url.Path`；`{dest}` |

已知实现问题：
- `zip`/`unzip` 的 `async=1` 分支会启动 goroutine，但**同时仍同步执行同一操作**；`zip` 的 goroutine 还误调用了 `fsutil.Unzip`（见 [06](06-build-run-deploy.md#7-已知缺陷与待清理项)）。
- `download_static` 使用 `http.Get` 无超时、无大小限制。

---

## 4. MySQL `/api/mysql/:id`

`:id` 是 **resource ID**（SQLite 中 `config_type=mysql` 的记录），不是 MySQL 实例名。每个请求通过 `container.GetMySQLDBByID(id)` 获取（缓存的）连接。

### 4.1 查询类

| 方法 | 路径 | 请求体/参数 | 返回 |
| --- | --- | --- | --- |
| POST | `/api/mysql/:id/query/:table` | `{where, page, order_by, page_size}` | `[]map[string]interface{}` |
| POST | `/api/mysql/:id/count/:table` | `{where}` | `{total}` |
| POST | `/api/mysql/:id/group_by_count/:table` | `{where, group_by}` | `[{<group_by>, "count": n}]`，默认按 `source` 分组，按 count 降序 |
| POST | `/api/mysql/:id/distinct/:column` | `{where}` | `[]string` |
| GET | `/api/mysql/:id/table` | — | `[]string` 表名 |
| GET | `/api/mysql/:id/desc/:table` | — | `[]map`（`desc` 原始结果） |
| GET | `/api/mysql/:id/show_create_table/:table` | — | `{sql}` |
| GET | `/api/mysql/:id/show_index/:table` | — | `[]map`（`show index` 原始结果） |
| GET | `/api/mysql/:id/global_variables` | `like`（可选） | `[]map`（`SHOW GLOBAL VARIABLES`） |

`QueryRequest.where` 的 key 语法（由 `module/mysql.BuildQueryRaws` 解释）：

| key 形式 | SQL |
| --- | --- |
| `col` | `col in (?)` |
| `not@col` | `col not in (?)` |
| `no@col` | `col != ?` |
| `gt@gte@lt@lte@col` | `col > / >= / < / <= ?` |
| `like@col` | `col like '%<value>%'` |
| `plike@col` | `col like '<value>%'` |

分页规则：`page<1 → 1`，`page_size<1 → 10`；`order_by` 为空时自动找 `Extra=auto_increment` 的列并 `desc` 排序（找不到则不排序）。

> `order_by` 原样进入 `Order()`；`where` 的 like 值被字符串拼接。仅应在受信环境使用，或补充白名单/参数化。

> 路由缺陷：`distinct` 注册为 `POST /api/mysql/:id/distinct/:column`，但 handler 读取 `ctx.Param("table")`，该值为空，实际执行会失败。前端 `mysql/index.tmpl` 拼接的是 `/api/mysql/<id>/distinct/<table>/<column>`，与路由不匹配。

### 4.2 写入类

| 方法 | 路径 | 请求/参数 | 返回 |
| --- | --- | --- | --- |
| POST | `/api/mysql/:id/update/:table/:row_id` | Body 为字段键值（表单或 JSON） | `{affected}`；`WHERE id = :row_id` |
| POST | `/api/mysql/:id/create/:table` | Body 为字段键值 | 创建后的 map |
| POST | `/api/mysql/:id/batch_create/:table` | Body 为 JSON 数组 | 原始请求字节（`[]byte` 被 JSON 编码为 base64 字符串，前端一般忽略） |
| POST | `/api/mysql/:id/delete/:table/:row_id` | — | `data=null`；`DELETE ... WHERE id = ?` |
| POST | `/api/mysql/:id/exec_sql` | `{"sql":"<base64+urlencode>"}` | `data=null`；解码后 `gorm.Exec` |

`exec_sql` 的 SQL 需要前端做 `encodeURIComponent` 再 Base64（`module/crypto.Base64Decode` 会先 Base64 解码再 URL 解码）。**该接口可执行任意 SQL**。

### 4.3 页面数据

`GET /mysql/:id` 由 `api/mysql.PageData` 生成 `PageData`：

```json
{
  "title": "MySQL - prod-mysql",
  "description": "10.0.0.1:3306 / app",
  "extension": {"config_id": 1, "config_name": "prod-mysql", "database": "app"}
}
```

---

## 5. Kubernetes / K3s `/api/k3s/:id`

> `:id` **未被使用**。客户端由 `getClientset()` 单例创建：优先 `KUBE_CONFIG` 文件，否则 `rest.InClusterConfig()`。即一套进程只服务一个集群。

支持的 `:resource`（不区分大小写，含别名）：

| 分组 | 资源 |
| --- | --- |
| core/v1（命名空间级） | `pods/pod`、`services/service/svc`、`configmaps/configmap/cm`、`secrets/secret`、`serviceaccounts/serviceaccount/sa` |
| core/v1（集群级） | `namespaces/namespace/ns`、`nodes/node` |
| apps/v1 | `deployments/deploy`、`daemonsets/ds`、`statefulsets/sts`、`replicasets/rs` |
| batch/v1 | `cronjobs/cj` |
| Traefik CRD | `ingressroutes/ir`、`ingressroutetcps/irtcp`（同时尝试 `traefik.io` 与 `traefik.containo.us` 两个 Group） |

### 5.1 接口表

| 方法 | 路径 | 参数 | 返回 |
| --- | --- | --- | --- |
| GET | `/api/k3s/:id/list/:resource` | `namespace`（默认 `default`） | 强类型 List 对象 JSON；IngressRoute 会合并两套 CRD 结果 |
| DELETE | `/api/k3s/:id/delete/:resource` | `name`（必填）、`namespace` | `{resource,name,namespace,deleted:true}` |
| POST | `/api/k3s/:id/apply` | query `namespace`（覆盖）、Body=YAML（支持多文档） | `[{kind,name,namespace,action,error}]`，`action` ∈ created/updated/error |
| GET | `/api/k3s/:id/describe/:resource` | `name`、`namespace` | `{output}`；目前只支持 `node`（`k8s.io/kubectl/pkg/describe`） |
| POST | `/api/k3s/:id/exec/:pod_name` | `namespace`、`container`；Body 为 JSON `{"command":[...]}` 或纯文本 shell 命令 | `{stdout,stderr,error?}` |
| GET | `/api/k3s/:id/log/:pod_name` | `namespace`、`container`、`tail`（默认 100，`0`=全量） | `{log}` |
| GET | `/api/k3s/:id/log/download` | `namespace`、`pods`（逗号或重复参数，最多 100 个）、`tail`、`container` | **zip 流**；单个 Pod 失败写入 `<pod>.error.txt` |
| POST | `/api/k3s/:id/rollout/restart/:resource` | `name`、`namespace` | `{resource,name,namespace,action:"restarted",restartedAt}`；支持 deployment/daemonset/statefulset |
| POST | `/api/k3s/:id/scale/:resource` | `name`、`replicas`（非负）、`namespace` | `{resource,name,namespace,action:"scaled",previousReplicas,currentReplicas}`；支持 deployment/statefulset |
| POST | `/api/k3s/:id/patch/:resource` | `name`、`namespace`、`type=strategic|merge|json`；Body=patch JSON | `{resource,name,namespace,action:"patched",patchType,resourceVersion}`；支持 deployment/statefulset/daemonset |
| GET | `/api/k3s/:id/node/list` | — | `[]NodeSummary{name,status,roles,version,internal_ip,pod_cidr,create_time,labels}` |
| POST | `/api/k3s/:id/node/label/add` | query `name`；Body `{"labels":{"k":"v"}}` | `{node}` |
| POST | `/api/k3s/:id/node/label/remove` | query `name`；Body `{"labelKeys":["k"]}` | `{node}` |

### 5.2 行为细节

- **apply**：用 `discovery + restmapper` 把 GVK 映射到 GVR；先 `Get`，不存在则 `Create`，存在则补 `resourceVersion` 后 `Update`（不是 server-side apply）。`namespace` query 只覆盖 namespace-scoped 资源；未指定时默认 `default`。
- **资源详情**：代码中另有 `Get` 处理器（支持 `format=yaml`，会回填 GVK、剥离 `managedFields` 与 `last-applied-configuration`），但**未在 main.go 注册**（即 `GET /api/k3s/:id/get/:resource` 当前不可用）。
- **日志下载**：常量 `maxDownloadLogPods = 100`（注释写 50，以代码为准）；pod 名需匹配 DNS-1123 子域名，避免 zip 路径穿越；客户端断开后停止后续 Pod。
- **describe**：仅映射了 `nodes/node`，其他资源返回 `unsupported resource for describe`。
- **exec**：不带 TTY，不支持 stdin；命令执行结果一次性返回。

---

## 6. Redis `/api/redis/:id`

`:id` 为 resource ID；`container.GetRedisClientByID` 建立并缓存客户端（查询前会 `PING`）。

| 方法 | 路径 | 参数 | 返回 |
| --- | --- | --- | --- |
| GET | `/api/redis/:id/scan` | `prefix`（必填） | `{prefix,match,count,limit:1000,reached,keys:[keyItem]}` |
| GET | `/api/redis/:id/get` | `key`（必填） | `keyItem` |
| GET | `/api/redis/:id/random` | — | `{count,limit:100,keys:[keyItem]}`；`RANDOMKEY` 多次取样去重 |
| DELETE | `/api/redis/:id/delete` | `key`（必填） | `{key,deleted}`；1=删除成功，0=不存在 |

`keyItem`：

```json
{"key":"user:1","type":"hash","ttl":3600,"value":{"a":"b"},"truncated":false,"error":""}
```

- `ttl`：秒，`-1` 永不过期，`-2` 不存在。
- 值读取策略：string 截断 20000 字符；list 最多 1000 个元素；set 最多 1000；zset 返回 `[{member,score}]` 最多 1000；hash 全量返回；stream 等类型提示“不支持预览”。
- 单次请求 15s 超时（继承请求 context）。

> 注意：handler 里 `defer client.Close()` 会关闭 `container` 缓存的客户端，导致后续请求复用已关闭的客户端。见缺陷清单。

---

## 7. FTP `/api/ftp/:id`

`:id` 为 resource ID；路径均相对配置中的 `prefix`。

| 方法 | 路径 | 参数 | 返回 |
| --- | --- | --- | --- |
| GET | `/api/ftp/:id/list` | `path`（默认 `.`） | `[]*ftp.Entry`（`Name/Size/Type/Time/Target`） |
| GET | `/api/ftp/:id/download` | `path`（必填） | 文件流（`http.ServeContent`） |
| POST | `/api/ftp/:id/upload` | multipart：`file`；form：`path`（默认 `.`）、`name`（默认上传文件名） | `{path}` |
| POST | `/api/ftp/:id/delete` | `{"path":"..."}` | `{path}` |
| POST | `/api/ftp/:id/rename` | `{"from":"...","to":"..."}` | `{from,to}` |
| POST | `/api/ftp/:id/mkdir` | `{"path":"..."}` | `{path}` |

路径安全：`checkPath` 拒绝包含 `..` 的路径段；`upload` 的 `name` 会先 `filepath.Base`。

> 缺陷：`GetFtpClient` 直接用配置中的密文密码登录，未调用 `AESGCMDecrypt`，按当前“密码加密入库”的模型会导致 FTP 登录失败。

---

## 8. S3 / MinIO `/api/s3/:id`

`:id` 为 resource ID；Bucket 由请求参数指定。

| 方法 | 路径 | 参数 | 返回 |
| --- | --- | --- | --- |
| GET | `/api/s3/:id/bucket/list` | — | `[]string` |
| GET | `/api/s3/:id/object/list` | `bucket`（必填）、`prefix` | `[{key,name,type,size,last_modified}]`，`type` ∈ directory/file |
| GET | `/api/s3/:id/object/download` | `bucket`；`key` 或 `prefix` | 单对象：文件流；前缀：下载所有对象 → 打 zip → 返回 zip |
| POST | `/api/s3/:id/object/upload` | `bucket`；`key`（单文件）或 `prefix`（zip 批量）；multipart `file` | 单文件 `{bucket,key}`；批量 `{files,uploadKeys}` |
| GET | `/api/s3/:id/object/peek_zip` | `bucket`、`key`（zip 对象）、`name`（zip 内文件） | `{content}`（文本内容） |
| GET | `/api/s3/:id/object/stat` | `bucket`、`key` | `HeadObject` 结果 |
| GET | `/api/s3/:id/token` | `bucket`、`prefix`（均必填） | `TempToken`（STS 临时凭证，3600s） |

行为细节：
- `prefix` 批量上传要求上传的是 **zip**，服务端解压后逐个 `PutObject` 到 `<prefix>/<相对路径>`；临时目录用后即删。
- 批量下载目录时先落盘 `/tmp/s3/download_<ts>/`，再 `fsutil.Zip` 为 `/tmp/s3/download_<ts>.zip`，用 `http.ServeFile` 返回。
- `object/list` 使用 `Delimiter="/"`，只列当前层。

> 缺陷：`QueryS3Config` 解码的 `S3Config` 字段名（`access_key/secret_key/endpoint`）与写入的 `ConnectionConfig`（`host/username/password`）不一致，且 secret 未解密；按新配置模型创建的 S3 资源，浏览接口实际拿不到 AK/SK/Endpoint。见 [06](06-build-run-deploy.md#7-已知缺陷与待清理项)。

---

## 9. 未注册 / 已注释的接口

| 接口 | 状态 |
| --- | --- |
| `GET /api/k3s/:id/get/:resource` | handler 存在（`api/k3s/get.go`），未注册 |
| `GET /api/k3s/:id/event` | handler 仅返回 Hello World，未注册 |
| `/api/ssh/exec`、`/api/ssh/test`、`/api/ssh/upload`、`/api/ssh/download`、`/api/ssh/file/*` | main.go 中整组注释；但 `module/ssh` 已实现完整能力 |
| `/api/execute/*`（`cp`、`k3s`、`crictl`、`mc`、`tar`） | 仅被历史页面调用，后端不存在 |
| `/api/system/info`、`/api/system/store_state` | 仅被 `system/*.tmpl` 调用，后端不存在 |
| `/api/user/login` | 仅被 `user/login.tmpl` 调用，后端不存在 |
| `/api/hashids/encode|decode` | 仅被 `text/hashids.tmpl` 调用，后端不存在 |
| `/api/project/txtzip/convert`、`/api/k3s/image/list`、`/api/ftp/config` | 历史页面调用，后端不存在 |

---

## 10. 调用示例

```bash
BASE=http://127.0.0.1:8874
AUTH=admin:change-me

# 新建 MySQL 配置
curl -u $AUTH -X POST $BASE/api/config/create \
  -H 'Content-Type: application/json' \
  -d '{"type":"mysql","name":"local","host":"127.0.0.1","port":3306,
       "username":"root","password":"root","extra":{"database":"app"}}'

# 列出配置（密码脱敏）
curl -u $AUTH "$BASE/api/config/list?type=mysql"

# 测试连通性
curl -u $AUTH -X POST $BASE/api/config/test/1

# 分页查询（id=1 的 MySQL，表 users，第 2 页）
curl -u $AUTH -X POST $BASE/api/mysql/1/query/users \
  -H 'Content-Type: application/json' \
  -d '{"where":{"status":"active","gt@age":18},"page":2,"page_size":20,"order_by":"id desc"}'

# K3s：列 default 命名空间的 Pod
curl -u $AUTH "$BASE/api/k3s/any/list/pods?namespace=default"

# K3s：apply 一段 YAML
curl -u $AUTH -X POST "$BASE/api/k3s/any/apply?namespace=default" \
  -H 'Content-Type: application/yaml' --data-binary @deploy.yaml

# Redis：按前缀扫描
curl -u $AUTH "$BASE/api/redis/1/scan?prefix=user:"

# S3：列对象
curl -u $AUTH "$BASE/api/s3/1/object/list?bucket=my-bucket&prefix=logs/"
```
