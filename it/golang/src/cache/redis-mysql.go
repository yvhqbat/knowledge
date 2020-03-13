package user

import (
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

// go-sql-driver 参考 https://studygolang.com/articles/13655

type UserInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
	AK       string `json:"ak"`
	SK       string `json:"sk"`
}

type Manager struct {
	RedisEndpoint string
	RedisConn     redis.Conn
	MysqlDSN      string
	DB            *sql.DB
	InitFlag      bool
}

func NewManager(redisEndpoint string, mysqlDsn string) *Manager {
	m := new(Manager)

	// redis
	m.RedisEndpoint = redisEndpoint
	conn, err := redis.Dial("tcp", m.RedisEndpoint)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	m.RedisConn = conn

	// mysql
	// m.MysqlDSN = "root:123456@tcp(127.0.0.1:3306)/ld"
	m.MysqlDSN = mysqlDsn
	db, err := sql.Open("mysql", m.MysqlDSN)
	if err != nil {
		log.Errorln(err)
		return nil
	}

	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(60*time.Second)

	SQL := `CREATE TABLE IF NOT EXISTS 
user(id varchar(64) PRIMARY KEY NOT NULL, 
name varchar(64), 
password varchar(64), 
ak varchar(64), 
sk varchar(64))`
	// _, err = db.Exec("CREATE TABLE IF NOT EXISTS user(id varchar(64) PRIMARY KEY NOT NULL, name varchar(64), password varchar(64), ak varchar(64), sk varchar(64))")
	_, err = db.Exec(SQL)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	m.DB = db

	// flag
	m.InitFlag = true

	return m
}

func (m *Manager) Close() error {
	if m.InitFlag == true {
		m.RedisConn.Close()
		m.DB.Close()
		m.InitFlag = false
	}
	return nil
}

// get
func (m *Manager) Get(id string) (*UserInfo, error) {
	userInfo := new(UserInfo)
	// 1. 从缓存获取
	data, err := redis.String(m.RedisConn.Do("HGET", "user:", id))
	if err == nil {
		// 1.1 命中， 返回
		err = json.Unmarshal([]byte(data), userInfo)
		if err != nil {
			log.Errorln(err)
			return nil, err
		}

		return userInfo, nil
	}

	// 1.2 未命中，从数据库获取
	SQL := `SELECT * FROM user WHERE id=?`
	row := m.DB.QueryRow(SQL, id)
	err = row.Scan(&userInfo.ID, &userInfo.Name, &userInfo.Password, &userInfo.AK, &userInfo.SK)
	if err != nil{
		// 1.2.1 数据库结果为空， 返回
		log.Errorln(err)
		return nil, err
	}

	// 1.2.2 数据库结果不为空，插入缓存后返回
	_, err = m.RedisConn.Do("HSET", "users:", userInfo.AK, userInfo.ID)
	if err != nil {
		log.Warnln(err)
		// 插入缓存失败，不返回错误
	}else{
		data, err := json.Marshal(userInfo)
		if err != nil {
			log.Warnln(err)
		}else{
			_, err = m.RedisConn.Do("HSET", "user:", userInfo.ID, data)
			if err != nil {
				log.Warnln(err)
			}
		}
	}

	return userInfo, nil
}

// update // 2. 延时双删
func (m *Manager) Put(userinfo *UserInfo) error {

	return nil
}

// insert CreateUser
func (m *Manager) CreateUser(userInfo *UserInfo) error {
	var err error

	// 1. 分布式锁
	lock_id := strings.Join([]string{"user", userInfo.ID}, ":")
	log.Printf("lock %s\n", lock_id)

	// 2. 插入数据库
	SQL := `INSERT INTO user(id, name, password, ak , sk) VALUES (?,?,?,?,?)`
	_, err = m.DB.Exec(SQL, userInfo.ID, userInfo.Name, userInfo.Password, userInfo.AK, userInfo.SK)
	if err != nil {
		log.Errorln(err)
		return err
	}

	// 3. 插入缓存
	// users:
	_, err = m.RedisConn.Do("HSET", "users:", userInfo.AK, userInfo.ID)
	if err != nil {
		log.Errorln(err)
		return err
	}

	// user:id
	//conn.Do("hset", "user:"+userinfo.ID, "id", userinfo.ID)
	//conn.Do("hset", "user:"+userinfo.ID, "name", userinfo.Name)
	//conn.Do("hset", "user:"+userinfo.ID, "password", userinfo.Password)
	//conn.Do("hset", "user:"+userinfo.ID, "ak", userinfo.AK)
	//conn.Do("hset", "user:"+userinfo.ID, "sk", userinfo.SK)

	data, err := json.Marshal(userInfo)
	if err != nil {
		log.Errorln(err)
		return err
	}
	_, err = m.RedisConn.Do("HSET", "user:", userInfo.ID, data)
	if err != nil {
		log.Errorln(err)
		return err
	}

	return nil
}

// delete
func (m *Manager) Delete(id string) error {
	// 1. 从缓存删除
	_, err := m.RedisConn.Do("HDEL", "user:", id)
	if err != nil {
		log.Errorln(err)
		return err
	}

	// 2. 从数据库删除
	SQL := `DELETE FROM user WHERE id=?`
	_, err = m.DB.Exec(SQL, id)
	if err != nil {
		log.Errorln(err)
		return err
	}

	// 3. 延时
	time.Sleep(1*time.Second)

	// 4. 再次从缓存删除
	_, err = m.RedisConn.Do("HDEL", "user:", id)
	if err != nil {
		log.Errorln(err)
		return err
	}

	return nil
}
