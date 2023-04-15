package goworker_test

import (
	"context"
	worker "github.com/miniyus/goworker"
	"github.com/redis/go-redis/v9"
	"log"
	"testing"
	"time"
)

// worker option for test
var opt = worker.Option{
	Name:        "default",       // 워커 이름
	MaxJobCount: 10,              // 워커에 담을 수 있는 작업 개수
	Delay:       time.Second * 3, // 작업 수행 후 다음 작업까지 딜레이 설정
}

// dispatcher option for test
var dispatcherOpt = worker.DispatcherOption{
	WorkerOptions: []worker.Option{
		opt,
	},
	// go-redis 세션이 끊어지는 이슈가 존재하여 현재는 redis 클라이언트를 생성해 줄 수 있는 함수로 받고 있다.
	Redis: func() *redis.Client {
		return redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		})
	},
}

var dispatcher = worker.NewDispatcher(dispatcherOpt)

var redisClient = dispatcherOpt.Redis()

func TestJobDispatcher(t *testing.T) {
	dispatcher.Run("default") // 입력된 워커만 실행
	dispatcher.Run()          // 입력된 값이 없을 경우 모든 워커가 실행된다.

	// Dispatch 메서드는 작업 id와 클로저를 받아 입력 받은 id로 작업을 생성하여, 클로저에 작성된 로직을 수행한다.
	err := dispatcher.Dispatch("t1", func(job *worker.Job) error {
		log.Printf("id %s status %s", job.JobId, job.Status)
		time.Sleep(time.Second)
		return nil
	})

	if err != nil {
		t.Error(err)
	}

	err = dispatcher.Dispatch("t2", func(job *worker.Job) error {
		log.Printf("id %s status %s", job.JobId, job.Status)
		time.Sleep(time.Second)
		return nil
	})

	if err != nil {
		t.Error(err)
	}

	loopCount := 0
	for {
		// Status 메서드는 현재 워커들의 현황을 확인 할 수 있다.
		stats := dispatcher.Status()
		stats.Print()
		completed := 0
		for _, w := range stats.Workers {
			if w.JobCount == 0 {
				log.Print("job count is zero")
				completed++
			}
		}

		if completed == stats.WorkerCount {
			log.Print("all workers is completed")
			break
		}

		time.Sleep(time.Second)

		loopCount++
		if loopCount >= 30 {
			break
		}
	}
	time.Sleep(time.Second)
}

func TestJobDispatcher_Hooks(t *testing.T) {
	// AfterJob 메서드는 작업이 종료된 후 부가적인 추가 작업을 등록하여 사용할 수 있다.
	dispatcher.AfterJob(func(j *worker.Job, err error) error {
		marshal, jErr := j.Marshal()
		if jErr != nil {
			return jErr
		}

		log.Printf("test after job %s %v", marshal, err)
		redisClient.LPush(context.Background(), j.WorkerName, marshal)
		return nil
	}, "default") // 특정 워커만 지정할 수 있다. 파라미터가 비어 있으면 모든 워커에 반영된다.

	// BeforeJob 메서드는 작업에 등록돈 클로저가 수행되기 전
	// 필요한 사전 작업을 등록할 수 있다.
	dispatcher.BeforeJob(func(j *worker.Job) error {
		marshal, err := j.Marshal()
		if err != nil {
			return err
		}

		log.Printf("test before job %s", marshal)
		redisClient.LPush(context.Background(), j.WorkerName, marshal)
		return nil
	}, "default.t4") // "." 구분자를 통해 특정 워커의 특정 작업에만 후킹 함수를 수행 시킬 수 있다.

	dispatcher.OnDispatch(func(j *worker.Job) error {
		marshal, err := j.Marshal()
		if err != nil {
			return err
		}
		log.Printf("test onDispatcher job %s", marshal)
		return nil
	}, "*.t3") // 모든 워커의 특정 작업에만 후킹 함수를 수행 시킬 수 있다.

	err := dispatcher.Dispatch("t3", func(job *worker.Job) error {
		log.Printf("id %s status %s", job.JobId, job.Status)
		job.Meta["TEST_1"] = 1
		time.Sleep(time.Second * 3)
		return nil
	})

	if err != nil {
		t.Error(err)
	}

	err = dispatcher.Dispatch("t4", func(job *worker.Job) error {
		log.Printf("id %s status %s", job.JobId, job.Status)
		job.Meta["TEST_2"] = 2
		time.Sleep(time.Second)
		return nil
	})

	if err != nil {
		t.Error(err)
	}

}

func TestJobDispatcher_AddWorker(t *testing.T) {
	dispatcher.AddWorker(worker.Option{
		Name:        "TEST_WORKER",
		MaxJobCount: 10,
		MaxPool:     10,
		// 워커를 기준으로 훅을 등록할 수 있다.
		// 지정된 훅은 워커의 모든 작업에 포함된다.
		BeforeJob: func(j *worker.Job) error {
			log.Print(j)
			return nil
		},
		Delay:  time.Second, // 작업을 끝낸 뒤 딜레이 시간
		Logger: nil,         // custom logger
	})

	if len(dispatcher.GetWorkers()) < 2 {
		t.Error("fail add worker")
	}
}

func TestJobDispatcher_RemoveWorker(t *testing.T) {
	dispatcher.RemoveWorker("TEST_WORKER")
	if len(dispatcher.GetWorkers()) >= 2 {
		t.Error("fail remove worker")
	}
}

func TestJob_UnMarshal(t *testing.T) {
	jsonStr := "{\"uuid\":\"dbb0353e-1474-4edb-bc58-49f64a82949b\",\"worker_name\":\"default\",\"job_id\":\"test\",\"status\":\"success\",\"created_at\":\"2023-02-04T11:35:02.793728793Z\",\"updated_at\":\"2023-02-04T11:35:05.799017503Z\"}"
	job := &worker.Job{}
	err := job.UnMarshal(jsonStr)
	if err != nil {
		t.Error(err)
	}
	log.Print(job)
}

func TestJobDispatcher_Stress(t *testing.T) {
	for i := 0; i < 100; i++ {
		err := dispatcher.Dispatch("1", func(j *worker.Job) error {
			log.Println(j)
			return nil
		})
		if err != nil {
			break
		}
	}
	stop := false
	for {
		if stop {
			break
		}
		workers := dispatcher.GetWorkers()
		for _, w := range workers {
			if w.GetName() == "default" && len(w.Queue().Jobs()) == 0 {
				stop = true
			} else {
				log.Println(w.Queue().Jobs())
			}
		}
		time.Sleep(time.Millisecond * 3)
	}

}

func TestJobDispatcher_Stop(t *testing.T) {
	dispatcher.Status().Print()
	time.Sleep(time.Second * 10)

	dispatcher.Status().Print()
	_ = dispatcher.Dispatch("STOP!", func(j *worker.Job) error {
		log.Println("STOP?")
		return nil
	})
	time.Sleep(time.Second * 10)

	dispatcher.Stop()
	for {
		isRun := dispatcher.GetWorkers()[0].IsRunning()
		if !isRun {
			break
		}
		time.Sleep(time.Second)
	}
	time.Sleep(time.Second * 3)
	log.Println("the end...")
}
