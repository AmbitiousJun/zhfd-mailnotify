// 开启 http 接口服务

package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// init 在程序启动的时候初始化一个 http 服务，接收配置更新请求
func init() {
	startHttpServerAsync()
}

// ConfigRefreshParams 刷新配置请求参数
type ConfigRefreshParams struct {
	// 刷新配置后是否立即重新发送邮件
	// 1 启用 2 不启用
	ReSendMail int `form:"re-send-mail"`
	// 要刷新的令牌
	Auth string `form:"auth"`
}

// configRefreshHandler 刷新配置处理器
func configRefreshHandler(c *gin.Context) {
	var params ConfigRefreshParams
	if err := c.ShouldBind(&params); err != nil {
		c.String(http.StatusBadRequest, "请求参数错误")
		return
	}
	if cache == nil {
		c.String(http.StatusBadRequest, "定时器至少触发 1 次后才能刷新配置")
		return
	}
	cache.Zhfd.Authorization = params.Auth
	if params.ReSendMail != 1 {
		c.String(http.StatusOK, "配置更新成功")
		return
	}
	// 需要重新发送邮件，但是邮件定时器还没有初始化好
	if MailSchedulerCache == nil {
		c.String(http.StatusInternalServerError, "定时器未初始化完成")
		return
	}
	if err := MailSchedulerCache.Send(false); err != nil {
		c.String(http.StatusInternalServerError, "发送邮件异常: "+err.Error())
		return
	}
	c.String(http.StatusOK, "配置更新成功")
}

// startHttpServerAsync 异步开启一个 http 服务器
func startHttpServerAsync() {
	go func() {
		router := gin.Default()

		router.GET("/config-refresh", configRefreshHandler)

		if err := router.Run(":54321"); err != nil {
			log.Fatal("http 服务运行异常", err)
		}
	}()
}
