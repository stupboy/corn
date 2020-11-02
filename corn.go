package corn

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	TaskTimeOut    = 500 //任务超时分配时间
	MinTaskNum     = 3   //最小携程数量不得小于3
	DefaultTaskNum = 500 //默认携程数量
)

var (
	CronList []cronItem    //定时列表
	record   itemRecord    //执行记录
	MaxTasks int           //最大任务数量
	TaskChan chan struct{} //任务携程
)

// 执行记录
type itemRecord struct {
	m map[string]struct{}
	s sync.Mutex
}

// 添加记录
func (i *itemRecord) add(key string) (has bool) {
	if key == "none" {
		return has
	}
	i.s.Lock()
	if _, ok := i.m[key]; !ok {
		has = false
		i.m[key] = struct{}{}
	} else {
		has = true
	}
	i.s.Unlock()
	return has
}

// 删除记录
func (i *itemRecord) del(key string) {
	if key == "none" {
		return
	}
	i.s.Lock()
	delete(i.m, key)
	i.s.Unlock()
}

// 定时项目
type cronItem struct {
	f   func()
	c   string
	key string
}

func init() {
	CronList = make([]cronItem, 0)
	record.m = make(map[string]struct{}, 0)

}

// 执行定时服务
func RunCorn(taskNum ...int) {
	MaxTasks = DefaultTaskNum
	if len(taskNum) > 0 {
		MaxTasks = taskNum[0]
	}
	if MaxTasks < MinTaskNum+len(CronList) {
		MaxTasks = MinTaskNum + len(CronList)
		log.Println("定时任务数量必须大于:", MinTaskNum+len(CronList))
	}
	TaskChan = make(chan struct{}, MaxTasks)
	go CronServer() //开启定时携程
}

// 添加定时列表
func AddCorn(f func(), c ...string) {
	var (
		key string
	)
	key = "none"
	if len(c) < 1 {
		return
	}
	if len(c) > 1 {
		key = c[1]
	}
	CronList = append(CronList, cronItem{
		f:   f,
		c:   c[0],
		key: key,
	})
}

// 遍历执行列表
func doCronList() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("定时任务出错")
		}
		<-TaskChan //列表携程退出
	}()
	t := time.Now().Format("2006-01-02 15:04:05")
	for _, i := range CronList {
		select {
		case TaskChan <- struct{}{}:
			go analyzeCron(i, t)
		case <-time.After(TaskTimeOut * time.Millisecond):
			log.Println("任务被跳过:", i.c, i.key)
		}
	}
}

// 解析定时
func analyzeCron(item cronItem, t string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("定时任务执行错误：", r)
			record.del(item.key)
		}
		<-TaskChan //定时明细退出
		//log.Println("携程数量:", len(TaskChan))
	}()
	cs := strings.Split(item.c, " ")
	if len(cs) != 6 {
		return
	}
	// 时 开始处理
	if analyzeTime(cs[2], t[11:13]) {
		return
	}
	if analyzeTime(cs[1], t[14:16]) {
		return
	}
	if analyzeTime(cs[0], t[17:19]) {
		return
	}
	if record.add(item.key) {
		return
	}
	item.f()
	record.del(item.key)
}

// 解析时间
func analyzeTime(a, b string) bool {
	if a == "*" {
		return false
	}
	if strings.Index(a, "-") > -1 {
		aa := strings.Split(a, "-")
		if len(aa) != 2 {
			return true
		}
		a1, _ := strconv.Atoi(aa[0])
		a2, _ := strconv.Atoi(aa[1])
		a3, _ := strconv.Atoi(b)
		if a3 >= a1 && a3 <= a2 {
			return false
		} else {
			return true
		}
	}
	if strings.Index(a, ",") > -1 {
		ok := true
		aa := strings.Split(a, ",")
		for _, k := range aa {
			if k == b {
				ok = false
				break
			}
		}
		return ok
	}
	if strings.Index(a, "/") > -1 {
		aa := strings.Split(a, "/")
		if len(aa) != 2 {
			return true
		}
		if aa[0] != "*" {
			return true
		}
		a1, _ := strconv.Atoi(aa[1])
		a2, _ := strconv.Atoi(b)
		if a2%a1 == 0 {
			return false
		} else {
			return true
		}
	}
	if a != b {
		return true
	}
	return false
}

// 定时服务
func CronServer() {
	for {
		serverGo()
	}
}

// 服务携程
func serverGo() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("错误", r)
		}
	}()
	select {
	case <-time.After(time.Second): //每秒执行一次
		//log.Println("正在执行携程数量:", len(TaskChan))
		select {
		case TaskChan <- struct{}{}:
			go doCronList()
		case <-time.After(TaskTimeOut * time.Millisecond): //500毫秒内未分配则跳过定时
			log.Println("任务执行队列已满，跳过时间点:", time.Now().Unix())
			log.Println("正在执行携程数量:", len(TaskChan))
		}
	}
}
