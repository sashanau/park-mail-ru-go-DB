package delivery

import (
	"DbGODZ/internal/app"
	"DbGODZ/internal/app/models"
	"DbGODZ/internal/pkg/res"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx"
	"github.com/valyala/fasthttp"
	"strconv"
	"strings"
)

type handler struct {
	forumRepo forum.Repository
}

func NewHandler(fr forum.Repository) *handler {
	return &handler{forumRepo: fr}
}

func (f *handler) AddForum(ctx *fasthttp.RequestCtx) {
	var newForum models.Forum
	err := json.Unmarshal(ctx.PostBody(), &newForum)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	newForumDB, err := f.forumRepo.AddForum(newForum)
	if pgerr, ok := err.(pgx.PgError); ok {
		switch pgerr.Code {
		case "23505":
			forumObj, err := f.forumRepo.GetBySlugForum(newForum.Slug)
			if err != nil {
				res.SendServerError(err.Error(), ctx)
				return
			}
			res.SendResponse(409, forumObj, ctx)
			return
		case "23503":
			err := res.HttpError{
				Message: fmt.Sprintf("Can't find user with nickname: %s", newForum.User),
			}
			res.SendResponse(404, err, ctx)
			return
		}

	}
	if err == pgx.ErrNoRows {
		err := res.HttpError{
			Message: fmt.Sprintf("Can't find user with nickname: %s", newForum.User),
		}
		res.SendResponse(404, err, ctx)
		return
	}

	if err != nil {
		res.SendResponse(400, err.Error(), ctx)
		return
	}

	res.SendResponse(201, newForumDB, ctx)
}

func (f *handler) GetForum(ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug").(string)
	if !ok {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	forumObj, err := f.forumRepo.GetBySlugForum(slug)
	switch err {
	case pgx.ErrNoRows:
		err := res.HttpError{
			Message: fmt.Sprintf("Can't find forum with slug: %s", slug),
		}
		res.SendResponse(404, err, ctx)
		return
	case nil:
	default:
		res.SendServerError(err.Error(), ctx)
	}

	res.SendResponseOK(forumObj, ctx)
	return
}

func (f *handler) AddThreadForum(ctx *fasthttp.RequestCtx) {
	forumSlug, found := ctx.UserValue("slug").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	newThread := models.Thread{Forum: forumSlug}

	err := json.Unmarshal(ctx.PostBody(), &newThread)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	newThreadDB, err := f.forumRepo.AddThreadForum(newThread)
	if pgerr, ok := err.(pgx.PgError); ok && pgerr.Code == "23505" {
		threadOld, err := f.forumRepo.GetThreadBySlugForum(newThread.Slug.String)
		if err != nil {
			res.SendServerError(err.Error(), ctx)
			return
		}
		res.SendResponse(409, threadOld, ctx)
		return
	}

	if err != nil {
		errHttp := res.HttpError{Message: err.Error()}
		res.SendResponse(404, errHttp, ctx)
		return
	}

	res.SendResponse(201, newThreadDB, ctx)
}

func extractBoolValueForum(ctx *fasthttp.RequestCtx, valueName string) (bool, error) {
	ValueStr := string(ctx.QueryArgs().Peek(valueName))
	var value bool
	var err error

	if ValueStr == "" {
		return false, nil
	}
	value, err = strconv.ParseBool(ValueStr)
	if err != nil {
		return false, err
	}

	return value, nil
}

func extractIntValueForum(ctx *fasthttp.RequestCtx, valueName string) (int, error) {
	ValueStr := string(ctx.QueryArgs().Peek(valueName))
	var value int
	var err error

	if ValueStr == "" {
		return 0, nil
	}

	value, err = strconv.Atoi(ValueStr)
	if err != nil {
		return -1, err
	}

	return value, nil
}

func (f *handler) GetThreadsForum(ctx *fasthttp.RequestCtx) {
	forumSlug, found := ctx.UserValue("slug").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	limit, err := extractIntValueForum(ctx, "limit")
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	since := string(ctx.QueryArgs().Peek("since"))

	desc, err := extractBoolValueForum(ctx, "desc")
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}
	threads, err := f.forumRepo.GetThreadsForum(forumSlug, limit, since, desc)
	if err == pgx.ErrNoRows || len(threads) == 0 {
		exists, err := f.forumRepo.CheckThreadExistsForum(forumSlug)
		if err != nil {
			res.SendServerError(err.Error(), ctx)
			return
		}
		if exists {
			data := make([]models.Thread, 0, 0)
			res.SendResponseOK(data, ctx)
			return
		}
		errHTTP := res.HttpError{
			Message: fmt.Sprintf("Can't find forum by slug: %s", forumSlug),
		}
		res.SendResponse(404, errHTTP, ctx)
		return
	}

	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}
	res.SendResponseOK(threads, ctx)
	return
}

