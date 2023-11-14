package goworker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

// InMemoryMap in memory job's storage
type InMemoryMap map[string]*Job

// JobStatus job's status
// success, wait, fail, progress
type JobStatus string

const (
	SUCCESS  JobStatus = "success"
	FAIL     JobStatus = "fail"
	WAIT     JobStatus = "wait"
	PROGRESS JobStatus = "progress"
)

// ctx  Redis context
var ctx = context.Background()

// InMemoryMap in memory job's storage variable
var inMemoryMap = InMemoryMap{}

// DefaultWorker default worker name
const DefaultWorker = "default"

// Job job's struct
type Job struct {
	UUID       uuid.UUID              `json:"uuid"`
	WorkerName string                 `json:"worker_name"`
	JobId      string                 `json:"job_id"`
	Status     JobStatus              `json:"status"`
	Closure    func(job *Job) error   `json:"-"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Meta       map[string]interface{} `json:"meta"`
}

// Marshal to json
func (j *Job) Marshal() (string, error) {
	marshal, err := json.Marshal(j)
	if err != nil {
		return "", err
	}

	return string(marshal), nil
}

// UnMarshal json to Job
func (j *Job) UnMarshal(jsonStr string) error {
	err := json.Unmarshal([]byte(jsonStr), &j)
	if err != nil {
		return err
	}
	return nil
}

// newJob create new job
func newJob(workerName string, jobId string, closure func(job *Job) error) Job {
	return Job{
		UUID:       uuid.New(),
		WorkerName: workerName,
		JobId:      jobId,
		Closure:    closure,
		Status:     WAIT,
		CreatedAt:  time.Now(),
		Meta:       map[string]interface{}{},
	}
}

// Queue job queue interface
type Queue interface {
	Enqueue(job Job) error
	Dequeue() (*Job, error)
	Clear()
	Count() int
	Next() *Job
	Jobs() []Job
}

// JobQueue job queue implement struct
type JobQueue struct {
	queue       []Job
	jobChan     chan Job
	maxJobCount int
}

// NewQueue new JobQueue instance
func NewQueue(maxJobCount int) Queue {
	return &JobQueue{
		queue:       make([]Job, 0),
		jobChan:     make(chan Job, maxJobCount),
		maxJobCount: maxJobCount,
	}
}

// Enqueue insert job in job queue
func (q *JobQueue) Enqueue(job Job) error {
	if q.maxJobCount > len(q.queue) {
		q.queue = append(q.queue, job)
		q.jobChan <- job
		return nil
	}

	return errors.New("can't enqueue job queue: over queue size")
}

// Dequeue remove job from job queue
func (q *JobQueue) Dequeue() (*Job, error) {
	if len(q.queue) == 0 {
		job := <-q.jobChan
		return &job, nil
	}

	job := q.queue[0]
	q.queue = q.queue[1:]
	jobChan := <-q.jobChan
	if job.JobId == jobChan.JobId {
		return &jobChan, nil
	}

	return nil, errors.New("can't match job id")
}

// Clear clean job queue, create new instance
func (q *JobQueue) Clear() {
	q.queue = make([]Job, 0)
	q.jobChan = make(chan Job)
}

// Count return job queue size
func (q *JobQueue) Count() int {
	return len(q.queue)
}

// Next return first job in queue
func (q *JobQueue) Next() *Job {
	if len(q.queue) == 0 {
		return nil
	}

	return &q.queue[0]
}

// Jobs return all jobs in queue
func (q *JobQueue) Jobs() []Job {
	return q.queue
}

// Hooks

// BeforeJob is a function that describes what it does.
type BeforeJob = func(j *Job) error

// AfterJob is a function that describes what it does.
type AfterJob = func(j *Job, err error) error

// OnAddJob is a function that adds a job to the specified scope.
type OnAddJob = func(j *Job) error

// Worker worker interface
type Worker interface {
	// GetName returns the name of the object.
	//
	// No parameters.
	// It returns a string.
	GetName() string

	// Run is a function that executes a task.
	//
	// It does not take any parameters.
	// It does not return any values.
	Run()

	// Resume description of the Go function.
	//
	// Resume does not have any parameters.
	// It does not have a return type.
	Resume()

	// Pending description of the Go function.
	Pending()

	// Stop stops the execution of the Go function.
	Stop()

	// AddJob adds a job to the system.
	//
	// job: the job to be added.
	// error: an error if the job could not be added.
	AddJob(job Job) error

	// IsRunning returns a boolean value indicating whether the process is running.
	//
	// No parameters.
	// Returns a boolean value.
	IsRunning() bool

	// IsPending returns a boolean value indicating whether the task is pending or not.
	//
	// This function does not take any parameters.
	// It returns a boolean value.
	IsPending() bool

	// MaxJobCount returns the maximum job count.
	//
	// It does not take any parameters.
	// It returns an integer value.
	MaxJobCount() int

	// JobCount returns the number of jobs.
	//
	// It does not take any parameters.
	// It returns an integer value.
	JobCount() int

	// Queue returns the Queue.
	//
	// No parameters.
	// Returns Queue.
	Queue() Queue

	// BeforeJob is a function that describes what it does.
	//
	// It takes a function as an argument, which is expected to have a pointer to a Job struct as its parameter, and returns an error.
	// The function can also have an optional variadic parameter of type string.
	BeforeJob(fn func(j *Job) error, scope ...string)

	// AfterJob is a function that executes the given function after a job has completed.
	//
	// The function takes in a function `fn` as its parameter that accepts a `*Job` and an `error`
	// and returns an `error`. The `fn` is executed after the job has completed and is passed the
	// job object and any error that occurred during the job execution.
	//
	// There is an optional variadic parameter `scope` that allows specifying a scope for the job.
	// The scope is a string that can be used to group related jobs together.
	AfterJob(fn func(j *Job, err error) error, scope ...string)

	// OnAddJob is a function that adds a job to the specified scope.
	//
	// fn: a function that takes a pointer to a Job struct and returns an error.
	// scope: optional string(s) specifying the scope(s) to add the job to.
	OnAddJob(fn func(j *Job) error, scope ...string)
}

// JobWorker worker implement
type JobWorker struct {
	Name        string
	queue       Queue
	jobChan     chan *Job
	pendChan    chan bool
	quitChan    chan bool
	redis       func() *redis.Client
	maxJobCount int
	isRunning   bool
	isPending   bool
	beforeJob   map[string]BeforeJob
	afterJob    map[string]AfterJob
	onAddJob    map[string]OnAddJob
	redisClient *redis.Client
	delay       time.Duration
	logger      Logger
}

// Config worker's configuration
type Config struct {
	Name        string
	Redis       func() *redis.Client
	MaxJobCount int
	BeforeJob   BeforeJob
	AfterJob    AfterJob
	OnAddJob    OnAddJob
	Delay       time.Duration
	Logger      Logger
}

// NewWorker returns a new Worker object based on the provided configuration.
//
// Parameters:
// - cfg: The configuration object containing the settings for the Worker.
//
// Return:
// - Worker: The newly created Worker object.
func NewWorker(cfg Config) Worker {
	var redisClient *redis.Client
	if cfg.Redis != nil {
		redisClient = cfg.Redis()
	}

	return &JobWorker{
		Name:        cfg.Name,
		queue:       NewQueue(cfg.MaxJobCount),
		jobChan:     make(chan *Job, cfg.MaxJobCount),
		pendChan:    make(chan bool),
		quitChan:    make(chan bool),
		redis:       cfg.Redis,
		maxJobCount: cfg.MaxJobCount,
		beforeJob:   map[string]BeforeJob{".": cfg.BeforeJob},
		afterJob:    map[string]AfterJob{".": cfg.AfterJob},
		onAddJob:    map[string]OnAddJob{".": cfg.OnAddJob},
		redisClient: redisClient,
		delay:       cfg.Delay,
		logger:      cfg.Logger,
	}
}

// saveJob saves a job with the given key in the JobWorker.
//
// Parameters:
// - key: The key used to save the job.
// - job: The job to be saved.
//
// Return type: None.
func (w *JobWorker) saveJob(key string, job Job) {
	if w.redisClient == nil {
		inMemoryMap[key] = &job
		return
	}

	jsonJob, err := job.Marshal()
	if err != nil {
		panic(err)
	}

	err = w.redisClient.Set(ctx, key, jsonJob, time.Minute).Err()
	if err != nil {
		panic(err)
	}
}

// getJob retrieves a Job from either the in-memory map or Redis based on the provided key.
//
// Parameters:
// - key: the key used to identify the Job.
//
// Returns:
// - *Job: the retrieved Job if found, or nil if not found.
// - error: any error that occurred during the retrieval process.
func (w *JobWorker) getJob(key string) (*Job, error) {
	if w.redisClient == nil {
		if job, ok := inMemoryMap[key]; ok {
			return job, nil
		}
		return nil, nil
	}

	val, err := w.redisClient.Get(ctx, key).Result()

	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		convJob := &Job{}
		err = convJob.UnMarshal(val)

		if err != nil {
			return nil, err
		}

		return convJob, nil
	}
}

// delJob deletes a job.
//
// The delJob function takes a key string as a parameter and deletes the job associated with that key. If the redisClient is not nil, it uses the redisClient to delete the job from the Redis server. Otherwise, it deletes the job from the inMemoryMap. The function does not return any value.
func (w *JobWorker) delJob(key string) {
	if w.redisClient == nil {
		delete(inMemoryMap, key)
		return
	}

	_, err := w.redisClient.Del(ctx, key).Result()
	if err != nil {
		panic(err)
	}
}

// work executes the job for the JobWorker.
//
// It takes in a job as a parameter and performs the following steps:
// - Generates a key based on the JobWorker's name and the job's jobId.
// - Sets the job's status to PROGRESS.
// - Saves the job using the generated key.
// - Logs the start of the job.
// - Calls the beforeJob handler if it exists and handles any errors.
// - Executes the job's closure and handles any errors.
// - Updates the job's status based on the success or failure of the closure.
// - Updates the job's updatedAt timestamp.
// - Saves the job again.
// - Calls the afterJob handler if it exists and handles any errors, passing the job and the closure error.
// - Marshals the job into JSON format and logs it.
// - Deletes the job using the generated key.
//
// The function does not have any return values.
func (w *JobWorker) work(job *Job) {
	key := fmt.Sprintf("%s.%s", w.Name, job.JobId)
	job.Status = PROGRESS

	w.saveJob(key, *job)

	log.Printf("start job: %s.%s", job.WorkerName, job.JobId)
	if w.logger != nil {
		w.logger.Infof("start job: %s.%s", job.WorkerName, job.JobId)
	}

	if w.beforeJob != nil {
		bErr := w.handleBeforeJob(job)
		if bErr != nil {
			log.Print(bErr)
			if w.logger != nil {
				w.logger.Error(bErr)
			}
		}
	}

	err := job.Closure(job)
	if err != nil {
		job.Status = FAIL
	} else {
		job.Status = SUCCESS
	}

	job.UpdatedAt = time.Now()
	w.saveJob(key, *job)

	if w.afterJob != nil {
		aErr := w.handleAfterJob(job, err)
		if aErr != nil {
			log.Print(aErr)
			if w.logger != nil {
				w.logger.Error(aErr)
			}
		}
	}

	jsonJob, err := job.Marshal()
	if err != nil {
		log.Print(err)
		if w.logger != nil {
			w.logger.Error(err)
		}
	}

	log.Printf("end job: %s", jsonJob)
	if w.logger != nil {
		w.logger.Infof("end job: %s", jsonJob)
	}

	w.delJob(key)
}

// routine is a Go function that represents the routine of a JobWorker. It performs the following tasks:
//  1. Logs the start of the routine.
//  2. Checks if the worker has a logger and logs the start of the routine with the logger if it is available.
//  3. Continuously loops and performs the following tasks:
//     a. Checks if the worker has a Redis client and initializes it.
//     b. Retrieves the next job from the queue and checks if it is available.
//     c. If a job is available, retrieves the corresponding job from the key-value store and checks if it is complete.
//     d. If the job is not complete, sets the canWork flag to false and logs a message.
//     e. If a job is available and complete, sets the jobChan variable to the job.
//     f. Sends the job to the jobChan channel if it is available.
//     g. If the worker is running and not pending, sets the isPending flag to true and logs an "auto pending" message.
//     Returns from the routine after this.
//     h. Waits for one of the following events to occur:
//     - A job is received from the jobChan channel. Calls the work function with the received job.
//     - A signal is received on the quitChan channel. Logs a "stopping" message and returns from the routine.
//     - A signal is received on the pendChan channel. Logs a "pending" message and returns from the routine.
//     - No events occur. Logs a "selected default" message.
//     i. If the worker has a Redis client, closes the client.
//     j. Sleeps for the specified delay duration.
//
// No parameters.
// No return values.
func (w *JobWorker) routine() {
	log.Printf("start rountine(worker: %s):", w.Name)
	if w.logger != nil {
		w.logger.Infof("start routine(worker: %s):", w.Name)
	}

	for {
		canWork := true
		if w.redis != nil {
			w.redisClient = w.redis()
		}

		next := w.queue.Next()
		if next == nil {
			canWork = false
		}

		if canWork {
			key := fmt.Sprintf("%s.%s", w.Name, next.JobId)
			convJob, err := w.getJob(key)
			if err != nil {
				log.Println(err)
				if w.logger != nil {
					w.logger.Error(err)
				}
			}

			if convJob != nil && convJob.Status != SUCCESS {
				canWork = false
				log.Printf("%s worker job(%s) is not complete...", w.Name, convJob.JobId)
			}
		}

		var jobChan *Job = nil
		if canWork {
			j, err := w.queue.Dequeue()
			if err != nil {
				canWork = false
				log.Println(err)
			} else {
				jobChan = j
			}
		}

		if canWork {
			w.jobChan <- jobChan
		} else if w.isRunning && !w.isPending {
			w.isPending = true
			log.Printf("worker %s auto pending\n", w.Name)
			if w.logger != nil {
				w.logger.Infof("worker %s auto pending\n", w.Name)
			}
			return
		}

		select {
		case job := <-w.jobChan:
			if job != nil {
				w.work(job)
			}
		case <-w.quitChan:
			log.Printf("worker %s stopping\n", w.Name)
			if w.logger != nil {
				w.logger.Infof("worker %s stopping\n", w.Name)
			}
			return
		case <-w.pendChan:
			log.Printf("worker %s pending\n", w.Name)
			if w.logger != nil {
				w.logger.Infof("worker %s pending\n", w.Name)
			}
			return
		default:
			log.Printf("worker %s selected default\n", w.Name)
		}

		if w.redis != nil {
			err := w.redisClient.Close()
			if err != nil {
				log.Println(err)
				if w.logger != nil {
					w.logger.Error(err)
				}
			}
		}

		time.Sleep(w.delay)
	}
}

// pendingRoutine is a method of the JobWorker struct that sets the isPending flag to true and sends a true value to the pendChan channel.
//
// No parameters.
// No return values.
func (w *JobWorker) pendingRoutine() {
	w.isPending = true
	w.pendChan <- true
}

// quitRoutine sets the isRunning flag to false and sends a value to the quitChan channel.
//
// No parameters.
// No return types.
func (w *JobWorker) quitRoutine() {
	w.isRunning = false
	w.quitChan <- true
}

// GetName returns the name of the JobWorker.
//
// No parameters.
// Returns a string.
func (w *JobWorker) GetName() string {
	return w.Name
}

// Run starts the JobWorker and runs its routine if it's not already running.
//
// No parameters.
// No return values.
func (w *JobWorker) Run() {
	if w.isRunning {
		log.Printf("%s worker is running", w.Name)
		if w.logger != nil {
			w.logger.Infof("%s worker is running", w.Name)
		}
		return
	}

	w.isRunning = true

	go w.routine()
}

// Resume resumes the JobWorker.
//
// This function does not take any parameters.
// It does not return anything.
func (w *JobWorker) Resume() {
	if !w.isRunning {
		log.Printf("%s worker is not running", w.Name)
		if w.logger != nil {
			w.logger.Infof("%s worker is not running", w.Name)
		}
		return
	}

	w.isPending = false
	go w.routine()
}

// Pending runs the pending routine if the worker is running.
//
// No parameters.
// No return types.
func (w *JobWorker) Pending() {
	if !w.isRunning {
		log.Printf("%s worker is not running", w.Name)
		if w.logger != nil {
			w.logger.Infof("%s worker is not running", w.Name)
		}
		return
	}

	go w.pendingRoutine()
}

// Stop stops the JobWorker.
//
// It checks if the worker is running. If not, it logs a message and returns. Otherwise, it starts a new goroutine to quit the worker.
// If the worker is in a pending state, it resumes the worker.
// If a Redis client is available, it deletes the worker's name from Redis.
func (w *JobWorker) Stop() {
	if !w.isRunning {
		log.Printf("%s worker is not running", w.Name)
		if w.logger != nil {
			w.logger.Infof("%s worker is not running", w.Name)
		}
		return
	}

	go w.quitRoutine()

	if w.isPending {
		w.Resume()
	}

	if w.redis != nil {
		w.redisClient = w.redis()

		w.redisClient.Del(ctx, w.Name)
	}
}

// AddJob adds a job to the JobWorker.
//
// It takes a Job parameter and returns an error.
func (w *JobWorker) AddJob(job Job) error {
	if w.onAddJob != nil {
		err := w.handleAddJob(&job)
		if err != nil {
			log.Print(err)
			if w.logger != nil {
				w.logger.Error(err)
			}
			return err
		}
	}

	err := w.queue.Enqueue(job)
	if err != nil {
		return err
	}

	if w.isPending {
		w.Resume()
	}

	return nil
}

// IsRunning returns a boolean value indicating whether the JobWorker is currently running.
//
// No parameters.
// Returns a boolean value.
func (w *JobWorker) IsRunning() bool {
	return w.isRunning
}

// IsPending returns a boolean value indicating whether the JobWorker is in a pending state.
//
// This function does not take any parameters.
// It returns a bool value.
func (w *JobWorker) IsPending() bool {
	return w.isPending
}

// MaxJobCount returns the maximum job count for the JobWorker.
//
// It does not take any parameters.
// It returns an integer value.
func (w *JobWorker) MaxJobCount() int {
	return w.maxJobCount
}

// JobCount returns the number of jobs in the queue.
//
// No parameters.
// Returns an integer.
func (w *JobWorker) JobCount() int {
	return w.queue.Count()
}

// Queue returns the queue associated with the JobWorker.
//
// No parameters.
// Returns a Queue.
func (w *JobWorker) Queue() Queue {
	return w.queue
}

// OnAddJob registers a function to be called when a new job is added.
//
// The function fn takes a pointer to a Job and returns an error.
// The optional scope parameter(s) specify the scope(s) in which the function should be called.
func (w *JobWorker) OnAddJob(fn func(j *Job) error, scope ...string) {
	if len(scope) == 0 {
		w.onAddJob["."] = fn
		return
	}

	for _, s := range scope {
		w.onAddJob[s] = fn
	}
}

// handleAddJob handles the addition of a job in the JobWorker.
//
// It iterates over the onAddJob map and executes the corresponding function
// for the given job ID. If the execution of the function fails, it returns
// the error.
//
// Parameters:
// - j: a pointer to the Job struct representing the job to be added.
//
// Return:
// - error: an error indicating the failure of the function execution, if any.
func (w *JobWorker) handleAddJob(j *Job) error {
	isExec := false
	for jobId, fn := range w.onAddJob {
		if jobId == j.JobId {
			err := fn(j)
			if err != nil {
				return err
			}
			isExec = true
		}
	}

	if !isExec {
		if w.onAddJob["."] != nil {
			err := w.onAddJob["."](j)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// BeforeJob sets a function to be executed before a job is processed.
//
// The first parameter, fn, is a function that takes a *Job pointer as input and returns an error.
// The second parameter, scope, is an optional variadic parameter that specifies the scope(s) in which the before job function should be applied.
// The function does not return anything.
func (w *JobWorker) BeforeJob(fn func(j *Job) error, scope ...string) {
	if len(scope) == 0 {
		w.beforeJob["."] = fn
		return
	}

	for _, s := range scope {
		w.beforeJob[s] = fn
	}
}

// handleBeforeJob handles the before job actions for a given job worker.
//
// It iterates over the beforeJob map and executes the function associated with the job ID.
// If an error occurs during the execution, it returns the error.
// If no function is associated with the job ID, it checks for a function associated with the key ".".
// If found, it executes the function associated with the key ".".
// Returns nil if no error occurs.
func (w *JobWorker) handleBeforeJob(j *Job) error {
	isExec := false
	for jobId, fn := range w.beforeJob {
		if jobId == j.JobId {
			err := fn(j)
			if err != nil {
				return err
			}
			isExec = true
		}
	}

	if !isExec {
		if w.beforeJob["."] != nil {
			err := w.beforeJob["."](j)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// AfterJob sets a function to be executed after a job has finished.
//
// fn: The function to be executed after the job has finished. It takes a Job pointer
//
//	and an error pointer as parameters and returns an error.
//
// scope: Optional parameter(s) specifying the scope(s) where the function should be executed.
//
//	If no scope is provided, the function will be executed for all scopes.
func (w *JobWorker) AfterJob(fn func(j *Job, err error) error, scope ...string) {
	if len(scope) == 0 {
		w.afterJob["."] = fn
		return
	}

	for _, s := range scope {
		w.afterJob[s] = fn
	}
}

// handleAfterJob handles the job after it is completed.
//
// It takes in a job and an error as parameters.
// It returns an error.
func (w *JobWorker) handleAfterJob(j *Job, err error) error {
	isExec := false
	for jobId, fn := range w.afterJob {
		if jobId == j.JobId {
			err := fn(j, err)
			if err != nil {
				return err
			}
			isExec = true
		}
	}

	if !isExec {
		if w.afterJob["."] != nil {
			err := w.afterJob["."](j, err)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
