package goworker

import (
	"encoding/json"
	"fmt"
	"github.com/meteormin/gocache"
	"github.com/redis/go-redis/v9"
	"log"
	"runtime"
	"strings"
	"time"
)

func init() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	gocache.New(uint(m.TotalAlloc))
}

// Logger interface
type Logger interface {
	Info(args ...interface{})
	// Infof implements message with Sprint, Sprintf, or neither.
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
	// Dispatch dispatches the job identified by the given jobId
	// to the closure function for processing.
	//
	// Parameters:
	// - jobId: The identifier of the job to be dispatched.
	// - closure: A function that takes a pointer to a Job
	//            and returns an error.
	//
	// Returns:
	// - An error if there was a problem dispatching the job.
	Dispatch(jobId string, closure func(j *Job) error) error

	// Run is a function that takes in an arbitrary number of string arguments and does something.
	//
	// It does not return anything.
	Run(names ...string)

	// Stop stops the specified names.
	//
	// names: The names to stop.
	Stop(names ...string)

	// SelectWorker selects a worker from the dispatcher based on the given name.
	//
	// name: the name of the worker to select.
	// returns: the selected worker as a Dispatcher.
	SelectWorker(name string) Dispatcher

	// GetWorkers returns an array of Worker objects.
	//
	// It returns an array of Worker objects.
	GetWorkers() []Worker

	// GetWorker returns a Worker based on the provided name.
	//
	// The name parameter specify the name of the worker to retrieve.
	// It can accept one or more names as variadic arguments.
	// If no name is provided, it returns the default worker.
	//
	// Returns the Worker object corresponding to the provided name.
	GetWorker(name ...string) Worker

	// GetRedis returns a function that returns a pointer to a redis.Client.
	//
	// Returns a function that returns a pointer to a redis.Client.
	GetRedis() func() *redis.Client

	// AddWorker adds a worker with the specified option.
	//
	// option: The option for the worker.
	AddWorker(option Option)

	// RemoveWorker removes a worker from the system.
	//
	// Parameters:
	// - nam: the name of the worker to be removed.
	RemoveWorker(nam string)

	// Status returns the status information.
	//
	// No parameters.
	// Returns a pointer to a StatusInfo object.
	Status() *StatusInfo

	// BeforeJob executes the given function before a job starts.
	//
	// The parameter `fn` is a function that takes a pointer to a `Job` structure
	// and returns an error. It represents the function that will be executed
	// before a job starts.
	//
	// The parameter `workerNames` is a variadic parameter of type string. It
	// represents the names of the workers that the job will be executed on.
	//
	// This function does not return anything.
	BeforeJob(fn func(j *Job) error, workerNames ...string)

	// AfterJob executes the given function after a job has been completed.
	//
	// The function takes a callback function as its first parameter, which will be
	// called with a pointer to the completed job and any error that occurred during
	// the job execution. The callback function should return an error.
	//
	// The workerNames parameter is variadic and allows specifying the names of the
	// workers that the callback function should be executed after. If no worker names
	// are provided, the callback function will be executed after any worker completes
	// a job.
	AfterJob(fn func(j *Job, err error) error, workerNames ...string)

	// OnDispatch is a function that performs a specified function on each job in a worker queue.
	//
	// fn: The function to be performed on each job.
	// workerNames: The names of the workers on which the function should be performed.
	//             If no worker names are provided, the function will be performed on all workers.
	OnDispatch(fn func(j *Job) error, workerNames ...string)

	// IsRunning checks if the given process name(s) is running.
	//
	// Parameters:
	// - name: a variadic parameter that represents the name(s) of the process(es) to check.
	//         If no name is provided, it returns the default worker.
	//
	// Returns:
	// - bool: true if the process is running, false otherwise.
	IsRunning(name ...string) bool

	// IsPending checks if the given name are pending.
	//
	// It accepts one or more name as arguments.
	// If no name is provided, it returns the default worker.
	// Returns a boolean value indicating whether the names are pending or not.
	IsPending(name ...string) bool
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
		workers = append(workers, NewWorker(Config{
			o.Name,
			opt.Redis,
			o.MaxJobCount,
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

	d.workers = append(d.workers, NewWorker(Config{
		option.Name,
		d.redis,
		option.MaxJobCount,
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

func (d *JobDispatcher) GetWorker(name ...string) Worker {
	if len(name) == 0 {
		return d.worker
	}

	for _, w := range d.workers {
		if w.GetName() == name[0] {
			return w
		}
	}

	return d.worker
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

// BeforeJob is a method of the JobDispatcher struct that is used to execute a function before a job is dispatched to the workers.
//
// The function accepts a function `fn` that takes a pointer to a Job struct and returns an error. It also accepts optional parameter `workerNames` which is a variadic slice of strings representing the names of the workers. If no worker names are provided, the function will be executed before the job is dispatched to all workers. If worker names are provided, the function will be executed before the job is dispatched to the specified workers.
//
// The function does not return any value.
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

// AfterJob dispatches a function to be executed after a job has completed.
//
// The first parameter `fn` is a function that takes a `*Job` and an `error` as parameters and returns an `error`.
// The remaining parameters `workerNames` are variadic and represent the names of the workers to which the function should be dispatched.
// The function can be dispatched to multiple workers by passing multiple worker names.
//
// This function does not return any values.
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

// OnDispatch dispatches a job to the specified worker(s).
//
// The function takes a function `fn` as its parameter, which represents the job to be dispatched. The function should accept a pointer to a `Job` and return an error.
// Additional parameters `workerNames` are optional and represent the names of the workers to which the job should be dispatched. If no worker names are provided, the job will be dispatched to all workers.
// The function does not return anything.
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

// Dispatch dispatches a job to the JobDispatcher.
//
// It takes a jobId string and a closure function as parameters.
// The closure function is a function that takes a Job pointer as its parameter and returns an error.
// The Dispatch function adds the job to the worker and returns any error that occurred during the process.
// It returns an error if the worker is nil or if there was an error in adding the job to the worker.
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

// Run dispatches jobs to the specified workers.
//
// It accepts a variadic parameter `workerNames` which specifies the names of the workers to dispatch the jobs to.
// If `workerNames` is empty, all workers in the `JobDispatcher` will be used.
//
// This function does not return any values.
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

// Stop stops the specified workers or all workers if no names are provided.
//
// The `workerNames` parameter is a variadic parameter that accepts a list of
// strings representing the names of the workers to be stopped. If no names are
// provided, all workers in the `JobDispatcher` will be stopped.
//
// This function does not return any values.
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

// Status returns the status information of the JobDispatcher.
//
// This function does not have any parameters.
// It returns a pointer to a StatusInfo struct.
func (d *JobDispatcher) Status() *StatusInfo {

	workers := make([]StatusWorkerInfo, 0)
	for _, w := range d.workers {
		workerInfo := StatusWorkerInfo{
			Name:        w.GetName(),
			IsRunning:   w.IsRunning(),
			IsPending:   w.IsPending(),
			JobCount:    w.JobCount(),
			MaxJobCount: w.MaxJobCount(),
			Queue:       w.Queue().Jobs(),
		}

		workers = append(workers, workerInfo)
	}

	return &StatusInfo{
		Workers:     workers,
		WorkerCount: len(workers),
	}
}

// IsRunning checks if any worker in the JobDispatcher is currently running.
//
// It takes a variable number of names as input parameters.
// It returns a boolean indicating whether any worker is running.
func (d *JobDispatcher) IsRunning(name ...string) bool {
	if len(name) != 0 {
		return d.GetWorker(name[0]).IsRunning()
	}

	for _, worker := range d.workers {
		if worker.IsRunning() {
			return true
		}
	}

	return false
}

// IsPending checks if the job with the specified name is pending.
//
// If one or more names are provided, it checks if the worker with the
// first name is pending. If no names are provided, it checks if all
// workers are pending.
//
// Returns true if the job(s) is/are pending, false otherwise.
func (d *JobDispatcher) IsPending(name ...string) bool {
	if len(name) != 0 {
		return d.GetWorker(name[0]).IsPending()
	}

	workerCount := len(d.workers)
	pendingCount := 0
	for _, worker := range d.workers {
		if worker.IsPending() {
			pendingCount++
		}
	}

	return workerCount == pendingCount
}
