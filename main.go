package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/AmbitiousJun/go-mailscheduler"
)

var MailSchedulerCache *mailscheduler.Scheduler

func main() {
	log.Println("正在读取配置文件...")
	cfg, err := ReadConfig(false)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("当前配置：\n", cfg)
	log.Println("配置文件读取完成！")

	log.Println("正在加载邮件模板...")
	normalTemplate, err := os.ReadFile("./template/normal.html")
	if err != nil {
		log.Fatal(err, "无法读取 html 模板")
	}
	fallbackTemplate, err := os.ReadFile("./template/fallback.html")
	if err != nil {
		log.Fatal(err, "无法读取 html 模板")
	}
	normalBuilder, fallbackBuilder := NewHtmlBuilder(normalTemplate), NewHtmlBuilder(fallbackTemplate)
	log.Println("邮件模板加载完成！")

	log.Println("正在初始化定时器...")
	sOpt := mailscheduler.SmtpOptions{
		Host:       cfg.Smtp.Host,
		Port:       cfg.Smtp.Port,
		Username:   cfg.Smtp.Username,
		Credential: cfg.Smtp.Credential,
	}

	var fallbackErr error
	bodyBuildFunc, err := BuildBody(&fallbackErr, normalBuilder)
	if err != nil {
		log.Fatal(err, "生成邮件初始化函数失败")
	}

	fallbackBodyBuildFunc := BuildFallbackBody(&fallbackErr, fallbackBuilder)

	mOpt := mailscheduler.MailOptions{
		From:                  cfg.Mail.From,
		To:                    cfg.Mail.To,
		Subject:               cfg.Mail.Subject,
		BodyType:              mailscheduler.MailBodyHtml,
		BodyBuildFunc:         bodyBuildFunc,
		FallbackBodyBuildFunc: fallbackBodyBuildFunc,
	}

	ms, err := mailscheduler.New(cfg.Cron, &mOpt, &sOpt)
	if err != nil {
		log.Fatal(err, "初始化定时器失败")
	}
	log.Println("定时器初始化完成！")

	ms.Start()
	log.Println("定时器启动成功 (*^▽^*)")
	MailSchedulerCache = ms

	// 阻塞主协程
	select {}
}

// BuildFallbackBody 返回一个生成失败邮件内容的函数
func BuildFallbackBody(fallbackErr *error, builder *HtmlBuilder) mailscheduler.MailFallbackBodyBuildFunc {
	return func() string {
		return builder.
			Set("error", (*fallbackErr).Error()).
			Build()
	}
}

// BuildBody 返回一个生成邮件内容的函数
// 请求数据时出现异常的话，会将异常设置到 fallbackErr 中
func BuildBody(fallbackErr *error, builder *HtmlBuilder) (mailscheduler.MailBodyBuildFunc, error) {
	cfg, err := ReadConfig(true)
	if err != nil {
		return nil, errors.Join(err, errors.New("读取缓存配置异常"))
	}

	// 初始化 http 请求客户端，连接超时 2 分钟，请求超时 5 分钟
	client := http.Client{
		Transport: &http.Transport{
			Dial:                  (&net.Dialer{Timeout: time.Minute * 2}).Dial,
			ResponseHeaderTimeout: time.Minute * 5,
		},
	}

	// 构造请求
	request, err := http.NewRequest(http.MethodGet, cfg.Zhfd.RequestUrl, nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("初始化请求失败"))
	}

	// 设置请求头
	request.Header.Set("Referer", cfg.Zhfd.Referer)
	request.Header.Set("Host", cfg.Zhfd.Host)
	request.Header.Set("Authorization", cfg.Zhfd.Authorization)

	return func() (res string, buildErr error) {
		defer func() {
			// 如果构造失败，就传递错误信息出去
			if buildErr != nil {
				*fallbackErr = buildErr
			}
		}()

		// 更新 token 为缓存中的最新值
		request.Header.Set("Authorization", cfg.Zhfd.Authorization)

		// 执行请求
		response, err := client.Do(request)
		if err != nil {
			return "", errors.Join(err, errors.New("请求失败"))
		}

		// 如果是 401 异常，需要发邮件通知用户刷新 token 令牌
		if response.StatusCode == http.StatusUnauthorized {
			refreshUrl := fmt.Sprintf("http://%s:54321/config-refresh?re-send-mail=1&auth=", cfg.ServerIp)
			return "", fmt.Errorf("请求失败, token 过期, 请在以下链接上拼接最新 token 后访问该链接刷新 token: \n%s", refreshUrl)
		}

		// 分析请求是否成功
		if response.StatusCode != http.StatusOK {
			return "", fmt.Errorf("请求失败, 错误码：%d", response.StatusCode)
		}

		// 分析业务请求是否成功
		if response.Body == nil {
			return "", errors.New("响应体为空")
		}
		defer response.Body.Close()

		resBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return "", errors.Join(err, errors.New("无法读取响应体"))
		}

		var result ZhfdResult
		if err = json.Unmarshal(resBytes, &result); err != nil {
			return "", errors.Join(err, fmt.Errorf("格式化响应体失败, 原始响应体: %s", string(resBytes)))
		}

		if !result.Success {
			return "", fmt.Errorf("业务响应失败: %s", result.Message)
		}

		// 检查返回数据是否可用
		if len(result.Data.Records) < 1 {
			return "", fmt.Errorf("查询不到水电数据, 原始响应体: %s", string(resBytes))
		}
		record := result.Data.Records[0]

		if len(record.MeterAddForms) != 1 || len(record.WaterAddForms) != 2 {
			return "", fmt.Errorf("水电数据异常, 原始响应体: %s", string(resBytes))
		}

		// 区分冷水和热水
		coldIdx, hotIdx := 0, 1
		if record.WaterAddForms[coldIdx].Remarks != "冷水" {
			coldIdx, hotIdx = hotIdx, coldIdx
		}

		// 构造 html
		htmlBody := builder.
			Set("title", cfg.Mail.Subject).
			Set("electricQuantity", record.MeterAddForms[0].ResidualElectricity).
			Set("coldWater", record.WaterAddForms[coldIdx].RechargeTonnage).
			Set("hotWater", record.WaterAddForms[hotIdx].RechargeTonnage).
			Build()

		return htmlBody, nil
	}, nil
}
