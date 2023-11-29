package goworker

import (
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"reflect"
	"testing"
	"time"
)

func TestJobQueue_Clear(t *testing.T) {
	type fields struct {
		queue       []Job
		jobChan     chan Job
		maxJobCount int
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				queue:       []Job{{UUID: uuid.New()}, {UUID: uuid.New()}},
				jobChan:     make(chan Job),
				maxJobCount: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &JobQueue{
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				maxJobCount: tt.fields.maxJobCount,
			}
			q.Clear()

			if len(q.queue) != 0 {
				t.Errorf("Clear() = %v, want %v", len(q.queue), 0)
			}
		})
	}
}

func TestJobQueue_Count(t *testing.T) {
	type fields struct {
		queue       []Job
		jobChan     chan Job
		maxJobCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "default",
			fields: fields{
				queue:       []Job{{UUID: uuid.New()}, {UUID: uuid.New()}},
				jobChan:     make(chan Job),
				maxJobCount: 10,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &JobQueue{
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				maxJobCount: tt.fields.maxJobCount,
			}
			if got := q.Count(); got != tt.want {
				t.Errorf("Count() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobQueue_Dequeue(t *testing.T) {
	type fields struct {
		queue       []Job
		jobChan     chan Job
		maxJobCount int
	}
	tests := []struct {
		name    string
		fields  fields
		want    *Job
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				queue:       []Job{{UUID: uuid.New()}, {UUID: uuid.New()}},
				jobChan:     make(chan Job),
				maxJobCount: 10,
			},
			want:    &Job{UUID: uuid.New()},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &JobQueue{
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				maxJobCount: tt.fields.maxJobCount,
			}
			got, err := q.Dequeue()
			if (err != nil) != tt.wantErr {
				t.Errorf("Dequeue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Dequeue() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobQueue_Enqueue(t *testing.T) {
	type fields struct {
		queue       []Job
		jobChan     chan Job
		maxJobCount int
	}
	type args struct {
		job Job
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				queue:       []Job{{UUID: uuid.New()}, {UUID: uuid.New()}},
				jobChan:     make(chan Job),
				maxJobCount: 10,
			},
			args:    args{job: Job{UUID: uuid.New()}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &JobQueue{
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				maxJobCount: tt.fields.maxJobCount,
			}
			if err := q.Enqueue(tt.args.job); (err != nil) != tt.wantErr {
				t.Errorf("Enqueue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobQueue_Jobs(t *testing.T) {
	type fields struct {
		queue       []Job
		jobChan     chan Job
		maxJobCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   []Job
	}{
		{
			name: "default",
			fields: fields{
				queue:       []Job{{UUID: uuid.New()}, {UUID: uuid.New()}},
				jobChan:     make(chan Job),
				maxJobCount: 10,
			},
			want: []Job{{UUID: uuid.New()}, {UUID: uuid.New()}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &JobQueue{
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				maxJobCount: tt.fields.maxJobCount,
			}
			if got := q.Jobs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Jobs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobQueue_Next(t *testing.T) {
	type fields struct {
		queue       []Job
		jobChan     chan Job
		maxJobCount int
	}
	tests := []struct {
		name   string
		fields fields
		want   *Job
	}{
		{
			name: "default",
			fields: fields{
				queue:       []Job{{UUID: uuid.New()}, {UUID: uuid.New()}},
				jobChan:     make(chan Job),
				maxJobCount: 10,
			},
			want: &Job{UUID: uuid.New()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &JobQueue{
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				maxJobCount: tt.fields.maxJobCount,
			}
			if got := q.Next(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobWorker_AddJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		job Job
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			args:    args{job: Job{UUID: uuid.New()}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if err := w.AddJob(tt.args.job); (err != nil) != tt.wantErr {
				t.Errorf("AddJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobWorker_AfterJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		fn    func(j *Job, err error) error
		scope []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			args: args{
				fn: func(j *Job, err error) error {
					if err != nil {
						return err
					}
					t.Log("after job")
					return nil
				},
				scope: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.AfterJob(tt.args.fn, tt.args.scope...)
		})
	}
}

func TestJobWorker_BeforeJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		fn    func(j *Job) error
		scope []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			args: args{
				fn: func(j *Job) error {
					t.Log("before job")
					return nil
				},
				scope: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.BeforeJob(tt.args.fn, tt.args.scope...)
		})
	}
}

func TestJobWorker_GetName(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			want: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if got := w.GetName(); got != tt.want {
				t.Errorf("GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobWorker_IsPending(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if got := w.IsPending(); got != tt.want {
				t.Errorf("IsPending() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobWorker_IsRunning(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if got := w.IsRunning(); got != tt.want {
				t.Errorf("IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobWorker_JobCount(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if got := w.JobCount(); got != tt.want {
				t.Errorf("JobCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobWorker_MaxJobCount(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			want: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if got := w.MaxJobCount(); got != tt.want {
				t.Errorf("MaxJobCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobWorker_OnAddJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		fn    func(j *Job) error
		scope []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			args: args{
				fn: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.OnAddJob(tt.args.fn, tt.args.scope...)
		})
	}
}

func TestJobWorker_Pending(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.Pending()
		})
	}
}

func TestJobWorker_Queue(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
		want   Queue
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			want: NewQueue(10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if got := w.Queue(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Queue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobWorker_Resume(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.Resume()
		})
	}
}

func TestJobWorker_Run(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.Run()
		})
	}
}

func TestJobWorker_Stop(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.Stop()
		})
	}
}

func TestJobWorker_delJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			args: args{
				key: "key",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.delJob(tt.args.key)
		})
	}
}

func TestJobWorker_getJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Job
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			args: args{
				key: "key",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			got, err := w.getJob(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getJob() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobWorker_handleAddJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		j *Job
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "default",
			fields: fields{
				queue:       NewQueue(10),
				jobChan:     make(chan *Job),
				pendChan:    make(chan bool),
				quitChan:    make(chan bool),
				redis:       func() *redis.Client { return nil },
				maxJobCount: 10,
				isRunning:   false,
				isPending:   false,
				beforeJob:   map[string]BeforeJob{".": nil},
				afterJob:    map[string]AfterJob{".": nil},
				onAddJob:    map[string]OnAddJob{".": nil},
				redisClient: nil,
				delay:       0,
				logger:      nil,
			},
			args: args{
				j: &Job{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if err := w.handleAddJob(tt.args.j); (err != nil) != tt.wantErr {
				t.Errorf("handleAddJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobWorker_handleAfterJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		j   *Job
		err error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if err := w.handleAfterJob(tt.args.j, tt.args.err); (err != nil) != tt.wantErr {
				t.Errorf("handleAfterJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobWorker_handleBeforeJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		j *Job
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			if err := w.handleBeforeJob(tt.args.j); (err != nil) != tt.wantErr {
				t.Errorf("handleBeforeJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobWorker_pendingRoutine(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.pendingRoutine()
		})
	}
}

func TestJobWorker_quitRoutine(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.quitRoutine()
		})
	}
}

func TestJobWorker_routine(t *testing.T) {
	type fields struct {
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
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.routine()
		})
	}
}

func TestJobWorker_saveJob(t *testing.T) {
	type fields struct {
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
	type args struct {
		key string
		job Job
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.saveJob(tt.args.key, tt.args.job)
		})
	}
}

func TestJobWorker_work(t *testing.T) {
	type fields struct {
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
	type args struct {
		job *Job
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &JobWorker{
				Name:        tt.fields.Name,
				queue:       tt.fields.queue,
				jobChan:     tt.fields.jobChan,
				pendChan:    tt.fields.pendChan,
				quitChan:    tt.fields.quitChan,
				redis:       tt.fields.redis,
				maxJobCount: tt.fields.maxJobCount,
				isRunning:   tt.fields.isRunning,
				isPending:   tt.fields.isPending,
				beforeJob:   tt.fields.beforeJob,
				afterJob:    tt.fields.afterJob,
				onAddJob:    tt.fields.onAddJob,
				redisClient: tt.fields.redisClient,
				delay:       tt.fields.delay,
				logger:      tt.fields.logger,
			}
			w.work(tt.args.job)
		})
	}
}

func TestJob_Marshal(t *testing.T) {
	type fields struct {
		UUID       uuid.UUID
		WorkerName string
		JobId      string
		Status     JobStatus
		Closure    func(job *Job) error
		CreatedAt  time.Time
		UpdatedAt  time.Time
		Meta       map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Job{
				UUID:       tt.fields.UUID,
				WorkerName: tt.fields.WorkerName,
				JobId:      tt.fields.JobId,
				Status:     tt.fields.Status,
				Closure:    tt.fields.Closure,
				CreatedAt:  tt.fields.CreatedAt,
				UpdatedAt:  tt.fields.UpdatedAt,
				Meta:       tt.fields.Meta,
			}
			got, err := j.Marshal()
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Marshal() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJob_UnMarshal(t *testing.T) {
	type fields struct {
		UUID       uuid.UUID
		WorkerName string
		JobId      string
		Status     JobStatus
		Closure    func(job *Job) error
		CreatedAt  time.Time
		UpdatedAt  time.Time
		Meta       map[string]interface{}
	}
	type args struct {
		jsonStr string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Job{
				UUID:       tt.fields.UUID,
				WorkerName: tt.fields.WorkerName,
				JobId:      tt.fields.JobId,
				Status:     tt.fields.Status,
				Closure:    tt.fields.Closure,
				CreatedAt:  tt.fields.CreatedAt,
				UpdatedAt:  tt.fields.UpdatedAt,
				Meta:       tt.fields.Meta,
			}
			if err := j.UnMarshal(tt.args.jsonStr); (err != nil) != tt.wantErr {
				t.Errorf("UnMarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewQueue(t *testing.T) {
	type args struct {
		maxJobCount int
	}
	tests := []struct {
		name string
		args args
		want Queue
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewQueue(tt.args.maxJobCount); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewQueue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewWorker(t *testing.T) {
	type args struct {
		cfg Config
	}
	tests := []struct {
		name string
		args args
		want Worker
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWorker(tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newJob(t *testing.T) {
	type args struct {
		workerName string
		jobId      string
		closure    func(job *Job) error
	}
	tests := []struct {
		name string
		args args
		want Job
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newJob(tt.args.workerName, tt.args.jobId, tt.args.closure); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newJob() = %v, want %v", got, tt.want)
			}
		})
	}
}
