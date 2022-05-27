package main

import (
	"status-feed-server/apiserver"
	"status-feed-server/globalctx"
)

func main() {
	globalctx.InitCfg()

	//go ytsource.Init()
	//
	//go wallpaper.Init()
	//
	//go ytsource.GetLastFMResource()
	//go keeper.GetLastFMResource()

	apiserver.RunServer()
}
