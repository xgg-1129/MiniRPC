package day7

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	CentrePath      =  "/Centre"
	defaultTimtOut  = time.Minute*5
)

type serviceItem struct {
	StartTime time.Time
	Addr string
}

type RegisterCentre struct {
	mu sync.Mutex
	services map[string]*serviceItem
	outTime time.Duration
}
//使用http协议实现通信
func NewCentre(time time.Duration)*RegisterCentre {
	centre:=&RegisterCentre{
		services: nil,
		outTime:  time,
	}
	return centre
}
var DefaultGeeRegister = NewCentre(defaultTimtOut)
//假设get是客户端获取服务列表，post是服务端注册服务
func (r *RegisterCentre) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("RPC-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		addr := req.Header.Get("RPC-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
func (r *RegisterCentre) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
}
func CentreHandleHttp() {
	DefaultGeeRegister.HandleHTTP(CentrePath)
}


func (r *RegisterCentre) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.services[addr]
	if item == nil{
		r.services[addr]=&serviceItem{
			StartTime: time.Now(),
			Addr:      addr,
		}
	}else{
		r.services[addr].StartTime=time.Now()
	}
}

func (r *RegisterCentre) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	res := make([]string,0)
	for key,value := range r.services{
		if r.outTime == 0 || value.StartTime.Add(r.outTime).After(time.Now()){
			res=append(res,key)
		}else{
			delete(r.services,key)
		}
	}
	return res
}
//讲道理这个应该集成在服务端
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		// make sure there is enough time to send heart beat
		// before it's removed from registry
		duration = defaultTimtOut - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("RPC-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}


