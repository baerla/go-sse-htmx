package main

import (
	"context"
	"time"
)

var allReactions = []Reaction{
	{
		ID:    "fire",
		Label: "🔥",
	},
	{
		ID:    "thinking",
		Label: "🤔",
	},
	{
		ID:    "heart",
		Label: "🩵",
	},
	{
		ID:    "laugh",
		Label: "😂",
	},
	{
		ID:    "sad",
		Label: "😢",
	},
}

func mustReactionByID(id string) Reaction {
	for _, r := range allReactions {
		if r.ID == id {
			return r
		}
	}

	panic("reaction not found")
}

type Reaction struct {
	ID    string
	Label string
}

type Post struct {
	ID        int
	Author    string
	Content   string
	CreatedAt time.Time
	Views     int
	Reactions map[string]int
}

type PostStats struct {
	ID              int
	Views           int
	ViewsUpdated    bool
	Reactions       map[string]int
	ReactionUpdated *string
}

type Repository struct {
	posts map[int]Post
}

type ErrPostNotFound struct {
	error
}

func NewRepository() *Repository {
	return &Repository{
		posts: make(map[int]Post),
	}
}

func (s *Repository) PostByID(ctx context.Context, id int) (Post, error) {
	post, ok := s.posts[id]
	if !ok {
		return Post{}, ErrPostNotFound{}
	}

	return post, nil
}

func (s *Repository) AllPosts(ctx context.Context) ([]Post, error) {
	posts := make([]Post, 0, len(s.posts))
	for _, post := range s.posts {
		posts = append(posts, post)
	}

	return posts, nil
}

func (s *Repository) UpdatePost(ctx context.Context, id int, updateFn func(*Post)) error {
	post, ok := s.posts[id]
	if !ok {
		return ErrPostNotFound{}
	}

	updateFn(&post)
	s.posts[id] = post

	return nil
}
