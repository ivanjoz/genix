package db

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gocql/gocql"
)

var mu sync.Mutex
var scyllaSession *gocql.Session
var isConnectingTime int64 = 0
var connectionParams ConnectionParameters = ConnectionParameters{}

type ConnectionParameters struct {
	Host      string
	Port      int
	User      string
	Password  string
	Reconnect bool
}

func SetConnectionParameters(args ConnectionParameters) {
	connectionParams = args
}

func ScyllaConnect() *gocql.Session {
	nowTime := time.Now().Unix()

	if (isConnectingTime + 12) > nowTime {
		for scyllaSession == nil {
			time.Sleep(2 * time.Millisecond)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if connectionParams.Reconnect {
		if scyllaSession != nil {
			scyllaSession.Close()
			scyllaSession = nil
		}
	} else if scyllaSession != nil {
		return scyllaSession
	}

	isConnectingTime = nowTime

	cluster := gocql.NewCluster(connectionParams.Host)
	fallback := gocql.RoundRobinHostPolicy()
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(fallback)
	cluster.Port = connectionParams.Port
	cluster.Consistency = gocql.LocalOne
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = time.Second * 10
	cluster.Compressor = gocql.SnappyCompressor{}
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username:              connectionParams.User,
		Password:              connectionParams.Password,
		AllowedAuthenticators: []string{"org.apache.cassandra.auth.PasswordAuthenticator"},
	}

	session, err := cluster.CreateSession() //  gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		log.Println(err)
		return nil
	}
	fmt.Println("Base de datos ScyllaDB Conectada!!")

	scyllaSession = session
	return scyllaSession
}
