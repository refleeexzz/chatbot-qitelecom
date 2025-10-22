package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// mockService implements ChatbotService for tests
type mockService struct{}

func (m *mockService) ProcessMessage(from, text string) (string, error) {
	return "ok", nil
}

func makePayload() string {
	// minimal valid-ish payload
	return `{"entry":[{"changes":[{"value":{"messages":[{"from":"5511999999999","id":"1","text":{"body":"hello"}}]}}]}]}`
}

func computeSignature(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// mockSender implements Sender and records calls
type mockSender struct {
	called  bool
	to      string
	message string
}

func (m *mockSender) Send(to, message string) error {
	m.called = true
	m.to = to
	m.message = message
	return nil
}

func TestWhatsAppWebhook_ValidSignature(t *testing.T) {
	secret := "testsecret123"
	os.Setenv("WHATSAPP_APP_SECRET", secret)
	defer os.Unsetenv("WHATSAPP_APP_SECRET")

	body := makePayload()
	sig := computeSignature(secret, body)

	ms := &mockSender{}
	h := NewWhatsAppWebhookHandler(&mockService{}, ms)
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	rr := httptest.NewRecorder()

	h.HandleWhatsAppWebhook(rr, req)

	if rr.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d, body=%s", rr.Result().StatusCode, rr.Body.String())
	}
}

func TestWhatsAppWebhook_MissingSignature(t *testing.T) {
	secret := "testsecret123"
	os.Setenv("WHATSAPP_APP_SECRET", secret)
	defer os.Unsetenv("WHATSAPP_APP_SECRET")

	body := makePayload()

	ms := &mockSender{}
	h := NewWhatsAppWebhookHandler(&mockService{}, ms)
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.HandleWhatsAppWebhook(rr, req)

	if rr.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing signature, got %d", rr.Result().StatusCode)
	}
}

func TestWhatsAppWebhook_InvalidSignature(t *testing.T) {
	secret := "testsecret123"
	os.Setenv("WHATSAPP_APP_SECRET", secret)
	defer os.Unsetenv("WHATSAPP_APP_SECRET")

	body := makePayload()
	// wrong secret
	sig := computeSignature("wrongsecret", body)

	ms := &mockSender{}
	h := NewWhatsAppWebhookHandler(&mockService{}, ms)
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	rr := httptest.NewRecorder()

	h.HandleWhatsAppWebhook(rr, req)

	if rr.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid signature, got %d", rr.Result().StatusCode)
	}
}
