package email

import (
	"bytes"
	"fmt"
	"html/template"
)

var otpTemplate = template.Must(template.New("otp").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;background:#f4f6f9;padding:40px 0;">
<div style="max-width:480px;margin:0 auto;background:#fff;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,0.08);padding:40px;">
  <h2 style="margin:0 0 8px;color:#1a1a2e;font-size:20px;">Verification Code</h2>
  <p style="color:#666;margin:0 0 24px;font-size:14px;">Use this code to verify your identity. It expires in {{.ExpiryMinutes}} minutes.</p>
  <div style="background:#f0f4ff;border:2px solid #3B82F6;border-radius:8px;padding:20px;text-align:center;margin-bottom:24px;">
    <span style="font-size:32px;font-weight:700;letter-spacing:8px;color:#1a1a2e;">{{.Code}}</span>
  </div>
  <p style="color:#999;font-size:12px;margin:0;">If you didn't request this code, you can safely ignore this email.</p>
</div>
<p style="text-align:center;color:#aaa;font-size:11px;margin-top:16px;">Powered by BitCode Engine</p>
</body>
</html>`))

func RenderOTPEmail(code string, expiryMinutes int) (string, error) {
	var buf bytes.Buffer
	err := otpTemplate.Execute(&buf, map[string]any{
		"Code":          code,
		"ExpiryMinutes": expiryMinutes,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render OTP email: %w", err)
	}
	return buf.String(), nil
}
