package repository

import (
	forum "DbGODZ/internal/app"
	"DbGODZ/internal/app/models"
	"errors"
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx"
	"strings"
	"time"
)

type postgresForumRepository struct {
	conn *pgx.ConnPool
}

func NewPostgresForumRepository(conn *pgx.ConnPool) forum.Repository {
	return &postgresForumRepository{
		conn: conn,
	}
}

func (p *postgresForumRepository) AddForum(forum models.Forum) (models.Forum, error) {
	query := `INSERT INTO forum(
    "user",
    slug,
    title)
	VALUES ($1, $2, $3) RETURNING *`

	userObj, err := p.GetByNick(forum.User)
	if err != nil {
		return models.Forum{}, err
	}

	var forumObj models.Forum
	err = p.conn.QueryRow(query, userObj.Nickname, forum.Slug, forum.Title).Scan(&forumObj.User, &forumObj.Posts, &forumObj.Slug, &forumObj.Threads, &forumObj.Title)
	return forumObj, err
}

func (p *postgresForumRepository) GetBySlugForum(slug string) (models.Forum, error) {
	query := `SELECT * FROM forum WHERE LOWER(slug)=LOWER($1)`

	var forumObj models.Forum
	err := p.conn.QueryRow(query, slug).Scan(&forumObj.User, &forumObj.Posts, &forumObj.Slug, &forumObj.Threads, &forumObj.Title)

	return forumObj, err
}

func (p *postgresForumRepository) AddThreadForum(thread models.Thread) (models.Thread, error) {
	query := `INSERT INTO thread(
    slug,
    author,
    created,
    message,
    title,
	forum)
	VALUES (NULLIF($1, ''), $2, $3, $4, $5, $6) RETURNING *`

	forumObj, err := p.GetBySlugForum(thread.Forum)
	if err != nil {
		return models.Thread{}, err
	}

	var threadObj models.Thread
	var created time.Time

	if thread.Created != "" {
		err = p.conn.QueryRow(query, thread.Slug, thread.Author,
			thread.Created, thread.Message, thread.Title, forumObj.Slug).Scan(&threadObj.Author,
			&created, &threadObj.Forum, &threadObj.Id, &threadObj.Message, &threadObj.Slug,
			&threadObj.Title, &threadObj.Votes)

	} else {
		err = p.conn.QueryRow(query, thread.Slug, thread.Author,
			time.Time{}, thread.Message, thread.Title, forumObj.Slug).Scan(&threadObj.Author,
			&created, &threadObj.Forum, &threadObj.Id, &threadObj.Message, &threadObj.Slug,
			&threadObj.Title, &threadObj.Votes)
	}
	threadObj.Created = strfmt.DateTime(created.UTC()).String()
	return threadObj, err
}

func (p *postgresForumRepository) GetThreadsForum(slug string, limit int, since string, desc bool) ([]models.Thread, error) {
	var whereExpression string
	var orderExpression string

	if since != "" && desc {
		whereExpression = fmt.Sprintf(`LOWER(forum)=LOWER('%s') AND created <= '%s'`, slug, since)
	} else if since != "" && !desc {
		whereExpression = fmt.Sprintf(`LOWER(forum)=LOWER('%s') AND created >= '%s'`, slug, since)
	} else {
		whereExpression = fmt.Sprintf(`LOWER(forum)=LOWER('%s')`, slug)
	}
	if desc {
		orderExpression = `DESC`
	} else {
		orderExpression = `ASC`
	}

	query := fmt.Sprintf("SELECT * FROM thread WHERE %s ORDER BY created %s LIMIT NULLIF(%d, 0)",
		whereExpression, orderExpression, limit)

	data := make([]models.Thread, 0, 0)
	row, err := p.conn.Query(query)

	if err != nil {
		return nil, err
	}

	defer func() {
		if row != nil {
			row.Close()
		}
	}()

	for row.Next() {
		var threadObj models.Thread
		var created time.Time

		err = row.Scan(&threadObj.Author, &created, &threadObj.Forum, &threadObj.Id, &threadObj.Message, &threadObj.Slug, &threadObj.Title, &threadObj.Votes)

		if err != nil {
			return nil, err
		}

		threadObj.Created = strfmt.DateTime(created.UTC()).String()

		data = append(data, threadObj)
	}

	return data, err
}

func (p *postgresForumRepository) CheckThreadExistsForum(slug string) (bool, error) {
	query := `select exists(select 1 from thread where LOWER(forum)=LOWER($1))`

	var exists bool

	err := p.conn.QueryRow(query, slug).Scan(&exists)
	return exists, err
}

