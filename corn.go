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

type ServerCron struct {
	CronList   []CronItem    //定时列表
	debug      bool          //是否调试模式
	ignoreList sync.Map      //过滤map
	taskChan   chan struct{} //任务channel
	record     itemRecord    //任务执行记录
	MaxTasks   int           //	最大执行任务数量
	status     bool          //定时任务状态 true开启 false 暂停
}

// 定时开启
func (s *ServerCron) Start() {
	s.status = true
}

// 定时暂停
func (s *ServerCron) Stop() {
	s.status = false
}

// 执行记录
type itemRecord struct {
	m map[string]struct{}
	s sync.Mutex
}

// 添加跳过记录
func (s *ServerCron) AddIgnore(key string) {
	s.ignoreList.Store(key, struct{}{})
}

// 删除跳过记录
func (s *ServerCron) DelIgnore(key string) {
	s.ignoreList.Delete(key)
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
type CronItem struct {
	f   func()
	C   string
	Key string
}

func New() (cron ServerCron) {
	cron.CronList = make([]CronItem, 0)
	cron.record.m = make(map[string]struct{}, 0)
	cron.status = true
	return cron
}

// 开启调试模式
func (s *ServerCron) StartDebug() {
	s.debug = true
}

// 执行定时服务
func (s *ServerCron) RunCorn(taskNum ...int) {
	s.MaxTasks = DefaultTaskNum
	if len(taskNum) > 0 {
		s.MaxTasks = taskNum[0]
	}
	if s.MaxTasks < MinTaskNum+len(s.CronList) {
		s.MaxTasks = MinTaskNum + len(s.CronList)
		log.Println("定时任务数量必须大于:", MinTaskNum+len(s.CronList))
	}
	s.taskChan = make(chan struct{}, s.MaxTasks)
	go s.cronServer() //开启定时携程
}

// 添加定时列表
func (s *ServerCron) AddCorn(f func(), TimeStr string, argus ...string) {
	var (
		key string
	)
	key = "none"
	if len(argus) > 0 {
		key = argus[0]
	}
	s.CronList = append(s.CronList, CronItem{
		f:   f,
		C:   TimeStr,
		Key: key,
	})
}

// 遍历执行列表
func (s *ServerCron) doCronList() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("定时任务出错")
		}
		<-s.taskChan //列表携程退出
	}()
	t := time.Now().Format("2006-01-02 15:04:05")
	for _, i := range s.CronList {
		_, ok := s.ignoreList.Load(i.Key)
		if ok {
			continue //在忽略列表中则退出
		}
		select {
		case s.taskChan <- struct{}{}:
			go s.analyzeCron(i, t)
		case <-time.After(TaskTimeOut * time.Millisecond):
			log.Println("任务被跳过:", i.C, i.Key)
		}
	}
}

// 解析定时
func (s *ServerCron) analyzeCron(item CronItem, t string) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("定时任务执行错误：", r)
			s.record.del(item.Key)
		}
		<-s.taskChan //定时明细退出
		//log.Println("携程数量:", len(TaskChan))
	}()
	cs := strings.Split(item.C, " ")
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
	if s.record.add(item.Key) {
		return
	}
	item.f()
	s.record.del(item.Key)
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
func (s *ServerCron) cronServer() {
	for {
		s.serverGo()
	}
}

// 服务携程
func (s *ServerCron) serverGo() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("错误", r)
		}
	}()
	if !s.status { //任务被暂停
		return
	}
	select {
	case <-time.After(time.Second): //每秒执行一次
		if s.debug {
			log.Println("正在执行携程数量:", len(s.taskChan))
		}
		select {
		case s.taskChan <- struct{}{}:
			go s.doCronList()
		case <-time.After(TaskTimeOut * time.Millisecond): //500毫秒内未分配则跳过定时
			log.Println("任务执行队列已满，跳过时间点:", time.Now().Unix())
			log.Println("执行携程数量:", len(s.taskChan))
		}
	}
}
