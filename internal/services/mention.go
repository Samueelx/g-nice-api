package services

import (
	"regexp"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// mentionRe matches @username tokens. Usernames may contain letters, digits,
// and underscores — consistent with the User.Username column constraints.
var mentionRe = regexp.MustCompile(`@([A-Za-z0-9_]+)`)

// maxMentionsPerContent caps how many unique usernames are resolved per piece
// of content to prevent a single request from triggering excessive DB lookups.
const maxMentionsPerContent = 10

// extractMentions returns a deduplicated slice of usernames found in content.
func extractMentions(content string) []string {
	matches := mentionRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		username := m[1]
		if _, dup := seen[username]; !dup {
			seen[username] = struct{}{}
			out = append(out, username)
		}
	}
	return out
}

// dispatchMentions parses @username tokens from content, resolves each to a
// user ID, and fires a mention notification. It is fully best-effort:
//   - unknown usernames are silently skipped
//   - individual notify failures are silently skipped
//   - at most maxMentionsPerContent usernames are processed
//
// Self-mentions are suppressed by NotificationService.Notify (no-op when
// actorID == userID), so no extra check is needed here.
//
// This function is intended to be called in a goroutine so it does not block
// the request path.
func dispatchMentions(
	content string,
	actorID uint,
	targetID *uint,
	targetType string,
	userRepo repository.UserRepository,
	notifSvc NotificationService,
) {
	usernames := extractMentions(content)
	if len(usernames) > maxMentionsPerContent {
		usernames = usernames[:maxMentionsPerContent]
	}

	for _, username := range usernames {
		user, err := userRepo.FindByUsername(username)
		if err != nil {
			// Unknown username or DB error — skip silently.
			continue
		}
		_ = notifSvc.Notify(user.ID, actorID, models.NotifTypeMention, targetID, targetType)
	}
}
