package tcp_server

import (
	"encoding/json"
	"github.com/anker-dev/infra/MAP"
	"github.com/anker-dev/infra/log"
	"io"
	"net"

	"bufio"
	"bytes"
	"github.com/astaxie/beego"
	. "tcp_project/define"
	"time"
)

var thePool *MAP.Map
var G_listener *net.TCPListener

func init() {
	thePool = new(MAP.Map)

}

func TcpServerStart() {
	addr, err := net.ResolveTCPAddr("tcp", ":"+beego.AppConfig.String("tcpport"))
	if err != nil {
		log.Infof("ResolveTCPAddr %s", err.Error())
		return
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Infof("ListenTCP %s", err.Error())
		return
	}

	G_listener = listener

	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		conn, err := G_listener.AcceptTCP()
		if err != nil {
			log.Errorf("TcpServer Error:%s", err.Error())
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Errorf("http: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}

		tempDelay = 0
		go server(conn)

	}
}

func TcpServerClose() {
	if G_listener != nil {
		G_listener.Close()
	}
	thePool.Range(func(iDeviceSN, iValue interface{}) bool {
		log.Infof("%v's connection is out", iDeviceSN)
		value, _ := iValue.(*pool_t)
		value.Exit <- 1
		deleteKey(value.ClientId)
		return true
	})
}

func server(conn *net.TCPConn) {
	wc := WriteChan(make(chan []byte, 1))
	exit := make(chan int, 1)
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("server panic %v", err)
			return
		}
	}()
	reader := bufio.NewReader(conn)
	message, err := readWithEnd(reader)
	if err != nil || len(message) == 0 {
		log.Errorf("read failed %s", err.Error())
		return
	}
	message = message[:len(message)-3]

	log.Infof("server recieved %s", string(message))

	var temp LogInRequest
	err = json.Unmarshal(message, &temp)
	if err != nil {
		log.Errorf("json unmarshal error:%s", err.Error())
		return
	}
	log.Infof("LogInRequest :%+v", temp)

	pool := newConnElement(conn, temp.Data.SN, wc, exit)
	storeConn(temp.Data.SN, conn.LocalAddr().String())
	go writeHandler(temp.Data.SN, pool.Conn, WriteChan(wc), exit)
	go echoHandler(temp.Data.SN, exit)

	HandlerResponse(&temp.ReqCommon, wc, message, temp.Data.SN)
	readHandler(temp.Data.SN, pool.Conn, WriteChan(wc), exit)
}

func writeTheChan(input []byte, c chan []byte) {
	t := 5 * time.Millisecond
	for i := 0; i < 5; i++ {
		select {
		case c <- input:
			return
		default:
			time.Sleep(t)
			t *= 2
		}
	}
}

func writeHandler(deviceSN string, conn *net.TCPConn, wc WriteChan, exit chan int) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("writeHandler %s panic %v", deviceSN, err)
		}
		log.Infof("Write out")
		conn.CloseWrite()
	}()

	for {
		select {
		case data, ok := <-wc:
			if ok {
				buf := bytes.NewBuffer(data)
				buf.WriteString("#\t#")
				conn.Write(buf.Bytes())
				log.Infof("%s write %s", deviceSN, data)
			}
		case <-exit:
			return
		}
	}

}

func readHandler(deviceSN string, conn *net.TCPConn, wc WriteChan, exit chan int) {
	defer func() {
		log.Infof("readHandler exit")
		conn.CloseRead()
	}()
	conn.SetReadBuffer(1048576)
	reader := bufio.NewReader(conn)
	for {
		var temp ReqCommon
		select {
		case <-exit:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		//n ,err := io.ReadFull(reader,message)
		//log.Infof("%d %v %s",n,err,message )
		message, err := readWithEnd(reader)
		if err == nil && len(message) >= 3 {
			message = message[:len(message)-3]
			log.Infof("read from socket get %s", string(message))
			err = json.Unmarshal([]byte(message), &temp)
			if err != nil {
				log.Errorf("json unmarshal error:%s len %v msg:%s", err.Error(), len(message), message)
				continue
			}
			renewElement(deviceSN)

			HandlerResponse(&temp, wc, message, deviceSN)

		} else if err == io.EOF {
			log.Infof("connection end")
			return
		} else {
			log.Infof("readWithEnd %v %s", err, message)
		}

	}
}

func echoHandler(deviceSN string, exit chan int) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("echoHandler %v panic %v", deviceSN, err)
		}
		close(exit)
		thePool.Delete(deviceSN)
		deleteKey(deviceSN)
		return
	}()

	for {
		time.Sleep(time.Second)
		temp, ok := thePool.Load(deviceSN)
		if !ok {
			log.Infof("echoHandler not found")
			return
		}

		pool, _ := temp.(*pool_t)
		if pool.UpdateAt.Add(50 * time.Second).Before(time.Now()) {
			log.Infof("echoHandler exit")
			return
		}
	}
}

func readWithEnd(reader *bufio.Reader) ([]byte, error) {
	message, err := reader.ReadBytes('#')
	if err != nil {
		return nil, err
	}

	a1, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	message = append(message, a1)
	if a1 != '\t' {
		message2, err := readWithEnd(reader)
		if err != nil {
			return nil, err
		}
		ret := append(message, message2...)
		return ret, nil
	}

	a2, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	message = append(message, a2)
	if a2 != '#' {
		message2, err := readWithEnd(reader)
		if err != nil {
			return nil, err
		}
		ret := append(message, message2...)
		return ret, nil
	}

	return message, nil
}

func readWithEnd2(reader *bufio.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(reader)
	split := func(data []byte, atEof bool) (advance int, token []byte, err error) {
		if atEof && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, []byte("#\t#")); i >= 0 {
			return i + 3, data[:i], nil
		}
		if atEof {
			return len(data), data, nil
		}

		return 0, nil, nil

	}

	scanner.Split(split)

	if scanner.Scan() {
		ret := scanner.Bytes()
		log.Infof("return %s ", ret)
		return ret, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}
