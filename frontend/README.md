# KamaChat Frontend

这是项目正式前端入口，不是测试页。

当前前端基于 Vue 3 + Element Plus，先完成后端 `setupPublicRoutes()` 对应的公开认证模块。

## 当前已接入

- `POST /auth/login`
- `POST /auth/register`
- `POST /auth/sms/login`
- `POST /sms/send`

`POST /auth/email/login` 暂时没有接，因为后端控制器还是空实现。

## 本地运行

```bash
cd frontend
npm install
npm run dev
```

默认前端开发地址：

```text
http://127.0.0.1:5173
```

## 代理说明

`vite.config.js` 已经把常用后端分组代理到：

```text
http://127.0.0.1:8080
```

这是按当前项目后端配置文件 [config.toml](D:/study_kamachat/configs/config.toml:2) 里的：

```toml
[mainConfig]
host = "127.0.0.1"
port = 8080
```

所以你现在看到的 `ECONNREFUSED 127.0.0.1:8000`，根因就是前端之前代理错端口了。

如果你以后要直接请求别的后端地址，可以设置：

```bash
VITE_API_BASE_URL=http://your-host:your-port
```

如果你想改开发代理目标，也可以设置：

```bash
VITE_PROXY_TARGET=http://your-host:your-port
VITE_PROXY_WS_TARGET=ws://your-host:your-port
```

## 主要文件

- `src/views/AuthView.vue`：用户登录、短信登录、注册页
- `src/views/HomeView.vue`：登录后的首页落点
- `src/api/http.js`：统一请求封装，适配后端 `JsonBack()` 返回结构
- `src/api/auth.js`：公开认证接口
- `src/router/index.js`：路由和登录态守卫
- `src/constants/ui-text.js`：中文产品文案和页面标题
