package router

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"strings"

	"github.com/bocheninc/msg-net/config"
	"github.com/bocheninc/msg-net/logger"
	pb "github.com/bocheninc/msg-net/protos"
)

func StartReport(r *Router) {
	//loop report...
	if !config.GetBool("report.on") {
		return
	}

	i := 0
	for {
		ReportOnce(i, r)
		i++
	}

}

func ReportOnce(n int, r *Router) {
	time.Sleep(config.GetDuration("report.interval"))
	f := make([]interface{}, 0)
	r.routerIterFunc(func(address string, router *pb.Router) {
		f = append(f, router.Address)
	})

	m := make(map[string]int)

	r.peerIterFunc(func(peer *pb.Peer, conn net.Conn) {
		key := strings.Split(peer.Id, ":")[0]
		if _, ok := m[key]; !ok {
			m[key] = 1
		} else {
			m[key]++
		}
	})

	format := ""
	for k, v := range m {
		format = format + fmt.Sprintf("%s:%d_", k, v)
	}

	URI := fmt.Sprintf("%s/msgnetreport/?routerAddress=%s&routercnts=%d&peers=%s&peercnts=%d&reporttimes=%d",
		config.GetString("report.serverIP"),
		r.address,
		len(f),
		format,
		len(m),
		n,
	)

	resp, err := http.Get(URI)
	if err != nil {
		// handle error
		logger.Errorln("report err: ", err)
		return
	}

	defer resp.Body.Close()
	_, _ = ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
		logger.Errorln("report err: ", err)
	}

}
