# 微信支付 Native 支付完整接入示例（Go + HTML）

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![WeChat Pay](https://img.shields.io/badge/WeChat%20Pay-APIv3-brightgreen.svg)](https://pay.weixin.qq.com/doc/v3/merchant/4012791874)
[![人言兑x阿小信](https://img.shields.io/badge/人言兑-阿小信-brightgreen.svg)](https://blog.axiaoxin.com)

> 从零开始的微信支付 Native 支付接入教程，包含完整的订单管理、支付、退款、回调处理等功能。适合独立开发者快速上手微信支付开发。
>
> [微信支付接入教程（独立开发者实战版）：Native 支付接入指南](https://blog.axiaoxin.com/post/wechat-pay-native-guide/)

## 项目结构

```
./wechatpay-native-demo
├── README.md
├── cert
├── config
│   └── config.go
├── go.mod
├── go.sum
├── handler
│   ├── notify.go
│   └── payment.go
├── main.go
├── service
│   ├── order_store.go
│   └── wechatpay.go
└── templates
    ├── index.html
    ├── orders.html
    ├── pay.html
    └── success.html
```

## 功能特性

- [x] **Native 支付** - PC 网站扫码支付，微信扫一扫完成付款
- [x] **订单管理** - 创建订单、查询订单、关闭订单
- [x] **退款功能** - 全额/部分退款，退款状态查询
- [x] **回调处理** - 支付回调、退款回调，自动更新订单状态
- [x] **公钥模式** - 支持微信支付公钥模式（2024年后新商户）
- [x] **内存存储** - 无需数据库，开箱即用
- [x] **响应式页面** - 支持 PC 和移动端查看

![主页](https://inews.gtimg.com/om_bt/OBujiHcDdyFP--Y2WOMU4n9rovwqt4BMShiLPH0nG0ZEkAA)

![扫码付款页](https://inews.gtimg.com/om_bt/OytUKN7e78NSCi-XMr8H1-X3WMtZUPizD3lBDVMzyBGH4AA)

![支付成功](https://inews.gtimg.com/om_bt/OgNOTB7sHHLACtdcoqL78u_iY7DJiKHPS86dp8s68fiRkAA)

![订单状态管理](https://inews.gtimg.com/om_bt/O0NB8cBdlegX9NRXTrRICoTR1cq87-6rquhklnCE0jWiMAA)

## 快速开始

### 1. 环境要求

- Go 1.21+
- 微信商户号（已开通 Native 支付）
- 商户 API 证书 + 微信支付公钥

### 2. 获取商户配置

登录 [微信支付商户平台](https://pay.weixin.qq.com)：

| 配置项          | 获取路径                                         | 说明                   |
| :-------------- | :----------------------------------------------- | :--------------------- |
| 商户号 (MCHID)  | 账户中心 → 商户信息 → 商户号                     | 10位数字               |
| APIv3 密钥      | 账户中心 → 安全中心 → API 安全 → 设置 APIv3 密钥 | 32位随机字符串         |
| 商户证书序列号  | 账户中心 → 安全中心 → API 安全 → 查看证书        | 32位十六进制           |
| 商户证书私钥    | 账户中心 → 安全中心 → API 安全 → 下载证书        | `apiclient_key.pem`    |
| 微信支付公钥 ID | 账户中心 → 安全中心 → 微信支付公钥 → 查看公钥 ID | 格式：`PUB_KEY_ID_xxx` |
| 微信支付公钥    | 账户中心 → 安全中心 → 微信支付公钥 → 下载公钥    | `pub_key.pem`          |

**证书 vs 公钥的区别：**

- **商户证书（私钥）**：用于对请求签名，证明你是商户
- **微信支付公钥**：用于验证微信响应的签名，证明是微信官方返回

### 3. 下载并运行

```bash
# 克隆项目
git clone https://github.com/axiaoxin-com/wechatpay-native-demo.git
cd wechatpay-native-demo

# 创建证书目录
mkdir -p cert

# 放置证书文件
cp /path/to/apiclient_key.pem cert/    # 商户私钥（签名用）
cp /path/to/pub_key.pem cert/          # 微信公钥（验签用）

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件，填入你的商户配置

# 运行
go mod tidy
go run .

# 访问 http://localhost:8080
```

### 4. 配置说明

编辑 `.env` 文件：

```bash
# 微信支付配置（必填）
WXPAY_APPID=wx1234567890abcdef          # 公众号/小程序/移动应用 AppID
WXPAY_MCHID=1234567890                  # 商户号
WXPAY_APIV3_KEY=YourAPIv3KeyHere        # APIv3 密钥（32位）

# 商户证书（用于请求签名）
WXPAY_CERT_SERIAL_NO=1234567890ABCDEF   # 商户证书序列号
WXPAY_PRIVATE_KEY_PATH=./cert/apiclient_key.pem  # 商户私钥路径

# 微信支付公钥（用于验证响应签名，2024年后新商户）
WXPAY_PUBLIC_KEY_ID=PUB_KEY_ID_xxx      # 微信支付公钥 ID
WXPAY_PUBLIC_KEY_PATH=./cert/pub_key.pem # 微信支付公钥路径

# 支付回调地址（必须是外网可访问的 HTTPS 地址）
WXPAY_NOTIFY_URL=https://yourdomain.com/api/notify

# 服务器配置
PORT=8080
```

## API 接口

| 方法 | 路径                    | 说明                      |
| :--- | :---------------------- | :------------------------ |
| GET  | `/`                     | 商品首页                  |
| GET  | `/orders.html`          | 订单管理页面              |
| GET  | `/pay?order_id=xxx`     | 支付页面（展示二维码）    |
| POST | `/api/order`            | 创建订单                  |
| GET  | `/api/orders`           | 订单列表                  |
| GET  | `/api/order/:id`        | 查询订单                  |
| POST | `/api/order/:id/close`  | 关闭未支付订单            |
| POST | `/api/order/:id/refund` | 申请退款                  |
| POST | `/api/notify`           | 微信支付回调（支付/退款） |

## 核心流程

### 支付流程

```
1. 创建订单  →  2. 生成二维码  →  3. 用户扫码支付  →  4. 微信回调通知  →  5. 更新订单状态
```

### 退款流程

```
1. 申请退款  →  2. 微信处理  →  3. 退款回调通知  →  4. 更新订单状态为已退款
```

## 常见问题

**Q: 为什么需要同时配置商户证书和微信支付公钥？**

商户证书（私钥）用于**对请求签名**，微信支付公钥用于**验证响应签名**。两者作用不同：

- 你发送请求 → 用商户私钥签名 → 微信用商户公钥验签
- 微信发送回调 → 用微信私钥签名 → 你用微信公钥验签

**Q: 为什么收不到回调通知？**

- 检查 `WXPAY_NOTIFY_URL` 是否为外网可访问的 HTTPS 地址
- 检查服务器防火墙是否放行微信支付回调 IP
- 查看日志确认是否收到请求
- 确保申请退款时传入了 `notify_url` 参数

**Q: 生产环境需要注意什么？**

- 必须使用 HTTPS
- 替换内存存储为数据库（MySQL/PostgreSQL 等）
- 添加请求限流、防重放攻击等安全措施
- 妥善保管证书文件，不要提交到代码仓库

## 技术栈

- **后端**: Go + Gin + [wechatpay-go](https://github.com/wechatpay-apiv3/wechatpay-go)
- **前端**: HTML5 + Vanilla JS + [QRCode.js](https://github.com/davidshimjs/qrcodejs)
- **支付**: 微信支付 Native 支付 APIv3

## 致谢

本项目基于微信支付官方文档和 [wechatpay-go](https://github.com/wechatpay-apiv3/wechatpay-go) SDK 开发，仅供学习交流使用。

如果这个项目对你有帮助，欢迎 Star ⭐
