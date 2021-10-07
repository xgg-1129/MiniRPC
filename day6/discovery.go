package day6

import (
	"errors"
	"math"
	"math/rand"
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


func NewDiscovery()Discovery{
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
