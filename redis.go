package tcp_server

import (
	"fmt"
	"github.com/anker-dev/infra/redisUtil"
	"github.com/anker-dev/infra/log"
)

const KEY = "tcp_conn:%s"

type connRedis struct {
	Address string
}

func deleteKey(deviceSn string) {
	key := fmt.Sprintf(KEY, deviceSn)
	err := redisUtil.Delete(key)
	if err != nil {
		log.Errorf("DeleteKey %s error:%s", deviceSn, err.Error())
	}
}

func storeConn(deviceSn string, address string) {
	key := fmt.Sprintf(KEY, deviceSn)
	var temp connRedis
	temp.Address = address
	err := redisUtil.SetObject(key, &temp)
	if err != nil {
		log.Errorf("StoreConn %s error:%s", deviceSn, err.Error())
	}
}
