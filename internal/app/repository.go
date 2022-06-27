package forum

import (
	"DbGODZ/internal/app/models"
)

type Repository interface {
	Add(user models.User) error
	GetByNickAndEmail(nickname, email string) ([]models.User, error)
	GetByNick(nickname string) (models.User, error)
	GetUsersByForum(slug string, limit int, since string, desc bool) ([]models.User, error)
	Update(user models.User) (models.User, error)
	AddForum(forum models.Forum) (models.Forum, error)
	GetBySlugForum(slug string) (models.Forum, error)
	AddThreadForum(thread models.Thread) (models.Thread, error)
	UpdateThreadForum(newThread models.Thread) (models.Thread, error)
	GetThreadsForum(slug string, limit int, since string, desc bool) ([]models.Thread, error)
	CheckThreadExistsForum(slug string) (bool, error)
	GetThreadBySlugForum(slug string) (models.Thread, error)
	GetThreadByIDForum(id int) (models.Thread, error)
	GetThreadIDBySlugForum(slug string) (int, error)
	GetThreadSlugByIDForum(id int) (string, error)
	AddPostsForum(posts []models.Post, threadID int) ([]models.Post, error)
	GetPostsForum(postSlugOrId models.Thread, limit, since int, sort string, desc bool) ([]models.Post, error)
	GetPostForum(id int, related []string) (map[string]interface{}, error)
	UpdatePostForum(newPost models.Post) (models.Post, error)
	AddVoteForum(vote models.Vote) error
	UpdateVoteForum(vote models.Vote) error
	GetServiceStatusForum() (map[string]int, error)
	ClearDatabaseForum() error
}
