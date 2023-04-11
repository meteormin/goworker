# Golang Job Queue Worker With Redis

## 주요 기능

1. 비동기 작업 큐
2. Redis 연동을 통한 작업 완료, 실패, 대기 여부 파악 및 동시성 제어
3. 여러개의 워커를 이용한 병렬 처리
4. 작업 전후 Hooking 제공

### 1. 비동기 작업 큐

- go channel과 routine을 활용하여 비동기 작업 큐 구현
- 최대 큐 사이즈 설정 가능
- worker 내부에서 비동기로 작업 루틴을 실행
  - JobId가 겹치지 않을 경우 비동기로 실행이 가능
  - worker 설정에 maxPool 설정이 worker내부에서 실행 가능한 비동기 풀 개수

### 2. Redis 연동을 통한 작업 완료, 실패, 대기 여부 파악

- Redis와 연동하여 작업 상태 + 작업 ID를 가지고 동시성 제어 가능
- Job ID를 겹치게 하여 이미 같은 작업 ID를 가진 작업이 수행 중일 경우 작업을 수행하지 않고 대기합니다.

### 3. 여러개의 워커를 이용한 병렬 처리

- 여러개의 워커 생성하여 병렬 처리가 가능합니다.
- 워커 내부에서 또한 JobId가 겹치지 않는 경우 비동기 수행으로 인한 병렬 처리가 가능합니다.
- 워커가 다를 경우 JobId가 겹치더라도 작업을 수행 할 수 있습니다.

### 4. 작업 전, 후 Hooking 제공

- OnDispatch() 메서드를 제공하여 작업이 dispatch 되었을 때 수행할 로직을 설정할 수 있습니다.
  - OnDispatch() 메서드는 Job 구조체의 Status 필드가 "wait"일 때 수행됩니다.
- BeforeJob() 메서드를 제공하여 작업 전에 수행할 로직을 설정할 수 있습니다.
  - BeforeJob() 메서드는 Job 구조체의 Status 필드가 "progress"일 때 수행됩니다.
- AfterJob() 메서드를 제공하여 작업 후에 수행할 로직을 설정할 수 있습니다.
  - AfterJob() 메서드는 Job 구조체의 Status 필드가 "success" or "fail"일 떄 수행됩니다.
- ex) 워커 당 작업 수행 히스토리 DB 저장

## 사용법

참고: [worker_test.go](./worker_test.go)
