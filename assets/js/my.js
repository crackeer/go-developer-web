function getQuery(key, value) {
    let url = new URLSearchParams(window.location.search);
    return url.get(key) || value;
}

function getAllQuery() {
    let params = new URLSearchParams(window.location.search);
    let retData = {};
    for (let param of params) {
        retData[param[0]] = param[1];
    }
    return retData;
}

function pushStateWith(query) {
    let newURL = window.location.pathname + "?" + httpBuildQuery(query);
    window.history.pushState(null, "", newURL);
}

function nanoid(t) {
    return crypto
        .getRandomValues(new Uint8Array(t))
        .reduce(
            (t, e) =>
            (t +=
                (e &= 63) < 36
                    ? e.toString(36)
                    : e < 62
                        ? (e - 26).toString(36).toUpperCase()
                        : e > 62
                            ? "-"
                            : "_"),
            ""
        );
}

function windowReload() {
    window.location.reload();
}

function windowReloadQuery(query) {
    window.location.href =
        window.location.pathname + "?" + httpBuildQuery(query);
}

function windowRedirect(path, query) {
    window.location.href = path + "?" + httpBuildQuery(query);
}

function goPrevPage() {
    let query = getAllQuery();
    if (query["page"] == undefined) {
        query["page"] = 1;
    }
    if (parseInt(query["page"]) <= 1) {
        return;
    }
    goPage(parseInt(query["page"]) - 1);
}

function goPage(page) {
    let query = getAllQuery();
    if (query["page"] == undefined) {
        query["page"] = 1;
    }
    if (parseInt(query["page"]) == page) {
        return;
    }
    let path = window.location.pathname;
    query["page"] = page;
    windowRedirect(path, query);
}

function goNextPage(totalPage) {
    let query = getAllQuery();
    let currentPage = 1
    if (query["page"] != undefined) {
        currentPage = parseInt(query["page"])
    }
    if (totalPage == undefined) {
        goPage(currentPage + 1);
        return;
    }
    if (currentPage >= totalPage) {
        return;
    }
    goPage(currentPage + 1);
}

function httpBuildQuery(query) {
    let params = new URLSearchParams("");
    Object.keys(query).forEach((k) => {
        params.append(k, query[k]);
    });
    return params.toString();
}

function cloneObject(data) {
    let raws = JSON.stringify(data);
    return JSON.parse(raws);
}

function saveFile(data, name) {
    //Blob为js的一个对象，表示一个不可变的, 原始数据的类似文件对象，这是创建文件中不可缺少的！
    var urlObject = window.URL || window.webkitURL || window;
    var export_blob = new Blob([data]);
    var save_link = document.createElementNS(
        "http://www.w3.org/1999/xhtml",
        "a"
    );
    save_link.href = urlObject.createObjectURL(export_blob);
    save_link.download = name;
    save_link.click();
}

//js 读取文件
function readFiles(file) {
    var reader = new FileReader(); //new一个FileReader实例
    if (/text+/.test(file.type)) {
        //判断文件类型，是不是text类型
        reader.onload = function (result) {
            console.log(result);
        };
        reader.readAsText(file);
    } else if (/image+/.test(file.type)) {
        //判断文件是不是imgage类型
        reader.onload = function (result) {
            console.log(result);
        };
        reader.readAsDataURL(file);
    }
}

function initJSONEditor(target, value) {
    let jsonEditor = "jsonEditor-" + target;
    if (window[jsonEditor] == undefined) {
        window[jsonEditor] = new JSONEditor(document.getElementById(target), {
            mode: "code",
            search: true,
            indentation: 4,
        });
    }
    try {
        let jsonValue = JSON.parse(value);
        window[jsonEditor].set(jsonValue);
    } catch (e) { }
    return window[jsonEditor];
}



function getAxiosConfig(method, fullUrl, data, headers) {
    let urlObject = new URL(fullUrl);
    /*
    let headerKeys = Object.keys(headers)
    let proxyHeaders = {}
    for (var i in headerKeys) {
        proxyHeaders['X-Proxy-' + headerKeys[i]] = headers[headerKeys[i]]
    }*/
    headers["X-Proxy-Host"] = urlObject.origin;

    let config = {
        url: "/proxy" + urlObject.pathname,
        method: method,
        params: null,
        data: null,
        withCredentials: false,
        headers: headers,
    };
    if (
        method.toLocaleUpperCase() == "POST" ||
        method.toLocaleUpperCase() == "PUT" ||
        method.toLocaleUpperCase() == "DELETE"
    ) {
        config.data = data;
    }
    if (method.toLocaleUpperCase() == "GET") {
        config.params = data;
    }
    return config;
}

