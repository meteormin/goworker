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

// JobStatus job's status
// success, wait, fail, progress
type JobStatus string

const (
	SUCCESS  JobStatus = "success"
	FAIL     JobStatus = "fail"
	WAIT     JobStatus = "wait"
	PROGRESS JobStatus = "progress"
)

// Redis context
var ctx = context.Background()

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

type Queue interface {
	Enqueue(job Job) error
	Dequeue() (*Job, error)
	Clear()
	Count() int
	Next() *Job
	Jobs() []Job
}

type JobQueue struct {
	queue       []Job
	jobChan     chan Job
	maxJobCount int
}

func NewQueue(maxJobCount int) Queue {
	return &JobQueue{
		queue:       make([]Job, 0),
		jobChan:     make(chan Job, maxJobCount),
		maxJobCount: maxJobCount,
	}
}

func (q *JobQueue) Enqueue(job Job) error {
	if q.maxJobCount > len(q.queue) {
		q.queue = append(q.queue, job)
		q.jobChan <- job
		return nil
	}

	return errors.New("can't enqueue job queue: over queue size")
}

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

func (q *JobQueue) Clear() {
	q.queue = make([]Job, 0)
	q.jobChan = make(chan Job)
}

func (q *JobQueue) Count() int {
	return len(q.queue)
}

func (q *JobQueue) Next() *Job {
	if len(q.queue) == 0 {
		return nil
	}

	return &q.queue[0]
}

func (q *JobQueue) Jobs() []Job {
	return q.queue
}

type BeforeJob = func(j *Job) error
type AfterJob = func(j *Job, err error) error
type OnAddJob = func(j *Job) error

type Worker interface {
	GetName() string
	Run()
	Resume()
	Pending()
	Stop()
	AddJob(job Job) error
	IsRunning() bool
	IsPending() bool
	MaxJobCount() int
	JobCount() int
	Queue() Queue
	BeforeJob(fn func(j *Job) error, scope ...string)
	AfterJob(fn func(j *Job, err error) error, scope ...string)
	OnAddJob(fn func(j *Job) error, scope ...string)
}

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

func NewWorker(cfg Config) Worker {
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
		redisClient: cfg.Redis(),
		delay:       cfg.Delay,
		logger:      cfg.Logger,
	}
}

func (w *JobWorker) saveJob(key string, job Job) {
	jsonJob, err := job.Marshal()
	if err != nil {
		panic(err)
	}

	err = w.redisClient.Set(ctx, key, jsonJob, time.Minute).Err()
	if err != nil {
		panic(err)
	}
}

func (w *JobWorker) getJob(key string) (*Job, error) {
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

func (w *JobWorker) delJob(key string) {
	_, err := w.redisClient.Del(ctx, key).Result()
	if err != nil {
		panic(err)
	}
}

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

func (w *JobWorker) routine() {
	log.Printf("start rountine(worker: %s):", w.Name)
	if w.logger != nil {
		w.logger.Infof("start routine(worker: %s):", w.Name)
	}

	for {
		canWork := true
		w.redisClient = w.redis()

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
				log.Print(err)
			}

			jobChan = j
		}

		if next == nil && w.isRunning && !w.isPending {
			w.Pending()
		} else if w.isRunning && !w.isPending {
			w.jobChan <- jobChan
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
		}
		time.Sleep(w.delay)
	}
}

func (w *JobWorker) resumeRoutine() {
	w.isPending = false
}

func (w *JobWorker) pendingRoutine() {
	w.isPending = true
	w.pendChan <- true
}

func (w *JobWorker) quitRoutine() {
	w.isRunning = false
	w.quitChan <- true
}

func (w *JobWorker) GetName() string {
	return w.Name
}

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

func (w *JobWorker) Resume() {
	if !w.isRunning {
		log.Printf("%s worker is not running", w.Name)
		if w.logger != nil {
			w.logger.Infof("%s worker is not running", w.Name)
		}
		return
	}

	go w.resumeRoutine()
	go w.routine()
}

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
	w.redisClient = w.redis()

	w.redisClient.Del(ctx, w.Name)
}

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

func (w *JobWorker) IsRunning() bool {
	return w.isRunning
}

func (w *JobWorker) IsPending() bool {
	return w.isPending
}

func (w *JobWorker) MaxJobCount() int {
	return w.maxJobCount
}

func (w *JobWorker) JobCount() int {
	return w.queue.Count()
}

func (w *JobWorker) Queue() Queue {
	return w.queue
}

func (w *JobWorker) OnAddJob(fn func(j *Job) error, scope ...string) {
	if len(scope) == 0 {
		w.onAddJob["."] = fn
		return
	}

	for _, s := range scope {
		w.onAddJob[s] = fn
	}
}

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

func (w *JobWorker) BeforeJob(fn func(j *Job) error, scope ...string) {
	if len(scope) == 0 {
		w.beforeJob["."] = fn
		return
	}

	for _, s := range scope {
		w.beforeJob[s] = fn
	}
}

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

func (w *JobWorker) AfterJob(fn func(j *Job, err error) error, scope ...string) {
	if len(scope) == 0 {
		w.afterJob["."] = fn
		return
	}

	for _, s := range scope {
		w.afterJob[s] = fn
	}
}

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
