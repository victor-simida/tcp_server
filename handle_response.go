package tcp_server

import (
	"github.com/anker-dev/infra/log"
	. "tcp_project/define"
)

type Handler func(WriteChan, []byte, string)

var codeToHandler map[int]Handler

func init() {
	codeToHandler = make(map[int]Handler)
	//codeToHandler[VERIFY] = verifyHandlerResponse
}

func HandlerResponse(input *ReqCommon, wc WriteChan, content []byte, deviceSN string) {
	h, ok := codeToHandler[input.InfoType]
	if !ok {
		log.Errorf("codeToHandler cannot find %v", input.InfoType)
		return
	}

	if h != nil {
		h(wc, content, deviceSN)
	}
}
