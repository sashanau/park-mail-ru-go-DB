package main

import (
	_Handlers "DbGODZ/internal/app/delivery"
	_Repo "DbGODZ/internal/app/repository"
	"github.com/fasthttp/router"
	"github.com/jackc/pgx"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func main() {
	pgxConn, err := pgx.ParseConnectionString("user=api password=password dbname=api sslmode=disable port=5432")
	if err != nil {
		log.Error().Msgf(err.Error())
		return
	}

	// CONFIG DB
	config := pgx.ConnPoolConfig{
		ConnConfig:     pgxConn,
		MaxConnections: 100,
		AfterConnect:   nil,
		AcquireTimeout: 0,
	}
	connPool, err := pgx.NewConnPool(config)
	if err != nil {
		log.Error().Msgf(err.Error())
	}
	forumRepo := _Repo.NewPostgresForumRepository(connPool)
	forumHandler := _Handlers.NewHandler(forumRepo)

	r := router.New()
	r.POST("/api/user/{nickname}/create", forumHandler.Add)
	r.GET("/api/user/{nickname}/profile", forumHandler.Get)
	r.POST("/api/user/{nickname}/profile", forumHandler.Update)
	r.GET("/api/forum/{slug}/users", forumHandler.GetByForum)
	r.POST("/api/forum/create", forumHandler.AddForum)
	r.GET("/api/forum/{slug}/details", forumHandler.GetForum)
	r.POST("/api/forum/{slug}/create", forumHandler.AddThreadForum)
	r.GET("/api/forum/{slug}/threads", forumHandler.GetThreadsForum)
	r.GET("/api/thread/{slug_or_id}/details", forumHandler.GetThreadDetailsSlugForum)
	r.POST("/api/thread/{slug_or_id}/details", forumHandler.UpdateThreadBySlugOrIDForum)
	r.POST("/api/thread/{slug_or_id}/create", forumHandler.AddPostSlugForum)
	r.GET("/api/thread/{slug_or_id}/posts", forumHandler.GetPostsSlugForum)
	r.GET("/api/post/{id:[0-9]+}/details", forumHandler.GetPostByIDForum)
	r.POST("/api/post/{id:[0-9]+}/details", forumHandler.UpdatePostForum)
	r.POST("/api/thread/{id:[0-9]+}/vote", forumHandler.AddVoteIDForum)
	r.POST("/api/thread/{slug}/vote", forumHandler.AddVoteSlugForum)
	r.GET("/api/service/status", forumHandler.GetServiceStatusForum)
	r.POST("/api/service/clear", forumHandler.ClearDataBaseForum)

	log.Error().Msgf(fasthttp.ListenAndServe(":5000", JSONSetContentType(r.Handler)).Error())
}

func JSONSetContentType(req fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Content-Type", "application/json")
		req(ctx)
	}
}
