package passkey

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	webauthn "github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

var errSessionNotFound = errors.New("Passkey 会话不存在或已过期")

const passkeyFlowTTL = 5 * time.Minute

type flowPayload struct {
	SessionData webauthn.SessionData      `json:"session_data"`
	Scope       string                    `json:"scope,omitempty"`
	Identity    model.AuthSessionIdentity `json:"identity"`
}

func CreateSessionDataFlow(purpose string, identity model.AuthSessionIdentity, scope string, data *webauthn.SessionData) (string, int64, error) {
	if data == nil {
		return "", 0, errors.New("Passkey 会话数据不能为空")
	}
	if purpose != model.AuthFlowPurposePasskeyLogin && (identity.UserID <= 0 || identity.SessionID == "" || identity.UserAuthVersion <= 0 || identity.SessionVersion <= 0) {
		return "", 0, model.ErrAuthFlowInvalid
	}
	payload, err := common.Marshal(flowPayload{SessionData: *data, Scope: scope, Identity: identity})
	if err != nil {
		return "", 0, err
	}
	expiresAt := time.Now().Add(passkeyFlowTTL)
	token, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose:   purpose,
		UserId:    identity.UserID,
		SessionId: identity.SessionID,
		Payload:   string(payload),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", 0, err
	}
	return token, expiresAt.Unix(), nil
}

func PopSessionDataFlow(token, purpose string, identity model.AuthSessionIdentity) (*webauthn.SessionData, string, error) {
	var payload flowPayload
	_, err := model.ConsumeAuthFlowWithAction(token, model.AuthFlowMatch{
		Purpose:   purpose,
		UserId:    identity.UserID,
		SessionId: identity.SessionID,
	}, func(tx *gorm.DB, flow *model.AuthFlow) error {
		if err := common.UnmarshalJsonStr(flow.Payload, &payload); err != nil {
			return err
		}
		if payload.Identity != identity {
			return model.ErrAuthFlowInvalid
		}
		if purpose != model.AuthFlowPurposePasskeyLogin {
			return model.ValidateAuthSessionWithTx(tx, identity)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, model.ErrAuthFlowInvalid) || errors.Is(err, model.ErrAuthFlowExpired) || errors.Is(err, model.ErrAuthFlowConsumed) {
			return nil, "", errSessionNotFound
		}
		return nil, "", err
	}
	return &payload.SessionData, payload.Scope, nil
}
