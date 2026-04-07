package email

import (
	"bytes"
	"fmt"
	"html/template"

	resend "github.com/resend/resend-go/v2"
)

// ResendSender implements Sender using the Resend transactional email API.
type ResendSender struct {
	client *resend.Client
	from   string
}

// NewResendSender constructs a ResendSender.
// apiKey  — your Resend API key (re_xxxx)
// from    — the verified sender address, e.g. "G-Nice <noreply@yourdomain.com>"
func NewResendSender(apiKey, from string) *ResendSender {
	return &ResendSender{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

// SendOTP sends a styled OTP email to the given address.
func (s *ResendSender) SendOTP(to, username, otp string) error {
	html, err := renderOTPEmail(username, otp)
	if err != nil {
		return fmt.Errorf("render OTP email: %w", err)
	}

	_, err = s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Your G-Nice verification code",
		Html:    html,
	})
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	return nil
}

// ── HTML template ─────────────────────────────────────────────────────────────

type otpData struct {
	Username string
	OTP      string
}

var otpTmpl = template.Must(template.New("otp").Parse(otpHTMLTemplate))

func renderOTPEmail(username, otp string) (string, error) {
	var buf bytes.Buffer
	if err := otpTmpl.Execute(&buf, otpData{Username: username, OTP: otp}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const otpHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Verify your email — G-Nice</title>
</head>
<body style="margin:0;padding:0;background-color:#0f0f13;font-family:'Segoe UI',Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background-color:#0f0f13;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="560" cellpadding="0" cellspacing="0" style="background-color:#1a1a24;border-radius:16px;overflow:hidden;border:1px solid #2a2a3a;">

          <!-- Header -->
          <tr>
            <td style="background:linear-gradient(135deg,#6c63ff 0%,#a855f7 100%);padding:36px 40px;text-align:center;">
              <h1 style="margin:0;color:#ffffff;font-size:28px;font-weight:700;letter-spacing:-0.5px;">
                G-Nice
              </h1>
              <p style="margin:8px 0 0;color:rgba(255,255,255,0.8);font-size:14px;">
                Community &amp; Social Platform
              </p>
            </td>
          </tr>

          <!-- Body -->
          <tr>
            <td style="padding:40px;">
              <p style="margin:0 0 8px;color:#a0a0b8;font-size:14px;text-transform:uppercase;letter-spacing:1px;font-weight:600;">
                Email Verification
              </p>
              <h2 style="margin:0 0 20px;color:#ffffff;font-size:22px;font-weight:700;">
                Hey {{.Username}}, welcome aboard! 👋
              </h2>
              <p style="margin:0 0 28px;color:#8888a8;font-size:15px;line-height:1.6;">
                Enter the 6-digit code below to verify your email address and activate your account.
              </p>

              <!-- OTP Box -->
              <table width="100%" cellpadding="0" cellspacing="0">
                <tr>
                  <td align="center">
                    <div style="display:inline-block;background:#111120;border:1px solid #6c63ff;border-radius:12px;padding:24px 48px;">
                      <span style="font-size:42px;font-weight:800;letter-spacing:14px;color:#a855f7;font-variant-numeric:tabular-nums;">
                        {{.OTP}}
                      </span>
                    </div>
                  </td>
                </tr>
              </table>

              <!-- Expiry notice -->
              <p style="margin:28px 0 0;text-align:center;color:#6666888;font-size:13px;">
                ⏱ This code expires in <strong style="color:#a0a0b8;">10 minutes</strong>.
                Do not share it with anyone.
              </p>
            </td>
          </tr>

          <!-- Divider -->
          <tr>
            <td style="padding:0 40px;">
              <hr style="border:none;border-top:1px solid #2a2a3a;margin:0;" />
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="padding:24px 40px;text-align:center;">
              <p style="margin:0;color:#555570;font-size:12px;line-height:1.6;">
                If you didn't create a G-Nice account, you can safely ignore this email.<br />
                &copy; 2025 G-Nice. All rights reserved.
              </p>
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
