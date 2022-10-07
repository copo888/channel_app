package main

import (
	"github.com/copo888/channel_app/common/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

func main() {
	payKey, _ := utils.MicroServiceEncrypt("FGHGasFd", "SFS47G6U")
	proxyKey, _ := utils.MicroServiceEncrypt("zrdfSeWd", "SFS47G6U")
	lineKey, _ := utils.MicroServiceEncrypt("GHddFKsa", "SFS47G6U")
	logx.Info("paykey  : " + payKey)
	logx.Info("proxyKey: " + proxyKey)
	logx.Info("LineKey: " + lineKey)

	//isOk, _ := utils.MicroServiceVerification(payKey, "FGHGasFd", "SFS47G6U")
	//log.Info("DesCBCDecrypt paykey  : " + strconv.FormatBool(isOk))
}
