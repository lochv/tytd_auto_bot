package tytd_go

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	"tytd_go/internal/model"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ThienmaInfoHandler reports cached Thiên Ma status and spiritual-ball
// resources for every event account owned by the requesting leader. It is
// deliberately read-only: it does not spend items or send game requests.
func (t *TeleBot) ThienmaInfoHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	if !t.isLeader(msg.Chat.ID) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID, Text: "leader only",
			ReplyParameters: &models.ReplyParameters{MessageID: msg.ID},
		})
		return
	}

	members, err := model.GetEventUsersByLeader(t.ydb.GetDB(), fmt.Sprintf("%d", msg.Chat.ID))
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID, Text: "failed to load event users: " + err.Error(),
			ReplyParameters: &models.ReplyParameters{MessageID: msg.ID},
		})
		return
	}
	if len(members) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID, Text: "Bạn chưa có event user nào.",
			ReplyParameters: &models.ReplyParameters{MessageID: msg.ID},
		})
		return
	}

	const header = "<b>🔮 Trạng thái Thiên Ma & Linh Lực</b>\n\n"
	var page strings.Builder
	page.WriteString(header)
	send := func() {
		if page.Len() <= len(header) {
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID, Text: page.String(), ParseMode: models.ParseModeHTML,
		})
		page.Reset()
		page.WriteString(header)
	}

	for i, member := range members {
		line := fmt.Sprintf("<b>%d. %s</b>\n", i+1, html.EscapeString(member.Username))
		uid, parseErr := strconv.ParseInt(member.ID, 10, 64)
		if parseErr != nil {
			line += "   ⚠️ ID tài khoản không hợp lệ\n\n"
		} else if value, ok := t.session.Load(uid); !ok {
			line += "   ⏹️ chưa login\n\n"
		} else if p, ok := value.(*EventPlayer); !ok || !p.Running {
			line += "   ⏹️ chưa login\n\n"
		} else {
			line += "   " + html.EscapeString(p.GodIslandResourceStatus(GIEventPeak)) + "\n\n"
		}
		if page.Len()+len(line) > 3500 {
			send()
		}
		page.WriteString(line)
	}
	send()
}
