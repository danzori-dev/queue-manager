package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type OwnerNotifier struct {
	bot        *telego.Bot
	ownerID    int64
	apiTimeout time.Duration
}

func NewOwnerNotifier(token string, ownerID int64, apiTimeout time.Duration) (*OwnerNotifier, error) {
	bot, err := telego.NewBot(token, telego.WithDiscardLogger())
	if err != nil {
		return nil, fmt.Errorf("create notifier bot: %w", err)
	}

	return &OwnerNotifier{
		bot:        bot,
		ownerID:    ownerID,
		apiTimeout: apiTimeout,
	}, nil
}

func (n *OwnerNotifier) Send(ctx context.Context, text string) error {
	reqCtx, cancel := context.WithTimeout(ctx, n.apiTimeout)
	defer cancel()

	if _, err := n.bot.SendMessage(reqCtx, tu.Message(tu.ID(n.ownerID), text)); err != nil {
		return fmt.Errorf("send message to owner: %w", err)
	}

	return nil
}
