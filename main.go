package main

import (
	"context"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-http/v2/pkg/http"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

type PostViewed struct {
	PostID int `json:"post_id"`
}

type PostReactionAdded struct {
	PostID     int    `json:"post_id"`
	ReactionID string `json:"reaction_id"`
}

type PostStatsUpdated struct {
	PostID          int            `json:"post_id"`
	Views           int            `json:"views"`
	ViewsUpdated    bool           `json:"views_updated"`
	Reactions       map[string]int `json:"reactions"`
	ReactionUpdated *string        `json:"reactions_updated"`
}

func main() {
	logger := watermill.NewStdLogger(false, false)

	publisher := gochannel.NewGoChannel(gochannel.Config{}, logger)

	eventBus, err := cqrs.NewEventBusWithConfig(publisher, cqrs.EventBusConfig{
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			return params.EventName, nil
		},
		Marshaler: cqrs.JSONMarshaler{},
		Logger:    logger,
	})
	if err != nil {
		panic(err)
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		panic(err)
	}
	router.AddMiddleware(middleware.Recoverer)

	eventProcessor, err := cqrs.NewEventProcessorWithConfig(router, cqrs.EventProcessorConfig{
		GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
			return params.EventName, nil
		},
		SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return publisher, nil
		},
		Marshaler: cqrs.JSONMarshaler{},
		Logger:    logger,
	})
	if err != nil {
		panic(err)
	}

	repo := NewRepository()

	err = eventProcessor.AddHandlers(
		cqrs.NewEventHandler(
			"UpdateViews",
			func(ctx context.Context, event *PostViewed) error {
				var views int
				var reactions map[string]int
				err = repo.UpdatePost(ctx, event.PostID, func(post *Post) {
					post.Views++
					views = post.Views
					reactions = post.Reactions
				})
				if err != nil {
					return err
				}

				statsUpdated := PostStatsUpdated{
					PostID:       event.PostID,
					ViewsUpdated: true,
					Views:        views,
					Reactions:    reactions,
				}

				return eventBus.Publish(ctx, statsUpdated)
			},
		),
		cqrs.NewEventHandler(
			"UpdateReactions",
			func(ctx context.Context, event *PostReactionAdded) error {
				var views int
				var reactions map[string]int
				err = repo.UpdatePost(ctx, event.PostID, func(post *Post) {
					post.Reactions[event.ReactionID]++
					views = post.Views
					reactions = post.Reactions
				})
				if err != nil {
					return err
				}

				statsUpdated := PostStatsUpdated{
					PostID:          event.PostID,
					Views:           views,
					ReactionUpdated: &event.ReactionID,
					Reactions:       reactions,
				}

				return eventBus.Publish(ctx, statsUpdated)
			},
		),
	)

	go func() {
		err := router.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	sseRouter, err := http.NewSSERouter(http.SSERouterConfig{
		UpstreamSubscriber: publisher,
		Marshaler:          http.StringSSEMarshaler{},
	}, logger)
	if err != nil {
		panic(err)
	}

	go func() {
		err := sseRouter.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	handler := NewHandler(repo, eventBus, sseRouter)

	err = handler.Start(":8080")
	if err != nil {
		panic(err)
	}
}
