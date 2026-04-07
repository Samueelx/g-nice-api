package email

// Sender is the interface for sending transactional emails.
// Any implementation (Resend, SendGrid, SMTP, mock) satisfies this contract.
type Sender interface {
	// SendOTP delivers a 6-digit verification code to the recipient.
	SendOTP(to, username, otp string) error
}
