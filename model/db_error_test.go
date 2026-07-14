package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestIsUniqueViolation(t *testing.T) {
	assert.False(t, isUniqueViolation(nil))
	assert.True(t, isUniqueViolation(gorm.ErrDuplicatedKey))
}

func TestIsUniqueViolationOnSQLite(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&User{
		Id:       1,
		Username: "one",
		AffCode:  "shared_code",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, DB.Create(&User{
		Id:       2,
		Username: "two",
		AffCode:  "two2",
		Status:   common.UserStatusEnabled,
	}).Error)

	err := DB.Model(&User{}).Where("id = ?", 2).Update("aff_code", "shared_code").Error
	require.Error(t, err)
	assert.True(t, isUniqueViolation(err))
}