func (p *postgresForumRepository) GetThreadBySlugForum(slug string) (models.Thread, error) {
	query := `SELECT * FROM thread WHERE LOWER(slug)=LOWER($1)`

	var threadObj models.Thread
	var created time.Time

	err := p.conn.QueryRow(query, slug).Scan(&threadObj.Author, &created, &threadObj.Forum,
		&threadObj.Id, &threadObj.Message, &threadObj.Slug, &threadObj.Title, &threadObj.Votes)

	threadObj.Created = strfmt.DateTime(created.UTC()).String()
	return threadObj, err
}

func (p *postgresForumRepository) GetThreadByIDForum(id int) (models.Thread, error) {
	query := `SELECT * FROM thread WHERE id=$1`

	var threadObj models.Thread
	var created time.Time

	err := p.conn.QueryRow(query, id).Scan(&threadObj.Author, &created, &threadObj.Forum,
		&threadObj.Id, &threadObj.Message, &threadObj.Slug, &threadObj.Title, &threadObj.Votes)
	threadObj.Created = strfmt.DateTime(created.UTC()).String()

	return threadObj, err
}

func (p *postgresForumRepository) GetThreadIDBySlugForum(slug string) (int, error) {
	query := `SELECT id FROM thread WHERE LOWER(slug)=LOWER($1)`

	var id int
	err := p.conn.QueryRow(query, slug).Scan(&id)
	return id, err
}

func (p *postgresForumRepository) GetThreadSlugByIDForum(id int) (string, error) {
	query := `SELECT slug FROM thread WHERE id=$1`

	var slug string
	err := p.conn.QueryRow(query, id).Scan(&slug)
	return slug, err
}

func (p *postgresForumRepository) getForumSlugForum(threadID int) (string, error) {
	query := `SELECT forum FROM thread WHERE id=$1`

	var slug string
	err := p.conn.QueryRow(query, threadID).Scan(&slug)
	return slug, err
}

func (p *postgresForumRepository) AddPostsForum(posts []models.Post, threadID int) ([]models.Post, error) {
	query := `INSERT INTO post(
                 author,
                 created,
                 message,
                 parent,
				 thread,
				 forum) VALUES `
	data := make([]models.Post, 0, 0)
	if len(posts) == 0 {
		return data, nil
	}

	slug, err := p.getForumSlugForum(threadID)
	if err != nil {
		return data, err
	}

	timeCreated := time.Now()
	var valuesNames []string
	var values []interface{}
	i := 1
	for _, element := range posts {
		valuesNames = append(valuesNames, fmt.Sprintf(
			"($%d, $%d, $%d, nullif($%d, 0), $%d, $%d)",
			i, i+1, i+2, i+3, i+4, i+5))
		i += 6
		values = append(values, element.Author, timeCreated, element.Message, element.Parent, threadID, slug)
	}

	query += strings.Join(valuesNames[:], ",")
	query += " RETURNING *"
	row, err := p.conn.Query(query, values...)

	if err != nil {
		return data, err
	}
	defer func() {
		if row != nil {
			row.Close()
		}
	}()

	for row.Next() {

		var post models.Post
		var created time.Time

		err = row.Scan(&post.Author, &created, &post.Forum, &post.Id, &post.IsEdited,
			&post.Message, &post.Parent, &post.Thread, &post.Path)

		if err != nil {
			return data, err
		}
		post.Created = strfmt.DateTime(created.UTC()).String()
		data = append(data, post)

	}

	return data, err
}

func (p *postgresForumRepository) AddVoteForum(vote models.Vote) error {
	query := `INSERT INTO vote(
				nickname,  
				voice,     
				idThread)
				VALUES ($1, $2, NULLIF($3, 0))`

	_, err := p.conn.Exec(query, vote.Nickname, vote.Voice, vote.IdThread)
	return err
}

func (p *postgresForumRepository) UpdateVoteForum(vote models.Vote) error {
	query := `UPDATE vote SET voice=$1 WHERE LOWER(nickname) = LOWER($2) AND idThread = $3`
	_, err := p.conn.Exec(query, vote.Voice, vote.Nickname, vote.IdThread)
	return err
}

