package main

import (
	"bufio"
	"business"
	"bytes"
	"calldb"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const RECV_PACK_LEN = 500

const (
	MaxJob      = 60
	MaxWorkPool = 10
)

type ClientConn struct {
	Cn    *net.Conn
	mutex sync.Mutex
}

//type msg_header struct {}

var g_client_map map[string]ClientConn
var g_client_slice []string

type Job struct {
	Cn      *net.Conn
	Recvbuf []byte
}

type JobQueue chan Job

var jobqueue JobQueue = make(JobQueue, MaxJob)

type Worker struct {
	WorkerPool chan chan Job
	Jobchan    chan Job
	Quit       chan bool
}

type Dispatcher struct {
	WorkPool chan chan Job
}

func NewDispatcher(maxWorkers int) *Dispatcher {
	pool := make(chan chan Job, maxWorkers)
	return &Dispatcher{WorkPool: pool}
}

func (d *Dispatcher) Run() {

	for i := 0; i < MaxWorkPool; i++ {
		worker := NewWorker(d.WorkPool)
		worker.Start()
	}
	go d.Dispatch()
}

func (d *Dispatcher) Dispatch() {
	for {
		select {
		case job := <-jobqueue:
			go func(job Job) {
				jobchan := <-d.WorkPool
				jobchan <- job
			}(job)
		}
	}
}

func NewWorker(workpools chan chan Job) Worker {
	return Worker{WorkerPool: workpools,
		Jobchan: make(chan Job),
		Quit:    make(chan bool)}
}
func remove_slice(slice []string, value string) []string {
	if len(slice) == 0 {
		return slice
	}
	for i, v := range slice {
		if v == value {
			slice = append(slice[:i], slice[i+1:]...)

			break
		}
	}
	return slice
}

func respond(job Job) {
	addr := (*(job.Cn)).RemoteAddr()

	if v, ok := g_client_map[addr.String()]; ok {

		anser, err := business.Transaction(job.Recvbuf)
		if err != nil {
			fmt.Printf("business.Transaction is error, err:%v\n", err)
		}
		v.mutex.Lock()
		(*(job.Cn)).Write([]byte(anser))
		v.mutex.Unlock()
		fmt.Printf("已发送数据到客户端\n")
	}

}

/* start work handler process */
func (work Worker) Start() {
	go func() {
		for {
			work.WorkerPool <- work.Jobchan
			select {
			case job := <-work.Jobchan:
				// execute job
				//fmt.Println(job)
				respond(job)
			case quit := <-work.Quit:
				// exit routine
				fmt.Println(quit)
				return
			}

		}
	}()
}

/* stop worker */
func (work Worker) Stop() {
	go func() {
		work.Quit <- true
	}()
}

func get_header_option(str [][]byte, sep []byte) (error, map[string]string) {

	count := len(str)
	var head_map map[string]string = make(map[string]string)
	if count <= 0 {
		return errors.New("header content is error!"), head_map
	}
	for i := 0; i < count; i++ {
		if i != 0 {
			k := bytes.Split(str[i], []byte(":"))
			head_map[string(k[0])] = string(k[1])
		}
	}
	return nil, head_map
}

func handler_proc(d *Dispatcher, cn *net.Conn) {

	for {
		var j Job
		j.Cn = cn
		j.Recvbuf = make([]byte, 4096)
		recv_data := make([]byte, RECV_PACK_LEN)
		//fmt.Println(recv_data)
		clientaddr := (*cn).RemoteAddr()
		/* 暂时没有考虑粘包，后续改进 add by 2018-08-17 */

		if _, ok := g_client_map[clientaddr.String()]; ok {

			rcounts, err := (*(j.Cn)).Read(recv_data)

			if err != nil {
				fmt.Printf("read client data is error, errno :%v", err)
				(*(j.Cn)).Close()
				//fmt.Printf("read client data is error, errno :%v", err)
				g_client_slice = remove_slice(g_client_slice, clientaddr.String())
				//(*(job.Cn)).Close()
				delete(g_client_map, clientaddr.String())
				return
			}
			fmt.Println(recv_data)
			//fmt.Printf("read count:%d,data:%s", rcounts, j.Recvbuf)
			fmt.Println(rcounts)
			param := bytes.Split(recv_data, []byte("\r\n\r\n"))
			//l := len(param)
			//fmt.Println(string(param[0]), string(param[1]))
			header_line := bytes.Split(param[0], []byte("\r\n"))
			var head_map map[string]string = make(map[string]string)
			err, head_map = get_header_option(header_line, []byte(":"))
			var body_len int
			for i, v := range head_map {
				if strings.TrimSpace(i) == "Content-Length" {
					//fmt.Println(i, v)
					body_len, err = strconv.Atoi(strings.TrimSpace(v))
					if err != nil {
						fmt.Println(err)
					}
					break
				}
			}
			//fmt.Println(body_len)
			body := param[1][:]
			//fmt.Printf("header:%d======len:%d,body:%s\n", bytes.Count(param[0], []byte(""))-1, bytes.Count(body, []byte(""))-1, string(body))
			if rcounts < RECV_PACK_LEN {
				fmt.Printf("body_len:%d,body:%d,byte:%d\n", body_len, len(string(body)), len(body))
				/* 当读出数据长度小于缓冲区长度RECV_PACK_LEN，这样只会读取一个业务 不会出现按连续两个业务被读取 不过这个RECV_PACK_LEN必须是小于两个业务的最小长度 add by 2018-08-21 */

				j.Recvbuf = body[:body_len]
				/* bytes.IndexByte删除结束符\0 */
				//index := bytes.IndexByte(body, 0)
				//j.Recvbuf = body[:index]
				//fmt.Println(j.Recvbuf)

			} else {
				if body_len > len(string(body)) {
					/**/
					recv_len := body_len - len(string(body))
					temp := make([]byte, recv_len)
					//fmt.Printf("temp:%s\n", string(temp))
					r, err := (*(j.Cn)).Read(temp)
					if err != nil {
						fmt.Printf("read client data is error, errno :%v", err)
						(*(j.Cn)).Close()
						//fmt.Printf("read client data is error, errno :%v", err)
						g_client_slice = remove_slice(g_client_slice, clientaddr.String())
						//(*(job.Cn)).Close()
						delete(g_client_map, clientaddr.String())
						return
					}
					fmt.Printf("r:%d\n", r)
					j.Recvbuf = body[:]
					j.Recvbuf = append(j.Recvbuf, temp...)
					fmt.Printf("recvbuf:%s\n", j.Recvbuf)
					//fmt.Printf("read all!")
				}
			}

			jobqueue <- j
			d.Run()
		}

	}

}

func server_init(d *Dispatcher, addr string) {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("create listener is error, errno:%v", err)
		os.Exit(-1)
	}
	for {
		cn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Accept is error , errno:%v", err)
			continue
		}
		clientaddr := cn.RemoteAddr()
		g_client_slice = append(g_client_slice, clientaddr.String())
		var clientconn ClientConn
		clientconn.Cn = &cn
		g_client_map[clientaddr.String()] = clientconn
		go handler_proc(d, &cn)
		//fmt.Println(1112)
	}
}