func (f *handler) createPostForum(ctx *fasthttp.RequestCtx, id int) {
	var newPosts []models.Post
	err := json.Unmarshal(ctx.PostBody(), &newPosts)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}
	if len(newPosts) == 0 {
		res.SendResponse(201, newPosts, ctx)
		return
	}
	newPostsAuthor := newPosts[0].Author
	newPosts, err = f.forumRepo.AddPostsForum(newPosts, id)
	if len(newPosts) == 0 {
		err = pgx.ErrNoRows
	}
	if err != nil {
		if pgerr, ok := err.(pgx.PgError); ok {
			switch pgerr.Code {
			case "00409":
				res.SendResponse(409, map[int]int{}, ctx)
				return
			}
		}

		if err == pgx.ErrNoRows {
			_, err = f.forumRepo.GetThreadByIDForum(id)
			if err == pgx.ErrNoRows {
				res.SendResponse(404, map[int]int{}, ctx)
				return
			}
			_, err = f.forumRepo.GetByNick(newPostsAuthor)
			if err == pgx.ErrNoRows {
				res.SendResponse(404, map[int]int{}, ctx)
				return
			}
			res.SendResponse(409, map[int]int{}, ctx)
			return
		}

		httpError := map[string]string{
			"message": err.Error(),
		}
		res.SendResponse(404, httpError, ctx)
		return
	}

	res.SendResponse(201, newPosts, ctx)
	return
}

func (f *handler) AddPostSlugForum(ctx *fasthttp.RequestCtx) {
	slugOrId, found := ctx.UserValue("slug_or_id").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}
	var id int
	id, err := strconv.Atoi(slugOrId)
	if err == nil {
		_, err = f.forumRepo.GetThreadByIDForum(id)
		if err != nil {
			errHTTP := res.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			res.SendResponse(404, errHTTP, ctx)

			return
		}
	} else {
		id, err = f.forumRepo.GetThreadIDBySlugForum(slugOrId)
		if err != nil {
			errHTTP := res.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			res.SendResponse(404, errHTTP, ctx)
			return
		}
	}

	f.createPostForum(ctx, id)
}

func (f *handler) AddVoteSlugForum(ctx *fasthttp.RequestCtx) {
	threadSlug, found := ctx.UserValue("slug").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	var newVote models.Vote
	err := json.Unmarshal(ctx.PostBody(), &newVote)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}
	threadID, _ := f.forumRepo.GetThreadIDBySlugForum(threadSlug)
	newVote.IdThread = int64(threadID)
	err = f.forumRepo.AddVoteForum(newVote)
	if err != nil {
		pgerr, ok := err.(pgx.PgError)
		if !ok {
			errHTTP := res.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			res.SendResponse(404, errHTTP, ctx)
			return
		}
		if pgerr.Code != "23505" {
			errHTTP := res.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			res.SendResponse(404, errHTTP, ctx)
			return
		}

	}

	updatedThread, err := f.forumRepo.GetThreadBySlugForum(threadSlug)
	if err != nil {
		errHTTP := res.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		res.SendResponse(404, errHTTP, ctx)
		return
	}

	res.SendResponseOK(updatedThread, ctx)
}

func (f *handler) AddVoteIDForum(ctx *fasthttp.RequestCtx) {
	ValueStr, found := ctx.UserValue("id").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	value, err := strconv.Atoi(ValueStr)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	var newVote models.Vote
	err = json.Unmarshal(ctx.PostBody(), &newVote)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}
	newVote.IdThread = int64(value)

	err = f.forumRepo.AddVoteForum(newVote)
	if err != nil {
		pgerr, ok := err.(pgx.PgError)
		if !ok {
			errHTTP := res.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			res.SendResponse(404, errHTTP, ctx)
			return
		}
		if pgerr.Code != "23505" {
			errHTTP := res.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			res.SendResponse(404, errHTTP, ctx)
			return
		} else {
			err = f.forumRepo.UpdateVoteForum(newVote)
			if err != nil {
				errHTTP := res.HttpError{
					Message: fmt.Sprintf(err.Error()),
				}
				res.SendResponse(404, errHTTP, ctx)
				return
			}
		}
	}
	updatedThread, err := f.forumRepo.GetThreadByIDForum(value)
	if err != nil {
		errHTTP := res.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		res.SendResponse(404, errHTTP, ctx)
		return
	}

	res.SendResponseOK(updatedThread, ctx)
}