func (p *postgresForumRepository) getPostsFlatForum(threadID, limit, since int,
	desc bool) ([]models.Post, error) {

	query := `SELECT * FROM post WHERE thread=$1 `

	if desc {
		if since > 0 {
			query += fmt.Sprintf("AND id < %d ", since)
		}
		query += `ORDER BY id DESC `
	} else {
		if since > 0 {
			query += fmt.Sprintf("AND id > %d ", since)
		}
		query += `ORDER BY id `
	}
	query += `LIMIT NULLIF($2, 0)`
	var posts []models.Post

	row, err := p.conn.Query(query, threadID, limit)

	if err != nil {
		return posts, err
	}
	defer func() {
		if row != nil {
			row.Close()
		}
	}()

	for row.Next() {
		var post models.Post
		var created time.Time

		err = row.Scan(&post.Author, &created, &post.Forum, &post.Id, &post.IsEdited, &post.Message,
			&post.Parent, &post.Thread, &post.Path)

		if err != nil {
			return posts, err
		}
		post.Created = strfmt.DateTime(created.UTC()).String()
		posts = append(posts, post)

	}
	return posts, err
}

func (p *postgresForumRepository) getPostsTreeForum(threadID, limit, since int,
	desc bool) ([]models.Post, error) {
	var query string
	sinceQuery := ""
	if since != 0 {
		if desc {
			sinceQuery = `AND PATH < `
		} else {
			sinceQuery = `AND PATH > `
		}
		sinceQuery += fmt.Sprintf(`(SELECT path FROM post WHERE id = %d)`, since)
	}
	if desc {
		query = fmt.Sprintf(
			`SELECT * FROM post WHERE thread=$1 %s ORDER BY path DESC, id DESC LIMIT NULLIF($2, 0);`, sinceQuery)
	} else {
		query = fmt.Sprintf(
			`SELECT * FROM post WHERE thread=$1 %s ORDER BY path, id LIMIT NULLIF($2, 0);`, sinceQuery)
	}
	var posts []models.Post
	row, err := p.conn.Query(query, threadID, limit)

	if err != nil {
		return posts, err
	}
	defer func() {
		if row != nil {
			row.Close()
		}
	}()

	for row.Next() {
		var post models.Post
		var created time.Time

		err = row.Scan(&post.Author, &created, &post.Forum, &post.Id, &post.IsEdited, &post.Message,
			&post.Parent, &post.Thread, &post.Path)

		if err != nil {
			return posts, err
		}
		post.Created = strfmt.DateTime(created.UTC()).String()
		posts = append(posts, post)

	}
	return posts, err
}

func (p *postgresForumRepository) getPostsParentTreeForum(threadID, limit, since int,
	desc bool) ([]models.Post, error) {
	var query string
	sinceQuery := ""
	if since != 0 {
		if desc {
			sinceQuery = `AND PATH[1] < `
		} else {
			sinceQuery = `AND PATH[1] > `
		}
		sinceQuery += fmt.Sprintf(`(SELECT path[1] FROM post WHERE id = %d)`, since)
	}

	parentsQuery := fmt.Sprintf(
		`SELECT id FROM post WHERE thread = $1 AND parent IS NULL %s`, sinceQuery)

	if desc {
		parentsQuery += `ORDER BY id DESC`
		if limit > 0 {
			parentsQuery += fmt.Sprintf(` LIMIT %d`, limit)
		}
		query = fmt.Sprintf(
			`SELECT * FROM post WHERE path[1] IN (%s) ORDER BY path[1] DESC, path, id;`, parentsQuery)
	} else {
		parentsQuery += `ORDER BY id`
		if limit > 0 {
			parentsQuery += fmt.Sprintf(` LIMIT %d`, limit)
		}
		query = fmt.Sprintf(
			`SELECT * FROM post WHERE path[1] IN (%s) ORDER BY path,id;`, parentsQuery)
	}
	var posts []models.Post
	row, err := p.conn.Query(query, threadID)

	if err != nil {
		return posts, err
	}

	defer func() {
		if row != nil {
			row.Close()
		}
	}()

	for row.Next() {
		var post models.Post
		var created time.Time

		err = row.Scan(&post.Author, &created, &post.Forum, &post.Id, &post.IsEdited, &post.Message,
			&post.Parent, &post.Thread, &post.Path)

		if err != nil {
			return posts, err
		}
		post.Created = strfmt.DateTime(created.UTC()).String()
		posts = append(posts, post)

	}
	return posts, err
}