async function httpRequest(method, url, params, headers) {
    try {
        if (headers == null || headers == undefined) {
            headers = {};
        }
        let axiosConfig = getAxiosConfig(method, url, params, headers);
        let res = await axios(axiosConfig);
        return res.data;
    } catch (e) {
        alert(e.message);
        return null;
    }
}

async function httpPost(url, data, headers) {
    if (headers == null || headers == undefined) {
        headers = {};
    }
    headers["X-Proxy-Content-Type"] = "application/json";
    return httpRequest("POST", url, data, headers);
}

async function httpGet(url, data, headers) {
    return httpRequest("GET", url, data, headers);
}

function extractURLs(data) {
    if (data == null) return [];
    let retData = [];
    // 对象
    if (typeof data == "object" && data.length == undefined) {
        Object.keys(data).forEach((key) => {
            if (typeof data[key] == "string") {
                if (isURL(data[key])) {
                    retData.push(data[key]);
                } else {
                    try {
                        let tmp = JSON.parse(data[key]);
                        retData.push(...extractURLs(tmp));
                    } catch (e) { }
                }
            } else if (typeof data[key] == "object") {
                retData.push(...extractURLs(data[key]));
            }
        });
    }
    if (typeof data == "object" && data.length != undefined) {
        for (let i in data) {
            retData.push(...extractURLs(data[i]));
        }
    }
    return retData;
}

function isURL(str) {
    return str.indexOf("http://") == 0 || str.indexOf("https://") == 0;
}

function groupURLByHost(urls) {
    let retData = {};
    for (let i = 0; i < urls.length; i++) {
        let data = new URL(urls[i]);
        let key = data.protocol + "//" + data.host;
        if (retData[key] == undefined) {
            retData[key] = [];
        }
        retData[key].push(urls[i]);
    }
    return retData;
}

function base64Encode(str) {
    console.log("base64Encode", str);
    return btoa(encodeURIComponent(str));
}

function base64Decode(str) {
    return decodeURIComponent(atob(str));
}

function bootboxLoading(message) {
    bootbox.hideAll();
    message = message || "loading";
    bootbox.dialog({
        message:
            '<div class="text-center"><i class="fas fa-spinfa-spinner"></i>' +
            message +
            "</div>",
        closeButton: false,
    });
}

function sleep(time) {
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve();
        }, time);
    });
}

function formatTimestamp(ts) {
    return dayjs.unix(ts).format("YYYY-MM-DD HH:mm:ss");
}







async function getAccessToken(host, ak, sk) {
    let result = await httpPost(host + "/auth/access_token", {
        app_key: ak,
        app_secret: sk,
    });
    if (result.code != 0) {
        return "";
    }
    let output = JSON.parse(result.data);
    return output.data.access_token;
}

function jsonDecode(data) {
    try {
        return JSON.parse(data);
    } catch (e) {
        console.log("jsonDecode error", e);
        return null;
    }
}

function showJSONString(item) {
    try {
        let input = JSON.parse(item);
        bootbox.alert(
            '<textarea rows="10"  class="form-control" >' +
            JSON.stringify(input, null, 4) +
            "</textarea>"
        );
    } catch (e) {
        bootbox.alert(
            '<textarea rows="10"  class="form-control" >' + item + "</textarea>"
        );
    }
}

function getNowTime() {
    return dayjs().format("YYYY-MM-DD HH:mm:ss");
}

function getNowTimeKey() {
    return dayjs().format("YYYY-MM-DD-HH-mm-ss");
}

function openPort(port) {
    console.log("//" + window.location.hostname + ":" + port);
    window.open("//" + window.location.hostname + ":" + port, "_blank");
}

function openHttpsPort(port) {
    window.open("https://" + window.location.hostname + ":" + port, "_blank");
}

function windowOpen(path, query) {
    window.open(path + "?" + httpBuildQuery(query), "_blank");
}




function formatCommandOutput(data, fromIndex, headers) {
    let parts = data.split("\n");

    if (parts.length < 1) {
        return [];
    }
    let retData = [];
    for (let i = fromIndex; i < parts.length; i++) {
        if (parts[i].length > 0) {
            let values = parts[i].split(/\s+/);
            let tmp = maapingRowData(values, headers);
            retData.push(tmp);
        }
    }
    return retData;
}

