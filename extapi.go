package main

import (
	"github.com/TarsCloud/TarsGo/tars"
	"github.com/bytefly/eos-wallet/NeexTrx"
	"log"
)

func storeTokenDepositTx(config *Config, token string, hash string, addr string, amount string) {
	comm := tars.NewCommunicator()
	obj := "NeexTrx.FreezingSysServer.FreezingSysObj"
	registry := config.RegistryAddr
	comm.SetProperty("locator", "tars.tarsregistry.QueryObj@tcp -h "+registry+" -p 17890")
	app := new(NeexTrx.FreezingSys)

	comm.StringToProxy(obj, app)

	ret, err := app.User_into_dc2(addr, token, hash, amount, int32(config.ChainId))
	if err != nil {
		log.Println("call freezing deposit err:", err)
		return
	}
	log.Println("call freezing deposit result:", ret)
}

func storeTokenWithdrawTx(config *Config, token string, hash string, addr string, amount string, fee string) {
	comm := tars.NewCommunicator()
	obj := "NeexTrx.FreezingSysServer.FreezingSysObj"
	registry := config.RegistryAddr
	comm.SetProperty("locator", "tars.tarsregistry.QueryObj@tcp -h "+registry+" -p 17890")
	app := new(NeexTrx.FreezingSys)

	comm.StringToProxy(obj, app)

	var rsp string
	ret, err := app.Commit_withdraw_dc(hash, token, amount, fee, &rsp)
	if err != nil {
		log.Println("call freezing withdraw err:", err)
		return
	}
	log.Println("call freezing withdraw result:", ret, ", rsp:", rsp, ", hash:", hash)
}
