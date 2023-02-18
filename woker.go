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

type JobStatus string

var ctx = context.Background()

const (
	SUCCESS  JobStatus = "success"
	FAIL     JobStatus = "fail"
	WAIT     JobStatus = "wait"
	PROGRESS JobStatus = "progress"
)

const DefaultWorker = "default"

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

func (j *Job) Marshal() (string, error) {
	marshal, err := json.Marshal(j)
	if err != nil {
		return "", err
	}

	return string(marshal), nil
}

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
	}
}

type Queue interface {
	Enqueue(job Job) error
	Dequeue() (*Job, error)
	Clear()
	Count() int
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

type Worker interface {
	GetName() string
	Run()
	Stop()
	AddJob(job Job) error
	IsRunning() bool
	MaxJobCount() int
	JobCount() int
	BeforeJob(fn func(j *Job) error)
	AfterJob(fn func(j *Job, err error) error)
	OnAddJob(fn func(j *Job) error)
}

type JobWorker struct {
	Name        string
	queue       Queue
	jobChan     chan Job
	quitChan    chan bool
	redis       func() *redis.Client
	maxJobCount int
	isRunning   bool
	beforeJob   func(j *Job) error
	afterJob    func(j *Job, err error) error
	onAddJob    func(j *Job) error
	redisClient *redis.Client
	delay       time.Duration
	logger      Logger
}

type Config struct {
	Name        string
	Redis       func() *redis.Client
	MaxJobCount int
	BeforeJob   func(j *Job) error
	AfterJob    func(j *Job, err error) error
	Delay       time.Duration
	Logger      Logger
}

func NewWorker(cfg Config) Worker {
	return &JobWorker{
		Name:        cfg.Name,
		queue:       NewQueue(cfg.MaxJobCount),
		jobChan:     make(chan Job, cfg.MaxJobCount),
		quitChan:    make(chan bool),
		redis:       cfg.Redis,
		maxJobCount: cfg.MaxJobCount,
		beforeJob:   cfg.BeforeJob,
		afterJob:    cfg.AfterJob,
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

func (w *JobWorker) work(job *Job) {
	job.Status = PROGRESS

	log.Printf("worker %s, job %s", w.Name, job.JobId)
	if w.logger != nil {
		w.logger.Infof("job working... worker: %s, job: %s", w.Name, job.JobId)
	}

	if w.beforeJob != nil {
		bErr := w.beforeJob(job)
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

	if w.afterJob != nil {
		aErr := w.afterJob(job, err)
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

	time.Sleep(w.delay)
}

func (w *JobWorker) routine() {
	log.Printf("start rountine(worker: %s):", w.Name)
	if w.logger != nil {
		w.logger.Infof("start routine(worker: %s):", w.Name)
	}
	for {
		jobChan, err := w.queue.Dequeue()
		if err != nil {
			log.Print(err)
			continue
		}

		w.jobChan <- *jobChan
		select {
		case job := <-w.jobChan:
			w.redisClient = w.redis()
			key := fmt.Sprintf("%s.%s", w.Name, job.JobId)
			convJob, err := w.getJob(key)
			if err != nil {
				log.Println(err)
				if w.logger != nil {
					w.logger.Error(err)
				}

			}

			if convJob != nil && convJob.Status != SUCCESS {
				err = w.queue.Enqueue(job)
				if err != nil {
					log.Println(err)
					if w.logger != nil {
						w.logger.Error(err)
					}
				}
				continue
			}

			w.work(&job)
			w.saveJob(key, job)
		case <-w.quitChan:
			log.Printf("worker %s stopping\n", w.Name)
			if w.logger != nil {
				w.logger.Infof("worker %s stopping\n", w.Name)
			}

			return
		}
	}
}

func (w *JobWorker) quitRoutine() {
	w.quitChan <- true
	w.isRunning = false
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

func (w *JobWorker) Stop() {
	if !w.isRunning {
		log.Printf("%s worker is not running", w.Name)
		if w.logger != nil {
			w.logger.Infof("%s worker is not running", w.Name)
		}
		return
	}

	go w.quitRoutine()
}

func (w *JobWorker) AddJob(job Job) error {
	if w.onAddJob != nil {
		err := w.onAddJob(&job)
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

	return nil
}

func (w *JobWorker) IsRunning() bool {
	return w.isRunning
}

func (w *JobWorker) MaxJobCount() int {
	return w.maxJobCount
}

func (w *JobWorker) JobCount() int {
	return w.queue.Count()
}

func (w *JobWorker) OnAddJob(fn func(j *Job) error) {
	w.onAddJob = fn
}

func (w *JobWorker) BeforeJob(fn func(j *Job) error) {
	w.beforeJob = fn
}

func (w *JobWorker) AfterJob(fn func(j *Job, err error) error) {
	w.afterJob = fn
}
