package projects

import (
	"context"
	"errors"
	"fmt"
	"main/db"
	"main/internal/dto"
	"main/internal/entity"
	"regexp"
	"strings"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ProjectService struct {
	pool *pgxpool.Pool
}

// New create new Project Service
func New() ProjectService {
	return ProjectService{db.Connection()}
}

type IProjectService interface {
	CreateProject(p dto.CreateProjectDTO) (int, error)
	FetchProject(id int) (entity.Project, error)
}

func (ps ProjectService) FetchProject(url string) (entity.Project, error) {
	var p entity.Project
	err := pgxscan.Get(context.Background(), ps.pool, &p, `select p.id, p.name, p.description, p.creating, p.banner, p.avatar, p.setting, p.created_at,
																													JSON_BUILD_OBJECT('name',c.name,'is_company',c.is_company,'id',c.id) as creator
																													from project p
																													inner join creator c on c.id = p.creator_id
																													inner join category ct on ct.id = p.category_id
																													where lower(page_url) = lower($1)
																													group by p.id,c.id`, url)
	if err != nil {
		fmt.Print(err.Error())
		return entity.Project{}, err
	}
	return p, nil
}
func (ps ProjectService) FetchProjectFeed(url string, user_id int) ([]entity.Feed, error) {
	var feeds []entity.Feed
	var tier entity.Tier
	var err error
	if user_id > 0 {
		err = pgxscan.Get(context.Background(), ps.pool, &tier, `
		select t.price,t.id,t.project_id
		from users u
		inner join subscription s on s.user_id = u.id
		inner join tier t on t.tier_id = s.tier_id and t.project_id = (select id from project where lower(page_url) = lower($2) limit 1)
		where id = $1
		`, user_id, url)
		err = pgxscan.Select(context.Background(), ps.pool, &feeds, `
		select p.id as id,case when p.min_price > $1 JSON_BUILD_OBJECT('min_price',p.min_price) as post, null as donation, null as subscription,'post' as type
		from project pr
		inner join post p on p.project_id = pr.id
		inner join post_tag pt on pt.post_id = p.id
		inner join tag t on t.id = pt.tag_id
		
	`)
	} else {

	}

	if err != nil {
		return nil, err
	}
	return feeds, err
}
func (ps ProjectService) CreateProject(p dto.CreateProjectDTO) error {
	_, err := ps.pool.Exec(context.Background(), "insert into project (name, page_url, description, creating, creator_id, cover_id, category_id, setting, created_at, updated_at) values ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())", p.Name, p.PageURL, p.Description, p.Creating, p.CreatorID, p.CoverID, p.CategoryID)
	if err != nil {
		return err
	}
	return nil
}

func (ps ProjectService) isPageURLAvailable(url string) (bool, error) {
	for _, page := range NotAllowedURL {
		lowerd_url := strings.ToLower(url)
		if page == lowerd_url {
			return false, nil
		}
	}
	matched, _ := regexp.MatchString("^[a-zA-Z0-9\u3040-\u309f\u30a0-\u30ff\uff00-\uff9f\u4e00-\u9faf\u3400-\u4dbf]*$", url)
	if !matched {
		return false, errors.New("許されていない文字が入っています。")
	}
	if len(url) > 16 {
		return false, errors.New("URLは16文字以下でお願いします。")
	}
	var project entity.Project
	err := pgxscan.Get(context.Background(), ps.pool, &project, `select id from project where lower(page_url) = lower($1)`, url)
	if err != nil {
		return false, err
	}
	if project.ID > 0 {
		return false, errors.New("このURLはもう使われています。")
	}
	return true, nil
}

//! Temporary, does not belong here
func (ps ProjectService) GetCreatorIDFromUserID(userID int) (int, error) {
	creatorID := 0
	err := ps.pool.QueryRow(context.Background(), "select id from creator where user_id=$1", userID).Scan(&creatorID)
	if err != nil {
		return -1, err
	}
	if creatorID <= 0 {
		return -1, errors.New("No creator found")
	}
	return creatorID, nil
}