func (p *postgresForumRepository) GetPostsForum(postSlugOrId models.Thread, limit, since int,
	sort string, desc bool) ([]models.Post, error) {
	var err error
	threadId := 0
	if postSlugOrId.Id <= 0 {
		threadId, err = p.GetThreadIDBySlugForum(postSlugOrId.Slug.String)
		if err != nil {
			return nil, err
		}
	} else {
		threadId = int(postSlugOrId.Id)
	}

	switch sort {
	case "flat":
		return p.getPostsFlatForum(threadId, limit, since, desc)
	case "tree":
		return p.getPostsTreeForum(threadId, limit, since, desc)
	case "parent_tree":
		return p.getPostsParentTreeForum(threadId, limit, since, desc)
	default:
		return nil, errors.New("THERE IS NO SORT WITH THIS NAME")
	}
}

func (p *postgresForumRepository) GetPostForum(id int, related []string) (map[string]interface{}, error) {
	query := `SELECT * FROM post WHERE id = $1;`
	var post models.Post
	var created time.Time

	err := p.conn.QueryRow(query, id).Scan(&post.Author, &created, &post.Forum,
		&post.Id, &post.IsEdited, &post.Message, &post.Parent, &post.Thread, &post.Path)
	post.Created = strfmt.DateTime(created.UTC()).String()

	returnMap := map[string]interface{}{
		"post": post,
	}

	for _, relatedObj := range related {
		switch relatedObj {
		case "user":
			author, err := p.GetByNick(post.Author)
			if err != nil {
				return returnMap, err
			}
			returnMap["author"] = author
		case "thread":
			thread, err := p.GetThreadByIDForum(int(post.Thread))
			if err != nil {
				return returnMap, err
			}
			returnMap["thread"] = thread
		case "forum":
			forumObj, err := p.GetBySlugForum(post.Forum)
			if err != nil {
				return returnMap, err
			}
			returnMap["forum"] = forumObj
		}
	}

	return returnMap, err
}

func (p *postgresForumRepository) UpdatePostForum(newPost models.Post) (models.Post, error) {
	query := `UPDATE post SET message = $1, isEdited = true WHERE id = $2 RETURNING *;`

	oldPost, err := p.GetPostForum(int(newPost.Id), []string{})
	if err != nil {
		return models.Post{}, err
	}
	if oldPost["post"].(models.Post).Message == newPost.Message {
		return oldPost["post"].(models.Post), nil
	}

	if newPost.Message == "" {
		query := `SELECT * FROM post WHERE id = $1`
		var post models.Post
		var created time.Time

		err := p.conn.QueryRow(query, newPost.Id).Scan(&post.Author, &created,
			&post.Forum, &post.Id, &post.IsEdited, &post.Message, &post.Parent, &post.Thread, &post.Path)

		post.Created = strfmt.DateTime(created.UTC()).String()
		return post, err
	}

	var post models.Post
	var created time.Time

	err = p.conn.QueryRow(query, newPost.Message, newPost.Id).Scan(&post.Author, &created,
		&post.Forum, &post.Id, &post.IsEdited, &post.Message, &post.Parent, &post.Thread, &post.Path)
	post.Created = strfmt.DateTime(created.UTC()).String()

	return post, err
}

func (p *postgresForumRepository) UpdateThreadForum(newThread models.Thread) (models.Thread, error) {
	query := `UPDATE thread SET message=COALESCE(NULLIF($1, ''), message), title=COALESCE(NULLIF($2, ''), title) WHERE `

	if newThread.Id > 0 {
		query += `id = $3 RETURNING *`
		var threadObj models.Thread
		var created time.Time
		err := p.conn.QueryRow(query, newThread.Message, newThread.Title, newThread.Id).Scan(
			&threadObj.Author, &created, &threadObj.Forum, &threadObj.Id, &threadObj.Message, &threadObj.Slug,
			&threadObj.Title, &threadObj.Votes)
		threadObj.Created = strfmt.DateTime(created.UTC()).String()
		return threadObj, err
	} else {
		query += `LOWER(slug) = LOWER($3) RETURNING *`
		var threadObj models.Thread
		var created time.Time
		err := p.conn.QueryRow(query, newThread.Message, newThread.Title, newThread.Slug).Scan(
			&threadObj.Author, &created, &threadObj.Forum, &threadObj.Id, &threadObj.Message, &threadObj.Slug,
			&threadObj.Title, &threadObj.Votes)
		threadObj.Created = strfmt.DateTime(created.UTC()).String()
		return threadObj, err
	}
}

