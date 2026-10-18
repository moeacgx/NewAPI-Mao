package passkey

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPasskeyFlowRejectsSessionChanges(t *testing.T) {
	for _, purpose := range []string{model.AuthFlowPurposePasskeyRegister, model.AuthFlowPurposePasskeyStepUp} {
		t.Run(purpose, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.AuthFlow{}))
			oldDB, oldType := model.DB, common.MainDatabaseType()
			model.DB = db
			common.SetMainDatabaseType(common.DatabaseTypeSQLite)
			t.Cleanup(func() { model.DB = oldDB; common.SetMainDatabaseType(oldType); _ = sqlDB.Close() })
			user := model.User{Username: "passkey-flow", Status: common.UserStatusEnabled, AuthVersion: 1}
			require.NoError(t, db.Create(&user).Error)
			session := model.UserSession{SID: "passkey-session", UserID: user.Id, Version: 1, UserAuthVersion: 1, Status: model.UserSessionStatusActive, ExpiresAt: time.Now().Add(time.Hour).Unix()}
			require.NoError(t, db.Create(&session).Error)
			identity := model.AuthSessionIdentity{UserID: user.Id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1}
			token, _, err := CreateSessionDataFlow(purpose, identity, "passkey.register", &webauthn.SessionData{Challenge: "challenge"})
			require.NoError(t, err)
			require.NoError(t, db.Model(&session).Update("version", 2).Error)
			_, _, err = PopSessionDataFlow(token, purpose, identity)
			assert.Error(t, err, "旧 challenge 不得跨越会话版本推进")
		})
	}
}
