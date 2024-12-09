package db

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gocql/gocql"
)

var mu sync.Mutex
var scyllaSession *gocql.Session
var isConnectingTime int64 = 0
var connParams ConnParams = ConnParams{}

type ConnParams struct {
	Host        string
	Port        int
	User        string
	Password    string
	Reconnect   bool
	ConnTimeout int64 //Seconds
	Keyspace    string
}

func SetScyllaConnection(params ConnParams) {
	if params.ConnTimeout == 0 {
		params.ConnTimeout = 10
	}
	connParams = params
}

func MakeScyllaConnection(params ConnParams) *gocql.Session {
	if params.ConnTimeout == 0 {
		params.ConnTimeout = 10
	}
	connParams = params
	return getScyllaConnection()
}

func getScyllaConnection() *gocql.Session {
	nowTime := time.Now().Unix()

	if (isConnectingTime + 12) > nowTime {
		for scyllaSession == nil {
			time.Sleep(2 * time.Millisecond)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if connParams.Reconnect {
		if scyllaSession != nil {
			scyllaSession.Close()
			scyllaSession = nil
		}
	} else if scyllaSession != nil {
		return scyllaSession
	}

	isConnectingTime = nowTime

	cluster := gocql.NewCluster(connParams.Host)
	fallback := gocql.RoundRobinHostPolicy()
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(fallback)
	cluster.Port = connParams.Port
	cluster.Consistency = gocql.LocalOne
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = time.Second * time.Duration(connParams.ConnTimeout)
	cluster.Compressor = gocql.SnappyCompressor{}
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username:              connParams.User,
		Password:              connParams.Password,
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

func QueryExecStatements(queryStatements []string) error {
	queryToExec := makeQueryStatement(queryStatements)
	return QueryExec(queryToExec)
}

func QueryExec(queryStr string) error {
	query := getScyllaConnection().Query(queryStr)
	if err := query.Exec(); err != nil {
		if strings.Contains(err.Error(), "no hosts available") {
			fmt.Println(`Error en conexión db: "no hosts available", reconectando...`)
			getScyllaConnection()
			fmt.Println(`Ejecutando query luego de reconexión...`)
			err = query.Exec()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
