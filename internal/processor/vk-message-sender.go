package processor

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "math/big"
    crand "crypto/rand"
    "net/http"
    "net/url"
    "strings"

    "vk-sender/internal/contract"
    "vk-sender/internal/logger"
)

type VkMessageSender struct {
    token      string
    apiBase    string
    httpClient *http.Client
    logger     logger.Logger
}

type Option struct {
    Token      string
    ApiBase    string
    HttpClient *http.Client
    Logger     logger.Logger
}

const (
    defaultVKAPIBase    = "https://api.vk.com/method"
    defaultVKAPIVersion = "5.199"
)

func NewVkMessageSender(opt Option) *VkMessageSender {
    apiBase := opt.ApiBase
    if apiBase == "" {
        apiBase = defaultVKAPIBase
    }
    httpClient := opt.HttpClient
    if httpClient == nil {
        httpClient = http.DefaultClient
    }
    return &VkMessageSender{
        token:      opt.Token,
        apiBase:    apiBase,
        httpClient: httpClient,
        logger:     opt.Logger,
    }
}

type vkAPIResponse struct {
    Response json.RawMessage `json:"response"`
    Error    *struct {
        ErrorCode int    `json:"error_code"`
        ErrorMsg  string `json:"error_msg"`
    } `json:"error"`
}

func (t *VkMessageSender) Handle(ctx context.Context, raw []byte) error {
    var requestMessage contract.SendMessageRequest
    if err := json.Unmarshal(raw, &requestMessage); err != nil {
        return err
    }

    endpoint := fmt.Sprintf("%s/messages.send", t.apiBase)

    form := url.Values{}
    form.Set("access_token", t.token)
    form.Set("v", defaultVKAPIVersion)
    form.Set("peer_id", fmt.Sprintf("%d", requestMessage.PeerID))
    form.Set("message", requestMessage.Message)
    form.Set("random_id", fmt.Sprintf("%d", randomID()))

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
    if err != nil {
        return fmt.Errorf("failed to build request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := t.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response: %w", err)
    }

    if resp.StatusCode >= 300 {
        return fmt.Errorf("VK API status=%s body=%s", resp.Status, string(body))
    }

    var apiResp vkAPIResponse
    if err := json.Unmarshal(body, &apiResp); err != nil {
        return fmt.Errorf("failed to parse VK response: %w body=%s", err, string(body))
    }
    if apiResp.Error != nil {
        return fmt.Errorf("VK API error: code=%d msg=%s", apiResp.Error.ErrorCode, apiResp.Error.ErrorMsg)
    }

    t.logger.Info("✅ Message sent to peer_id=%d", requestMessage.PeerID)
    return nil
}

func randomID() int64 {
    n, err := crand.Int(crand.Reader, big.NewInt(1<<31-1))
    if err != nil {
        return 1
    }
    return n.Int64()
}