func broadcast(data []byte) {
	for {
		time.Sleep(time.Duration(5) * time.Second)
		for _, v := range g_client_slice {
			fmt.Printf("send notice to client,ip:%v\n", v)
			if value, ok := g_client_map[v]; ok {
				value.mutex.Lock()
				(*(value.Cn)).Write(data)
				value.mutex.Unlock()
				fmt.Printf("broadcast is finished!\n")
				time.Sleep(time.Duration(5) * time.Second)
			}
		}
	}
}

func read_ini_string(ini_file, node, key, default_value string) (string, error) {
	fp, err := os.Open(ini_file)
	if err != nil {
		fmt.Printf("open ini file is error, err:%v\n", err)
		return default_value, err
	}
	defer fp.Close()
	node_tmp := fmt.Sprintf("[%s]", node)
	//key_tmp := fmt.Sprintf("%s=", key)
	//key_len := len(key_tmp)
	rd := bufio.NewReader(fp)
	for {
		line, err := rd.ReadBytes('\n')
		if err != nil { //遇到任何错误立即返回，并忽略 EOF 错误信息
			if err == io.EOF {
				return default_value, nil
			}
			return default_value, err
		}
		index := bytes.Index(line, []byte("#"))
		if index != -1 {
			//#是注释
			line = line[:index]
		}
		line = bytes.TrimSpace(line)

		//start := bytes.Index(line,[]byte('['))
		//fmt.Printf("byte:%s,string:%s\n", string(line), node_tmp)
		if bytes.Contains(line, []byte(node_tmp)) == true {
			for {
				key_line, err := rd.ReadBytes('\n')
				if err != nil { //遇到任何错误立即返回，并忽略 EOF 错误信息
					if err == io.EOF {
						return default_value, nil
					}
					return default_value, err
				}
				key_line = bytes.TrimSpace(key_line)
				key_index := bytes.Index(key_line, []byte("="))
				//key_index := bytes.Index(key_line, []byte(key_tmp))
				if key_index == -1 {
					continue
				}
				//fmt.Println(string(bytes.TrimSpace(key_line[:key_index])))
				if bytes.Contains(bytes.TrimSpace(key_line[:key_index]), []byte(key)) == true {
					value := bytes.TrimSpace(key_line[key_index+1:])
					return string(value), nil
				}
				//value := key_line[key_index+key_len:]
				//return string(value), nil
			}
		} else {
			continue
		}

	}
}