func (f *handler) GetThreadDetailsSlugForum(ctx *fasthttp.RequestCtx) {
	threadSlug, found := ctx.UserValue("slug_or_id").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	id, err := strconv.Atoi(threadSlug)
	if err != nil {
		id, err = f.forumRepo.GetThreadIDBySlugForum(threadSlug)
		if err != nil {
			errHTTP := res.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			res.SendResponse(404, errHTTP, ctx)
			return
		}
	}

	forumObj, err := f.forumRepo.GetThreadByIDForum(id)
	if err != nil {
		errHTTP := res.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		res.SendResponse(404, errHTTP, ctx)
		return
	}

	res.SendResponseOK(forumObj, ctx)
	return
}

func (f *handler) UpdateThreadBySlugOrIDForum(ctx *fasthttp.RequestCtx) {
	threadSlugOrID, found := ctx.UserValue("slug_or_id").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}
	var newThread models.Thread
	if id, err := strconv.Atoi(threadSlugOrID); err == nil {
		newThread.Id = int32(id)
	} else {
		newThread.Slug = models.JsonNullString{
			NullString: sql.NullString{Valid: true, String: threadSlugOrID},
		}
	}

	err := json.Unmarshal(ctx.PostBody(), &newThread)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	thread, err := f.forumRepo.UpdateThreadForum(newThread)
	if err != nil {
		res.SendResponse(404, err, ctx)
		return
	}

	res.SendResponseOK(thread, ctx)
	return
}

func (f *handler) GetPostsSlugForum(ctx *fasthttp.RequestCtx) {
	threadSlugOrID, found := ctx.UserValue("slug_or_id").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	limit, err := extractIntValueForum(ctx, "limit")
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	since, err := extractIntValueForum(ctx, "since")
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	sortType := string(ctx.QueryArgs().Peek("sort"))
	if sortType == "" {
		sortType = "flat"
	}

	desc, err := extractBoolValueForum(ctx, "desc")
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}
	var slugOrID models.Thread
	if id, err := strconv.Atoi(threadSlugOrID); err == nil {
		slugOrID.Id = int32(id)
	}
	slug := sql.NullString{String: threadSlugOrID, Valid: true}
	slugJSON := models.JsonNullString{NullString: slug}
	slugOrID.Slug = slugJSON

	posts, err := f.forumRepo.GetPostsForum(slugOrID, limit, since, sortType, desc)
	if err != nil {
		errHTTP := res.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		res.SendResponse(404, errHTTP, ctx)
		return
	}

	if posts == nil {
		if slugOrID.Id != 0 {
			_, err := f.forumRepo.GetThreadByIDForum(int(slugOrID.Id))
			if err == pgx.ErrNoRows {
				httpErr := res.HttpError{Message: err.Error()}
				res.SendResponse(404, httpErr, ctx)
				return
			}
		}
		res.SendResponseOK([]int{}, ctx)
		return
	}

	res.SendResponseOK(posts, ctx)
	return
}

func (f *handler) GetPostByIDForum(ctx *fasthttp.RequestCtx) {
	ValueStr, found := ctx.UserValue("id").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	id, err := strconv.Atoi(ValueStr)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	related := string(ctx.QueryArgs().Peek("related"))

	post, err := f.forumRepo.GetPostForum(id, strings.Split(related, ","))
	if err != nil {
		httpErr := res.HttpError{Message: err.Error()}
		res.SendResponse(404, httpErr, ctx)
		return
	}

	res.SendResponseOK(post, ctx)
	return
}

func (f *handler) UpdatePostForum(ctx *fasthttp.RequestCtx) {
	ValueStr, found := ctx.UserValue("id").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	id, err := strconv.Atoi(ValueStr)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	newPost := models.Post{
		Id: int64(id),
	}

	err = json.Unmarshal(ctx.PostBody(), &newPost)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	newPost, err = f.forumRepo.UpdatePostForum(newPost)
	if err != nil {
		httpErr := res.HttpError{Message: err.Error()}
		res.SendResponse(404, httpErr, ctx)
		return
	}

	res.SendResponseOK(newPost, ctx)
	return
}

