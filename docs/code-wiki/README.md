# go-developer-web Code Wiki

> 面向研发的代码维基：说明这个项目“是什么、怎么分层、有哪些模块/关键类型与函数、接口怎么用、如何运行与部署”。
> 本文档基于仓库源码静态分析，并在本地完成了 `go build` 与接口冒烟验证（见 [06-build-run-deploy.md](06-build-run-deploy.md)）。

## 1. 项目是什么

`go-developer-web`（构建产物名 `devops-console`）是一个**单进程、单二进制**的开发者/运维 Web 控制台，使用 Go + Gin 编写，页面由服务端模板 + Vue 3（全局构建版）渲染，静态资源与页面模板通过 `go:embed` 打包进二进制。

核心能力：

| 能力 | 说明 | 主要代码 |
| --- | --- | --- |
| 资源配置管理 | 统一管理 MySQL / FTP / SSH / S3 / Redis 连接配置，密码 AES-GCM 加密后落库 SQLite | [api/resource](../../api/resource)、[container/resource.go](../../container/resource.go) |
| MySQL 浏览与运维 | 表列表、分页查询、条件构造、增删改、建表语句、索引、全局变量、任意 SQL 执行 | [api/mysql](../../api/mysql)、[module/mysql](../../module/mysql) |
| Kubernetes / K3s 运维 | Pod/Deployment/Node 等资源列表与详情、apply、exec、日志、批量下载日志、重启、扩缩容、patch、节点标签 | [api/k3s](../../api/k3s) |
| Redis 浏览 | 按前缀 SCAN、按 key 查看（string/list/set/zset/hash）、随机取样、删除 | [api/redis/query.go](../../api/redis/query.go) |
| 对象存储 S3 / MinIO | Bucket/Object 列表、上传、下载（目录打包 zip）、stat、zip 内文件预览、STS 临时凭证 | [api/s3client](../../api/s3client)、[module/s3](../../module/s3) |
| FTP 文件管理 | 列表、上传、下载、删除、重命名、新建目录（相对配置的 prefix） | [api/ftp](../../api/ftp)、[module/ftp](../../module/ftp) |
| 本机文件管理 | 列目录、上传、下载、打包/解压、删除、新建、重命名、目录大小、按 URL 下载 | [api/file](../../api/file) |
| JSON 工具 | 纯前端 JSON 编辑器/格式化页面 | [pages/tool/json.tmpl](../../pages/tool/json.tmpl) |

> 重要背景：仓库处于一次“从早期个人工具集 → 配置驱动控制台”的重构中。后端主链路（配置管理、MySQL、K3s、Redis、S3、FTP、文件）已按新模型实现，但 `pages/` 下仍保留了大量**未挂载路由/调用已删除接口**的历史页面。详见 [04-frontend-pages.md](04-frontend-pages.md) 与 [06-build-run-deploy.md](06-build-run-deploy.md#7-已知缺陷与待清理项)。

## 2. 文档导航

| 文档 | 内容 |
| --- | --- |
| [01-architecture.md](01-architecture.md) | 整体架构、分层、目录结构、关键请求链路、技术选型与依赖矩阵 |
| [02-backend-modules.md](02-backend-modules.md) | `container` 与 `module/*` 的职责、关键类型/函数说明、调用关系 |
| [03-api-reference.md](03-api-reference.md) | 全部已注册 HTTP API：路径、参数、请求体、返回、注意事项 |
| [04-frontend-pages.md](04-frontend-pages.md) | 模板渲染机制、Vue 页面结构、页面与 API 对照、遗留页面清单 |
| [05-data-and-security.md](05-data-and-security.md) | SQLite 数据模型、配置 JSON、加密与缓存、鉴权、安全风险 |
| [06-build-run-deploy.md](06-build-run-deploy.md) | 环境变量、编译/运行、开发模式、K8s 部署要求、冒烟结果、缺陷清单 |
| [07-code-map.md](07-code-map.md) | 文件级代码地图与关键函数索引（按包） |

## 3. 五分钟快速开始

```bash
# 1) 准备环境变量（项目启动时 godotenv/autoload 会自动加载当前目录 .env）
cp .env.example .env
# 至少修改：
#   LOGIN_PASSWORD=change-me      # Basic Auth 密码（用户名固定 admin）
#   SQLITE_PATH=./data/devops-console.db
#   CONFIG_SECRET=<随机值>         # 连接配置密码的 AES-GCM 密钥

# 2) 运行
go run .

# 3) 访问（默认端口见 PORT，.env.example 中为 8874）
#    http://127.0.0.1:8874/
#    浏览器会弹出 Basic Auth：用户名 admin，密码为 LOGIN_PASSWORD
```

生产构建：

```bash
go build -o devops-console .
./devops-console
```

> 页面模板和 `assets/` 已通过 `go:embed` 打进二进制；只有设置 `ENV=develop` 时才从本地 `./pages`、`./assets` 读取。

## 4. 技术栈速览

- 语言/运行时：Go 1.24（`go.mod`），本地验证使用 Go 1.25.1。
- Web：Gin v1.9.1；模板引擎 `unrolled/render` v1.6.1。
- 配置存储：GORM + `glebarez/sqlite`（纯 Go SQLite 驱动，无需 CGO）。
- 外部资源客户端：`gorm.io/driver/mysql`、`go-redis/redis/v8`、`jlaffaye/ftp`、`pkg/sftp` + `x/crypto/ssh`、AWS SDK v2（S3/STS）。
- 集群运维：`k8s.io/client-go`、`k8s.io/api`、`k8s.io/apimachinery`、`k8s.io/kubectl`。
- 压缩/归档：`mholt/archiver/v4`。
- 前端：Vue 3 global build、Bootstrap 3.4、axios、JSONEditor、bootbox、dayjs。

## 5. 关键事实（先记住这些）

1. **单集群 K3s 客户端是单例**：`api/k3s` 的 `getClientset()` 使用 `sync.Once`，路由里的 `:id` 实际未被使用。
2. **所有 HTTP 路由（含静态资源）都挂了 Gin BasicAuth**：用户名固定 `admin`，密码取 `LOGIN_PASSWORD`。
3. **配置密码落库即加密**：`CONFIG_SECRET` 经 SHA-256 派生 32 字节密钥，AES-256-GCM，密文格式 `base64(nonce||ciphertext)`。
4. **MySQL / Redis 连接按配置 ID 缓存**：`container.GetMySQLDBByID` / `GetRedisClientByID`，配置更新/删除时 `ClearClientCache` 失效。
5. **首页是 `NoRoute` 兜底渲染的 `index` 模板**：`GET /` 与任意未匹配路径都会渲染“首页/页面不存在”的配置管理页。
6. **部分页面路由引用了不存在的模板**：`/ftp/:id`、`/ssh/:id`、`/s3/:id`、`/redis/:id` 指向 `service/*`，仓库中没有该模板，实测返回 500。

---

维护建议：改动接口/模型时同步更新 `03-api-reference.md`、`05-data-and-security.md`；新增页面时同步 `04-frontend-pages.md` 的“页面-路由-API 对照表”。
