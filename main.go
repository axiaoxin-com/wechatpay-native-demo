package main

import (
	"log"
	"os"

	"wechatpay-native-demo/config"
	"wechatpay-native-demo/handler"
	"wechatpay-native-demo/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("未找到.env文件，使用系统环境变量")
	}

	// 加载配置
	cfg := config.LoadConfig()

	// 验证必要配置
	if cfg.AppID == "" || cfg.MchID == "" || cfg.APIv3Key == "" || cfg.PublicKeyID == "" {
		log.Fatal("缺少必要的微信支付配置，请检查环境变量")
	}

	// 创建内存订单存储
	orderStore := service.NewOrderStore()

	// 创建微信支付服务
	wechatPay, err := service.NewWechatPayService(cfg)
	if err != nil {
		log.Fatalf("初始化微信支付服务失败: %v", err)
	}

	// 创建处理器
	paymentHandler := handler.NewPaymentHandler(wechatPay, orderStore)
	notifyHandler, err := handler.NewNotifyHandler(cfg.APIv3Key, cfg.PublicKeyID, cfg.PublicKeyPath, orderStore)
	if err != nil {
		log.Fatalf("初始化通知处理器失败: %v", err)
	}

	// 创建Gin路由
	r := gin.Default()

	// 加载HTML模板
	r.LoadHTMLGlob("templates/*")

	// 静态文件服务
	r.Static("/static", "./static")

	// 页面路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.GET("/orders.html", func(c *gin.Context) {
		c.HTML(200, "orders.html", nil)
	})

	r.GET("/pay", func(c *gin.Context) {
		c.HTML(200, "pay.html", nil)
	})

	r.GET("/success", func(c *gin.Context) {
		c.HTML(200, "success.html", gin.H{
			"order_id": c.Query("order_id"),
		})
	})

	// API路由
	api := r.Group("/api")
	{
		// 订单管理
		api.POST("/order", paymentHandler.CreateOrder)                  // 创建订单
		api.GET("/orders", paymentHandler.ListOrders)                   // 订单列表
		api.GET("/order/:order_id", paymentHandler.QueryOrder)          // 查询订单
		api.POST("/order/:order_id/close", paymentHandler.CloseOrder)   // 关闭订单
		api.POST("/order/:order_id/refund", paymentHandler.RefundOrder) // 申请退款

		// 微信支付回调
		api.POST("/notify", notifyHandler.PaymentNotify)
	}

	// 启动服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("服务器启动在 http://localhost:%s", port)
	log.Printf("订单管理页面: http://localhost:%s/orders.html", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
