package api

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go-admin/database"
	"go-admin/global"
	mycasbin "go-admin/pkg/casbin"
	"go-admin/router"
	"go-admin/tools"
	"go-admin/tools/config"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	configYml string
	port      string
	mode      string
	StartCmd  = &cobra.Command{
		Use:     "server",
		Short:   "Start API server",
		Example: "go-admin server config/settings.yml",
		PreRun: func(cmd *cobra.Command, args []string) {
			usage()
			setup()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configYml, "config", "c", "config/settings.yml", "Start server with provided configuration file")
	StartCmd.PersistentFlags().StringVarP(&port, "port", "p", "8000", "Tcp port server listening on")
	StartCmd.PersistentFlags().StringVarP(&mode, "mode", "m", "dev", "server mode ; eg:dev,test,prod")
}

func usage() {
	usageStr := `starting api server`
	log.Printf("%s\n", usageStr)
}

func setup() {
	//1. 读取配置
	config.ConfigSetup(configYml)
	//2. 设置日志
	tools.InitLogger()
	//3. 初始化数据库链接
	database.Setup(config.DatabaseConfig.Driver)

	mycasbin.Setup()

}

func run() error {
	if mode != "" {
		config.SetConfig(configYml, "settings.application.mode", mode)
	}
	if viper.GetString("settings.application.mode") == string(tools.ModeProd) {
		gin.SetMode(gin.ReleaseMode)
	}

	r := router.InitRouter()

	defer global.Eloquent.Close()

	srv := &http.Server{
		Addr:    config.ApplicationConfig.Host + ":" + config.ApplicationConfig.Port,
		Handler: r,
	}

	go func() {
		// 服务连接
		if config.ApplicationConfig.IsHttps {
			if err := srv.ListenAndServeTLS(config.SslConfig.Pem, config.SslConfig.KeyStr); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s \r\n", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s \r\n", err)
			}
		}
	}()
	content, _ := ioutil.ReadFile("./static/go-admin.txt")
	fmt.Println(string(content))
	fmt.Printf("%s Server Run http://%s:%s/ \r\n", tools.GetCurrentTimeStr(), config.ApplicationConfig.Host, config.ApplicationConfig.Port)
	fmt.Printf("%s Swagger URL http://%s:%s/swagger/index.html \r\n", tools.GetCurrentTimeStr(), config.ApplicationConfig.Host, config.ApplicationConfig.Port)
	fmt.Printf("%s Enter Control + C Shutdown Server \r\n", tools.GetCurrentTimeStr())
	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	fmt.Printf("%s Shutdown Server ... \r\n", tools.GetCurrentTimeStr())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
	return nil
}
