package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
	"sync"
	"time"
)

func lock(key string, cli *clientv3.Client){
	// session
	s, err := concurrency.NewSession(cli)
	if err!=nil{
		log.Errorf("new session failed\n")
		return
	}

	m := concurrency.NewMutex(s,key)
	//m := new(concurrency.Mutex)
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx ,5*time.Second)
	defer cancel()
	err = m.Lock(ctx)
	if err!=nil{
		log.Errorf("lock failed\n")
		return
	}
	log.Info("locked successfully")
	ctx = context.Background()
	ctx, cancel = context.WithTimeout(ctx ,5*time.Second)
	defer cancel()
	err = m.Unlock(ctx)
	if err!=nil{
		log.Errorf("unlock failed\n")
		return
	}
}
func demo(){
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"10.192.71.31:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Errorf("connect failed %s", err)
		return
	}
	defer cli.Close()

	lock("lock_ld", cli)

	// put
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	res, err := cli.Put(ctx,"key1", "value1")
	cancel()
	if err!=nil{
		log.Errorf("put failed %s", err)
		return
	}
	log.Infof("%v", res)

	// get
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	getRes, err := cli.Get(ctx, "key1")
	cancel()
	if err!=nil{
		log.Errorf("put failed %s", err)
		return
	}
	log.Printf("%v", getRes)
	for pos, kv := range getRes.Kvs{
		log.Printf("pos:%d, key:%s, value:%s\n", pos, kv.Key, kv.Value)
	}

	// watch
	// ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		log.Infof("enter go func")
		ctxWatch := context.Background()
		chWatch := cli.Watcher.Watch(ctxWatch, "key1")
		for e := range chWatch{
			log.Infof("%v", e)
		}
		log.Infof("leave go func")
		wg.Done()
	}()

	// lease
	ctx = context.Background()
	//ctx = context.WithValue(ctx, "name", "liudong")
	//log.Infof("%s\n", ctx.Value("name"))
	leaseRes, err := cli.Lease.Grant(ctx, 10)
	if err!=nil{
		log.Errorf("%s\n", err)
		return
	}
	log.Infof("ID:%d\n", leaseRes.ID)

	for i:=0;i<5;i++{
		ctx = context.Background()
		leaseLiveRes, err := cli.Lease.TimeToLive(ctx, leaseRes.ID)
		if err!=nil{
			log.Errorf("%s\n", err)
			return
		}
		log.Infof("TTL:%d\n", leaseLiveRes.TTL)
		time.Sleep(1*time.Second)
	}

	// lease keepalive once
	ctx = context.Background()
	leaseKeepaliveRes, err := cli.Lease.KeepAliveOnce(ctx, leaseRes.ID)
	if err!=nil{
		log.Errorf("%s\n", err)
		return
	}
	log.Infof("ID:%d, ttl:%d", leaseKeepaliveRes.ID, leaseKeepaliveRes.TTL)

	// put with lease
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	putRes, err := cli.Put(ctx,"key2", "lease 10", clientv3.WithLease(leaseRes.ID))
	cancel()
	if err!=nil{
		log.Errorf("put failed %s", err)
		return
	}
	log.Infof("%v", putRes)

	// get with prefix
	ctx = context.Background()
	getRes, err = cli.KV.Get(ctx, "key",clientv3.WithPrefix())
	if err!=nil{
		log.Errorf("put failed %s", err)
		return
	}
	for pos, kv := range getRes.Kvs{
		log.Printf("get with prefix, pos:%d, key:%s, value:%s\n", pos, kv.Key, kv.Value)
	}


	// delete
	ctx = context.Background()
	deleteRes, err := cli.Delete(ctx, "key1")
	if err!=nil{
		log.Errorf("%s\n", err)
		return
	}
	log.Infof("%v\n", deleteRes)

	// wait for go func
	//for{
	//	time.Sleep(1*time.Second)
	//}
	wg.Wait()
}

func main(){
	// demo()
	log.Infof("hello world")
}
