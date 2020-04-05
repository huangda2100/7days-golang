package geecache

import (
	"fmt"
	"log"
	"sync"
)

// A Group is a cache namespace and associated data loaded spread over
type Group struct {  //group是一个整体的抽象，对外就只暴露group的api
	name      string
	getter    Getter
	mainCache cache
}

// A Getter loads data for a key.
type Getter interface {  //精华，是依赖注入吗？根据不同的数据源来获取不同源的数据
	Get(key string) ([]byte, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc func(key string) ([]byte, error)  //GetterFunc实现了Get函数，所以是Getter类型

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup(name string) *Group {
	mu.RLock()   //以只读锁来获取group
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

//从数据源获取数据，如mysql，mongo
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key) //使用group特有的getter的get方法，get方法可以根据不同的原始存储实体进行实例化
	if err != nil {
		return ByteView{}, err

	}
	// 选择 byte 类型是为了能够支持任意的数据类型的存储，例如字符串、图片等。
	//b 是只读的，使用 ByteSlice() 方法返回一个拷贝，防止缓存值被外部程序修改。
	value := ByteView{b: cloneBytes(bytes)} //为什么要cloneBytes？
	g.populateCache(key, value) //将值存入cache
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
