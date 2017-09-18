package tcp_server

import (
	"github.com/anker-dev/infra/log"
	"net"
	"sync"
	. "tcp_project/define"
	"time"
)

type pool_t struct {
	ClientId         string
	UpdateAt         time.Time
	Conn             *net.TCPConn
	Exit             chan int
	Wc               WriteChan
	VersionInfoChan  chan int
	WifiInfoChan     chan int
	SimpleActionChan chan int
	StartCleanChan   chan int
}

type privateConn struct {
	Conn *net.TCPConn
	*sync.RWMutex
}

func renewElement(deviceSn string) {
	temp, ok := thePool.Load(deviceSn)
	if !ok {
		log.Errorf("Cannot get the pool which key is %s", deviceSn)
		return
	}

	pool, _ := temp.(*pool_t)
	pool.UpdateAt = time.Now()
	thePool.Store(deviceSn, pool)
}

func newConnElement(conn *net.TCPConn, deviceSN string, wc WriteChan, exit chan int) *pool_t {
	pool := pool_t{}
	pool.ClientId = deviceSN
	pool.UpdateAt = time.Now()
	pool.Conn = conn
	pool.Wc = wc
	pool.Exit = exit
	pool.VersionInfoChan = make(chan int, 10)
	pool.WifiInfoChan = make(chan int, 10)
	pool.SimpleActionChan = make(chan int, 10)
	pool.StartCleanChan = make(chan int, 10)

	thePool.Store(pool.ClientId, &pool)
	return &pool
}

func CheckConnExist(deviceSn string) bool {
	_, ok := thePool.Load(deviceSn)
	return ok
}
