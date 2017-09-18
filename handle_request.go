package tcp_server

import (
	"github.com/anker-dev/infra/log"
	. "tcp_project/define"
)

type requestHandler func(string, WriteChan, interface{}) error

var requestHandlerMap map[int]requestHandler

func init() {
	requestHandlerMap = make(map[int]requestHandler)
	//requestHandlerMap[ACTION] = actionHandlerRequest
}

func SendRequestToDevice(deviceSn string, requestCode int, input interface{}) error {
	temp, ok := thePool.Load(deviceSn)
	if !ok {
		return log.Errorf("No Such Device Sn %v", deviceSn)
	}

	h, ok := requestHandlerMap[requestCode]
	if !ok {
		return log.Errorf("No Such Action %v", requestCode)
	}

	pool, _ := temp.(*pool_t)
	err := h(deviceSn, pool.Wc, input)
	if err != nil {
		log.Errorf("sendRequest %v %v Error:%s", deviceSn, requestCode, err.Error())
		return err
	}
	return nil

}