func read_ini_int(ini_file, node, key, default_value string) (int, error) {
	tmp, err := read_ini_string(ini_file, node, key, default_value)
	if err != nil {
		fmt.Println(err)
		tmp, _ := strconv.Atoi(default_value)
		return tmp, err
	}
	value, err := strconv.Atoi(tmp)
	return value, err
}

func main() {
	//read close chan bool, value is false
	//n := make(chan bool)
	//close(n)
	//w := <- n
	//fmt.Println(w)
	fmt.Println(runtime.GOOS)
	ip, err := read_ini_string("./config/config.ini", "SERVER", "IP", "10.10.1.58")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(ip)
	mysql_user, err := read_ini_string("./config/config.ini", "MYSQL", "USER", "admin")
	if err != nil {
		fmt.Println(err)
	}
	mysql_pwd, err := read_ini_string("./config/config.ini", "MYSQL", "PASSWORD", "admin")
	if err != nil {
		fmt.Println(err)
	}
	mysql_ip, err := read_ini_string("./config/config.ini", "MYSQL", "MYSQL_IP", "10.10.1.58")
	if err != nil {
		fmt.Println(err)
	}
	mysql_port, err := read_ini_int("./config/config.ini", "MYSQL", "MYSQL_PORT", "3306")
	if err != nil {
		fmt.Println(err)
	}
	db_name, err := read_ini_string("./config/config.ini", "MYSQL", "DB", "tj_selflot")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(mysql_user, mysql_pwd, mysql_port, db_name)
	g_client_map = make(map[string]ClientConn)
	g_client_slice = make([]string, 65500)
	calldb.Mysql_init(mysql_user, mysql_pwd, mysql_ip, db_name, mysql_port)
	//calldb.Mysql_init("admin", "admin", "192.168.0.101", "tj_selflot", 3306)
	var d *Dispatcher
	d = NewDispatcher(MaxWorkPool)
	//go broadcast([]byte("HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\n123456abcd"))
	server_init(d, ":3000")

}
