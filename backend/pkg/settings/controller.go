package settings

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/bun"

	"github.com/watch-tower-org/watchdog/backend/internal/logger"
	"github.com/watch-tower-org/watchdog/backend/internal/mailer"
	"github.com/watch-tower-org/watchdog/backend/internal/model"
	"github.com/watch-tower-org/watchdog/backend/internal/validator"
)

type Controller struct {
	db *bun.DB
}

func NewController(db *bun.DB) *Controller {
	return &Controller{db: db}
}

func (c *Controller) loadSettings(ctx context.Context) (*model.Settings, error) {
	var s model.Settings
	err := c.db.NewSelect().
		Model(&s).
		Limit(1).
		OrderBy("id", bun.OrderAsc).
		Scan(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("Error getting settings: %v", err)
		return nil, errors.New("Failed to retrieve settings.")
	}

	return &s, nil
}

func (c *Controller) GetSettings(ctx context.Context) (*model.Settings, error) {
	s, err := c.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	s.AdminPassword = ""
	s.APIKeyHash = ""
	return s, nil
}

func (c *Controller) GetEmailSettings(ctx context.Context) (*model.Settings, error) {
	s, err := c.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	s.SMTPPassword = ""
	return s, nil
}

func (c *Controller) UpdateEmailSettings(ctx context.Context, req *model.UpdateEmailSettingsRequest) (*model.Settings, error) {
	s, err := c.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	if req.SMTPHost != nil {
		s.SMTPHost = *req.SMTPHost
	}
	if req.SMTPPort != nil {
		s.SMTPPort = *req.SMTPPort
	}
	if req.SMTPUsername != nil {
		s.SMTPUsername = *req.SMTPUsername
	}
	if req.SMTPPassword != nil && *req.SMTPPassword != "" {
		s.SMTPPassword = *req.SMTPPassword
	}
	if req.SMTPFromEmail != nil {
		s.SMTPFromEmail = *req.SMTPFromEmail
	}
	if req.SMTPFromName != nil {
		s.SMTPFromName = *req.SMTPFromName
	}

	s.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(s).
		Where("id = ?", s.ID).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update email settings: %v", err)
		return nil, errors.New("Failed to update email settings.")
	}

	s.SMTPPassword = ""
	return s, nil
}

func (c *Controller) UpdateThrottle(ctx context.Context, req *model.UpdateThrottleRequest) (*model.Settings, error) {
	s, err := c.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	if req.ThrottleWindow != nil {
		s.ThrottleWindow = *req.ThrottleWindow
	}

	s.UpdatedAt = time.Now()

	_, err = c.db.NewUpdate().
		Model(s).
		Where("id = ?", s.ID).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to update throttle settings: %v", err)
		return nil, errors.New("Failed to update throttle settings.")
	}

	return s, nil
}

func (c *Controller) TestSMTP(ctx context.Context, req *model.TestEmailRequest) error {
	to := req.To

	if !validator.IsValidEmail(to) {
		return errors.New("Invalid email address format.")
	}

	var host, username, password, fromEmail, fromName string
	var port int

	// If the admin provided SMTP fields in the request, test those
	// (not-yet-saved) values. Otherwise fall back to the saved settings.
	if req.SMTPHost != nil || req.SMTPPort != nil || req.SMTPFromEmail != nil {
		if req.SMTPHost == nil || req.SMTPPort == nil || req.SMTPFromEmail == nil {
			return errors.New("When testing unsaved SMTP settings, smtp_host, smtp_port and smtp_from_email are all required.")
		}
		host = *req.SMTPHost
		port = *req.SMTPPort
		if req.SMTPUsername != nil {
			username = *req.SMTPUsername
		}
		if req.SMTPPassword != nil {
			password = *req.SMTPPassword
		}
		fromEmail = *req.SMTPFromEmail
		if req.SMTPFromName != nil {
			fromName = *req.SMTPFromName
		}
	} else {
		s, err := c.loadSettings(ctx)
		if err != nil {
			return err
		}
		if s.SMTPHost == "" || s.SMTPFromEmail == "" {
			return errors.New("SMTP settings are not configured. Please configure them first.")
		}
		host = s.SMTPHost
		port = s.SMTPPort
		username = s.SMTPUsername
		password = s.SMTPPassword
		fromEmail = s.SMTPFromEmail
		fromName = s.SMTPFromName
	}

	subject := "Test Email from WatchTower"
	body := "This is a test email to verify your SMTP settings."

	if err := mailer.Send(host, port, username, password, fromName, fromEmail, to, subject, body); err != nil {
		logger.Ctx(ctx).Error().Msgf("smtp test send to %s failed: %v", to, err)
		return errors.New("Failed to send test email. Please check the SMTP settings.")
	}

	return nil
}

func (c *Controller) CreateDefaultSettings(adminUsername, adminPasswordHash string) error {
	ctx := context.Background()

	nb, err := c.db.NewSelect().
		Model((*model.Settings)(nil)).
		Count(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("count settings: %v", err)
		return err
	}

	if nb != 0 {
		return nil
	}

	defaultSettings := &model.Settings{
		SetupComplete:  false,
		AdminUsername:  adminUsername,
		AdminPassword:  adminPasswordHash,
		ThrottleWindow: 60,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = c.db.NewInsert().
		Model(defaultSettings).
		Exec(ctx)

	if err != nil {
		logger.Ctx(ctx).Error().Msgf("failed to create default settings: %v", err)
		return errors.New("An unexpected error occurred. Please try again.")
	}

	return nil
}