function maapingRowData(values, headers) {
    let tmp = {};

    for (let j = 0; j < values.length; j++) {
        if (values[j].length < 1) {
        } else if (j < headers.length) {
            tmp[headers[j]] =
                tmp[headers[j]] == undefined
                    ? values[j]
                    : tmp[headers[j]] + " " + values[j];
        } else {
            tmp[j + ""] = values[j];
        }
    }
    return tmp;
}



async function doHttpRequest(method, url, query, body, headers, timeout) {
    if (timeout == null || timeout == undefined) {
        timeout = 30;
    }
    let result = await axios.post("/api/http/handle", {
        url: url,
        method: method,
        query: query,
        body: body,
        headers: headers,
        timeout: timeout
    });
    return result.data;
}


function generateUppercasePassword(totalLength) {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
    let password = "";
    for (let i = 0; i < totalLength; i++) {
        const randomIndex = Math.floor(Math.random() * chars.length);
        password += chars[randomIndex];
    }
    return password;
}


function exportJsonToCsv(jsonData, filename) {
    if (!Array.isArray(jsonData) || jsonData.length === 0) {
        console.warn("数据为空或不是数组");
        return;
    }

    // 获取所有字段（以第一个对象的键为准）
    const headers = Object.keys(jsonData[0]);

    // 构建 CSV 内容
    const csvRows = [];

    // 添加表头
    csvRows.push(
        headers.map((header) => `"${header.replace(/"/g, '""')}"`).join(",")
    );

    // 添加数据行
    for (const row of jsonData) {
        const values = headers.map((header) => {
            const val =
                row[header] === null || row[header] === undefined
                    ? ""
                    : row[header].toString();
            // 处理包含逗号、引号或换行符的值（用双引号包裹，并转义内部引号）
            return `"${val.replace(/"/g, '""')}"`;
        });
        csvRows.push(values.join(","));
    }

    // 合并为完整 CSV 字符串
    const csvString = csvRows.join("\n");
    const bom = '\uFEFF';
    // 创建 Blob 对象
    const blob = new Blob([bom + csvString], { type: "text/csv;charset=utf-8;" });

    // 创建下载链接
    const link = document.createElement("a");
    const url = URL.createObjectURL(blob);
    link.setAttribute("href", url);
    link.setAttribute("download", filename);
    link.style.visibility = "hidden";

    // 触发下载
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url); // 释放内存
}







function markNavActive() {
    try {
        // 给顶部导航栏匹配到当前路径的菜单项加 active（Bootstrap3：active 通常加在 li 上）
        const currentPathRaw = window.location && window.location.pathname ? window.location.pathname : ''
        const normalizePath = (p) => {
            let s = String(p || '')
            if (s.length > 1 && s.endsWith('/')) {
                s = s.slice(0, -1)
            }
            return s
        }
        const currentPath = normalizePath(currentPathRaw)

        $('.navbar-nav a[href]').each(function () {
            const href = $(this).attr('href')
            if (!href) return
            if (href.indexOf('javascript:') === 0) return
            if (href === '#') return

            let linkPath = ''
            try {
                // 菜单大多是绝对路径 '/xxx'，这里也兼容相对路径
                linkPath = href.startsWith('/') ? href : new URL(href, window.location.origin).pathname
            } catch (e) {
                linkPath = ''
            }
            linkPath = normalizePath(linkPath)
            if (!linkPath) return

            // 前缀匹配但带边界：/a 匹配 /a 或 /a/xxx，不匹配 /a2
            let matched = false
            if (linkPath === '/') {
                matched = currentPath === '/'
            } else {
                matched = currentPath === linkPath || currentPath.startsWith(linkPath + '/')
            }

            if (matched) {
                $(this).closest('li').addClass('active')
            }
        })
    } catch (e) { }
}

function showSpinner() {
    if (window.SpinnerId == null || window.SpinnerId == undefined) {
        console.error('SpinnerId is not set');
        return;
    }
    if (window.SpinnerId == "<none>") {
        console.error('SpinnerId is <none>');
        return;
    }
    var target = document.getElementById(window.SpinnerId);
    if (target == null || target == undefined) {
        console.error('Spinner target not found: ' + window.SpinnerId);
        return;
    }
    window.spinner = new window.Spinner().spin(target);
}

function hideSpinner() {
    if (window.spinner == null || window.spinner == undefined) {
        console.error('Spinner is not set');
        return;
    }
    window.spinner.stop();
    window.spinner = null;
}


