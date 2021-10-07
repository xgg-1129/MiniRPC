package day7

import (
	"errors"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)
type selectMole = int

const (
	RandomSelect  selectMole  = iota
	RoundRobinSelect
)

//Discovery集成在客户端
type Discovery interface {
	Get(mole selectMole) (string,error)
	GetAll()([]string,error)
	Refresh() error // refresh from remote registry
	Update(servers []string) error
}
type MultiServersDiscovery struct{
	//保存每台主机的地址
	services []string
	mu sync.Mutex

	//轮盘算法使用
	index int
	//随机算法使用
	seek *rand.Rand
}


func NewDiscovery() *MultiServersDiscovery {
	d:= &MultiServersDiscovery{
		index:    0,
		seek:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	d.index = d.seek.Intn(math.MaxInt32 - 1)
	return d
}

func (m *MultiServersDiscovery) Get(mole selectMole) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	servicesLen:=len(m.services)
	if servicesLen== 0{
		return "",errors.New("no avaiable service")
	}
	switch mole {
	case RandomSelect:
		return m.services[m.seek.Intn(servicesLen)],nil
	case RoundRobinSelect:
		s:=m.services[m.index]
		m.index=(m.index+1)/servicesLen
		return s,nil
	default:
		return "", errors.New("the selectMole is illegal")
	}
}

func (m *MultiServersDiscovery) GetAll() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	//这里要返回一个副本
	copyServices:=make([]string,len(m.services),len(m.services))
	copy(copyServices,m.services)
	return copyServices,nil
}

func (m *MultiServersDiscovery) Refresh() error {
	return nil

}

func (m *MultiServersDiscovery) Update(servers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services=servers
	return nil
}

type GeeRegistryDiscovery struct {
	*MultiServersDiscovery
	registry   string
	timeout    time.Duration
	lastUpdate time.Time
}

const defaultUpdateTimeout = time.Second * 10

func NewGeeRegistryDiscovery(registerAddr string, timeout time.Duration) *GeeRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &GeeRegistryDiscovery{
		MultiServersDiscovery: NewDiscovery(),
		registry:              registerAddr,
		timeout:               timeout,
	}
	return d
}
func (d *GeeRegistryDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.services = servers
	d.lastUpdate = time.Now()
	return nil
}

func (d *GeeRegistryDiscovery) Refresh() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.lastUpdate.Add(d.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh servers from registry", d.registry)
	resp, err := http.Get(d.registry)
	if err != nil {
		log.Println("rpc registry refresh err:", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("RPC-Servers"), ",")
	d.services = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			d.services = append(d.services, strings.TrimSpace(server))
		}
	}
	d.lastUpdate = time.Now()
	return nil
}

func (d *GeeRegistryDiscovery) Get(mode selectMole) (string, error) {
	if err := d.Refresh(); err != nil {
		return "", err
	}
	return d.MultiServersDiscovery.Get(mode)
}

func (d *GeeRegistryDiscovery) GetAll() ([]string, error) {
	if err := d.Refresh(); err != nil {
		return nil, err
	}
	return d.MultiServersDiscovery.GetAll()
}