func (p *postgresForumRepository) GetServiceStatusForum() (map[string]int, error) {
	query := `SELECT * FROM (SELECT COUNT(*) FROM forum) as fC, (SELECT COUNT(*) FROM post) as pC,
              (SELECT COUNT(*) FROM thread) as tC, (SELECT COUNT(*) FROM users) as uC;`

	a, err := p.conn.Query(query)
	if err != nil {
		return nil, err
	}

	if a.Next() {
		forumCount, postCount, threadCount, usersCount := 0, 0, 0, 0
		err := a.Scan(&forumCount, &postCount, &threadCount, &usersCount)
		if err != nil {
			return nil, err
		}
		return map[string]int{
			"forum":  forumCount,
			"post":   postCount,
			"thread": threadCount,
			"user":   usersCount,
		}, nil
	}
	return nil, errors.New("no info available")
}

func (p *postgresForumRepository) ClearDatabaseForum() error {
	query := `TRUNCATE users, forum, thread, post, vote, users_forum;`

	_, err := p.conn.Exec(query)
	return err
}

func (p *postgresForumRepository) Add(user models.User) error {
	query := `INSERT INTO users(
    about,
    email,
    fullname,
    nickname)
	VALUES ($1, $2, $3, $4)`

	_, err := p.conn.Exec(query, user.About, user.Email, user.FullName, user.Nickname)
	return err
}

func (p *postgresForumRepository) GetByNickAndEmail(nickname, email string) ([]models.User, error) {
	query := `SELECT * FROM users WHERE LOWER(Nickname)=LOWER($1) OR Email=$2`

	var data []models.User

	row, err := p.conn.Query(query, nickname, email)

	if err != nil {
		return nil, err
	}

	defer func() {
		if row != nil {
			row.Close()
		}
	}()

	for row.Next() {

		var u models.User

		err = row.Scan(&u.About, &u.Email, &u.FullName, &u.Nickname)

		if err != nil {
			return nil, err
		}

		data = append(data, u)
	}

	return data, err
}

func (p *postgresForumRepository) GetByNick(nickname string) (models.User, error) {
	query := `SELECT * FROM users WHERE LOWER(Nickname)=LOWER($1)`

	var userObj models.User
	err := p.conn.QueryRow(query, nickname).Scan(&userObj.About, &userObj.Email, &userObj.FullName, &userObj.Nickname)
	return userObj, err
}

func (p *postgresForumRepository) Update(user models.User) (models.User, error) {
	query := `UPDATE users SET 
                 about=COALESCE(NULLIF($1, ''), about),
                 email=COALESCE(NULLIF($2, ''), email),
                 fullname=COALESCE(NULLIF($3, ''), fullname) 
	WHERE LOWER(nickname) = LOWER($4) RETURNING *`

	var userObj models.User
	err := p.conn.QueryRow(query, user.About, user.Email, user.FullName, user.Nickname).Scan(&userObj.About, &userObj.Email, &userObj.FullName, &userObj.Nickname)
	return userObj, err
}

func (p *postgresForumRepository) GetUsersByForum(slug string, limit int, since string, desc bool) ([]models.User, error) {
	var query string
	if desc {
		if since != "" {
			query = fmt.Sprintf(`SELECT users.about, users.Email, users.FullName, users.Nickname FROM users
    	inner join users_forum uf on users.Nickname = uf.nickname
        WHERE uf.slug =$1 AND uf.nickname < '%s'
        ORDER BY lower(users.Nickname) DESC LIMIT NULLIF($2, 0)`, since)
		} else {
			query = `SELECT users.about, users.Email, users.FullName, users.Nickname FROM users
    	inner join users_forum uf on users.Nickname = uf.nickname
        WHERE uf.slug =$1
        ORDER BY lower(users.Nickname) DESC LIMIT NULLIF($2, 0)`
		}
	} else {
		query = fmt.Sprintf(`SELECT users.about, users.Email, users.FullName, users.Nickname FROM users
    	inner join users_forum uf on users.Nickname = uf.nickname
        WHERE uf.slug =$1 AND uf.nickname > '%s'
        ORDER BY lower(users.Nickname) LIMIT NULLIF($2, 0)`, since)
	}
	var data []models.User
	row, err := p.conn.Query(query, slug, limit)

	if err != nil {
		return data, nil
	}

	defer func() {
		if row != nil {
			row.Close()
		}
	}()

	for row.Next() {

		var u models.User

		err = row.Scan(&u.About, &u.Email, &u.FullName, &u.Nickname)

		if err != nil {
			return data, err
		}

		data = append(data, u)
	}

	return data, err
}