func (f *handler) GetServiceStatusForum(ctx *fasthttp.RequestCtx) {
	info, err := f.forumRepo.GetServiceStatusForum()
	if err != nil {
		res.SendResponse(404, err.Error(), ctx)
		return
	}
	res.SendResponseOK(info, ctx)
	return
}

func (f *handler) ClearDataBaseForum(ctx *fasthttp.RequestCtx) {
	err := f.forumRepo.ClearDatabaseForum()
	if err != nil {
		res.SendResponse(404, err.Error(), ctx)
		return
	}
	res.SendResponseOK("", ctx)
	return
}

func (f *handler) Add(ctx *fasthttp.RequestCtx) {
	nickname, ok := ctx.UserValue("nickname").(string)
	if !ok {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	newUser := models.User{Nickname: nickname}

	err := json.Unmarshal(ctx.PostBody(), &newUser)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	err = f.forumRepo.Add(newUser)

	if err != nil {
		users, err := f.forumRepo.GetByNickAndEmail(newUser.Nickname, newUser.Email)
		if err != nil {
			res.SendServerError(err.Error(), ctx)
		}
		res.SendResponse(409, users, ctx)
		return
	}

	res.SendResponse(201, newUser, ctx)
	return
}

func (f *handler) Get(ctx *fasthttp.RequestCtx) {
	nickname, found := ctx.UserValue("nickname").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	userObj, err := f.forumRepo.GetByNick(nickname)
	if err != nil {
		err := res.HttpError{
			Message: fmt.Sprintf("Can't find user by nickname: %s", nickname),
		}
		res.SendResponse(404, err, ctx)
		return
	}

	res.SendResponseOK(userObj, ctx)
	return
}

func (f *handler) Update(ctx *fasthttp.RequestCtx) {
	nickname, found := ctx.UserValue("nickname").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	newUser := models.User{Nickname: nickname}

	err := json.Unmarshal(ctx.PostBody(), &newUser)
	if err != nil {
		res.SendServerError(err.Error(), ctx)
		return
	}

	userDB, err := f.forumRepo.Update(newUser)
	if pgerr, ok := err.(pgx.PgError); ok {
		switch pgerr.Code {
		case "23505":
			err := res.HttpError{
				Message: fmt.Sprintf("This email is already registered by user: %s", newUser.Email),
			}
			res.SendResponse(409, err, ctx)
			return
		}
	}
	if err != nil {
		err := res.HttpError{
			Message: fmt.Sprintf("Can't find user by nickname: %s", newUser.Nickname),
		}
		res.SendResponse(404, err, ctx)
		return
	}

	res.SendResponseOK(userDB, ctx)
	return
}

func extractBoolValue(ctx *fasthttp.RequestCtx, valueName string) (bool, error) {
	ValueStr := string(ctx.QueryArgs().Peek(valueName))
	var value bool
	var err error

	if ValueStr == "" {
		return false, nil
	}
	value, err = strconv.ParseBool(ValueStr)
	if err != nil {
		return false, err
	}

	return value, nil
}

func extractIntValue(ctx *fasthttp.RequestCtx, valueName string) (int, error) {
	ValueStr := string(ctx.QueryArgs().Peek(valueName))
	var value int
	var err error

	if ValueStr == "" {
		return 0, nil
	}

	value, err = strconv.Atoi(ValueStr)
	if err != nil {
		return -1, err
	}

	return value, nil
}

func (f *handler) GetByForum(ctx *fasthttp.RequestCtx) {
	slug, found := ctx.UserValue("slug").(string)
	if !found {
		res.SendResponse(400, "bad request", ctx)
		return
	}

	limit, err := extractIntValue(ctx, "limit")
	if err != nil {
		res.SendResponse(400, err, ctx)
		return
	}

	since := string(ctx.QueryArgs().Peek("since"))

	desc, err := extractBoolValue(ctx, "desc")
	if err != nil {
		res.SendResponse(400, err, ctx)
		return
	}

	users, err := f.forumRepo.GetUsersByForum(slug, limit, since, desc)
	if err != nil {
		res.SendResponse(404, err, ctx)
		return
	}

	if users == nil {
		_, err = f.forumRepo.GetBySlugForum(slug)
		if err != nil {
			res.SendResponse(404, err, ctx)
			return
		}
		res.SendResponseOK([]models.User{}, ctx)
		return
	}

	res.SendResponseOK(users, ctx)
	return
}
