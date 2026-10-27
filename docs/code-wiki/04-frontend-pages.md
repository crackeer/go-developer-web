# 04 前端与页面

项目没有独立前端工程（无 npm / 构建步骤），所有页面都是**服务端模板 + 浏览器端 Vue 3 运行时**：Go 用 `unrolled/render` 渲染 `pages/*.tmpl`，每个模板内嵌一段 `<script>` 定义 `window.VueObject`，由 `layout.tmpl` 统一挂载到 `#app`。静态资源通过 `go:embed` 打进二进制。

## 1. 模板渲染机制

### 1.1 render 客户端

`main.go` 中的 [newRenderClient](file:///workspace/main.go#L35-L53) 初始化渲染器：

```go
object := rollRender.New(rollRender.Options{
    Layout:     "layout",
    Directory:  "pages",
    Extensions: []string{".tmpl"},
    Delims:     rollRender.Delims{Left: "{[{", Right: "}]}"},
    IsDevelopment:               true,
    RenderPartialsWithoutPrefix: true,
    FileSystem:                  fileSystem,
})
```

| 配置 | 值 | 说明 |
| --- | --- | --- |
| `Layout` | `layout` | 对应 `pages/layout.tmpl` |
| `Directory` | `pages` | 页面名以目录相对路径表示，如 `mysql/index`、`tool/json` |
| `Extensions` | `.tmpl` | 渲染时传页面名不带扩展名 |
| `Delims` | `{[{` / `}]}` | 避开 Vue 的 `{{ }}` 插值语法，两者可共存于同一文件 |
| `FileSystem` | `EmbedFileSystem` / `LocalFileSystem` | `ENV=develop` 时读磁盘 `./pages`，否则读 `go:embed` 的 `tmplPage` |
| `IsDevelopment` | 恒为 `true` | 仅影响模板缓存策略（每次重新解析），与 `ENV` 无关 |

渲染调用统一走 `renderClient.HTML(ctx.Writer, http.StatusOK, page, binding)`，**错误页也返回 HTTP 200**（业务错误靠页面内容表达）。

### 1.2 pageBinding：模板变量

[pageBinding](file:///workspace/main.go#L216-L241) 把 `*api.PageData` 转成模板 map：

| key | 来源 | 说明 |
| --- | --- | --- |
| `title` | `PageData.Title` | `<title>` |
| `description` | `PageData.Description` | `<meta name="description">` |
| `breadcrumbs` | `PageData.Breadcrumbs` | 自动在最前面补 `{首页, "/"}`；`HideBreadcrumbs` 为真时**不注入该 key** |
| `extension` | `PageData.Extension` | 注入前端 `window.PageData`，缺省 `{}` |
| `error` | `err.Error()` | 仅在错误页出现 |

`api.PageData` / `api.Breadcrumb` / `api.DataFunc` 定义见 [api/page.go](file:///workspace/api/page.go#L5-L21)。

### 1.3 layout.tmpl 与 Vue 挂载

[layout.tmpl](file:///workspace/pages/layout.tmpl) 是整个站点的唯一骨架：

1. `<head>` 统一引入 jQuery、Vue 3 global build、axios、dayjs、JSONEditor、Bootstrap 3.4、bootbox、marked、editormd、filesize、qrcode、json-bigint、`my.js`。
2. 内联脚本把服务端数据写入全局：

```js
window.PageData = {[{ .extension }]} || {}
window.VueObject = null
window.VueMount = "#app"
window.Vm = null
```

`html/template` 对 `<script>` 内的 JSON 会做 JS 转义，因此 `extension` 中的字符串是安全的。

3. 页面脚本（写在模板尾部）负责给 `window.VueObject` 赋值；`DOMContentLoaded` 后**延迟 500ms** 执行 `Vue.createApp(...).mount("#app")`。延迟是为了等待页面依赖的额外 `<script src>`（如 `json-to-go.js`）加载完成。
4. `body` 中依次输出面包屑、`<div id="app"></div>`、`{[{ yield }]}`。

页面模板的标准结构：

```html
<template id="template">
    <!-- Vue 模板内容 -->
</template>
<script>
    window.VueObject = {
        data() { return {} },
        computed: {},
        async mounted() {},
        methods: {},
        template: '#template',
    }
</script>
```

注意：`window.VueObject` 在每个模板中**独立定义**，`index.tmpl` 等页面不会互相复用。

### 1.4 页面处理器

| 处理器 | 用途 | 错误处理 |
| --- | --- | --- |
| [createPageHandlerV2](file:///workspace/main.go#L191-L195) | 静态页面数据，`dataFunc` 固定为 `nil` | 无 |
| [createPageHandlerV1](file:///workspace/main.go#L198-L213) | 带 `api.DataFunc` 的动态页面 | `dataFunc` 返回 error 时渲染 `error` 模板，HTTP 仍为 200 |

`error` 模板只输出一个 Bootstrap `alert-danger`，并把 `window.VueObject` 覆盖为空组件，避免错误信息被 Vue 模板吞掉。

## 2. 静态资源

| 位置 | 内容 |
| --- | --- |
| `assets/js` | 第三方库 + `my.js` 全局工具 + `json-to-go.js` / `nanoid.js` / `util.js` |
| `assets/js/codemirror` | CodeMirror 及 YAML 模式（YAML 编辑场景用） |
| `assets/css` | Bootstrap 3.4、JSONEditor、editormd 预览、markdown、spin、`my.css` |
| `assets/images` | favicon 等 |

生产模式由 [main.go](file:///workspace/main.go#L64-L72) 用 `fs.Sub + StaticFS` 暴露 `/assets`；开发模式直接 `router.Static("/assets", "./assets")`。**没有版本号/指纹**，浏览器缓存需靠手动刷新。

## 3. 已注册页面与可达性

`main.go` 只注册了 6 个页面路由，其余模板均无路由：

| 路径 | 模板 | 处理器 | 页面数据 | 状态 |
| --- | --- | --- | --- | --- |
| `/`（及所有未匹配路径） | `index` | `createPageHandlerV2` | 标题“首页”，描述“页面不存在” | 可用 |
| `/mysql/:id` | `mysql/index` | `createPageHandlerV1` + `mysqlHandler.PageData` | `config_id` / `config_name` / `database` | 可用（配置不存在时渲染错误页） |
| `/tool/json` | `tool/json` | `createPageHandlerV2` | 标题“JSON 工具” | 可用（纯前端） |
| `/ftp/:id` | `service/ftp` | `createPageHandlerV1` | `nil` | **500**：模板不存在 |
| `/ssh/:id` | `service/ssh` | `createPageHandlerV1` | `nil` | **500**：模板不存在 |
| `/s3/:id` | `service/s3` | `createPageHandlerV1` | `nil` | **500**：模板不存在 |
| `/redis/:id` | `service/redis` | `createPageHandlerV1` | `nil` | **500**：模板不存在 |

`NoRoute` 覆盖了其余所有路径，因此 `/file/manage`、`/k3s/index`、`/system/info`、`/setting`、`/text/*`、`/user/login` 等历史页面**当前不可访问**——访问时拿到的是首页模板。

`/ftp/:id` 等 4 个路由指向的 `pages/service/` 目录在仓库中不存在，属于遗留路由。

## 4. 模板清单

| 模板文件 | 标题（同目录 `.json`） | 用途 | 有无路由 |
| --- | --- | --- | --- |
| `pages/layout.tmpl` | - | 全站骨架 | 布局 |
| `pages/index.tmpl` | - | 资源管理器：MySQL/FTP/OSS/SSH/Redis 配置 CRUD 与连通性测试 | `/` 及 NoRoute |
| `pages/error.tmpl` | - | 错误提示页 | DataFunc 出错时 |
| `pages/mysql/index.tmpl` | 无 json | 单库数据浏览、表结构、SQL 执行、全局变量 | `/mysql/:id` |
| `pages/tool/json.tmpl` | JSON编辑/验证 | JSON 编辑、格式化、转 Go Struct、递归解析 | `/tool/json` |
| `pages/ftp/index.tmpl` | FTP管理 | FTP 浏览/上传/下载/重命名/配置 | 无（`/ftp/:id` 模板缺失） |
| `pages/ssh/file.tmpl` | SSH文件管理 | 远程文件管理 | 无 |
| `pages/redis/index.tmpl` | Redis查询 | Key 扫描/取值/删除 | 无 |
| `pages/s3browse/index.tmpl` | S3文件查看 | 依赖 `mc` 命令的 S3 文件浏览 | 无 |
| `pages/file/manage.tmpl` | 文件上传 / 下载 | 本机文件管理（上传/下载/压缩/解压） | 无 |
| `pages/k3s/index.tmpl` | k3s容器管理 | K3s 资源浏览、YAML 编辑、日志、扩缩容 | 无 |
| `pages/k3s/image.tmpl` | k3s容器管理 | 节点镜像导入导出（依赖 `/api/execute/*`） | 无 |
| `pages/k3s/image2.tmpl` | 镜像管理 | 镜像列表与分发（依赖 `/api/k3s/image/*`、`/api/ssh/*`） | 无 |
| `pages/system/info.tmpl` | System信息 | 主机/系统信息 | 无 |
| `pages/system/state.tmpl` | 服务状态 | 加密狗、MySQL/Redis、业务接口探活 | 无 |
| `pages/setting/index.tmpl` | 设置&配置 | 应用配置表编辑（表名硬编码） | 无 |
| `pages/user/login.tmpl` | 登录 | 登录表单（后端无对应接口） | 无 |
| `pages/http/request.tmpl` | HTTP 请求 | HTTP 调试工具 | 无 |
| `pages/text/*.tmpl` | Base64 / 文本编辑 / HashIds / 二维码 / 运行JS代码 | 文本工具集 | 无 |

### 4.1 `.json` 元数据未被使用

每个模板同目录下的 `*.json`（`title`、`page_data`、`no_navigation` 等）看起来来自另一套页面框架。当前 Go 代码**从不读取这些文件**，`title` 完全由 `api.PageData` 提供；`page_data` 也不参与渲染（例如 `s3browse/index.json` 的 `alias`、`public_bucket` 只在模板里通过 `window.PageData` 读取，但该页无路由、也无处理器注入）。可视为历史残留。

`pages/k3s/*.yml`、`pages/redis/redis_key_prefixes.json` 是页面用到的静态数据/清单文件，同样没有 Go 读取逻辑，供人工或外部脚本使用。

## 5. 页面 → API 对照

### 5.1 `index.tmpl`（首页资源管理器）

| 动作 | 接口 | 后端是否注册 |
| --- | --- | --- |
| 加载类型元数据 | `GET /api/config/types` | 是 |
| 加载全部配置 | `GET /api/config/list` | 是 |
| 新建 | `POST /api/config/create` | 是 |
| 编辑 | `POST /api/config/update/:id` | 是 |
| 删除 | `POST /api/config/delete/:id` | 是 |
| 连通性测试 | `POST /api/config/test/:id` | 是 |

前端一次拉全量配置，按 `type` 在前端过滤并计算各 Tab 计数；`?type=mysql` 可指定初始 Tab（`getQuery('type')`）。MySQL 行名称链接到 `/mysql/:id`，其余类型无跳转。

### 5.2 `mysql/index.tmpl`

所有请求经 `api(path)` 拼接为 `/api/mysql/${configId}/${path}`，`configId` 来自 `window.PageData.config_id`。

| 动作 | 接口 |
| --- | --- |
| 表列表 | `GET .../table` |
| 表结构 | `GET .../desc/:table` |
| 分页查询 | `POST .../query/:table` |
| 总数 | `POST .../count/:table` |
| 新增行 | `POST .../create/:table` |
| 更新行 | `POST .../update/:table/:row_id` |
| 删除行 | `POST .../delete/:table/:row_id` |
| 执行 SQL | `POST .../exec_sql` |
| 建表语句 | `GET .../show_create_table/:table` |
| 索引 | `GET .../show_index/:table` |
| 全局变量 | `GET .../global_variables` |

另有 `GET /mysql_data/struct/:database.json`（表结构快照），该静态路径未注册，请求会落到 `NoRoute` 返回首页 HTML。

前端未使用后端的 `distinct` 接口。

### 5.3 其余页面（均为历史页面，接口未注册或缺少 `:id`）

| 页面 | 前端调用 | 后端现状 |
| --- | --- | --- |
| `ftp/index.tmpl` | `/api/ftp/list`、`download`、`upload`、`delete`、`rename`、`mkdir`、`GET/POST /api/ftp/config` | 实际路由是 `/api/ftp/:id/...`，且无 `/config` 接口 |
| `redis/index.tmpl` | `/api/redis/scan`、`get`、`random`、`DELETE delete` | 实际路由是 `/api/redis/:id/...` |
| `ssh/file.tmpl` | `/api/ssh/file/nodes`、`list`、`download`、`mkdir`、`rename`、`delete`、`upload` | `main.go` 中整组路由被注释 |
| `k3s/index.tmpl` | `/api/k3s/node/list`、`describe/:resource`、`list/:resource`、`get/:resource`、`apply`、`log/:pod`、`delete/:resource`、`rollout/restart/:resource`、`scale/:resource`、`patch/:resource`、`node/label/add|remove`、`list/namespace` | 实际路由均带 `/api/k3s/:id` 前缀；`get/:resource`、`list/namespace` 未注册 |
| `k3s/image.tmpl` | `/api/execute/crictl`、`/api/execute/k3s`、`/api/file/*` | `/api/execute/*` 未注册 |
| `k3s/image2.tmpl` | `/api/k3s/image/list`、`/api/ssh/exec|upload|download`、`/api/mysql/query/vrapi/strategy` | 均未注册（SSH 路由被注释） |
| `s3browse/index.tmpl` | `/api/execute/mc`、`/api/execute/tar`、`/api/file/*` | `/api/execute/*` 未注册 |
| `file/manage.tmpl` | `/api/file/list|download|upload|zip|unzip|delete|mkdirall|rename`、`/api/execute/cp|k3s`、`/api/project/txtzip/convert` | `/api/file/*` 已注册；其余未注册 |
| `setting/index.tmpl` | `/api/mysql/query|create|delete/...` | 缺 `:id`，且表名 `saas_config.app`、`vr_open_platform.app` 为历史业务表 |
| `system/info.tmpl` | `/api/system/info` | 未注册 |
| `system/state.tmpl` | `/api/system/store_state` | 未注册 |
| `user/login.tmpl` | `/api/user/login` | 未注册（鉴权由 BasicAuth 承担） |
| `text/hashids.tmpl` | `/api/hashids/encode|decode` | 未注册 |
| `http/request.tmpl` | `doHttpRequest()` → `/api/http/handle` | 未注册 |
| `my.js` 的 `httpRequest` | `/proxy...` | 未注册 |

结论：除首页与 MySQL 页外，前端页面与当前后端契约**整体脱节**，属于上一代工程迁移后的残留。

## 6. 全局 JS 工具函数（`assets/js/my.js`）

| 函数 | 用途 |
| --- | --- |
`getQuery` / `getAllQuery` | 读取 URL query |
`pushStateWith` / `windowReloadQuery` / `windowRedirect` | 修改地址栏并保持 query |
`goPrevPage` / `goPage` / `goNextPage` | 基于 `?page=` 的分页跳转 |
`httpBuildQuery` | 对象转 query string |
`cloneObject` | JSON 深拷贝 |
`saveFile` / `readFiles` | 浏览器本地文件读写 |
`initJSONEditor` | 初始化/复用 JSONEditor 实例（`mode: code`） |
`getAxiosConfig` / `httpRequest` / `httpPost` / `httpGet` | 把任意 URL 代理到 `/proxy`，附带 `X-Proxy-Host`/`X-Proxy-Content-Type` 头（后端未实现） |
`extractURLs` / `isURL` / `groupURLByHost` | 从任意 JSON 中递归提取 URL |
`base64Encode` / `base64Decode` | UTF-8 安全的 Base64 |
`bootboxLoading` / `sleep` | 加载遮罩与延时 |
`formatTimestamp` / `getNowTime` / `getNowTimeKey` | dayjs 时间格式化 |
`jsonDecode` / `showJSONString` | JSON 解析与弹窗展示 |
`formatCommandOutput` / `maapingRowData` | 把命令行表格文本按表头映射为对象 |
`exportJsonToCsv` | 前端导出 CSV（带 BOM） |
`markNavActive` | 按 pathname 给导航项加 `active`（当前 layout 无导航栏，实际未生效） |
`showSpinner` / `hideSpinner` | 基于 `spin.js` 的加载动画 |

`assets/js/util.js`、`nanoid.js`、`json-to-go.js` 为独立工具脚本，仅 `tool/json` 页显式引用了 `json-to-go.js`。

## 7. 前端开发约定

新增页面时按以下步骤：

1. 建 `pages/<name>/index.tmpl`，遵循 `<template id="template">` + `window.VueObject` 结构。
2. 在 `main.go` 注册路由，选择 `createPageHandlerV2`（静态数据）或 `createPageHandlerV1`（需要 `DataFunc`）。
3. 页面需要服务端数据时，在 `DataFunc` 中返回 `api.PageData{Extension: {...}}`，前端用 `window.PageData.xxx` 读取。
4. 所有接口调用加 `/api/` 前缀；带资源的接口必须拼上 `:id`（如 `/api/redis/${id}/scan`）。
5. 不要使用 `{{ }}`——模板插值统一用 `{[{ }]}`，Vue 表达式保持 `{{ }}`。

相关阅读：[03-api-reference.md](03-api-reference.md)、[05-data-and-security.md](05-data-and-security.md)。
