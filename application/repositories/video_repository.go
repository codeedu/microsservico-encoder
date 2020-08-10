package repositories

import (
	"encoder/domain"
	"fmt"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type VideoRepository interface {
	Insert(video *domain.Video) (*domain.Video, error)
	Find(id string) (*domain.Video, error)
}

type VideoRepositoryDb struct {
	Db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) *VideoRepositoryDb {
	return &VideoRepositoryDb{Db: db}
}

func (repo VideoRepositoryDb) Insert(video *domain.Video) (*domain.Video, error) {

	if video.ID == "" {
		video.ID = uuid.NewV4().String()
	}

	err := repo.Db.Create(video).Error

	if err != nil {
		return nil, err
	}

	return video, nil

}

func (repo VideoRepositoryDb) Find(id string) (*domain.Video, error) {

	var video domain.Video
	repo.Db.Preload("Jobs").First(&video, "id = ?", id)

	if video.ID == "" {
		return nil, fmt.Errorf("video does not exist")
	}

	return &video, nil

}
