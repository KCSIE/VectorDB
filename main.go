package main

import (
	"fmt"

	"vectordb/db"
	"vectordb/logger"
	"vectordb/router"
	"vectordb/settings"
)

// start app
func main() {
	// load config file
	if err := settings.Init(); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}

	// init log
	if err := logger.Init(settings.Conf.LogConfig, settings.Conf.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}

	// init db
	if err := db.Init(settings.Conf.DBConfig.PersistPath); err != nil {
		fmt.Printf("init db failed, err:%v\n", err)
		return
	}
	defer db.Close()

	// register router
	r := router.SetupRouter(settings.Conf.Mode)
	if err := r.Run(fmt.Sprintf("%s:%d", settings.Conf.Host, settings.Conf.Port)); err != nil {
		fmt.Printf("run server failed, err:%v\n", err)
		return
	}
}
