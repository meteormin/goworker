package goworker

import (
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"strings"
	"time"
)

type Logger interface {
	Info(args ...interface{})

	//Infof implements message with Sprint, Sprintf, or neither.
	Infof(template string, args ...interface{})
	Infoln(args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Errorln(args ...interface{})
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Debugln(args ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Warnln(args ...interface{})
}

// Dispatcher dispatcher
// manage workers
type Dispatcher interface {
	Dispatch(jobId string, closure func(j *Job) error) error
	Run(names ...string)
	Stop(names ...string)
	SelectWorker(name string) Dispatcher
	GetWorkers() []Worker
	GetRedis() func() *redis.Client
	AddWorker(option Option)
	RemoveWorker(nam string)
	Status() *StatusInfo
	BeforeJob(fn func(j *Job) error, workerNames ...string)
	AfterJob(fn func(j *Job, err error) error, workerNames ...string)
	OnDispatch(fn func(j *Job) error, workerNames ...string)
}

// JobDispatcher implements Dispatcher
type JobDispatcher struct {
	workers []Worker
	worker  Worker
	redis   func() *redis.Client
	logger  Logger
}

// Option JobWorker's option
type Option struct {
	Name        string
	MaxJobCount int
	MaxPool     int
	BeforeJob   func(j *Job) error
	AfterJob    func(j *Job, err error) error
	OnAddJob    func(j *Job) error
	Delay       time.Duration
	Logger      Logger
}

// DispatcherOption dispatcher option
type DispatcherOption struct {
	WorkerOptions []Option
	Redis         func() *redis.Client
}

// defaultWorkerOption default option setting
var defaultWorkerOption = []Option{
	{
		Name:        DefaultWorker,
		MaxJobCount: 10,
		MaxPool:     1,
	},
}

// NewDispatcher make dispatcher
func NewDispatcher(opt DispatcherOption) Dispatcher {
	workers := make([]Worker, 0)

	if len(opt.WorkerOptions) == 0 {
		opt.WorkerOptions = defaultWorkerOption
	}

	for _, o := range opt.WorkerOptions {
		if o.MaxJobCount == 0 {
			o.MaxJobCount = 10
		}

		if o.MaxPool == 0 {
			o.MaxPool = 1
		}

		workers = append(workers, NewWorker(Config{
			o.Name,
			opt.Redis,
			o.MaxJobCount,
			o.MaxPool,
			o.BeforeJob,
			o.AfterJob,
			o.OnAddJob,
			o.Delay,
			o.Logger,
		}))
	}

	return &JobDispatcher{
		workers: workers,
		worker:  nil,
		redis:   opt.Redis,
	}
}

// AddWorker add worker in runtime
func (d *JobDispatcher) AddWorker(option Option) {
	if option.MaxJobCount == 0 {
		option.MaxJobCount = 10
	}

	if option.MaxPool == 0 {
		option.MaxPool = 1
	}

	d.workers = append(d.workers, NewWorker(Config{
		option.Name,
		d.redis,
		option.MaxJobCount,
		option.MaxPool,
		option.BeforeJob,
		option.AfterJob,
		option.OnAddJob,
		option.Delay,
		option.Logger,
	}))
}

// RemoveWorker remove worker in runtime
func (d *JobDispatcher) RemoveWorker(name string) {
	var rmIndex *int = nil
	for i, worker := range d.workers {
		if worker.GetName() == name {
			rmIndex = &i
		}
	}

	if rmIndex != nil {
		d.workers = append(d.workers[:*rmIndex], d.workers[*rmIndex+1:]...)
	}
}

// GetRedis redis client make function
func (d *JobDispatcher) GetRedis() func() *redis.Client {
	return d.redis
}

// GetWorkers get this dispatcher's workers
func (d *JobDispatcher) GetWorkers() []Worker {
	return d.workers
}

// SelectWorker select worker by worker name
func (d *JobDispatcher) SelectWorker(name string) Dispatcher {
	if name == "" {
		for _, w := range d.workers {
			if w.GetName() == "default" {
				d.worker = w
			}
		}

	}

	for _, w := range d.workers {
		if w.GetName() == name {
			d.worker = w
		}
	}

	return d
}

// BeforeJob 해당 Job 수행 전 실행할 클로저 설정
func (d *JobDispatcher) BeforeJob(fn func(j *Job) error, workerNames ...string) {
	if len(workerNames) == 0 {
		for _, w := range d.workers {
			w.BeforeJob(fn)
		}
	} else {
		for _, w := range d.workers {
			for _, name := range workerNames {
				scope := strings.Split(name, ".")
				if w.GetName() == scope[0] {
					if len(scope) >= 2 {
						w.BeforeJob(fn, scope[1])
					} else {
						w.BeforeJob(fn, ".")
					}
				} else if scope[0] == "*" {
					if len(scope) >= 2 {
						w.BeforeJob(fn, scope[1])
					} else {
						w.BeforeJob(fn, ".")
					}
				}
			}
		}
	}
}

// AfterJob 해당 Job 수행 후 실행할 클로저 설정, error가 발생할 수도 있기 때문에 error로 함께 넘겨 받는다.
func (d *JobDispatcher) AfterJob(fn func(j *Job, err error) error, workerNames ...string) {
	if len(workerNames) == 0 {
		for _, w := range d.workers {
			w.AfterJob(fn)
		}
	} else {
		for _, w := range d.workers {
			for _, name := range workerNames {
				scope := strings.Split(name, ".")
				if w.GetName() == scope[0] {
					if len(scope) >= 2 {
						w.AfterJob(fn, scope[1])
					} else {
						w.AfterJob(fn)
					}
				} else if scope[0] == "*" {
					if len(scope) >= 2 {
						w.AfterJob(fn, scope[1])
					} else {
						w.AfterJob(fn)
					}
				}
			}
		}
	}
}

func (d *JobDispatcher) OnDispatch(fn func(j *Job) error, workerNames ...string) {
	if len(workerNames) == 0 {
		for _, w := range d.workers {
			w.OnAddJob(fn)
		}
	} else {
		for _, w := range d.workers {
			for _, wn := range workerNames {
				scope := strings.Split(wn, ".")
				if w.GetName() == scope[0] {
					if len(scope) >= 2 {
						w.OnAddJob(fn, scope[1])
					} else {
						w.OnAddJob(fn)
					}
				} else if scope[0] == "*" {
					if len(scope) >= 2 {
						w.OnAddJob(fn, scope[1])
					} else {
						w.OnAddJob(fn)
					}
				}
			}
		}
	}

}

// Dispatch job을 생성하고 worker에 등록하여 수행할 준비를 한다.
func (d *JobDispatcher) Dispatch(jobId string, closure func(j *Job) error) error {
	if d.worker == nil {
		for _, w := range d.workers {
			if w.GetName() == DefaultWorker {
				d.worker = w
			}
		}
	}

	err := d.worker.AddJob(newJob(d.worker.GetName(), jobId, closure))
	if err != nil {
		return err
	}

	return nil
}

// Run dispatcher의 worker들을 가동 시켜 대기열 루틴을 돌 수 있게 실행,
// 특정 워커들만 수행하고 싶을 경우 workerNames 파라미터를 이용
func (d *JobDispatcher) Run(workerNames ...string) {
	workers := make([]Worker, 0)

	if len(workerNames) == 0 {
		workers = d.workers
	} else {
		for _, w := range d.workers {
			for _, wn := range workerNames {
				if wn == w.GetName() {
					workers = append(workers, w)
				}
			}
		}
	}

	for _, w := range workers {
		w.Run()
	}
}

// Stop 모든 worker의 작업을 종료한다.
// 일부만 종료하고 싶을 경우 workerNames 파라미터를 이용
func (d *JobDispatcher) Stop(workerNames ...string) {
	workers := make([]Worker, 0)

	if len(workerNames) == 0 {
		workers = d.workers
	} else {
		for _, w := range d.workers {
			for _, wn := range workerNames {
				if wn == w.GetName() {
					workers = append(workers, w)
				}
			}
		}
	}

	for _, w := range workers {
		w.Stop()
	}
}

type StatusWorkerInfo struct {
	Name        string `json:"name"`
	IsRunning   bool   `json:"is_running"`
	IsPending   bool   `json:"is_pending"`
	JobCount    int    `json:"job_count"`
	MaxJobCount int    `json:"max_job_count"`
	MaxPool     int    `json:"max_pool"`
	Pool        int    `json:"pool"`
	Queue       []Job  `json:"queue"`
}

type StatusInfo struct {
	Workers     []StatusWorkerInfo `json:"workers"`
	WorkerCount int                `json:"worker_count"`
}

// Print StatusInfo to Console log
func (si *StatusInfo) Print() {
	for _, w := range si.Workers {
		prefix := fmt.Sprintf("[worker: %s]", w.Name)
		marshal, err := json.Marshal(w)
		if err != nil {
			log.Printf("%s %v", prefix, err)
		} else {
			log.Printf("%s %s", prefix, string(marshal))
		}

	}
}

// Status 현재 worker들의 상태를 조회한다.
func (d *JobDispatcher) Status() *StatusInfo {

	workers := make([]StatusWorkerInfo, 0)
	for _, w := range d.workers {
		workerInfo := StatusWorkerInfo{
			Name:        w.GetName(),
			IsRunning:   w.IsRunning(),
			IsPending:   w.IsPending(),
			JobCount:    w.JobCount(),
			MaxJobCount: w.MaxJobCount(),
			MaxPool:     w.MaxPool(),
			Pool:        w.Pool(),
			Queue:       w.Queue().Jobs(),
		}

		workers = append(workers, workerInfo)
	}

	return &StatusInfo{
		Workers:     workers,
		WorkerCount: len(workers),
	}
}
