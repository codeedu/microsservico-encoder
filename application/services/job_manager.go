package services

import (
	"encoder/application/repositories"
	"encoder/domain"
	"encoder/framework/queue"
	"encoding/json"
	"github.com/jinzhu/gorm"
	"github.com/streadway/amqp"
	"log"
	"os"
	"strconv"
)

type JobManager struct {
	Db               *gorm.DB
	Domain           domain.Job
	MessageChannel   chan amqp.Delivery
	JobReturnChannel chan JobWorkerResult
	RabbitMQ         *queue.RabbitMQ
}

type JobNotificationError struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func NewJobManager(db *gorm.DB, rabbitMQ *queue.RabbitMQ, jobReturnChannel chan JobWorkerResult, messageChannel chan amqp.Delivery) *JobManager {
	return &JobManager{
		Db:               db,
		Domain:           domain.Job{},
		MessageChannel:   messageChannel,
		JobReturnChannel: jobReturnChannel,
		RabbitMQ:         rabbitMQ,
	}
}

func (j *JobManager) Start(ch *amqp.Channel) {

	videoService := NewVideoService()
	videoService.VideoRepository = repositories.VideoRepositoryDb{Db: j.Db}

	jobService := JobService{
		JobRepository: repositories.JobRepositoryDb{Db: j.Db},
		VideoService:  videoService,
	}

	concurrency, err := strconv.Atoi(os.Getenv("CONCURRENCY_WORKERS"))

	if err != nil {
		log.Fatalf("error loading var: CONCURRENCY_WORKERS.")
	}

	for qtdProcesses := 0; qtdProcesses < concurrency; qtdProcesses++ {
		go JobWorker(j.MessageChannel, j.JobReturnChannel, jobService, j.Domain, qtdProcesses)
	}

	for jobResult := range j.JobReturnChannel {
		if jobResult.Error != nil {
			err = j.checkParseErrors(jobResult)
		} else {
			err = j.notifySuccess(jobResult, ch)
		}

		if err != nil {
			jobResult.Message.Reject(false)
		}
	}
}

func (j *JobManager) notifySuccess(jobResult JobWorkerResult, ch *amqp.Channel) error {

	Mutex.Lock()
	jobJson, err := json.Marshal(jobResult.Job)
	Mutex.Unlock()

	if err != nil {
		return err
	}

	err = j.notify(jobJson)

	if err != nil {
		return err
	}

	err = jobResult.Message.Ack(false)

	if err != nil {
		return err
	}

	return nil
}

func (j *JobManager) checkParseErrors(jobResult JobWorkerResult) error {
	if jobResult.Job.ID != "" {
		log.Printf("MessageID: %v. Error during the job: %v with video: %v. Error: %v",
			jobResult.Message.DeliveryTag, jobResult.Job.ID, jobResult.Job.Video.ID, jobResult.Error.Error())
	} else {
		log.Printf("MessageID: %v. Error parsing message: %v", jobResult.Message.DeliveryTag, jobResult.Error)
	}

	errorMsg := JobNotificationError{
		Message: string(jobResult.Message.Body),
		Error:   jobResult.Error.Error(),
	}

	jobJson, err := json.Marshal(errorMsg)

	err = j.notify(jobJson)

	if err != nil {
		return err
	}

	err = jobResult.Message.Reject(false)

	if err != nil {
		return err
	}

	return nil
}

func (j *JobManager) notify(jobJson []byte) error {

	err := j.RabbitMQ.Notify(
		string(jobJson),
		"application/json",
		os.Getenv("RABBITMQ_NOTIFICATION_EX"),
		os.Getenv("RABBITMQ_NOTIFICATION_ROUTING_KEY"),
	)

	if err != nil {
		return err
	}

	return nil
}
