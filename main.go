package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shoaibaziz-afk/Blog_aggregator/internal/config"
	"github.com/shoaibaziz-afk/Blog_aggregator/internal/database"

	_ "github.com/lib/pq"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.handlers[cmd.name]

	if !exists {
		return errors.New("command does not exist")
	}

	return handler(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

func middlewareLoggedIn(
	handler func(s *state, cmd command, user database.User) error,
) func(*state, command) error {

	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(
			context.Background(),
			s.cfg.CurrentUserName,
		)

		if err != nil {
			return err
		}

		return handler(s, cmd, user)
	}
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = s.db.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		return err
	}

	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}

	fmt.Printf("Fetching feed: %s\n", feed.Name)

	for _, item := range rssFeed.Channel.Item {

		publishedAt := sql.NullTime{}

		if item.PubDate != "" {
			parsedTime, err := time.Parse(time.RFC1123Z, item.PubDate)
			if err == nil {
				publishedAt.Valid = true
				publishedAt.Time = parsedTime
			}
		}

		_, err = s.db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				Title: item.Title,
				Url:   item.Link,
				Description: sql.NullString{
					String: item.Description,
					Valid:  item.Description != "",
				},
				PublishedAt: publishedAt,
				FeedID:      feed.ID,
			},
		)

		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("Saved post:", item.Title)
	}

	fmt.Println("========================================")

	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("username is required")
	}

	username := cmd.args[0]

	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return errors.New("user does not exist")
	}

	err = s.cfg.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("User has been set to %s\n", username)

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("username is required")
	}

	username := cmd.args[0]

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	})

	if err != nil {
		return err
	}

	err = s.cfg.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Println("User created successfully!")
	fmt.Println(user)

	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return err
	}

	fmt.Println("Database reset successful")

	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: agg <time_between_reqs>")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %v\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)

	for ; ; <-ticker.C {
		err := scrapeFeeds(s)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return errors.New("usage: addfeed <name> <url>")
	}

	feed, err := s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			Name:   cmd.args[0],
			Url:    cmd.args[1],
			UserID: user.ID,
		},
	)

	if err != nil {
		return err
	}

	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			UserID: user.ID,
			FeedID: feed.ID,
		},
	)

	if err != nil {
		return err
	}

	fmt.Println("Feed created successfully!")
	fmt.Println(feed)

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		fmt.Println("Name:", feed.Name)
		fmt.Println("URL:", feed.Url)
		fmt.Println("User:", feed.UserName)
		fmt.Println()
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: follow <url>")
	}

	feed, err := s.db.GetFeedByURL(
		context.Background(),
		cmd.args[0],
	)

	if err != nil {
		return err
	}

	feedFollow, err := s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			UserID: user.ID,
			FeedID: feed.ID,
		},
	)

	if err != nil {
		return err
	}

	fmt.Println("Feed:", feedFollow.FeedName)
	fmt.Println("User:", feedFollow.UserName)

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	feedFollows, err := s.db.GetFeedFollowsForUser(
		context.Background(),
		user.ID,
	)

	if err != nil {
		return err
	}

	for _, follow := range feedFollows {
		fmt.Println(follow.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return errors.New("usage: unfollow <url>")
	}

	feed, err := s.db.GetFeedByURL(
		context.Background(),
		cmd.args[0],
	)

	if err != nil {
		return err
	}

	err = s.db.DeleteFeedFollow(
		context.Background(),
		database.DeleteFeedFollowParams{
			UserID: user.ID,
			FeedID: feed.ID,
		},
	)

	if err != nil {
		return err
	}

	fmt.Println("Feed unfollowed successfully")

	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {

	limit := 2

	if len(cmd.args) == 1 {
		fmt.Sscanf(cmd.args[0], "%d", &limit)
	}

	posts, err := s.db.GetPostsForUser(
		context.Background(),
		database.GetPostsForUserParams{
			UserID: user.ID,
			Limit:  int32(limit),
		},
	)

	if err != nil {
		return err
	}

	for _, post := range posts {
		fmt.Println("Title:", post.Title)
		fmt.Println("URL:", post.Url)

		if post.Description.Valid {
			fmt.Println("Description:", post.Description.String)
		}

		fmt.Println()
	}

	return nil
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	dbQueries := database.New(db)

	s := &state{
		db:  dbQueries,
		cfg: &cfg,
	}

	cmds := commands{
		handlers: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("feeds", handlerFeeds)

	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))

	if len(os.Args) < 2 {
		fmt.Println("not enough arguments provided")
		os.Exit(1)
	}

	cmd := command{
		name: os.Args[1],
		args: os.Args[2:],
	}

	err = cmds.run(s, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
